package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	"gorm.io/gorm"
)

type authHandlerUserStoreStub struct {
	byID       map[int]*identitydomain.User
	byUsername map[string]*identitydomain.User
	errByID    error
	errByName  error
}

type handlerTokenVersionReader struct {
	version int
}

func (reader handlerTokenVersionReader) GetTokenVersion(context.Context, int) (int, error) {
	return reader.version, nil
}

func (stub *authHandlerUserStoreStub) GetAuthUserByID(id int) (*identitydomain.User, error) {
	if stub.errByID != nil {
		return nil, stub.errByID
	}
	user, ok := stub.byID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func (stub *authHandlerUserStoreStub) FindAuthUserByUsername(username string) (*identitydomain.User, error) {
	if stub.errByName != nil {
		return nil, stub.errByName
	}
	user, ok := stub.byUsername[username]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func newAuthHandlerForTest(store *authHandlerUserStoreStub, accessExpire, refreshExpire time.Duration) (*AuthHandler, *auth.JWTManager) {
	manager := auth.NewJWTManager("test-secret-key-32-chars-long!!", accessExpire, refreshExpire)
	return NewAuthHandler(service.NewAuthFacade(service.NewAuthCommandService(store, service.NewAuthPasswordVerifier(), manager))), manager
}

func performAuthRequest(t *testing.T, route string, handler gin.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST(route, handler)

	req := httptest.NewRequest(http.MethodPost, route, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestAuthHandlerLogin(t *testing.T) {
	hashedPassword, err := auth.HashPassword("pass123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	t.Run("success returns token pair and user info", func(t *testing.T) {
		handler, _ := newAuthHandlerForTest(&authHandlerUserStoreStub{
			byUsername: map[string]*identitydomain.User{
				"alice": {
					ID:       7,
					Username: "alice",
					Email:    "alice@example.com",
					IsActive: true,
					Password: hashedPassword,
				},
			},
		}, 15*time.Minute, time.Hour)

		recorder := performAuthRequest(t, "/v1/sessions", handler.Login, `{"username":"alice","password":"pass123"}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body LoginResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal login response: %v", err)
		}
		if body.AccessToken == "" || body.RefreshToken == "" || body.ExpiresIn != 900 {
			t.Fatalf("unexpected login tokens: %+v", body)
		}
		if body.User.ID != 7 || body.User.Username != "alice" || body.User.Email != "alice@example.com" {
			t.Fatalf("unexpected login user: %+v", body.User)
		}
	})

	t.Run("invalid credentials map to unauthorized", func(t *testing.T) {
		handler, _ := newAuthHandlerForTest(&authHandlerUserStoreStub{
			byUsername: map[string]*identitydomain.User{},
		}, 15*time.Minute, time.Hour)

		recorder := performAuthRequest(t, "/v1/sessions", handler.Login, `{"username":"ghost","password":"pass123"}`)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"UNAUTHORIZED"`) || !strings.Contains(recorder.Body.String(), `"message":"Invalid username or password"`) {
			t.Fatalf("unexpected unauthorized response: %s", recorder.Body.String())
		}
	})

	t.Run("missing required field fast-fails with validation error", func(t *testing.T) {
		handler, _ := newAuthHandlerForTest(&authHandlerUserStoreStub{}, 15*time.Minute, time.Hour)

		recorder := performAuthRequest(t, "/v1/sessions", handler.Login, `{"username":"alice"}`)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"VALIDATION_ERROR"`) || !strings.Contains(recorder.Body.String(), `"field":"Password"`) {
			t.Fatalf("unexpected validation response: %s", recorder.Body.String())
		}
	})

	t.Run("disabled user maps to unauthorized", func(t *testing.T) {
		handler, _ := newAuthHandlerForTest(&authHandlerUserStoreStub{
			byUsername: map[string]*identitydomain.User{
				"alice": {
					ID:       7,
					Username: "alice",
					IsActive: false,
					Password: hashedPassword,
				},
			},
		}, 15*time.Minute, time.Hour)

		recorder := performAuthRequest(t, "/v1/sessions", handler.Login, `{"username":"alice","password":"pass123"}`)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"message":"User account is disabled"`) {
			t.Fatalf("unexpected disabled-user response: %s", recorder.Body.String())
		}
	})

	t.Run("unexpected upstream error maps to internal error", func(t *testing.T) {
		handler, _ := newAuthHandlerForTest(&authHandlerUserStoreStub{
			errByName: errors.New("db unavailable"),
		}, 15*time.Minute, time.Hour)

		recorder := performAuthRequest(t, "/v1/sessions", handler.Login, `{"username":"alice","password":"pass123"}`)
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"INTERNAL_ERROR"`) || !strings.Contains(recorder.Body.String(), `"message":"Failed to login"`) {
			t.Fatalf("unexpected internal-error response: %s", recorder.Body.String())
		}
	})
}

func TestAuthHandlerGetCurrentUser(t *testing.T) {
	t.Run("missing claims maps to unauthorized", func(t *testing.T) {
		handler, _ := newAuthHandlerForTest(&authHandlerUserStoreStub{}, 15*time.Minute, time.Hour)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.GET("/v1/users/current", handler.GetCurrentUser)

		req := httptest.NewRequest(http.MethodGet, "/v1/users/current", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"message":"Not authenticated"`) {
			t.Fatalf("unexpected unauthorized response: %s", recorder.Body.String())
		}
	})

	t.Run("success and public error mappings remain stable", func(t *testing.T) {
		manager := auth.NewJWTManager("test-secret-key-32-chars-long!!", 15*time.Minute, time.Hour)
		accessToken, _, err := manager.GenerateAccessToken(7, "alice", 0)
		if err != nil {
			t.Fatalf("generate access token: %v", err)
		}

		runRequest := func(store *authHandlerUserStoreStub) *httptest.ResponseRecorder {
			handler := NewAuthHandler(service.NewAuthFacade(service.NewAuthCommandService(store, service.NewAuthPasswordVerifier(), manager)))
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(middleware.AuthMiddleware(manager, handlerTokenVersionReader{}))
			router.GET("/v1/users/current", handler.GetCurrentUser)

			req := httptest.NewRequest(http.MethodGet, "/v1/users/current", nil)
			req.Header.Set("Authorization", "Bearer "+accessToken)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			return recorder
		}

		recorder := runRequest(&authHandlerUserStoreStub{
			byID: map[int]*identitydomain.User{
				7: {ID: 7, Username: "alice", Email: "alice@example.com"},
			},
		})
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		var body UserInfo
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal current user response: %v", err)
		}
		if body.ID != 7 || body.Username != "alice" || body.Email != "alice@example.com" {
			t.Fatalf("unexpected current user response: %+v", body)
		}

		recorder = runRequest(&authHandlerUserStoreStub{byID: map[int]*identitydomain.User{}})
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"NOT_FOUND"`) || !strings.Contains(recorder.Body.String(), `"message":"User not found"`) {
			t.Fatalf("unexpected not-found response: %s", recorder.Body.String())
		}

		recorder = runRequest(&authHandlerUserStoreStub{errByID: errors.New("user lookup unavailable")})
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"INTERNAL_ERROR"`) || !strings.Contains(recorder.Body.String(), `"message":"Failed to get current user"`) {
			t.Fatalf("unexpected internal-error response: %s", recorder.Body.String())
		}
	})
}

