package repository

import (
	"context"

	model "github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// GetTokenVersionByID reads only the revocation version needed by protected requests.
func (r *UserRepository) GetTokenVersionByID(ctx context.Context, id int) (int, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Select("token_version").First(&user, id).Error; err != nil {
		return 0, err
	}
	return user.TokenVersion, nil
}

// GetByID finds a user by ID
func (r *UserRepository) GetByID(id int) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsername finds a user by username
func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// List returns users with pagination
func (r *UserRepository) List(page, pageSize int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	if err := r.db.Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Scopes(
		scope.WithPagination(page, pageSize),
		scope.OrderBy("id", true),
	).Find(&users).Error

	return users, total, err
}

// ExistsByUsername checks if username exists
func (r *UserRepository) ExistsByUsername(username string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}
