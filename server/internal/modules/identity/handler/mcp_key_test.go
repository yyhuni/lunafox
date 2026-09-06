package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	identityapp "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

type mcpKeyHandlerStore struct {
	key        *identitydomain.MCPKey
	byDigest   map[string]*identitydomain.MCPKey
	replaces   int
	lastUser   int
	lastDigest string
}

func (store *mcpKeyHandlerStore) GetByUserID(context.Context, int) (*identitydomain.MCPKey, error) {
	if store.key == nil {
		return nil, identitydomain.ErrMCPKeyNotFound
	}
	copyKey := *store.key
	return &copyKey, nil
}

func (store *mcpKeyHandlerStore) GetByDigest(_ context.Context, digest string) (*identitydomain.MCPKey, error) {
	key, ok := store.byDigest[digest]
	if !ok {
		return nil, identitydomain.ErrMCPKeyNotFound
	}
	copyKey := *key
	return &copyKey, nil
}

func (store *mcpKeyHandlerStore) Replace(_ context.Context, userID int, digest string) (*identitydomain.MCPKey, error) {
	store.replaces++
	store.lastUser = userID
	store.lastDigest = digest
	now := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC).Add(time.Duration(store.replaces) * time.Minute)
	if store.key == nil {
		store.key = &identitydomain.MCPKey{ID: 17, UserID: userID, CreatedAt: now}
	} else {
		delete(store.byDigest, store.key.KeyDigest)
	}
	store.key.KeyDigest = digest
	store.key.UpdatedAt = now
	if store.byDigest == nil {
		store.byDigest = make(map[string]*identitydomain.MCPKey)
	}
	store.byDigest[digest] = store.key
	copyKey := *store.key
	return &copyKey, nil
}

type mcpKeyHandlerGenerator struct{ secrets []string }

func (generator *mcpKeyHandlerGenerator) Generate() (string, string, error) {
	if len(generator.secrets) == 0 {
		return "", "", errors.New("no secret")
	}
	secret := generator.secrets[0]
	generator.secrets = generator.secrets[1:]
	return secret, "digest:" + secret, nil
}

func (generator *mcpKeyHandlerGenerator) Digest(secret string) string { return "digest:" + secret }

func newMCPKeyHandlerTest(t *testing.T, store *mcpKeyHandlerStore, generator *mcpKeyHandlerGenerator) (*UserHandler, *identityapp.MCPKeyLifecycleService) {
	t.Helper()
	keyService := identityapp.NewMCPKeyLifecycleService(store, generator)
	return NewUserHandler(nil, keyService), keyService
}

func performMCPKeyHandlerRequest(t *testing.T, method, route string, handler gin.HandlerFunc, claims *auth.Claims) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if claims != nil {
		manager := auth.NewJWTManager("test-secret-key-32-chars-long!!", 15*time.Minute, time.Hour)
		router.Use(middleware.AuthMiddleware(manager, handlerTokenVersionReader{version: claims.TokenVersion}))
		token, _, err := manager.GenerateAccessToken(claims.UserID, claims.Username, claims.TokenVersion)
		if err != nil {
			t.Fatalf("generate access token: %v", err)
		}
		router.Handle(method, route, handler)
		req := httptest.NewRequest(method, route, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder
	}
	router.Handle(method, route, handler)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(method, route, nil))
	return recorder
}

func TestUserHandlerMCPKeyActionsRequireJWT(t *testing.T) {
	handler := NewUserHandler(nil)
	for _, test := range []struct {
		name   string
		method string
		route  string
		handle gin.HandlerFunc
	}{
		{name: "status", method: http.MethodGet, route: "/v1/users/me/mcpKey", handle: handler.GetCurrentMCPKey},
		{name: "generate", method: http.MethodPost, route: "/v1/users/me:generateMcpKey", handle: handler.GenerateCurrentMCPKey},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := performMCPKeyHandlerRequest(t, test.method, test.route, test.handle, nil)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestUserHandlerMCPKeyStatusOmitsSecretMaterial(t *testing.T) {
	store := &mcpKeyHandlerStore{}
	handler, keyService := newMCPKeyHandlerTest(t, store, &mcpKeyHandlerGenerator{secrets: []string{"lf_mcp_test"}})
	claims := &auth.Claims{UserID: 7, Username: "alice"}

	recorder := performMCPKeyHandlerRequest(t, http.MethodGet, "/v1/users/me/mcpKey", handler.GetCurrentMCPKey, claims)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"configured":false`) {
		t.Fatalf("unconfigured status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "lf_mcp_test") || strings.Contains(recorder.Body.String(), "digest") {
		t.Fatalf("unconfigured status leaked secret material: %s", recorder.Body.String())
	}

	if _, err := keyService.Generate(context.Background(), claims.UserID); err != nil {
		t.Fatalf("seed key: %v", err)
	}
	recorder = performMCPKeyHandlerRequest(t, http.MethodGet, "/v1/users/me/mcpKey", handler.GetCurrentMCPKey, claims)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"configured":true`) {
		t.Fatalf("configured status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "lf_mcp_test") || strings.Contains(recorder.Body.String(), "digest") || strings.Contains(recorder.Body.String(), `"key"`) {
		t.Fatalf("configured status leaked secret material: %s", recorder.Body.String())
	}
}

func TestUserHandlerMCPKeyGenerationReturnsOneTimeSecretAndRotates(t *testing.T) {
	store := &mcpKeyHandlerStore{}
	handler, keyService := newMCPKeyHandlerTest(t, store, &mcpKeyHandlerGenerator{secrets: []string{"lf_mcp_first", "lf_mcp_second"}})
	claims := &auth.Claims{UserID: 7, Username: "alice"}

	first := performMCPKeyHandlerRequest(t, http.MethodPost, "/v1/users/me:generateMcpKey", handler.GenerateCurrentMCPKey, claims)
	if first.Code != http.StatusOK || !strings.Contains(first.Body.String(), `"key":"lf_mcp_first"`) {
		t.Fatalf("first generation = %d body=%s", first.Code, first.Body.String())
	}
	if strings.Contains(first.Body.String(), "digest:") || store.lastUser != claims.UserID || store.lastDigest != "digest:lf_mcp_first" {
		t.Fatalf("first generation persisted wrong material: body=%s store=%+v", first.Body.String(), store)
	}

	second := performMCPKeyHandlerRequest(t, http.MethodPost, "/v1/users/me:generateMcpKey", handler.GenerateCurrentMCPKey, claims)
	if second.Code != http.StatusOK || !strings.Contains(second.Body.String(), `"key":"lf_mcp_second"`) {
		t.Fatalf("rotation = %d body=%s", second.Code, second.Body.String())
	}
	if _, err := keyService.Authenticate(context.Background(), "lf_mcp_first"); !errors.Is(err, identitydomain.ErrMCPKeyNotFound) {
		t.Fatalf("old key authentication error = %v", err)
	}
	if _, err := keyService.Authenticate(context.Background(), "lf_mcp_second"); err != nil {
		t.Fatalf("new key authentication error = %v", err)
	}
}
