package router

import (
	"github.com/assethub/assethub/internal/constants"
	"github.com/assethub/assethub/internal/middleware"
	"github.com/gin-gonic/gin"
)

// registerShared wires the external shared interface. Requests authenticate with
// a team credential (X-Api-Key) instead of an employee JWT. Every call is
// scope-guarded, quota-limited and written to the credential access log.
func registerShared(api *gin.RouterGroup, h Handlers, credentialService *middleware.SharedCredentialServices) {
	shared := api.Group("/shared")
	shared.Use(middleware.CredentialAccessLog(credentialService))
	shared.Use(middleware.SharedAuth(credentialService))

	readAssets := shared.Group("/assets")
	readAssets.Use(middleware.RequireScope(constants.CredentialScopeAssetsRead))
	readAssets.Use(middleware.CredentialQuota(credentialService))
	readAssets.GET("", h.Shared.ListAssets)
	readAssets.GET("/:id", h.Shared.GetAsset)

	downloads := shared.Group("/assets")
	downloads.Use(middleware.RequireScope(constants.CredentialScopeDownloadsWrite))
	downloads.Use(middleware.CredentialQuota(credentialService))
	downloads.POST("/:id/download", h.Shared.RegisterDownload)

	collections := shared.Group("/collections")
	collections.Use(middleware.RequireScope(constants.CredentialScopeCollectionsWrite))
	collections.Use(middleware.CredentialQuota(credentialService))
	collections.POST("", h.Shared.CreateCollection)
	collections.GET("", h.Shared.ListCollections)
	collections.POST("/:id/assets", h.Shared.AddCollectionAsset)
	collections.DELETE("/:id/assets/:assetId", h.Shared.RemoveCollectionAsset)
}
