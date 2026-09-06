package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type userCommandStoreStub struct {
	userByID          map[int]*identitydomain.User
	users             []identitydomain.User
	total             int64
	usernameExists    map[string]bool
	createdUser       *identitydomain.User
	updatedUser       *identitydomain.User
	findByIDErr       error
	listErr           error
	existsByNameErr   error
	createErr         error
	updateErr         error
	forceCreateUserID int
	lastListPage      int
	lastListPageSize  int
}

func (stub *userCommandStoreStub) GetUserByID(id int) (*identitydomain.User, error) {
	if stub.findByIDErr != nil {
		return nil, stub.findByIDErr
	}
	user, ok := stub.userByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func (stub *userCommandStoreStub) ExistsByUsername(username string) (bool, error) {
	if stub.existsByNameErr != nil {
		return false, stub.existsByNameErr
	}
	return stub.usernameExists[username], nil
}

func (stub *userCommandStoreStub) Create(user *identitydomain.User) error {
	if stub.createErr != nil {
		return stub.createErr
	}
	if stub.forceCreateUserID > 0 {
		user.ID = stub.forceCreateUserID
	}
	copyUser := *user
	stub.createdUser = &copyUser
	return nil
}

func (stub *userCommandStoreStub) Update(user *identitydomain.User) error {
	if stub.updateErr != nil {
		return stub.updateErr
	}
	copyUser := *user
	stub.updatedUser = &copyUser
	return nil
}

func (stub *userCommandStoreStub) List(page, pageSize int) ([]identitydomain.User, int64, error) {
	stub.lastListPage = page
	stub.lastListPageSize = pageSize
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	result := make([]identitydomain.User, len(stub.users))
	copy(result, stub.users)
	return result, stub.total, nil
}

type passwordHasherStub struct {
	hashValue  string
	hashErr    error
	verifyFunc func(password, hashed string) bool
}

func (stub passwordHasherStub) HashPassword(password string) (string, error) {
	if stub.hashErr != nil {
		return "", stub.hashErr
	}
	if stub.hashValue != "" {
		return stub.hashValue, nil
	}
	return "hashed-" + password, nil
}

func (stub passwordHasherStub) VerifyPassword(password, hashed string) bool {
	if stub.verifyFunc != nil {
		return stub.verifyFunc(password, hashed)
	}
	return "hashed-"+password == hashed
}

func TestUserCommandServiceCreateUser(t *testing.T) {
	t.Run("username already exists", func(t *testing.T) {
		store := &userCommandStoreStub{
			usernameExists: map[string]bool{"alice": true},
		}
		service := NewUserCommandService(store, passwordHasherStub{})

		_, err := service.CreateUser(context.Background(), "alice", "pass123", "a@example.com")
		if !errors.Is(err, ErrUsernameExists) {
			t.Fatalf("expected ErrUsernameExists, got %v", err)
		}
	})

	t.Run("create user succeeds", func(t *testing.T) {
		store := &userCommandStoreStub{
			usernameExists:    map[string]bool{},
			forceCreateUserID: 100,
		}
		hasher := passwordHasherStub{hashValue: "secure-hash"}
		service := NewUserCommandService(store, hasher)

		user, err := service.CreateUser(context.Background(), "alice", "pass123", "a@example.com")
		if err != nil {
			t.Fatalf("create user failed: %v", err)
		}
		if store.createdUser == nil {
			t.Fatalf("expected store.Create called")
		}
		if store.createdUser.Password != "secure-hash" {
			t.Fatalf("expected hashed password saved, got %s", store.createdUser.Password)
		}
		if user.Username != "alice" || user.Email != "a@example.com" {
			t.Fatalf("unexpected user: %+v", user)
		}
	})
}

func TestUserCommandServiceUpdateUserPassword(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		store := &userCommandStoreStub{
			userByID: map[int]*identitydomain.User{},
		}
		service := NewUserCommandService(store, passwordHasherStub{})

		err := service.UpdateUserPassword(context.Background(), 1, "old", "new")
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
		}
	})

	t.Run("old password mismatch", func(t *testing.T) {
		store := &userCommandStoreStub{
			userByID: map[int]*identitydomain.User{1: {
				ID:       1,
				Username: "alice",
				Password: "stored-hash",
			}},
		}
		hasher := passwordHasherStub{
			verifyFunc: func(password, hashed string) bool {
				return false
			},
		}
		service := NewUserCommandService(store, hasher)

		err := service.UpdateUserPassword(context.Background(), 1, "wrong", "new-pass")
		if !errors.Is(err, ErrInvalidPassword) {
			t.Fatalf("expected ErrInvalidPassword, got %v", err)
		}
	})

	t.Run("update password succeeds", func(t *testing.T) {
		store := &userCommandStoreStub{
			userByID: map[int]*identitydomain.User{1: {
				ID:       1,
				Username: "alice",
				Password: "old-hash",
			}},
		}
		hasher := passwordHasherStub{
			hashValue: "new-hash",
			verifyFunc: func(password, hashed string) bool {
				return password == "old-pass" && hashed == "old-hash"
			},
		}
		service := NewUserCommandService(store, hasher)

		err := service.UpdateUserPassword(context.Background(), 1, "old-pass", "new-pass")
		if err != nil {
			t.Fatalf("update password failed: %v", err)
		}
		if store.updatedUser == nil {
			t.Fatalf("expected store.Update called")
		}
		if store.updatedUser.Password != "new-hash" {
			t.Fatalf("expected password updated to new hash, got %s", store.updatedUser.Password)
		}
	})
}

