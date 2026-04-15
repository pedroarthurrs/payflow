package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"i9pay/internal/api/dto"
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	i9token "i9pay/internal/auth/token"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MemberHandler handles organization member management and per-user permission overrides.
type MemberHandler struct {
	memberSvc *service.MemberService
	permSvc   *service.PermissionService
}

// NewMemberHandler creates a MemberHandler.
func NewMemberHandler(memberSvc *service.MemberService, permSvc *service.PermissionService) *MemberHandler {
	return &MemberHandler{memberSvc: memberSvc, permSvc: permSvc}
}

// RegisterRoutes registers member, permission and activity log routes.
func (h *MemberHandler) RegisterRoutes(rg *gin.RouterGroup) {
	members := rg.Group("/members")
	members.GET("", middleware.RequireRole(domain.RoleAdmin), h.List)
	members.POST("", middleware.RequireRole(domain.RoleOwner), h.Invite)
	members.GET("/:user_id", middleware.RequireRole(domain.RoleAdmin), h.GetOne)
	members.PUT("/:user_id/role", middleware.RequireRole(domain.RoleOwner), h.UpdateRole)
	members.DELETE("/:user_id", middleware.RequireRole(domain.RoleOwner), h.Remove)

	perms := members.Group("/:user_id/permissions")
	perms.GET("", middleware.RequireRole(domain.RoleAdmin), h.ListPermissions)
	perms.PUT("", middleware.RequireRole(domain.RoleOwner), h.BulkSetPermissions)
	perms.DELETE("/:permission", middleware.RequireRole(domain.RoleOwner), h.DeletePermission)

	logs := members.Group("/logs")
	logs.GET("", middleware.RequireRole(domain.RoleAdmin), h.ListActivityLogs)
	logs.GET("/:log_id", middleware.RequireRole(domain.RoleAdmin), h.GetActivityLog)
}

type memberResponse struct {
	ID             uuid.UUID   `json:"id"`
	UserAccountID  uuid.UUID   `json:"user_id"`
	OrganizationID uuid.UUID   `json:"organization_id"`
	Email          string      `json:"email"`
	Name           string      `json:"name"`
	Role           domain.Role `json:"role"`
	JoinedAt       time.Time   `json:"joined_at"`
}

func toMemberResponse(m *domain.OrganizationMember) memberResponse {
	return memberResponse{
		ID:             m.ID,
		UserAccountID:  m.UserAccountID,
		OrganizationID: m.OrganizationID,
		Email:          m.Email,
		Name:           m.Name,
		Role:           m.Role,
		JoinedAt:       m.JoinedAt,
	}
}

type inviteMemberResponse struct {
	memberResponse
	TempPassword *string `json:"temp_password,omitempty"`
}

type permissionEntry struct {
	Permission domain.Permission `json:"permission"`
	Source     string            `json:"source"` // "role" or "grant" or "revoke"
}

type effectivePermissionsResponse struct {
	Role        domain.Role       `json:"role"`
	Permissions []permissionEntry `json:"permissions"`
	Revoked     []permissionEntry `json:"revoked"`
}

type inviteMemberRequest struct {
	Email string      `json:"email" binding:"required"`
	Name  string      `json:"name"  binding:"required"`
	Role  domain.Role `json:"role"  binding:"required"`
}

type updateRoleRequest struct {
	Role domain.Role `json:"role" binding:"required"`
}

type bulkPermissionsRequest struct {
	Grants      []domain.Permission `json:"grants"`
	Revocations []domain.Permission `json:"revocations"`
}

func parseMemberID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid user_id"))
		return uuid.UUID{}, false
	}
	return id, true
}

// List handles GET /api/v1/members — returns all members of the organization.
func (h *MemberHandler) List(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}

	members, _, err := h.memberSvc.ListMembers(c.Request.Context(), orgID, pag.ToListParams())
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]memberResponse, len(members))
	for i := range members {
		resp[i] = toMemberResponse(&members[i])
	}

	c.JSON(http.StatusOK, dto.PagedResponse[memberResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
	})
}

