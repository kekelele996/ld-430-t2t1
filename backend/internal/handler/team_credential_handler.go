package handler

import (
	"strconv"
	"time"

	"github.com/assethub/assethub/internal/constants"
	"github.com/assethub/assethub/internal/dto"
	"github.com/assethub/assethub/internal/middleware"
	"github.com/assethub/assethub/internal/model"
	"github.com/assethub/assethub/internal/service"
	"github.com/assethub/assethub/internal/util"
	"github.com/gin-gonic/gin"
)

// TeamCredentialHandler exposes admin endpoints for managing team credentials.
type TeamCredentialHandler struct {
	credentialService *service.TeamCredentialService
}

// NewTeamCredentialHandler creates a TeamCredentialHandler.
func NewTeamCredentialHandler(credentialService *service.TeamCredentialService) *TeamCredentialHandler {
	return &TeamCredentialHandler{credentialService: credentialService}
}

// Create handles POST /team-credentials.
func (h *TeamCredentialHandler) Create(c *gin.Context) {
	var req dto.CreateTeamCredentialRequest
	if err := middleware.ValidateJSON(c, &req); err != nil {
		_ = c.Error(err)
		return
	}
	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			_ = c.Error(err)
			return
		}
		if !parsed.After(time.Now()) {
			respondBusiness(c, constants.CodeValidationFailed, "expires_at must be in the future")
			return
		}
		expiresAt = &parsed
	}

	adminID := mustObjectID(c.GetString(middleware.ContextUserIDKey))
	credential, plaintext, err := h.credentialService.Create(
		c.Request.Context(), req.Name, req.Scopes, req.DailyLimit, expiresAt, adminID,
	)
	if err != nil {
		_ = c.Error(err)
		return
	}
	middleware.SetAuditAction(c, "team_credential.create", credential.Name)
	util.Created(c, dto.TeamCredentialCreatedResponse{
		ID:         credential.ID.Hex(),
		Name:       credential.Name,
		APIKey:     plaintext,
		Scopes:     credential.Scopes,
		DailyLimit: credential.DailyLimit,
		Status:     credential.Status,
		CreatedAt:  credential.CreatedAt,
	})
}

// List handles GET /team-credentials. The list keeps only name, scopes, daily
// quota and the key's last four characters.
func (h *TeamCredentialHandler) List(c *gin.Context) {
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 64)
	skip := util.Offset(page, pageSize)
	pageSize = util.NormalizePageSize(pageSize)

	credentials, err := h.credentialService.List(c.Request.Context(), skip, pageSize)
	if err != nil {
		_ = c.Error(err)
		return
	}
	summaries := make([]dto.TeamCredentialSummary, 0, len(credentials))
	for i := range credentials {
		summaries = append(summaries, toCredentialSummary(&credentials[i]))
	}
	util.OK(c, summaries)
}

// Get handles GET /team-credentials/:id.
func (h *TeamCredentialHandler) Get(c *gin.Context) {
	credential, err := h.credentialService.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.OK(c, credential)
}

// Revoke handles POST /team-credentials/:id/revoke.
func (h *TeamCredentialHandler) Revoke(c *gin.Context) {
	adminID := mustObjectID(c.GetString(middleware.ContextUserIDKey))
	if err := h.credentialService.Revoke(c.Request.Context(), c.Param("id"), adminID); err != nil {
		_ = c.Error(err)
		return
	}
	middleware.SetAuditAction(c, "team_credential.revoke", c.Param("id"))
	util.OK(c, gin.H{"revoked": true})
}

// AccessLogs handles GET /team-credentials/:id/access-logs.
func (h *TeamCredentialHandler) AccessLogs(c *gin.Context) {
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 64)
	skip := util.Offset(page, pageSize)
	pageSize = util.NormalizePageSize(pageSize)

	logs, err := h.credentialService.AccessLogs(c.Request.Context(), c.Param("id"), skip, pageSize)
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.OK(c, logs)
}

func toCredentialSummary(c *model.TeamCredential) dto.TeamCredentialSummary {
	return dto.TeamCredentialSummary{
		ID:         c.ID.Hex(),
		Name:       c.Name,
		Scopes:     c.Scopes,
		DailyLimit: c.DailyLimit,
		LastFour:   c.LastFour,
	}
}
