package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/auth"
	model "github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserRepositoryPublicBehavior(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)
	if repo == nil || repo.db != db {
		t.Fatalf("expected repository to hold provided db, got %+v", repo)
	}

	alice := &model.User{
		Username:   "alice",
		Password:   "hash-1",
		Email:      "alice@example.com",
		IsActive:   true,
		DateJoined: time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC),
	}
	bob := &model.User{
		Username:   "bob",
		Password:   "hash-2",
		Email:      "bob@example.com",
		IsActive:   true,
		DateJoined: time.Date(2026, 3, 11, 8, 0, 0, 0, time.UTC),
	}

	for _, user := range []*model.User{alice, bob} {
		if err := repo.Create(user); err != nil {
			t.Fatalf("create user %s failed: %v", user.Username, err)
		}
		if user.ID == 0 {
			t.Fatalf("expected create to backfill id for %s", user.Username)
		}
	}

	gotByID, err := repo.GetByID(alice.ID)
	if err != nil {
		t.Fatalf("get user by id failed: %v", err)
	}
	if gotByID.Username != "alice" || gotByID.Email != "alice@example.com" {
		t.Fatalf("unexpected user by id: %+v", gotByID)
	}

	gotByUsername, err := repo.FindByUsername("bob")
	if err != nil {
		t.Fatalf("find user by username failed: %v", err)
	}
	if gotByUsername.ID != bob.ID || gotByUsername.Email != "bob@example.com" {
		t.Fatalf("unexpected user by username: %+v", gotByUsername)
	}

	exists, err := repo.ExistsByUsername("alice")
	if err != nil {
		t.Fatalf("exists by username failed: %v", err)
	}
	if !exists {
		t.Fatal("expected existing username to return true")
	}

	exists, err = repo.ExistsByUsername("ghost")
	if err != nil {
		t.Fatalf("exists by username for missing user failed: %v", err)
	}
	if exists {
		t.Fatal("expected missing username to return false")
	}

	pageOne, total, err := repo.List(1, 1)
	if err != nil {
		t.Fatalf("find all first page failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2 users, got %d", total)
	}
	if len(pageOne) != 1 || pageOne[0].ID != bob.ID {
		t.Fatalf("expected first page to return latest id first, got %+v", pageOne)
	}

	pageTwo, total, err := repo.List(2, 1)
	if err != nil {
		t.Fatalf("find all second page failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2 users on second page, got %d", total)
	}
	if len(pageTwo) != 1 || pageTwo[0].ID != alice.ID {
		t.Fatalf("expected second page to return older user, got %+v", pageTwo)
	}

	bob.Email = "updated@example.com"
	bob.IsActive = false
	if err := repo.Update(bob); err != nil {
		t.Fatalf("update user failed: %v", err)
	}

	gotByID, err = repo.GetByID(bob.ID)
	if err != nil {
		t.Fatalf("get updated user failed: %v", err)
	}
	if gotByID.Email != "updated@example.com" || gotByID.IsActive {
		t.Fatalf("expected updated fields to persist, got %+v", gotByID)
	}
}

func TestUserRepositoryDatabaseErrors(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	closeUserRepositorySQLDB(t, db)

	repo := NewUserRepository(db)
	for _, tc := range []struct {
		name string
		call func() error
	}{
		{
			name: "get by id",
			call: func() error {
				_, err := repo.GetByID(1)
				return err
			},
		},
		{
			name: "find by username",
			call: func() error {
				_, err := repo.FindByUsername("alice")
				return err
			},
		},
		{
			name: "get token version",
			call: func() error {
				_, err := repo.GetTokenVersionByID(context.Background(), 1)
				return err
			},
		},
		{
			name: "reset administrator password",
			call: func() error {
				return repo.ResetAdminPassword(context.Background(), "hash")
			},
		},
		{
			name: "find all",
			call: func() error {
				_, _, err := repo.List(1, 10)
				return err
			},
		},
		{
			name: "exists by username",
			call: func() error {
				_, err := repo.ExistsByUsername("alice")
				return err
			},
		},
		{
			name: "create",
			call: func() error {
				return repo.Create(&model.User{Username: "alice", Password: "hash", DateJoined: time.Now().UTC()})
			},
		},
		{
			name: "update",
			call: func() error {
				return repo.Update(&model.User{ID: 1, Username: "alice", Password: "hash", DateJoined: time.Now().UTC()})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err == nil {
				t.Fatalf("expected %s to return database error", tc.name)
			}
		})
	}
}

