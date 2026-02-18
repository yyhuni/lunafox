package application

import identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"

type UserCommandStore interface {
	GetUserByID(id int) (*identitydomain.User, error)
	ExistsByUsername(username string) (bool, error)
	Create(user *identitydomain.User) error
	Update(user *identitydomain.User) error
}
