package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

// UserHandler handles user endpoints
type UserHandler struct {
	svc *service.UserFacade
}

// NewUserHandler creates a new user handler
func NewUserHandler(svc *service.UserFacade) *UserHandler {
	return &UserHandler{svc: svc}
}

// CreateUser creates a new user
// POST /api/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	user, err := h.svc.CreateUser(&req)
	if err != nil {
		if errors.Is(err, service.ErrUsernameExists) {
			dto.BadRequest(c, "Username already exists")
			return
		}
		dto.InternalError(c, "Failed to create user")
		return
	}

	dto.Created(c, dto.UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		IsActive:    user.IsActive,
		IsSuperuser: user.IsSuperuser,
		DateJoined:  user.DateJoined,
		LastLogin:   user.LastLogin,
	})
}

// ListUsers returns paginated users
// GET /api/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	var query dto.PaginationQuery
	if !dto.BindQuery(c, &query) {
		return
	}

	users, total, err := h.svc.ListUsers(&query)
	if err != nil {
		dto.InternalError(c, "Failed to list users")
		return
	}

	var resp []dto.UserResponse
	for _, u := range users {
		resp = append(resp, dto.UserResponse{
			ID:          u.ID,
			Username:    u.Username,
			Email:       u.Email,
			IsActive:    u.IsActive,
			IsSuperuser: u.IsSuperuser,
			DateJoined:  u.DateJoined,
			LastLogin:   u.LastLogin,
		})
	}

	dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
}

// UpdateCurrentUserPassword updates current user's password
// PUT /api/users/me/password
func (h *UserHandler) UpdateCurrentUserPassword(c *gin.Context) {
	// Get current user from context
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		dto.Unauthorized(c, "Not authenticated")
		return
	}

	var req dto.UpdatePasswordRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	err := h.svc.UpdateUserPassword(claims.UserID, &req)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			dto.NotFound(c, "User not found")
			return
		}
		if errors.Is(err, service.ErrInvalidPassword) {
			dto.BadRequest(c, "Invalid old password")
			return
		}
		dto.InternalError(c, "Failed to update password")
		return
	}

	dto.Success(c, gin.H{"message": "Password updated"})
}
