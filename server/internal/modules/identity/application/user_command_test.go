package application

import (
	"context"
	"errors"
	"testing"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	"gorm.io/gorm"
)

type userCommandStoreStub struct {
	userByID          map[int]*identitydomain.User
	usernameExists    map[string]bool
	createdUser       *identitydomain.User
	updatedUser       *identitydomain.User
	findByIDErr       error
	existsByNameErr   error
	createErr         error
	updateErr         error
	forceCreateUserID int
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
	copyUser := *user
	if stub.forceCreateUserID > 0 {
		copyUser.ID = stub.forceCreateUserID
	}
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
