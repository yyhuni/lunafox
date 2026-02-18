package application

import identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"

type UserQueryStore interface {
	GetUserByID(id int) (*identitydomain.User, error)
	FindAll(page, pageSize int) ([]identitydomain.User, int64, error)
}
