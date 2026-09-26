package middleware

import (
	"context"

	"github.com/assethub/assethub/internal/constants"
	"github.com/assethub/assethub/internal/model"
	"github.com/assethub/assethub/internal/util"
	"github.com/gin-gonic/gin"
)

// CredentialQuotaConsumer increments and checks the credential's daily quota.
type CredentialQuotaConsumer interface {
	ConsumeQuota(ctx context.Context, credential *model.TeamCredential) (exceeded bool, err error)
	TouchLastUsed(ctx context.Context, credential *model.TeamCredential)
}

// CredentialQuota enforces the per-credential daily call limit. Exhaustion yields
// 429. It must run after SharedAuth (and, typically, after RequireScope so that
// out-of-scope calls do not consume quota).
func CredentialQuota(consumer CredentialQuotaConsumer) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := c.Get(ContextCredentialKey)
		if !ok {
			util.Fail(c, 401, constants.CodeUnauthorized, constants.MsgUnauthorized)
			return
		}
		credential, ok := raw.(*model.TeamCredential)
		if !ok {
			util.Fail(c, 401, constants.CodeUnauthorized, constants.MsgUnauthorized)
			return
		}
		exceeded, err := consumer.ConsumeQuota(c.Request.Context(), credential)
		if err != nil {
			util.Fail(c, httpStatusForCode(constants.CodeRateLimited), constants.CodeRateLimited, constants.MsgRateLimited)
			return
		}
		if exceeded {
			util.Fail(c, 429, constants.CodeRateLimited, constants.MsgCredentialQuotaUsed)
			return
		}
		consumer.TouchLastUsed(c.Request.Context(), credential)
		c.Next()
	}
}