// Invite handles POST /api/v1/members — invites a user by email.
// If the email has no account yet, a temp password is returned.
func (h *MemberHandler) Invite(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	var invitedBy *uuid.UUID
	if claims, ok := c.Get(middleware.ClaimsKey); ok {
		if cl, ok := claims.(*i9token.Claims); ok && cl.UserID != "" {
			if uid, err := uuid.Parse(cl.UserID); err == nil {
				invitedBy = &uid
			}
		}
	}

	var req inviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apierr.BadRequest(err.Error()))
		return
	}

	member, tempPass, err := h.memberSvc.InviteMember(c.Request.Context(), orgID, req.Email, req.Name, req.Role, invitedBy)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) || errors.Is(err, domain.ErrAlreadyMember) {
			_ = c.Error(apierr.Conflict(err.Error()))
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			_ = c.Error(apierr.Forbidden(err.Error()))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, inviteMemberResponse{
		memberResponse: toMemberResponse(member),
		TempPassword:   tempPass,
	})
}

// GetOne handles GET /api/v1/members/:user_id.
func (h *MemberHandler) GetOne(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	memberID, ok := parseMemberID(c)
	if !ok {
		return
	}

	member, err := h.memberSvc.GetMember(c.Request.Context(), orgID, memberID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			_ = c.Error(apierr.NotFound("member not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toMemberResponse(member))
}

// UpdateRole handles PUT /api/v1/members/:user_id/role.
func (h *MemberHandler) UpdateRole(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	memberID, ok := parseMemberID(c)
	if !ok {
		return
	}

	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apierr.BadRequest(err.Error()))
		return
	}

	member, err := h.memberSvc.UpdateMemberRole(c.Request.Context(), orgID, memberID, req.Role)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			_ = c.Error(apierr.NotFound("member not found"))
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			_ = c.Error(apierr.Forbidden(err.Error()))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toMemberResponse(member))
}

// Remove handles DELETE /api/v1/members/:user_id.
func (h *MemberHandler) Remove(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	memberID, ok := parseMemberID(c)
	if !ok {
		return
	}

	if err := h.memberSvc.RemoveMember(c.Request.Context(), orgID, memberID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			_ = c.Error(apierr.NotFound("member not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListPermissions handles GET /api/v1/members/:user_id/permissions.
// Returns the effective permission set with source annotation (role/grant/revoke).
func (h *MemberHandler) ListPermissions(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	memberID, ok := parseMemberID(c)
	if !ok {
		return
	}

	member, err := h.memberSvc.GetMember(c.Request.Context(), orgID, memberID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			_ = c.Error(apierr.NotFound("member not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	effectivePerms, err := h.permSvc.EffectivePermissions(c.Request.Context(), orgID, memberID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// Build source-annotated response.
	roleDefaults := make(map[domain.Permission]bool)
	for _, p := range domain.DefaultPermissionsFor(member.Role) {
		roleDefaults[p] = true
	}

	effectiveSet := make(map[domain.Permission]bool, len(effectivePerms))
	for _, p := range effectivePerms {
		effectiveSet[p] = true
	}

	granted := []permissionEntry{}
	revoked := []permissionEntry{}

	for _, p := range effectivePerms {
		if roleDefaults[p] {
			granted = append(granted, permissionEntry{Permission: p, Source: "role"})
		} else {
			granted = append(granted, permissionEntry{Permission: p, Source: "grant"})
		}
	}

	for p := range roleDefaults {
		if !effectiveSet[p] {
			revoked = append(revoked, permissionEntry{Permission: p, Source: "revoke"})
		}
	}

	c.JSON(http.StatusOK, effectivePermissionsResponse{
		Role:        member.Role,
		Permissions: granted,
		Revoked:     revoked,
	})
}

// BulkSetPermissions handles PUT /api/v1/members/:user_id/permissions.
// Accepts {grants: [...], revocations: [...]} to set multiple overrides atomically.
func (h *MemberHandler) BulkSetPermissions(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	memberID, ok := parseMemberID(c)
	if !ok {
		return
	}

	var req bulkPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apierr.BadRequest(err.Error()))
		return
	}

	ctx := c.Request.Context()
	for _, p := range req.Grants {
		if err := h.permSvc.SetPermission(ctx, orgID, memberID, p, true); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				_ = c.Error(apierr.NotFound("member not found"))
				return
			}
			_ = c.Error(err)
			return
		}
	}
	for _, p := range req.Revocations {
		if err := h.permSvc.SetPermission(ctx, orgID, memberID, p, false); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				_ = c.Error(apierr.NotFound("member not found"))
				return
			}
			_ = c.Error(err)
			return
		}
	}

	c.Status(http.StatusNoContent)
}

// DeletePermission handles DELETE /v1/members/:user_id/permissions/:permission.
func (h *MemberHandler) DeletePermission(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	memberID, ok := parseMemberID(c)
	if !ok {
		return
	}

	perm := domain.Permission(c.Param("permission"))
	if perm == "" {
		_ = c.Error(apierr.BadRequest("permission is required"))
		return
	}

	if err := h.permSvc.RemovePermission(c.Request.Context(), orgID, memberID, perm); err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrPermissionNotFound) {
			_ = c.Error(apierr.NotFound("permission override not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListActivityLogs handles GET /v1/members/logs — lists member activity logs for the organization.
func (h *MemberHandler) ListActivityLogs(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	var limit int64 = 20
	if l := c.Query("limit"); l != "" {
		if v, err := parseMemberLogLimit(l); err == nil {
			limit = v
		}
	}

	var cursor *domain.CursorToken
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		if parsed, err := service.DecodeCursor(cursorStr); err == nil {
			cursor = parsed
		}
	}

	input := domain.ListMemberLogsInput{
		Limit:  int(limit),
		Cursor: cursor,
	}

	if userID := c.Query("user_id"); userID != "" {
		if id, err := uuid.Parse(userID); err == nil {
			input.MemberID = &id
		}
	}
	if action := c.Query("action"); action != "" {
		input.Action = &action
	}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			input.From = &t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			input.To = &t
		}
	}
	if statusCode := c.Query("status_code"); statusCode != "" {
		if sc, err := parseMemberLogInt32(statusCode); err == nil {
			input.StatusCode = &sc
		}
	}
	if method := c.Query("method"); method != "" {
		input.Method = &method
	}
	if path := c.Query("path"); path != "" {
		input.PathPrefix = &path
	}

	records, nextCursor, err := h.memberSvc.ListActivityLogs(c.Request.Context(), orgID, input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]memberActivityLogResponse, len(records))
	for i := range records {
		resp[i] = toMemberActivityLogResponse(&records[i])
	}

	c.JSON(http.StatusOK, dto.CursorPagedResponse[memberActivityLogResponse]{
		Data:       resp,
		NextCursor: nextCursor,
		Limit:      limit,
	})
}

