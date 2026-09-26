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
	"github.com/assethub/assethub/internal/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TeamCredentialService manages team credential lifecycle and authenticates
// shared-interface requests.
type TeamCredentialService struct {
	repo    *repository.TeamCredentialRepository
	logRepo *repository.CredentialAccessLogRepository
	redis   DailyQuotaCounter
	logger  *slog.Logger
}

// DailyQuotaCounter counts a credential's calls within the current UTC day.
type DailyQuotaCounter interface {
	IncrDailyCredentialUsage(ctx context.Context, credentialID string, limit int) (count int64, exceeded bool, err error)
}

// NewTeamCredentialService creates a TeamCredentialService.
func NewTeamCredentialService(
	repo *repository.TeamCredentialRepository,
	logRepo *repository.CredentialAccessLogRepository,
	redis DailyQuotaCounter,
	logger *slog.Logger,
) *TeamCredentialService {
	return &TeamCredentialService{repo: repo, logRepo: logRepo, redis: redis, logger: logger}
}

// Create issues a new team credential and returns the persisted credential
// together with the one-time plaintext key.
func (s *TeamCredentialService) Create(ctx context.Context, name string, scopes []string, dailyLimit int, expiresAt *time.Time, createdBy primitive.ObjectID) (*model.TeamCredential, string, error) {
	for _, scope := range scopes {
		if !constants.ValidCredentialScopes[constants.CredentialScope(scope)] {
			return nil, "", errors.NewBusinessError(constants.CodeValidationFailed, "invalid scope: "+scope)
		}
	}
	key, err := util.GenerateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("generate api key: %w", err)
	}
	credential := &model.TeamCredential{
		Name:       name,
		KeyID:      key.KeyID,
		SecretHash: util.HashAPIKeySecret(key.Secret),
		LastFour:   key.LastFour,
		Scopes:     scopes,
		DailyLimit: dailyLimit,
		Status:     string(constants.CredentialStatusActive),
		ExpiresAt:  expiresAt,
		CreatedBy:  createdBy,
	}
	if err := s.repo.Create(ctx, credential); err != nil {
		return nil, "", err
	}
	s.logger.Info("team credential created", "credential_id", credential.ID.Hex(), "name", name, "by", createdBy.Hex())
	return credential, key.Full, nil
}

// List returns all team credentials.
func (s *TeamCredentialService) List(ctx context.Context, skip, limit int64) ([]model.TeamCredential, error) {
	return s.repo.List(ctx, skip, limit)
}

// Get returns a single team credential.
func (s *TeamCredentialService) Get(ctx context.Context, id string) (*model.TeamCredential, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.NewBusinessError(constants.CodeBadRequest, "invalid credential id")
	}
	credential, err := s.repo.FindByID(ctx, oid)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewBusinessError(constants.CodeNotFound, constants.MsgNotFound)
		}
		return nil, fmt.Errorf("get credential: %w", err)
	}
	return credential, nil
}

// Revoke immediately disables a credential so subsequent requests are rejected.
func (s *TeamCredentialService) Revoke(ctx context.Context, id string, revokedBy primitive.ObjectID) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.NewBusinessError(constants.CodeBadRequest, "invalid credential id")
	}
	if err := s.repo.Revoke(ctx, oid, revokedBy, time.Now()); err != nil {
		if errors.IsNotFound(err) {
			return errors.NewBusinessError(constants.CodeNotFound, "credential not found or already revoked")
		}
		return fmt.Errorf("revoke credential: %w", err)
	}
	s.logger.Info("team credential revoked", "credential_id", id, "by", revokedBy.Hex())
	return nil
}

// AccessLogs returns the access log for a credential.
func (s *TeamCredentialService) AccessLogs(ctx context.Context, id string, skip, limit int64) ([]model.CredentialAccessLog, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.NewBusinessError(constants.CodeBadRequest, "invalid credential id")
	}
	return s.logRepo.ListByCredential(ctx, oid, skip, limit)
}

// Authenticate validates a presented key and returns the active credential.
// It returns a BusinessError with CodeUnauthorized for missing/unknown/expired/
// revoked credentials.
func (s *TeamCredentialService) Authenticate(ctx context.Context, apiKey string) (*model.TeamCredential, error) {
	keyID, secret, ok := util.ParseAPIKey(apiKey)
	if !ok {
		return nil, errors.NewBusinessError(constants.CodeUnauthorized, constants.MsgCredentialInvalid)
	}
	credential, err := s.repo.FindByKeyID(ctx, keyID)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewBusinessError(constants.CodeUnauthorized, constants.MsgCredentialInvalid)
		}
		return nil, fmt.Errorf("authenticate credential: %w", err)
	}
	if !util.VerifyAPIKeySecret(secret, credential.SecretHash) {
		return nil, errors.NewBusinessError(constants.CodeUnauthorized, constants.MsgCredentialInvalid)
	}
	if credential.Status == string(constants.CredentialStatusRevoked) {
		return nil, errors.NewBusinessError(constants.CodeUnauthorized, constants.MsgCredentialRevoked)
	}
	if credential.ExpiresAt != nil && time.Now().After(*credential.ExpiresAt) {
		return nil, errors.NewBusinessError(constants.CodeUnauthorized, constants.MsgCredentialExpired)
	}
	return credential, nil
}

// ConsumeQuota increments the credential's daily call counter and reports whether
// the configured daily limit has been exceeded.
func (s *TeamCredentialService) ConsumeQuota(ctx context.Context, credential *model.TeamCredential) (bool, error) {
	count, exceeded, err := s.redis.IncrDailyCredentialUsage(ctx, credential.ID.Hex(), credential.DailyLimit)
	if err != nil {
		return false, fmt.Errorf("consume credential quota: %w", err)
	}
	if exceeded {
		s.logger.Warn("team credential daily quota exceeded", "credential_id", credential.ID.Hex(), "count", count, "limit", credential.DailyLimit)
	}
	return exceeded, nil
}

// TouchLastUsed updates the last usage timestamp; failures are non-fatal.
func (s *TeamCredentialService) TouchLastUsed(ctx context.Context, credential *model.TeamCredential) {
	if err := s.repo.TouchLastUsed(ctx, credential.ID, time.Now()); err != nil {
		s.logger.Warn("touch credential last used failed", "error", err.Error())
	}
}

// LogAccess records the result of a shared-interface call.
func (s *TeamCredentialService) LogAccess(ctx context.Context, entry *model.CredentialAccessLog) {
	if err := s.logRepo.Create(ctx, entry); err != nil {
		s.logger.Warn("write credential access log failed", "error", err.Error())
	}
}

// HasScope reports whether the credential grants the given scope.
func HasScope(credential *model.TeamCredential, scope constants.CredentialScope) bool {
	for _, granted := range credential.Scopes {
		if granted == string(scope) {
			return true
		}
	}
	return false
}
