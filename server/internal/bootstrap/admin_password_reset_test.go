package bootstrap

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/auth"
	identitywiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/identity"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	identityapp "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
	identitymodel "github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestResetAdminPasswordUsesOnlyIdentityDatabasePath(t *testing.T) {
	db := newAdminPasswordResetTestDB(t)
	originalHash, err := auth.HashPassword("original-password")
	if err != nil {
		t.Fatalf("hash original password: %v", err)
	}
	if err := db.Create(&identitymodel.User{
		Username:   "admin",
		Password:   originalHash,
		IsActive:   true,
		DateJoined: time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}

	password, err := resetAdminPassword(context.Background(), db)
	if err != nil {
		t.Fatalf("reset admin password: %v", err)
	}
	if password == "" {
		t.Fatal("reset returned an empty password")
	}
	admin, err := identityrepo.NewUserRepository(db).FindByUsername("admin")
	if err != nil {
		t.Fatalf("load reset admin: %v", err)
	}
	if !auth.VerifyPassword(password, admin.Password) || auth.VerifyPassword("original-password", admin.Password) {
		t.Fatal("reset did not persist exactly the returned password")
	}
	if admin.TokenVersion != 1 {
		t.Fatalf("token version = %d, want 1", admin.TokenVersion)
	}
}

func TestResetAdminPasswordMissingAdminFastFails(t *testing.T) {
	password, err := resetAdminPassword(context.Background(), newAdminPasswordResetTestDB(t))
	if password != "" || !errors.Is(err, identityapp.ErrAdminNotFound) {
		t.Fatalf("missing admin result password=%q err=%v", password, err)
	}
}

func TestResetAdminPasswordImmediatelyRejectsExistingTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAdminPasswordResetTestDB(t)
	originalHash, err := auth.HashPassword("original-password")
	if err != nil {
		t.Fatalf("hash original password: %v", err)
	}
	admin := &identitymodel.User{
		Username:   "admin",
		Password:   originalHash,
		IsActive:   true,
		DateJoined: time.Now().UTC(),
	}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}

	repo := identityrepo.NewUserRepository(db)
	tokens := auth.NewJWTManager("test-secret-key-32-chars-long!!", time.Minute, time.Hour)
	tokenPair, err := tokens.GenerateTokenPair(admin.ID, admin.Username, 0)
	if err != nil {
		t.Fatalf("generate pre-reset tokens: %v", err)
	}

	resetService := identityapp.NewAdminPasswordResetService(
		identitywiring.NewIdentityAdminPasswordResetStoreAdapter(repo),
		identityapp.NewAuthPasswordHasher(),
		fixedAdminPasswordGenerator("replacement-password"),
	)
	if _, err := resetService.Reset(context.Background()); err != nil {
		t.Fatalf("reset administrator password: %v", err)
	}

	refreshService := identityapp.NewAuthCommandService(
		identitywiring.NewIdentityAuthUserStoreAdapter(repo),
		identityapp.NewAuthPasswordVerifier(),
		tokens,
	)
	if _, err := refreshService.RefreshToken(context.Background(), tokenPair.RefreshToken); !errors.Is(err, identityapp.ErrInvalidRefreshToken) {
		t.Fatalf("refresh after reset error = %v, want invalid refresh token", err)
	}

	handled := false
	router := gin.New()
	router.Use(middleware.AuthMiddleware(tokens, identitywiring.NewIdentityTokenVersionReaderAdapter(repo)))
	router.GET("/protected", func(c *gin.Context) {
		handled = true
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized || handled {
		t.Fatalf("pre-reset access token status=%d handled=%t, want rejected before handler", recorder.Code, handled)
	}
}

func TestRunAdminPasswordResetRejectsMissingInputs(t *testing.T) {
	if _, err := RunAdminPasswordReset(nil, nil, embed.FS{}); err == nil {
		t.Fatal("expected missing context to fail before initialization")
	}
	if _, err := RunAdminPasswordReset(context.Background(), nil, embed.FS{}); err == nil {
		t.Fatal("expected missing database configuration to fail before initialization")
	}
}

func newAdminPasswordResetTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:admin-password-reset-%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite connection: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
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
		t.Fatalf("create auth_user table: %v", err)
	}
	return db
}

type fixedAdminPasswordGenerator string

func (generator fixedAdminPasswordGenerator) Generate() (string, error) {
	return string(generator), nil
}
