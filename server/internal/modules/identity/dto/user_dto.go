package dto

import "time"

type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=150"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email" binding:"omitempty,email"`
}

type UpdatePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

type UserResponse struct {
	ID          int        `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	IsActive    bool       `json:"isActive"`
	IsSuperuser bool       `json:"isSuperuser"`
	DateJoined  time.Time  `json:"dateJoined"`
	LastLogin   *time.Time `json:"lastLogin"`
}