// GetActivityLog handles GET /v1/members/logs/:log_id.
func (h *MemberHandler) GetActivityLog(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("log_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid log_id"))
		return
	}

	log, err := h.memberSvc.GetActivityLogByID(c.Request.Context(), orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrMemberLogNotFound) {
			_ = c.Error(apierr.NotFound(err.Error()))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toMemberActivityLogResponse(log))
}

type memberActivityLogResponse struct {
	ID             string      `json:"id"`
	OrganizationID string      `json:"organization_id"`
	MemberID       string      `json:"member_id"`
	Action         string      `json:"action"`
	Method         *string     `json:"method,omitempty"`
	Path           *string     `json:"path,omitempty"`
	StatusCode     *int32      `json:"status_code,omitempty"`
	LatencyMs      *int32      `json:"latency_ms,omitempty"`
	IPAddress      *string     `json:"ip_address,omitempty"`
	UserAgent      *string     `json:"user_agent,omitempty"`
	Metadata       interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
}

func toMemberActivityLogResponse(log *domain.MemberLogRecord) memberActivityLogResponse {
	resp := memberActivityLogResponse{
		ID:             log.ID.String(),
		OrganizationID: log.OrganizationID.String(),
		MemberID:       log.MemberID.String(),
		Action:         log.Action,
		Method:         log.Method,
		Path:           log.Path,
		StatusCode:     log.StatusCode,
		LatencyMs:      log.LatencyMs,
		IPAddress:      log.IPAddress,
		UserAgent:      log.UserAgent,
		CreatedAt:      log.CreatedAt,
	}
	if len(log.Metadata) > 0 {
		var m interface{}
		_ = json.Unmarshal(log.Metadata, &m)
		resp.Metadata = m
	}
	return resp
}

func parseMemberLogLimit(s string) (int64, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	if v < 1 || v > 100 {
		return 0, errors.New("limit must be between 1 and 100")
	}
	return v, nil
}

func parseMemberLogInt32(s string) (int32, error) {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}
