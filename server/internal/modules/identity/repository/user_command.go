package repository

import (
	"github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
)

// Create creates a new user
func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// Update updates a user
func (r *UserRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}
