package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/assethub/assethub/internal/client"
	"github.com/assethub/assethub/internal/constants"
	"github.com/assethub/assethub/internal/errors"
	"github.com/assethub/assethub/internal/model"
	"github.com/assethub/assethub/internal/repository"
	"github.com/assethub/assethub/internal/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// quotaWindow keeps a daily counter key alive slightly past the day boundary
// so concurrent late requests cannot race on a missing/reset key.
const quotaWindow = 48 * time.Hour

// sharedCredentialStore is the persistence surface used by the service.
type sharedCredentialStore interface {
	Create(ctx context.Context, credential *model.SharedCredential) error
	FindByHash(ctx context.Context, keyHash string) (*model.SharedCredential, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*model.SharedCredential, error)
	List(ctx context.Context) ([]model.SharedCredential, error)
	Revoke(ctx context.Context, id primitive.ObjectID) error
}

// sharedCallLogStore is the call-log persistence surface used by the service.
type sharedCallLogStore interface {
	Create(ctx context.Context, logEntry *model.SharedCallLog) error
	ListByCredential(ctx context.Context, credentialID primitive.ObjectID, skip, limit int64) ([]model.SharedCallLog, error)
}

// dailyQuotaCounter is the atomic counter surface backed by Redis in production.
type dailyQuotaCounter interface {
	IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

// SharedCredentialService manages team shared credentials, daily quotas and call logs.
type SharedCredentialService struct {
	credRepo sharedCredentialStore
	logRepo  sharedCallLogStore
	redis    dailyQuotaCounter
	logger   *slog.Logger
}

// NewSharedCredentialService creates a SharedCredentialService.
func NewSharedCredentialService(credRepo *repository.SharedCredentialRepository, logRepo *repository.SharedCallLogRepository, redis *client.RedisClient, logger *slog.Logger) *SharedCredentialService {
	return &SharedCredentialService{credRepo: credRepo, logRepo: logRepo, redis: redis, logger: logger}
}

// Create issues a new shared credential and returns the persisted document plus the
// plaintext key, which is never stored and can only be returned to the caller here.
func (s *SharedCredentialService) Create(ctx context.Context, name string, scopes []string, dailyLimit int64, expiresAt *time.Time, createdBy primitive.ObjectID) (*model.SharedCredential, string, error) {
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return nil, "", errors.NewBusinessError(constants.CodeBadRequest, "expires_at must be in the future")
	}
	normalized := make([]string, 0, len(scopes))
	seen := make(map[string]bool, len(scopes))
	for _, scope := range scopes {
		if !constants.ValidSharedScopes[constants.SharedScope(scope)] {
			return nil, "", errors.NewBusinessError(constants.CodeValidationFailed, "invalid scope: "+scope)
		}
		if !seen[scope] {
			seen[scope] = true
			normalized = append(normalized, scope)
		}
	}

	plain, err := util.GenerateSharedKey()
	if err != nil {
		return nil, "", err
	}
	credential := &model.SharedCredential{
		Name:       name,
		KeyHash:    util.HashSharedKey(plain),
		KeyLast4:   util.SharedKeyLast4(plain),
		Scopes:     normalized,
		DailyLimit: dailyLimit,
		ExpiresAt:  expiresAt,
		CreatedBy:  createdBy,
	}
	if err := s.credRepo.Create(ctx, credential); err != nil {
		return nil, "", err
	}
	s.logger.Info(constants.LogSharedCredCreated, "credential_id", credential.ID.Hex(), "name", name, "scopes", normalized)
	return credential, plain, nil
}

// List returns every issued shared credential.
func (s *SharedCredentialService) List(ctx context.Context) ([]model.SharedCredential, error) {
	return s.credRepo.List(ctx)
}

// Revoke immediately invalidates a shared credential.
func (s *SharedCredentialService) Revoke(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.NewBusinessError(constants.CodeBadRequest, "invalid credential id")
	}
	if err := s.credRepo.Revoke(ctx, oid); err != nil {
		return err
	}
	s.logger.Info(constants.LogSharedCredRevoked, "credential_id", oid.Hex())
	return nil
}

// Authenticate resolves a presented key and enforces revocation/expiry.
// On success the credential is returned; otherwise a 401 business error is returned.
func (s *SharedCredentialService) Authenticate(ctx context.Context, plainKey string) (*model.SharedCredential, error) {
	if plainKey == "" {
		return nil, errors.NewBusinessError(constants.CodeUnauthorized, constants.MsgSharedCredentialInvalid)
	}
	credential, err := s.credRepo.FindByHash(ctx, util.HashSharedKey(plainKey))
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, errors.NewBusinessError(constants.CodeUnauthorized, constants.MsgSharedCredentialInvalid)
		}
		return nil, err
	}
	if credential.Revoked {
		return credential, errors.NewBusinessError(constants.CodeUnauthorized, constants.MsgSharedCredentialRevoked)
	}
	if credential.ExpiresAt != nil && time.Now().After(*credential.ExpiresAt) {
		return credential, errors.NewBusinessError(constants.CodeUnauthorized, constants.MsgSharedCredentialExpired)
	}
	return credential, nil
}

// ConsumeDailyQuota increments the credential's UTC-day call counter and rejects the
// call once the configured daily limit is reached.
func (s *SharedCredentialService) ConsumeDailyQuota(ctx context.Context, credentialID primitive.ObjectID, dailyLimit int64) error {
	key := fmt.Sprintf("sharedquota:%s:%s", credentialID.Hex(), time.Now().UTC().Format("20060102"))
	count, err := s.redis.IncrWithTTL(ctx, key, quotaWindow)
	if err != nil {
		s.logger.Error("shared quota counter failed", "error", err.Error(), "credential_id", credentialID.Hex())
		return errors.NewBusinessError(constants.CodeRateLimited, constants.MsgRateLimited)
	}
	if count > dailyLimit {
		return errors.NewBusinessError(constants.CodeRateLimited, constants.MsgSharedQuotaExhausted)
	}
	return nil
}

// RecordCall persists one shared credential call log entry.
func (s *SharedCredentialService) RecordCall(ctx context.Context, entry *model.SharedCallLog) error {
	return s.logRepo.Create(ctx, entry)
}

// ListCallLogs returns paginated call logs for a credential.
func (s *SharedCredentialService) ListCallLogs(ctx context.Context, credentialID string, skip, limit int64) ([]model.SharedCallLog, error) {
	oid, err := primitive.ObjectIDFromHex(credentialID)
	if err != nil {
		return nil, errors.NewBusinessError(constants.CodeBadRequest, "invalid credential id")
	}
	if _, err := s.credRepo.FindByID(ctx, oid); err != nil {
		return nil, err
	}
	return s.logRepo.ListByCredential(ctx, oid, skip, limit)
}