func TestAuthHandlerRefreshToken(t *testing.T) {
	t.Run("success returns a new access token", func(t *testing.T) {
		handler, manager := newAuthHandlerForTest(&authHandlerUserStoreStub{
			byID: map[int]*identitydomain.User{9: {ID: 9, Username: "bob", TokenVersion: 0}},
		}, 15*time.Minute, time.Hour)
		tokenPair, err := manager.GenerateTokenPair(9, "bob", 0)
		if err != nil {
			t.Fatalf("generate token pair: %v", err)
		}

		recorder := performAuthRequest(t, "/v1/sessions:renew", handler.RefreshToken, `{"refreshToken":"`+tokenPair.RefreshToken+`"}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body RefreshResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal refresh response: %v", err)
		}
		if body.AccessToken == "" || body.ExpiresIn != 900 {
			t.Fatalf("unexpected refresh response: %+v", body)
		}
	})

	t.Run("expired refresh token maps to unauthorized", func(t *testing.T) {
		handler, manager := newAuthHandlerForTest(&authHandlerUserStoreStub{
			byID: map[int]*identitydomain.User{9: {ID: 9, Username: "bob", TokenVersion: 0}},
		}, 15*time.Minute, time.Millisecond)
		tokenPair, err := manager.GenerateTokenPair(9, "bob", 0)
		if err != nil {
			t.Fatalf("generate token pair: %v", err)
		}
		time.Sleep(20 * time.Millisecond)

		recorder := performAuthRequest(t, "/v1/sessions:renew", handler.RefreshToken, `{"refreshToken":"`+tokenPair.RefreshToken+`"}`)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"UNAUTHORIZED"`) || !strings.Contains(recorder.Body.String(), `"message":"Invalid or expired refresh token"`) {
			t.Fatalf("unexpected expired-token response: %s", recorder.Body.String())
		}
	})
}
