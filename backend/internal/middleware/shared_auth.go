package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/assethub/assethub/internal/constants"
	apperrors "github.com/assethub/assethub/internal/errors"
	"github.com/assethub/assethub/internal/model"
	"github.com/assethub/assethub/internal/util"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Context keys attached by shared credential authentication.
const (
	ContextSharedCredentialKey = "shared_credential"
	ContextSharedKeyHeader     = "X-Api-Key"
)

// SharedAuthenticator resolves a plaintext key to a credential, enforcing expiry/revocation.
type SharedAuthenticator interface {
	Authenticate(ctx context.Context, plainKey string) (*model.SharedCredential, error)
}

// SharedQuotaConsumer enforces the per-credential daily call quota.
type SharedQuotaConsumer interface {
	ConsumeDailyQuota(ctx context.Context, credentialID primitive.ObjectID, dailyLimit int64) error
}

// SharedCallRecorder persists shared credential call logs.
type SharedCallRecorder interface {
	RecordCall(ctx context.Context, entry *model.SharedCallLog) error
}

// SharedAuth validates the shared credential presented in X-Api-Key (or Bearer).
// Expired or revoked credentials fail with 401; the credential is still placed in
// the context so the call logger can record the denial.
func SharedAuth(authenticator SharedAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		credential, err := authenticator.Authenticate(c.Request.Context(), extractSharedKey(c))
		if credential != nil {
			c.Set(ContextSharedCredentialKey, credential)
			// Downstream handlers act on behalf of the team admin that issued the credential.
			c.Set(ContextUserIDKey, credential.CreatedBy.Hex())
		}
		if err != nil {
			respondBusiness(c, err)
			return
		}
		c.Next()
	}
}

// RequireSharedScope rejects calls whose credential lacks the required scope with 403.
func RequireSharedScope(required constants.SharedScope) gin.HandlerFunc {
	return func(c *gin.Context) {
		credential, ok := sharedCredentialFromContext(c)
		if !ok {
			util.Fail(c, http.StatusUnauthorized, constants.CodeUnauthorized, constants.MsgSharedCredentialInvalid)
			return
		}
		for _, scope := range credential.Scopes {
			if scope == string(required) {
				c.Next()
				return
			}
		}
		util.Fail(c, http.StatusForbidden, constants.CodeForbidden, constants.MsgSharedScopeForbidden)
	}
}

// SharedDailyQuota consumes one daily call and rejects once the limit is exhausted (429).
func SharedDailyQuota(consumer SharedQuotaConsumer) gin.HandlerFunc {
	return func(c *gin.Context) {
		credential, ok := sharedCredentialFromContext(c)
		if !ok {
			util.Fail(c, http.StatusUnauthorized, constants.CodeUnauthorized, constants.MsgSharedCredentialInvalid)
			return
		}
		if err := consumer.ConsumeDailyQuota(c.Request.Context(), credential.ID, credential.DailyLimit); err != nil {
			respondBusiness(c, err)
			return
		}
		c.Next()
	}
}

// SharedCallLogger records every call attributable to a presented credential, including
// denials (401 expired/revoked, 403 scope, 429 quota). Unknown keys are not logged.
func SharedCallLogger(recorder SharedCallRecorder, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		credential, ok := sharedCredentialFromContext(c)
		if !ok {
			return
		}
		status := resolveSharedStatus(c)
		entry := &model.SharedCallLog{
			CredentialID: credential.ID,
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			StatusCode:   status,
			Result:       sharedResultForStatus(status),
			IP:           c.ClientIP(),
		}
		if err := recorder.RecordCall(c.Request.Context(), entry); err != nil {
			logger.Warn("write shared call log failed", "error", err.Error())
		}
	}
}

func sharedCredentialFromContext(c *gin.Context) (*model.SharedCredential, bool) {
	raw, ok := c.Get(ContextSharedCredentialKey)
	if !ok {
		return nil, false
	}
	credential, ok := raw.(*model.SharedCredential)
	return credential, ok
}

// extractSharedKey reads X-Api-Key first, falling back to an Authorization bearer key.
func extractSharedKey(c *gin.Context) string {
	if key := strings.TrimSpace(c.GetHeader(ContextSharedKeyHeader)); key != "" {
		return key
	}
	header := c.GetHeader("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	}
	return ""
}

func respondBusiness(c *gin.Context, err error) {
	var bizErr *apperrors.BusinessError
	if errors.As(err, &bizErr) {
		util.Fail(c, httpStatusForCode(bizErr.Code), bizErr.Code, bizErr.Message)
		return
	}
	util.Fail(c, http.StatusInternalServerError, constants.CodeInternal, constants.MsgInternalError)
}

// resolveSharedStatus mirrors the error handler because, for handler errors, the final
// status is written by the outer ErrorHandler only after this deferred logic runs.
func resolveSharedStatus(c *gin.Context) int {
	if len(c.Errors) > 0 {
		err := c.Errors.Last().Err
		var bizErr *apperrors.BusinessError
		var valErr *apperrors.ValidationError
		switch {
		case errors.As(err, &bizErr):
			return httpStatusForCode(bizErr.Code)
		case errors.As(err, &valErr):
			return http.StatusUnprocessableEntity
		default:
			return http.StatusInternalServerError
		}
	}
	if c.Writer.Status() != 0 {
		return c.Writer.Status()
	}
	return http.StatusOK
}

func sharedResultForStatus(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "success"
	case status == http.StatusUnauthorized:
		return "unauthorized"
	case status == http.StatusForbidden:
		return "forbidden"
	case status == http.StatusTooManyRequests:
		return "quota_exhausted"
	case status >= 400 && status < 500:
		return "client_error"
	default:
		return "server_error"
	}
}
