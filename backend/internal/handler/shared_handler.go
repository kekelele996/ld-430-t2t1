package handler

import (
	"strconv"

	"github.com/assethub/assethub/internal/dto"
	"github.com/assethub/assethub/internal/middleware"
	"github.com/assethub/assethub/internal/service"
	"github.com/assethub/assethub/internal/util"
	"github.com/gin-gonic/gin"
)

// SharedHandler exposes the external shared interface consumed by team
// credentials.
type SharedHandler struct {
	sharedService *service.SharedService
}

// NewSharedHandler creates a SharedHandler.
func NewSharedHandler(sharedService *service.SharedService) *SharedHandler {
	return &SharedHandler{sharedService: sharedService}
}

// ListAssets handles GET /shared/assets.
func (h *SharedHandler) ListAssets(c *gin.Context) {
	var q dto.AssetQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		respondValidation(c, err)
		return
	}
	page, _ := strconv.ParseInt(q.Page, 10, 64)
	pageSize, _ := strconv.ParseInt(q.PageSize, 10, 64)
	result, err := h.sharedService.ListAssets(c.Request.Context(), service.AssetQueryParams{
		Keyword:     q.Keyword,
		FileType:    q.FileType,
		LicenseType: q.LicenseType,
		Tag:         q.Tag,
		CategoryID:  q.CategoryID,
		Sort:        q.Sort,
		Page:        util.NormalizePage(page),
		PageSize:    util.NormalizePageSize(pageSize),
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.OK(c, result)
}

// GetAsset handles GET /shared/assets/:id.
func (h *SharedHandler) GetAsset(c *gin.Context) {
	asset, err := h.sharedService.GetAsset(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.OK(c, asset)
}

// CreateCollection handles POST /shared/collections.
func (h *SharedHandler) CreateCollection(c *gin.Context) {
	var req dto.CollectionRequest
	if err := middleware.ValidateJSON(c, &req); err != nil {
		_ = c.Error(err)
		return
	}
	actorID := mustObjectID(c.GetString(middleware.ContextActorIDKey))
	collection, err := h.sharedService.CreateCollection(c.Request.Context(), actorID, req.Name, req.Description, req.CoverImageURL, req.IsPublic)
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.Created(c, collection)
}

// ListCollections handles GET /shared/collections.
func (h *SharedHandler) ListCollections(c *gin.Context) {
	actorID := mustObjectID(c.GetString(middleware.ContextActorIDKey))
	collections, err := h.sharedService.ListCollections(c.Request.Context(), actorID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.OK(c, collections)
}

// AddCollectionAsset handles POST /shared/collections/:id/assets.
func (h *SharedHandler) AddCollectionAsset(c *gin.Context) {
	var req dto.AddAssetRequest
	if err := middleware.ValidateJSON(c, &req); err != nil {
		_ = c.Error(err)
		return
	}
	actorID := mustObjectID(c.GetString(middleware.ContextActorIDKey))
	if err := h.sharedService.AddCollectionAsset(c.Request.Context(), c.Param("id"), req.AssetID, actorID); err != nil {
		_ = c.Error(err)
		return
	}
	util.OK(c, gin.H{"added": true})
}

// RemoveCollectionAsset handles DELETE /shared/collections/:id/assets/:assetId.
func (h *SharedHandler) RemoveCollectionAsset(c *gin.Context) {
	actorID := mustObjectID(c.GetString(middleware.ContextActorIDKey))
	if err := h.sharedService.RemoveCollectionAsset(c.Request.Context(), c.Param("id"), c.Param("assetId"), actorID); err != nil {
		_ = c.Error(err)
		return
	}
	util.OK(c, gin.H{"removed": true})
}

// RegisterDownload handles POST /shared/assets/:id/download.
func (h *SharedHandler) RegisterDownload(c *gin.Context) {
	var req dto.DownloadRequest
	if err := middleware.ValidateJSON(c, &req); err != nil {
		_ = c.Error(err)
		return
	}
	actorID := mustObjectID(c.GetString(middleware.ContextActorIDKey))
	asset, err := h.sharedService.RegisterDownload(c.Request.Context(), c.Param("id"), req.Purpose, req.LicenseVersion, c.ClientIP(), actorID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.OK(c, gin.H{"download_url": asset.FileURL, "asset_id": asset.ID.Hex()})
}
