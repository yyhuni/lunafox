package identitywiring

import (
	"context"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
)

type identityUserStoreAdapter struct {
	repo *identityrepo.UserRepository
}

func newIdentityUserStoreAdapter(repo *identityrepo.UserRepository) *identityUserStoreAdapter {
	return &identityUserStoreAdapter{repo: repo}
}

func (adapter *identityUserStoreAdapter) GetUserByID(id int) (*identitydomain.User, error) {
	return adapter.getUserByID(id)
}

func (adapter *identityUserStoreAdapter) GetAuthUserByID(id int) (*identitydomain.User, error) {
	return adapter.getUserByID(id)
}

func (adapter *identityUserStoreAdapter) FindAuthUserByUsername(username string) (*identitydomain.User, error) {
	return adapter.findUserByUsername(username)
}

func (adapter *identityUserStoreAdapter) GetTokenVersion(ctx context.Context, userID int) (int, error) {
	return adapter.repo.GetTokenVersionByID(ctx, userID)
}

func (adapter *identityUserStoreAdapter) ResetAdminPassword(ctx context.Context, hashedPassword string) error {
	return adapter.repo.ResetAdminPassword(ctx, hashedPassword)
}

func (adapter *identityUserStoreAdapter) ExistsByUsername(username string) (bool, error) {
	return adapter.repo.ExistsByUsername(username)
}

func (adapter *identityUserStoreAdapter) List(page, pageSize int) ([]identitydomain.User, int64, error) {
	users, total, err := adapter.repo.List(page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return identityModelUsersToIdentityDomainUsers(users), total, nil
}

func (adapter *identityUserStoreAdapter) Create(user *identitydomain.User) error {
	modelUser := identityDomainUserToIdentityModelUser(user)
	if err := adapter.repo.Create(modelUser); err != nil {
		return err
	}
	*user = *identityModelUserToIdentityDomainUser(modelUser)
	return nil
}

func (adapter *identityUserStoreAdapter) Update(user *identitydomain.User) error {
	return adapter.repo.Update(identityDomainUserToIdentityModelUser(user))
}

func (adapter *identityUserStoreAdapter) getUserByID(id int) (*identitydomain.User, error) {
	user, err := adapter.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return identityModelUserToIdentityDomainUser(user), nil
}

func (adapter *identityUserStoreAdapter) findUserByUsername(username string) (*identitydomain.User, error) {
	user, err := adapter.repo.FindByUsername(username)
	if err != nil {
		return nil, err
	}
	return identityModelUserToIdentityDomainUser(user), nil
}
