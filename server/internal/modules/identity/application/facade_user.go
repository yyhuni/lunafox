package application

import (
	"context"
	"errors"

	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

// UserFacade handles user business logic.
type UserFacade struct {
	queryService *UserQueryService
	cmdService   *UserCommandService
}

// NewUserFacade creates a new user service.
func NewUserFacade(queryService *UserQueryService, cmdService *UserCommandService) *UserFacade {
	return &UserFacade{
		queryService: queryService,
		cmdService:   cmdService,
	}
}

// CreateUser creates a new user.
func (service *UserFacade) CreateUser(req *dto.CreateUserRequest) (*identitydomain.User, error) {
	user, err := service.cmdService.CreateUser(context.Background(), req.Username, req.Password, req.Email)
	if err != nil {
		if errors.Is(err, ErrUsernameExists) {
			return nil, ErrUsernameExists
		}
		return nil, err
	}
	return user, nil
}

// ListUsers returns paginated users.
func (service *UserFacade) ListUsers(query *httpdto.PaginationQuery) ([]identitydomain.User, int64, error) {
	users, total, err := service.queryService.ListUsers(context.Background(), query.GetPage(), query.GetPageSize())
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// GetUserByID returns a user by ID.
func (service *UserFacade) GetUserByID(id int) (*identitydomain.User, error) {
	user, err := service.queryService.GetUserByID(context.Background(), id)
	if err != nil {
		return nil, mapUserBoundaryError(err)
	}
	return user, nil
}

// UpdateUserPassword updates user password.
func (service *UserFacade) UpdateUserPassword(id int, req *dto.UpdatePasswordRequest) error {
	err := service.cmdService.UpdateUserPassword(context.Background(), id, req.OldPassword, req.NewPassword)
	if err != nil {
		if isIdentityRecordNotFound(err) {
			return ErrUserNotFound
		}
		if errors.Is(err, ErrInvalidPassword) {
			return ErrInvalidPassword
		}
		return err
	}
	return nil
}

func NewAuthPasswordHasher() PasswordHasher {
	return authPasswordHasher{}
}

func NewAuthPasswordVerifier() PasswordVerifier {
	return authPasswordVerifier{}
}

type authPasswordHasher struct{}

func (authPasswordHasher) HashPassword(password string) (string, error) {
	return auth.HashPassword(password)
}

func (authPasswordHasher) VerifyPassword(password, hashed string) bool {
	return auth.VerifyPassword(password, hashed)
}

type authPasswordVerifier struct{}

func (authPasswordVerifier) VerifyPassword(password, hashed string) bool {
	return auth.VerifyPassword(password, hashed)
}
