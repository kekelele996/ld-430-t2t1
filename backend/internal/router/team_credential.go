package router

import (
	"github.com/assethub/assethub/internal/constants"
	"github.com/assethub/assethub/internal/middleware"
	"github.com/assethub/assethub/internal/util"
	"github.com/gin-gonic/gin"
)

func registerTeamCredential(api *gin.RouterGroup, h Handlers, jwtManager *util.JWTManager) {
	credentials := api.Group("/team-credentials")
	credentials.Use(middleware.Auth(jwtManager))
	credentials.Use(middleware.Roles(constants.RoleAdmin))
	credentials.POST("", h.TeamCredential.Create)
	credentials.GET("", h.TeamCredential.List)
	credentials.GET("/:id", h.TeamCredential.Get)
	credentials.POST("/:id/revoke", h.TeamCredential.Revoke)
	credentials.GET("/:id/access-logs", h.TeamCredential.AccessLogs)
}
