package application

import (
	"context"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

type UserQueryService struct {
	store UserQueryStore
}

func NewUserQueryService(store UserQueryStore) *UserQueryService {
	return &UserQueryService{store: store}
}

func (service *UserQueryService) ListUsers(ctx context.Context, page, pageSize int) ([]identitydomain.User, int64, error) {
	_ = ctx
	return service.store.FindAll(page, pageSize)
}

func (service *UserQueryService) GetUserByID(ctx context.Context, id int) (*identitydomain.User, error) {
	_ = ctx
	return service.store.GetUserByID(id)
}
