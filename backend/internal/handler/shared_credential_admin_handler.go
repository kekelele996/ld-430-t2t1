package handler

import (
	"strconv"

	"github.com/assethub/assethub/internal/dto"
	"github.com/assethub/assethub/internal/middleware"
	"github.com/assethub/assethub/internal/service"
	"github.com/assethub/assethub/internal/util"
	"github.com/gin-gonic/gin"
)

// SharedCredentialAdminHandler exposes admin-only shared credential management.
type SharedCredentialAdminHandler struct {
	sharedService *service.SharedCredentialService
}

// NewSharedCredentialAdminHandler creates a SharedCredentialAdminHandler.
func NewSharedCredentialAdminHandler(sharedService *service.SharedCredentialService) *SharedCredentialAdminHandler {
	return &SharedCredentialAdminHandler{sharedService: sharedService}
}

// Create handles POST /shared-credentials.
func (h *SharedCredentialAdminHandler) Create(c *gin.Context) {
	var req dto.CreateSharedCredentialRequest
	if err := middleware.ValidateJSON(c, &req); err != nil {
		_ = c.Error(err)
		return
	}
	adminID := mustObjectID(c.GetString(middleware.ContextUserIDKey))
	credential, plainKey, err := h.sharedService.Create(c.Request.Context(), req.Name, req.Scopes, req.DailyLimit, req.ExpiresAt, adminID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	middleware.SetAuditAction(c, "shared_credential.create", credential.Name)
	util.Created(c, dto.SharedCredentialCreated{
		ID:         credential.ID.Hex(),
		Name:       credential.Name,
		Scopes:     credential.Scopes,
		DailyLimit: credential.DailyLimit,
		ExpiresAt:  credential.ExpiresAt,
		Key:        plainKey,
		KeyLast4:   credential.KeyLast4,
		CreatedAt:  credential.CreatedAt,
	})
}

// List handles GET /shared-credentials. Only name, scopes, quota and last four are exposed.
func (h *SharedCredentialAdminHandler) List(c *gin.Context) {
	credentials, err := h.sharedService.List(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	items := make([]dto.SharedCredentialItem, 0, len(credentials))
	for _, credential := range credentials {
		items = append(items, dto.SharedCredentialItem{
			ID:         credential.ID.Hex(),
			Name:       credential.Name,
			Scopes:     credential.Scopes,
			DailyLimit: credential.DailyLimit,
			KeyLast4:   credential.KeyLast4,
			ExpiresAt:  credential.ExpiresAt,
			Revoked:    credential.Revoked,
			CreatedAt:  credential.CreatedAt,
		})
	}
	util.OK(c, items)
}

// Revoke handles DELETE /shared-credentials/:id.
func (h *SharedCredentialAdminHandler) Revoke(c *gin.Context) {
	if err := h.sharedService.Revoke(c.Request.Context(), c.Param("id")); err != nil {
		_ = c.Error(err)
		return
	}
	middleware.SetAuditAction(c, "shared_credential.revoke", c.Param("id"))
	util.OK(c, gin.H{"revoked": true})
}

// CallLogs handles GET /shared-credentials/:id/call-logs.
func (h *SharedCredentialAdminHandler) CallLogs(c *gin.Context) {
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 64)
	skip := util.Offset(page, pageSize)
	pageSize = util.NormalizePageSize(pageSize)
	logs, err := h.sharedService.ListCallLogs(c.Request.Context(), c.Param("id"), skip, pageSize)
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.OK(c, logs)
}
