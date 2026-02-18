package application

import (
	"context"
	"errors"
	"testing"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

type userQueryStoreStub struct {
	users    []identitydomain.User
	total    int64
	userByID map[int]*identitydomain.User
	listErr  error
	getErr   error
}

func (stub *userQueryStoreStub) GetUserByID(id int) (*identitydomain.User, error) {
	if stub.getErr != nil {
		return nil, stub.getErr
	}
	user, ok := stub.userByID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	copyUser := *user
	return &copyUser, nil
}

func (stub *userQueryStoreStub) FindAll(page, pageSize int) ([]identitydomain.User, int64, error) {
	_ = page
	_ = pageSize
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	result := make([]identitydomain.User, len(stub.users))
	copy(result, stub.users)
	return result, stub.total, nil
}

func TestUserQueryServiceListUsers(t *testing.T) {
	store := &userQueryStoreStub{
		users: []identitydomain.User{
			{ID: 1, Username: "alice"},
			{ID: 2, Username: "bob"},
		},
		total: 2,
	}
	service := NewUserQueryService(store)

	users, total, err := service.ListUsers(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("list users failed: %v", err)
	}
	if total != 2 || len(users) != 2 {
		t.Fatalf("unexpected list result: total=%d len=%d", total, len(users))
	}
}

func TestUserQueryServiceGetUserByID(t *testing.T) {
	store := &userQueryStoreStub{
		userByID: map[int]*identitydomain.User{
			7: {ID: 7, Username: "charlie"},
		},
	}
	service := NewUserQueryService(store)

	user, err := service.GetUserByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	if user.ID != 7 || user.Username != "charlie" {
		t.Fatalf("unexpected user: %+v", user)
	}
}
