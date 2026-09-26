package middleware

import (
	"context"
	"strings"

	"github.com/assethub/assethub/internal/constants"
	apperrors "github.com/assethub/assethub/internal/errors"
	"github.com/assethub/assethub/internal/model"
	"github.com/assethub/assethub/internal/util"
	"github.com/gin-gonic/gin"
)

// Context keys populated for authenticated shared-interface requests.
const (
	ContextCredentialKey   = "team_credential"
	ContextCredentialID    = "credential_id"
	ContextCredentialName  = "credential_name"
	ContextCredentialKeyID = "credential_key_id"
	ContextActorIDKey      = "credential_actor_id"
)

// apiKeyHeader is the primary header carrying the team credential.
const apiKeyHeader = "X-Api-Key"

// CredentialAuthenticator authenticates presented API keys.
type CredentialAuthenticator interface {
	Authenticate(ctx context.Context, apiKey string) (*model.TeamCredential, error)
}

// SharedCredentialServices aggregates the credential capabilities the shared
// middleware chain depends on.
type SharedCredentialServices struct {
	Authenticator CredentialAuthenticator
	Quota         CredentialQuotaConsumer
	AccessLogger  CredentialAccessLogger
}

// Authenticate implements CredentialAuthenticator.
func (s *SharedCredentialServices) Authenticate(ctx context.Context, apiKey string) (*model.TeamCredential, error) {
	return s.Authenticator.Authenticate(ctx, apiKey)
}

// ConsumeQuota implements CredentialQuotaConsumer.
func (s *SharedCredentialServices) ConsumeQuota(ctx context.Context, credential *model.TeamCredential) (bool, error) {
	return s.Quota.ConsumeQuota(ctx, credential)
}

// TouchLastUsed implements CredentialQuotaConsumer.
func (s *SharedCredentialServices) TouchLastUsed(ctx context.Context, credential *model.TeamCredential) {
	s.Quota.TouchLastUsed(ctx, credential)
}

// LogAccess implements CredentialAccessLogger.
func (s *SharedCredentialServices) LogAccess(ctx context.Context, entry *model.CredentialAccessLog) {
	s.AccessLogger.LogAccess(ctx, entry)
}

// SharedAuth authenticates external requests through a team credential.
// Missing/invalid/expired/revoked credentials are rejected with 401.
func SharedAuth(authenticator CredentialAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := extractAPIKey(c)
		if apiKey == "" {
			util.Fail(c, 401, constants.CodeUnauthorized, constants.MsgCredentialInvalid)
			return
		}
		if keyID, _, ok := util.ParseAPIKey(apiKey); ok {
			c.Set(ContextCredentialKeyID, keyID)
		}
		credential, err := authenticator.Authenticate(c.Request.Context(), apiKey)
		if err != nil {
			code := constants.CodeUnauthorized
			message := constants.MsgCredentialInvalid
			var bizErr *apperrors.BusinessError
			if apperrors.AsBusinessError(err, &bizErr) {
				code = bizErr.Code
				message = bizErr.Message
			}
			util.Fail(c, 401, code, message)
			return
		}
		c.Set(ContextCredentialKey, credential)
		c.Set(ContextCredentialID, credential.ID.Hex())
		c.Set(ContextCredentialName, credential.Name)
		c.Set(ContextActorIDKey, credential.CreatedBy.Hex())
		c.Next()
	}
}

func extractAPIKey(c *gin.Context) string {
	if key := strings.TrimSpace(c.GetHeader(apiKeyHeader)); key != "" {
		return key
	}
	header := c.GetHeader("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	}
	return ""
}
