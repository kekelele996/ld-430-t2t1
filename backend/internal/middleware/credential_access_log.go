package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/assethub/assethub/internal/model"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CredentialAccessLogger writes access logs for credential calls.
type CredentialAccessLogger interface {
	LogAccess(ctx context.Context, entry *model.CredentialAccessLog)
}

// CredentialAccessLog records the credential, path, result and timestamp of every
// shared-interface call, including authentication and quota failures.
func CredentialAccessLog(logger CredentialAccessLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		entry := &model.CredentialAccessLog{
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			StatusCode: c.Writer.Status(),
			Result:     accessResult(c.Writer.Status()),
			IP:         c.ClientIP(),
			CreatedAt:  time.Now(),
		}
		if keyID, exists := c.Get(ContextCredentialKeyID); exists {
			entry.KeyID = keyID.(string)
		}
		if idHex, exists := c.Get(ContextCredentialID); exists {
			if oid, err := primitive.ObjectIDFromHex(idHex.(string)); err == nil {
				entry.CredentialID = oid
			}
		}
		logger.LogAccess(c.Request.Context(), entry)
	}
}

func accessResult(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "success"
	case status == http.StatusForbidden:
		return "forbidden"
	case status == http.StatusUnauthorized:
		return "unauthorized"
	case status == http.StatusTooManyRequests:
		return "rate_limited"
	case status >= 400:
		return "client_error"
	default:
		return "error"
	}
}
