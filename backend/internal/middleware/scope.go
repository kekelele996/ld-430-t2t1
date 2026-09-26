package middleware

import (
	"github.com/assethub/assethub/internal/constants"
	"github.com/assethub/assethub/internal/model"
	"github.com/assethub/assethub/internal/util"
	"github.com/gin-gonic/gin"
)

// RequireScope rejects shared requests whose credential lacks the given scope.
// It must run after SharedAuth.
func RequireScope(scope constants.CredentialScope) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := c.Get(ContextCredentialKey)
		if !ok {
			util.Fail(c, 401, constants.CodeUnauthorized, constants.MsgUnauthorized)
			return
		}
		credential, ok := raw.(*model.TeamCredential)
		if !ok || !hasCredentialScope(credential, scope) {
			util.Fail(c, 403, constants.CodeForbidden, constants.MsgForbidden)
			return
		}
		c.Next()
	}
}

func hasCredentialScope(credential *model.TeamCredential, scope constants.CredentialScope) bool {
	for _, granted := range credential.Scopes {
		if granted == string(scope) {
			return true
		}
	}
	return false
}
