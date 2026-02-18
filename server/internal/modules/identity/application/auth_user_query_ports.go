package application

import identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"

type AuthUserStore interface {
	GetAuthUserByID(id int) (*identitydomain.User, error)
	FindAuthUserByUsername(username string) (*identitydomain.User, error)
}
