package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	svc *service.AuthFacade
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(svc *service.AuthFacade) *AuthHandler {
	return &AuthHandler{
		svc: svc,
	}
}

// LoginRequest represents login request body
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response
type LoginResponse struct {
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	ExpiresIn    int64    `json:"expiresIn"`
	User         UserInfo `json:"user"`
}

// UserInfo represents basic user information
type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// RefreshRequest represents refresh token request
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// RefreshResponse represents refresh token response
type RefreshResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"`
}

// Login handles user login
// POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	result, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			dto.Unauthorized(c, "Invalid username or password")
			return
		}
		if errors.Is(err, service.ErrUserDisabled) {
			dto.Unauthorized(c, "User account is disabled")
			return
		}
		dto.InternalError(c, "Failed to login")
		return
	}

	dto.Success(c, LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		User: UserInfo{
			ID:       result.User.ID,
			Username: result.User.Username,
			Email:    result.User.Email,
		},
	})
}

// RefreshToken handles token refresh
// POST /api/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	result, err := h.svc.RefreshToken(req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			dto.Unauthorized(c, "Invalid or expired refresh token")
			return
		}
		dto.InternalError(c, "Failed to refresh token")
		return
	}

	dto.Success(c, RefreshResponse{
		AccessToken: result.AccessToken,
		ExpiresIn:   result.ExpiresIn,
	})
}

// GetCurrentUser returns current user info
// GET /api/auth/me
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		dto.Unauthorized(c, "Not authenticated")
		return
	}

	user, err := h.svc.GetCurrentUser(claims.UserID)
	if err != nil {
		if errors.Is(err, service.ErrAuthUserNotFound) {
			dto.NotFound(c, "User not found")
			return
		}
		dto.InternalError(c, "Failed to get current user")
		return
	}

	dto.Success(c, UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	})
}
