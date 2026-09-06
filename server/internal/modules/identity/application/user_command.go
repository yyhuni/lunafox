package application

import (
	"context"
	"errors"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

type PasswordHasher interface {
	HashPassword(password string) (string, error)
	VerifyPassword(password, hashed string) bool
}

type UserCommandService struct {
	store  UserCommandStore
	hasher PasswordHasher
}

func NewUserCommandService(store UserCommandStore, hasher PasswordHasher) *UserCommandService {
	return &UserCommandService{store: store, hasher: hasher}
}

func (service *UserCommandService) CreateUser(ctx context.Context, username, password, email string) (*identitydomain.User, error) {
	_ = ctx

	normalizedUsername := identitydomain.NormalizeUsername(username)

	exists, err := service.store.ExistsByUsername(normalizedUsername)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameExists
	}

	hashedPassword, err := service.hasher.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := identitydomain.NewUser(username, email, hashedPassword)

	if err := service.store.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (service *UserCommandService) UpdateUserPassword(ctx context.Context, id int, oldPassword, newPassword string) error {
	_ = ctx

	user, err := service.store.GetUserByID(id)
	if err != nil {
		return err
	}

	if !service.hasher.VerifyPassword(oldPassword, user.Password) {
		return ErrInvalidPassword
	}

	hashedPassword, err := service.hasher.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.UpdatePassword(hashedPassword)
	if err := service.store.Update(user); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}
