package handler

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

// UserHandler handles user endpoints
type UserHandler struct {
	svc       *service.UserFacade
	mcpKeySvc MCPKeyLifecycle
}

// MCPKeyLifecycle is the narrow key-management dependency needed by user actions.
// Authentication at the /mcp transport remains owned by the transport boundary.
type MCPKeyLifecycle interface {
	Status(context.Context, int) (service.MCPKeyStatus, error)
	Generate(context.Context, int) (*service.MCPKeyGeneration, error)
}

// NewUserHandler creates a new user handler
func NewUserHandler(svc *service.UserFacade, mcpKeySvc ...MCPKeyLifecycle) *UserHandler {
	var keyService MCPKeyLifecycle
	if len(mcpKeySvc) > 0 {
		keyService = mcpKeySvc[0]
	}
	return &UserHandler{svc: svc, mcpKeySvc: keyService}
}

// CreateUser creates a new user
// POST /v1/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	user, err := h.svc.CreateUser(&req)
	if err != nil {
		if errors.Is(err, service.ErrUsernameExists) {
			httpdto.BadRequest(c, "Username already exists")
			return
		}
		httpdto.InternalError(c, "Failed to create user")
		return
	}

	httpdto.Created(c, dto.UserResponse{
		ID:          user.ID,
		Name:        httpdto.UserName(user.ID),
		Username:    user.Username,
		Email:       user.Email,
		IsActive:    user.IsActive,
		IsSuperuser: user.IsSuperuser,
		DateJoined:  user.DateJoined,
		LastLogin:   user.LastLogin,
	})
}

// List returns paginated users
// GET /v1/users
func (h *UserHandler) List(c *gin.Context) {
	var query httpdto.PaginationQuery
	if !httpdto.BindQuery(c, &query) {
		return
	}

	users, total, err := h.svc.ListUsers(&query)
	if err != nil {
		httpdto.InternalError(c, "Failed to list users")
		return
	}

	var resp []dto.UserResponse
	for _, u := range users {
		resp = append(resp, dto.UserResponse{
			ID:          u.ID,
			Name:        httpdto.UserName(u.ID),
			Username:    u.Username,
			Email:       u.Email,
			IsActive:    u.IsActive,
			IsSuperuser: u.IsSuperuser,
			DateJoined:  u.DateJoined,
			LastLogin:   u.LastLogin,
		})
	}

	httpdto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

// UpdateCurrentUserPassword updates current user's password
// POST /v1/users/me:changePassword
func (h *UserHandler) UpdateCurrentUserPassword(c *gin.Context) {
	// Get current user from context
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}

	var req dto.UpdatePasswordRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	err := h.svc.UpdateUserPassword(claims.UserID, &req)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			httpdto.NotFound(c, "User not found")
			return
		}
		if errors.Is(err, service.ErrInvalidPassword) {
			httpdto.BadRequest(c, "Invalid old password")
			return
		}
		httpdto.InternalError(c, "Failed to update password")
		return
	}

	httpdto.Success(c, gin.H{"message": "Password updated"})
}

// GetCurrentMCPKey returns non-secret MCP key metadata for the current user.
// GET /v1/users/me/mcpKey
func (h *UserHandler) GetCurrentMCPKey(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok || h.mcpKeySvc == nil {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	status, err := h.mcpKeySvc.Status(c.Request.Context(), claims.UserID)
	if err != nil {
		httpdto.InternalError(c, "Failed to get MCP key status")
		return
	}
	status = service.MCPKeyStatusTimestampsAreUTC(status)
	httpdto.Success(c, dto.MCPKeyStatusResponse{
		Configured: status.Configured,
		CreatedAt:  status.CreatedAt,
		UpdatedAt:  status.UpdatedAt,
	})
}

// GenerateCurrentMCPKey generates or rotates the current user's MCP key.
// POST /v1/users/me:generateMcpKey
func (h *UserHandler) GenerateCurrentMCPKey(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok || h.mcpKeySvc == nil {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	result, err := h.mcpKeySvc.Generate(c.Request.Context(), claims.UserID)
	if err != nil {
		httpdto.InternalError(c, "Failed to generate MCP key")
		return
	}
	httpdto.Success(c, dto.MCPKeyGenerationResponse{
		Key:        result.Secret,
		Configured: result.Configured,
		CreatedAt:  result.CreatedAt.UTC(),
		UpdatedAt:  result.UpdatedAt.UTC(),
	})
}
