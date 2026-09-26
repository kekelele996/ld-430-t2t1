package handler

import (
	"strconv"

	"github.com/assethub/assethub/internal/constants"
	"github.com/assethub/assethub/internal/dto"
	"github.com/assethub/assethub/internal/errors"
	"github.com/assethub/assethub/internal/middleware"
	"github.com/assethub/assethub/internal/service"
	"github.com/assethub/assethub/internal/util"
	"github.com/gin-gonic/gin"
)

// SharedHandler exposes the external, credential-authenticated subset of the API.
type SharedHandler struct {
	assetService    *service.AssetService
	downloadService *service.DownloadService
}

// NewSharedHandler creates a SharedHandler.
func NewSharedHandler(assetService *service.AssetService, downloadService *service.DownloadService) *SharedHandler {
	return &SharedHandler{assetService: assetService, downloadService: downloadService}
}

// ListAssets handles GET /shared/assets. Only published assets are ever returned.
func (h *SharedHandler) ListAssets(c *gin.Context) {
	var q dto.SharedAssetQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		_ = c.Error(errors.NewValidationError(err))
		return
	}
	page, _ := strconv.ParseInt(q.Page, 10, 64)
	pageSize, _ := strconv.ParseInt(q.PageSize, 10, 64)
	page = util.NormalizePage(page)
	pageSize = util.NormalizePageSize(pageSize)
	result, err := h.assetService.List(c.Request.Context(), service.AssetQueryParams{
		Keyword:     q.Keyword,
		FileType:    q.FileType,
		LicenseType: q.LicenseType,
		Status:      string(constants.AssetStatusPublished),
		Tag:         q.Tag,
		CategoryID:  q.CategoryID,
		Sort:        q.Sort,
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.OK(c, result)
}

// GetAsset handles GET /shared/assets/:id.
func (h *SharedHandler) GetAsset(c *gin.Context) {
	asset, err := h.assetService.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	if asset.Status != string(constants.AssetStatusPublished) {
		_ = c.Error(errors.NewBusinessError(constants.CodeNotFound, constants.MsgNotFound))
		return
	}
	util.OK(c, asset)
}

// HotAssets handles GET /shared/assets/hot.
func (h *SharedHandler) HotAssets(c *gin.Context) {
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	assets, err := h.assetService.Hot(c.Request.Context(), limit)
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.OK(c, assets)
}

// Download handles POST /shared/assets/:id/download.
func (h *SharedHandler) Download(c *gin.Context) {
	var req dto.SharedDownloadRequest
	if err := middleware.ValidateJSON(c, &req); err != nil {
		_ = c.Error(err)
		return
	}
	// Calls are made on behalf of the admin that owns the credential, which bypasses
	// the Viewer-only commercial license restriction.
	teamID := mustObjectID(c.GetString(middleware.ContextUserIDKey))
	asset, err := h.downloadService.Download(c.Request.Context(), c.Param("id"), teamID, req.Purpose, req.LicenseVersion, c.ClientIP(), string(constants.RoleAdmin))
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.OK(c, gin.H{"download_url": asset.FileURL, "asset_id": asset.ID.Hex()})
}
