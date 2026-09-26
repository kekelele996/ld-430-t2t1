package router

import (
	"log/slog"

	"github.com/assethub/assethub/internal/middleware"
	"github.com/assethub/assethub/internal/service"
	"github.com/assethub/assethub/internal/util"
	"github.com/gin-gonic/gin"
)

// registerShared mounts both the admin management routes (JWT + Admin) and the
// external shared routes (team shared credential, scope + daily quota enforcement).
func registerShared(api *gin.RouterGroup, h Handlers, jwtManager *util.JWTManager, sharedService *service.SharedCredentialService, logger *slog.Logger) {
	registerSharedAdmin(api, h, jwtManager)
	registerSharedExternal(api, h, sharedService, logger)
}

func registerSharedAdmin(api *gin.RouterGroup, h Handlers, jwtManager *util.JWTManager) {
	admin := api.Group("/shared-credentials")
	admin.Use(middleware.Auth(jwtManager), middleware.Roles("Admin"))
	admin.POST("", h.SharedCredential.Create)
	admin.GET("", h.SharedCredential.List)
	admin.DELETE("/:id", h.SharedCredential.Revoke)
	admin.GET("/:id/call-logs", h.SharedCredential.CallLogs)
}

func registerSharedExternal(api *gin.RouterGroup, h Handlers, sharedService *service.SharedCredentialService, logger *slog.Logger) {
	shared := api.Group("/shared")
	// Call logger is outermost so expired/revoked denials are recorded too;
	// auth then scope then quota, so out-of-scope calls do not consume quota.
	shared.Use(
		middleware.SharedCallLogger(sharedService, logger),
		middleware.SharedAuth(sharedService),
	)

	readAssets := shared.Group("/assets")
	readAssets.Use(
		middleware.RequireSharedScope("assets:read"),
		middleware.SharedDailyQuota(sharedService),
	)
	readAssets.GET("", h.Shared.ListAssets)
	readAssets.GET("/hot", h.Shared.HotAssets)
	readAssets.GET("/:id", h.Shared.GetAsset)

	downloads := shared.Group("/assets")
	downloads.Use(
		middleware.RequireSharedScope("downloads:write"),
		middleware.SharedDailyQuota(sharedService),
	)
	downloads.POST("/:id/download", h.Shared.Download)

	collections := shared.Group("/collections")
	collections.Use(
		middleware.RequireSharedScope("collections:write"),
		middleware.SharedDailyQuota(sharedService),
	)
	collections.POST("", h.Collection.Create)
	collections.GET("", h.Collection.List)
	collections.GET("/:id", h.Collection.Get)
	collections.PUT("/:id", h.Collection.Update)
	collections.POST("/:id/assets", h.Collection.AddAsset)
	collections.DELETE("/:id/assets/:assetId", h.Collection.RemoveAsset)
}
