package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/assethub/assethub/internal/constants"
	"github.com/assethub/assethub/internal/errors"
	"github.com/assethub/assethub/internal/model"
	"github.com/assethub/assethub/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SharedService implements the external shared interface used by team
// credentials. All actions are performed on behalf of the credential creator and
// are restricted to published assets.
type SharedService struct {
	assetRepo    *repository.AssetRepository
	downloadRepo *repository.DownloadRecordRepository
	search       *SearchService
	collections  *CollectionService
	logger       *slog.Logger
}

// NewSharedService creates a SharedService.
func NewSharedService(
	assetRepo *repository.AssetRepository,
	downloadRepo *repository.DownloadRecordRepository,
	search *SearchService,
	collections *CollectionService,
	logger *slog.Logger,
) *SharedService {
	return &SharedService{
		assetRepo:    assetRepo,
		downloadRepo: downloadRepo,
		search:       search,
		collections:  collections,
		logger:       logger,
	}
}

// ListAssets returns published assets matching the query (external callers may
// never see drafts or archived assets).
func (s *SharedService) ListAssets(ctx context.Context, q AssetQueryParams) (*AssetListResult, error) {
	q.Status = string(constants.AssetStatusPublished)
	return s.searchResult(ctx, q)
}

func (s *SharedService) searchResult(ctx context.Context, q AssetQueryParams) (*AssetListResult, error) {
	assets, total, err := s.search.Search(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("shared list assets: %w", err)
	}
	return &AssetListResult{Items: assets, Total: total}, nil
}

// GetAsset returns a published asset by id.
func (s *SharedService) GetAsset(ctx context.Context, id string) (*model.Asset, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.NewBusinessError(constants.CodeBadRequest, "invalid asset id")
	}
	asset, err := s.assetRepo.FindByID(ctx, oid)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewBusinessError(constants.CodeNotFound, constants.MsgNotFound)
		}
		return nil, fmt.Errorf("shared get asset: %w", err)
	}
	if asset.Status != string(constants.AssetStatusPublished) {
		return nil, errors.NewBusinessError(constants.CodeNotFound, constants.MsgNotFound)
	}
	return asset, nil
}

// CreateCollection creates a favorites collection owned by the credential creator.
func (s *SharedService) CreateCollection(ctx context.Context, ownerID primitive.ObjectID, name, description, coverImageURL string, isPublic bool) (*model.Collection, error) {
	return s.collections.Create(ctx, ownerID, name, description, coverImageURL, isPublic)
}

// ListCollections returns collections visible to the credential creator.
func (s *SharedService) ListCollections(ctx context.Context, ownerID primitive.ObjectID) ([]model.Collection, error) {
	return s.collections.List(ctx, ownerID, false)
}

// AddCollectionAsset adds an asset to a collection the credential creator maintains.
func (s *SharedService) AddCollectionAsset(ctx context.Context, collectionID, assetID string, ownerID primitive.ObjectID) error {
	return s.collections.AddAsset(ctx, collectionID, assetID, ownerID)
}

// RemoveCollectionAsset removes an asset from a collection the credential creator maintains.
func (s *SharedService) RemoveCollectionAsset(ctx context.Context, collectionID, assetID string, ownerID primitive.ObjectID) error {
	return s.collections.RemoveAsset(ctx, collectionID, assetID, ownerID)
}

// RegisterDownload validates the published asset and records a download on behalf
// of the credential creator.
func (s *SharedService) RegisterDownload(ctx context.Context, assetID, purpose, licenseVersion, ip string, actorID primitive.ObjectID) (*model.Asset, error) {
	oid, err := primitive.ObjectIDFromHex(assetID)
	if err != nil {
		return nil, errors.NewBusinessError(constants.CodeBadRequest, "invalid asset id")
	}
	asset, err := s.assetRepo.FindByID(ctx, oid)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewBusinessError(constants.CodeNotFound, constants.MsgNotFound)
		}
		return nil, fmt.Errorf("shared register download: %w", err)
	}
	if asset.Status != string(constants.AssetStatusPublished) {
		// Do not leak the existence of non-published assets to external callers.
		return nil, errors.NewBusinessError(constants.CodeNotFound, constants.MsgNotFound)
	}
	record := &model.DownloadRecord{
		AssetID:        oid,
		DownloaderID:   actorID,
		DownloadedAt:   time.Now(),
		Purpose:        purpose,
		LicenseVersion: licenseVersion,
		IP:             ip,
	}
	if err := s.downloadRepo.Create(ctx, record); err != nil {
		return nil, err
	}
	if err := s.assetRepo.IncDownloadCount(ctx, oid); err != nil {
		s.logger.Warn("increment download count failed", "error", err.Error())
	}
	s.logger.Info(constants.LogDownloadAsset, "asset_id", oid.Hex(), "credential_actor", actorID.Hex())
	return asset, nil
}