func TestUserFacade(t *testing.T) {
	t.Run("constructor-backed facade exposes public user flows", func(t *testing.T) {
		oldHash, err := auth.HashPassword("old-pass")
		if err != nil {
			t.Fatalf("hash old password: %v", err)
		}

		store := &userCommandStoreStub{
			userByID: map[int]*identitydomain.User{
				1: {ID: 1, Username: "alice", Email: "alice@example.com", Password: oldHash, IsActive: true},
			},
			users: []identitydomain.User{
				{ID: 1, Username: "alice", Email: "alice@example.com", IsActive: true},
			},
			total:          1,
			usernameExists: map[string]bool{},
		}
		facade := NewUserFacade(NewUserQueryService(store), NewUserCommandService(store, NewAuthPasswordHasher()))

		created, err := facade.CreateUser(&dto.CreateUserRequest{
			Username: "  bob  ",
			Password: "pass123",
			Email:    "  BOB@Example.com ",
		})
		if err != nil {
			t.Fatalf("create user failed: %v", err)
		}
		if created.Username != "bob" || created.Email != "bob@example.com" || !created.IsActive {
			t.Fatalf("unexpected created user: %+v", created)
		}
		if store.createdUser == nil || !auth.VerifyPassword("pass123", store.createdUser.Password) {
			t.Fatalf("expected created user password hashed, stored=%+v", store.createdUser)
		}

		users, total, err := facade.ListUsers(&httpdto.PaginationQuery{})
		if err != nil {
			t.Fatalf("list users failed: %v", err)
		}
		if total != 1 || len(users) != 1 || users[0].Username != "alice" {
			t.Fatalf("unexpected list result: total=%d users=%+v", total, users)
		}
		if store.lastListPage != 1 || store.lastListPageSize != 20 {
			t.Fatalf("unexpected list args: page=%d pageSize=%d", store.lastListPage, store.lastListPageSize)
		}

		user, err := facade.GetUserByID(1)
		if err != nil {
			t.Fatalf("get user failed: %v", err)
		}
		if user.ID != 1 || user.Email != "alice@example.com" {
			t.Fatalf("unexpected user detail: %+v", user)
		}

		err = facade.UpdateUserPassword(1, &dto.UpdatePasswordRequest{
			OldPassword: "old-pass",
			NewPassword: "new-pass",
		})
		if err != nil {
			t.Fatalf("update password failed: %v", err)
		}
		if store.updatedUser == nil || !auth.VerifyPassword("new-pass", store.updatedUser.Password) {
			t.Fatalf("expected updated password hash, updated=%+v", store.updatedUser)
		}
	})

	t.Run("public error mappings and passthroughs remain stable", func(t *testing.T) {
		duplicateStore := &userCommandStoreStub{
			usernameExists: map[string]bool{"taken": true},
		}
		facade := NewUserFacade(NewUserQueryService(duplicateStore), NewUserCommandService(duplicateStore, NewAuthPasswordHasher()))
		_, err := facade.CreateUser(&dto.CreateUserRequest{
			Username: "taken",
			Password: "pass123",
			Email:    "taken@example.com",
		})
		if !errors.Is(err, ErrUsernameExists) {
			t.Fatalf("expected ErrUsernameExists, got %v", err)
		}

		createStoreErr := errors.New("insert failed")
		createStore := &userCommandStoreStub{
			usernameExists: map[string]bool{},
			createErr:      createStoreErr,
		}
		facade = NewUserFacade(NewUserQueryService(createStore), NewUserCommandService(createStore, NewAuthPasswordHasher()))
		_, err = facade.CreateUser(&dto.CreateUserRequest{
			Username: "alice",
			Password: "pass123",
			Email:    "alice@example.com",
		})
		if !errors.Is(err, createStoreErr) {
			t.Fatalf("expected create error passthrough, got %v", err)
		}

		hashFailureStore := &userCommandStoreStub{
			usernameExists: map[string]bool{},
		}
		facade = NewUserFacade(NewUserQueryService(hashFailureStore), NewUserCommandService(hashFailureStore, NewAuthPasswordHasher()))
		_, err = facade.CreateUser(&dto.CreateUserRequest{
			Username: "oversized",
			Password: strings.Repeat("a", 73),
			Email:    "oversized@example.com",
		})
		if !errors.Is(err, bcrypt.ErrPasswordTooLong) {
			t.Fatalf("expected bcrypt.ErrPasswordTooLong passthrough, got %v", err)
		}
		if hashFailureStore.createdUser != nil {
			t.Fatalf("expected create skipped after hash failure, got %+v", hashFailureStore.createdUser)
		}

		listStoreErr := errors.New("list failed")
		listStore := &userCommandStoreStub{listErr: listStoreErr}
		facade = NewUserFacade(NewUserQueryService(listStore), NewUserCommandService(listStore, NewAuthPasswordHasher()))
		_, _, err = facade.ListUsers(&httpdto.PaginationQuery{})
		if !errors.Is(err, listStoreErr) {
			t.Fatalf("expected list error passthrough, got %v", err)
		}

		getStore := &userCommandStoreStub{userByID: map[int]*identitydomain.User{}}
		facade = NewUserFacade(NewUserQueryService(getStore), NewUserCommandService(getStore, NewAuthPasswordHasher()))
		_, err = facade.GetUserByID(99)
		if !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("expected ErrUserNotFound, got %v", err)
		}

		getStoreErr := errors.New("lookup failed")
		getStore = &userCommandStoreStub{findByIDErr: getStoreErr}
		facade = NewUserFacade(NewUserQueryService(getStore), NewUserCommandService(getStore, NewAuthPasswordHasher()))
		_, err = facade.GetUserByID(1)
		if !errors.Is(err, getStoreErr) {
			t.Fatalf("expected get user error passthrough, got %v", err)
		}

		updateNotFoundStore := &userCommandStoreStub{userByID: map[int]*identitydomain.User{}}
		facade = NewUserFacade(NewUserQueryService(updateNotFoundStore), NewUserCommandService(updateNotFoundStore, NewAuthPasswordHasher()))
		err = facade.UpdateUserPassword(99, &dto.UpdatePasswordRequest{
			OldPassword: "old-pass",
			NewPassword: "new-pass",
		})
		if !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("expected ErrUserNotFound, got %v", err)
		}

		validHash, err := auth.HashPassword("old-pass")
		if err != nil {
			t.Fatalf("hash valid password: %v", err)
		}

		invalidPasswordStore := &userCommandStoreStub{
			userByID: map[int]*identitydomain.User{
				1: {ID: 1, Username: "alice", Password: validHash, IsActive: true},
			},
		}
		facade = NewUserFacade(NewUserQueryService(invalidPasswordStore), NewUserCommandService(invalidPasswordStore, NewAuthPasswordHasher()))
		err = facade.UpdateUserPassword(1, &dto.UpdatePasswordRequest{
			OldPassword: "wrong-pass",
			NewPassword: "new-pass",
		})
		if !errors.Is(err, ErrInvalidPassword) {
			t.Fatalf("expected ErrInvalidPassword, got %v", err)
		}

		updateStoreErr := errors.New("update failed")
		updateStore := &userCommandStoreStub{
			userByID: map[int]*identitydomain.User{
				1: {ID: 1, Username: "alice", Password: validHash, IsActive: true},
			},
			updateErr: updateStoreErr,
		}
		facade = NewUserFacade(NewUserQueryService(updateStore), NewUserCommandService(updateStore, NewAuthPasswordHasher()))
		err = facade.UpdateUserPassword(1, &dto.UpdatePasswordRequest{
			OldPassword: "old-pass",
			NewPassword: "new-pass",
		})
		if !errors.Is(err, updateStoreErr) {
			t.Fatalf("expected update error passthrough, got %v", err)
		}
	})
}
