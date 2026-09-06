package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/yyhuni/lunafox/server/internal/auth"
)

type tokenVersionReaderStub struct {
	version int
	err     error
}

func (stub tokenVersionReaderStub) GetTokenVersion(context.Context, int) (int, error) {
	return stub.version, stub.err
}

func TestAuthMiddleware_ValidatesPersistedTokenVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := auth.NewJWTManager("test-secret-key-32-chars-long!!", time.Minute, time.Hour)

	t.Run("matching version continues", func(t *testing.T) {
		token, _, err := manager.GenerateAccessToken(7, "alice", 2)
		if err != nil {
			t.Fatalf("generate token: %v", err)
		}

		router := gin.New()
		router.Use(AuthMiddleware(manager, tokenVersionReaderStub{version: 2}))
		router.GET("/protected", func(c *gin.Context) {
			claims, ok := GetUserClaims(c)
			if !ok || claims.TokenVersion != 2 {
				t.Fatalf("unexpected claims: %+v", claims)
			}
			c.Status(http.StatusNoContent)
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/protected", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusNoContent, recorder.Body.String())
		}
	})

	t.Run("mismatch rejects before handler", func(t *testing.T) {
		token, _, err := manager.GenerateAccessToken(7, "alice", 2)
		if err != nil {
			t.Fatalf("generate token: %v", err)
		}

		handled := false
		router := gin.New()
		router.Use(AuthMiddleware(manager, tokenVersionReaderStub{version: 3}))
		router.GET("/protected", func(c *gin.Context) {
			handled = true
			c.Status(http.StatusNoContent)
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/protected", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized || handled {
			t.Fatalf("mismatched token status = %d, handler=%t", recorder.Code, handled)
		}
	})

	t.Run("reader failure rejects before handler", func(t *testing.T) {
		token, _, err := manager.GenerateAccessToken(7, "alice", 2)
		if err != nil {
			t.Fatalf("generate token: %v", err)
		}

		router := gin.New()
		router.Use(AuthMiddleware(manager, tokenVersionReaderStub{err: errors.New("database unavailable")}))
		router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/protected", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("reader failure status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
	})
}

func TestAuthMiddleware_AcceptsLegacyZeroVersionToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "test-secret-key-32-chars-long!!"
	legacy := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":   7,
		"username": "alice",
		"exp":      time.Now().Add(time.Hour).Unix(),
		"iat":      time.Now().Unix(),
		"nbf":      time.Now().Unix(),
	})
	token, err := legacy.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign legacy token: %v", err)
	}

	router := gin.New()
	router.Use(AuthMiddleware(auth.NewJWTManager(secret, time.Minute, time.Hour), tokenVersionReaderStub{version: 0}))
	router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("legacy token status = %d, want %d: %s", recorder.Code, http.StatusNoContent, recorder.Body.String())
	}
}

func TestAuthMiddleware_ExpiredTokenUsesRenewableCanonicalReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "test-secret-key-32-chars-long!!"
	expiredManager := auth.NewJWTManager(secret, -time.Minute, time.Hour)
	token, _, err := expiredManager.GenerateAccessToken(7, "alice", 0)
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}

	router := gin.New()
	router.Use(AuthMiddleware(auth.NewJWTManager(secret, time.Minute, time.Hour), tokenVersionReaderStub{version: 0}))
	router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(recorder.Body.String(), "TOKEN_EXPIRED") {
		t.Fatalf("expired token response lacks TOKEN_EXPIRED: %s", recorder.Body.String())
	}
}