func TestUserRepositoryResetAdminPassword(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)
	oldHash, err := auth.HashPassword("before-reset")
	if err != nil {
		t.Fatalf("hash original password: %v", err)
	}
	admin := &model.User{
		Username:   administratorUsername,
		Password:   oldHash,
		IsActive:   true,
		DateJoined: time.Now().UTC(),
	}
	if err := repo.Create(admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}

	newHash, err := auth.HashPassword("after-reset")
	if err != nil {
		t.Fatalf("hash replacement password: %v", err)
	}
	if err := repo.ResetAdminPassword(context.Background(), newHash); err != nil {
		t.Fatalf("reset admin password: %v", err)
	}

	updated, err := repo.FindByUsername(administratorUsername)
	if err != nil {
		t.Fatalf("load admin after reset: %v", err)
	}
	if !auth.VerifyPassword("after-reset", updated.Password) || auth.VerifyPassword("before-reset", updated.Password) {
		t.Fatalf("admin password hash was not replaced: %q", updated.Password)
	}
	if updated.TokenVersion != 1 {
		t.Fatalf("admin token version = %d, want 1", updated.TokenVersion)
	}

	version, err := repo.GetTokenVersionByID(context.Background(), admin.ID)
	if err != nil {
		t.Fatalf("get token version: %v", err)
	}
	if version != 1 {
		t.Fatalf("token version reader returned %d, want 1", version)
	}
}

func TestUserRepositoryResetAdminPasswordDoesNotPartiallyCommit(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)
	oldHash, err := auth.HashPassword("before-reset")
	if err != nil {
		t.Fatalf("hash original password: %v", err)
	}
	admin := &model.User{
		Username:   administratorUsername,
		Password:   oldHash,
		IsActive:   true,
		DateJoined: time.Now().UTC(),
	}
	if err := repo.Create(admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if err := db.Exec(`CREATE TRIGGER reject_admin_password_reset
		BEFORE UPDATE ON auth_user
		WHEN OLD.username = 'admin'
		BEGIN
			SELECT RAISE(ABORT, 'forced reset failure');
		END;`).Error; err != nil {
		t.Fatalf("create rollback trigger: %v", err)
	}

	if err := repo.ResetAdminPassword(context.Background(), "replacement-hash"); err == nil {
		t.Fatal("expected reset failure")
	}

	updated, err := repo.FindByUsername(administratorUsername)
	if err != nil {
		t.Fatalf("load admin after failed reset: %v", err)
	}
	if updated.Password != oldHash || updated.TokenVersion != 0 {
		t.Fatalf("failed reset partially committed: %+v", updated)
	}
}

func TestUserRepositoryResetAdminPasswordFailsWhenAdminIsMissing(t *testing.T) {
	repo := NewUserRepository(newUserRepositoryTestDB(t))
	if err := repo.ResetAdminPassword(context.Background(), "replacement-hash"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing admin error = %v, want record not found", err)
	}
}

func TestUserPersistenceTableName(t *testing.T) {
	if got := (model.User{}).TableName(); got != "auth_user" {
		t.Fatalf("unexpected user table name: %s", got)
	}
}

func newUserRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:identity-user-%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db failed: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := db.Exec(`CREATE TABLE auth_user (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		password TEXT NOT NULL,
		token_version INTEGER NOT NULL DEFAULT 0,
		last_login DATETIME,
		is_superuser BOOLEAN NOT NULL DEFAULT FALSE,
		username TEXT NOT NULL UNIQUE,
		first_name TEXT NOT NULL DEFAULT '',
		last_name TEXT NOT NULL DEFAULT '',
		email TEXT NOT NULL DEFAULT '',
		locale TEXT NOT NULL DEFAULT 'en',
		locale_initialized_at DATETIME,
		is_staff BOOLEAN NOT NULL DEFAULT FALSE,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		date_joined DATETIME NOT NULL
	)`).Error; err != nil {
		t.Fatalf("create auth_user table failed: %v", err)
	}

	return db
}

func closeUserRepositorySQLDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db failed: %v", err)
	}
	if err := sqlDB.Close(); err != nil && !errors.Is(err, sql.ErrConnDone) {
		t.Fatalf("close sql db failed: %v", err)
	}
}
