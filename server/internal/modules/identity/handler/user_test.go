package handler

import (
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
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
	"gorm.io/gorm"
)

type userHandlerStoreStub struct {
	userByID          map[int]*identitydomain.User
	users             []identitydomain.User
	total             int64
	usernameExists    map[string]bool
	createdUser       *identitydomain.User
	updatedUser       *identitydomain.User
	getErr            error
	listErr           error
	existsByNameErr   error
	createErr         error
	updateErr         error
	forceCreateUserID int
	lastListPage      int
	lastListPageSize  int
}

func (stub *userHandlerStoreStub) GetUserByID(id int) (*identitydomain.User, error) {
	if stub.getErr != nil {
		return nil, stub.getErr
	}
	user, ok := stub.userByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func (stub *userHandlerStoreStub) ExistsByUsername(username string) (bool, error) {
	if stub.existsByNameErr != nil {
		return false, stub.existsByNameErr
	}
	return stub.usernameExists[username], nil
}

func (stub *userHandlerStoreStub) Create(user *identitydomain.User) error {
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

func (stub *userHandlerStoreStub) Update(user *identitydomain.User) error {
	if stub.updateErr != nil {
		return stub.updateErr
	}
	copyUser := *user
	stub.updatedUser = &copyUser
	return nil
}

func (stub *userHandlerStoreStub) List(page, pageSize int) ([]identitydomain.User, int64, error) {
	stub.lastListPage = page
	stub.lastListPageSize = pageSize
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	result := make([]identitydomain.User, len(stub.users))
	copy(result, stub.users)
	return result, stub.total, nil
}

func newUserHandlerForTest(store *userHandlerStoreStub) *UserHandler {
	return NewUserHandler(service.NewUserFacade(service.NewUserQueryService(store), service.NewUserCommandService(store, service.NewAuthPasswordHasher())))
}

func performUpdatePasswordRequest(t *testing.T, handler gin.HandlerFunc, body string, claims *auth.Claims) *httptest.ResponseRecorder {
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

		req := httptest.NewRequest(http.MethodPost, "/v1/users/me:changePassword", strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()
		router.POST("/v1/users/me:changePassword", handler)
		router.ServeHTTP(recorder, req)
		return recorder
	}

	router.POST("/v1/users/me:changePassword", handler)
	req := httptest.NewRequest(http.MethodPost, "/v1/users/me:changePassword", strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestNewUserHandler(t *testing.T) {
	svc := &service.UserFacade{}

	handler := NewUserHandler(svc)

	if handler == nil {
		t.Fatal("expected handler instance")
	}
	if handler.svc != svc {
		t.Fatal("expected service to be stored on handler")
	}
}

func TestUserHandlerCreateUser(t *testing.T) {
	t.Run("success returns created user payload", func(t *testing.T) {
		store := &userHandlerStoreStub{
			usernameExists:    map[string]bool{},
			forceCreateUserID: 42,
		}
		handler := newUserHandlerForTest(store)

		recorder := performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/users",
			"/v1/users",
			handler.CreateUser,
			`{"username":"  bob  ","password":"pass123","email":"BOB@Example.com"}`,
		)

		if recorder.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body dto.UserResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal create user response: %v", err)
		}
		if body.ID != 42 || body.Username != "bob" || body.Email != "bob@example.com" || !body.IsActive {
			t.Fatalf("unexpected created user response: %+v", body)
		}
		if store.createdUser == nil || !auth.VerifyPassword("pass123", store.createdUser.Password) {
			t.Fatalf("expected created user persisted with hashed password, got %+v", store.createdUser)
		}
	})

	t.Run("duplicate username maps to bad request", func(t *testing.T) {
		handler := newUserHandlerForTest(&userHandlerStoreStub{
			usernameExists: map[string]bool{"alice": true},
		})

		recorder := performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/users",
			"/v1/users",
			handler.CreateUser,
			`{"username":"alice","password":"pass123","email":"alice@example.com"}`,
		)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"BAD_REQUEST"`) || !strings.Contains(recorder.Body.String(), `"message":"Username already exists"`) {
			t.Fatalf("unexpected bad-request response: %s", recorder.Body.String())
		}
	})

	t.Run("missing required field fast-fails before service access", func(t *testing.T) {
		handler := NewUserHandler(nil)

		recorder := performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/users",
			"/v1/users",
			handler.CreateUser,
			`{"password":"pass123"}`,
		)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"VALIDATION_ERROR"`) || !strings.Contains(recorder.Body.String(), `"field":"Username"`) {
			t.Fatalf("unexpected validation response: %s", recorder.Body.String())
		}
	})

	t.Run("unexpected facade error maps to internal error", func(t *testing.T) {
		handler := newUserHandlerForTest(&userHandlerStoreStub{
			usernameExists: map[string]bool{},
			createErr:      errors.New("insert failed"),
		})

		recorder := performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/users",
			"/v1/users",
			handler.CreateUser,
			`{"username":"alice","password":"pass123","email":"alice@example.com"}`,
		)

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"INTERNAL_ERROR"`) || !strings.Contains(recorder.Body.String(), `"message":"Failed to create user"`) {
			t.Fatalf("unexpected internal-error response: %s", recorder.Body.String())
		}
	})
}

func TestUserHandlerList(t *testing.T) {
	t.Run("success returns paginated users and shared default pagination", func(t *testing.T) {
		lastLogin := time.Date(2026, 3, 19, 15, 4, 5, 0, time.UTC)
		store := &userHandlerStoreStub{
			users: []identitydomain.User{
				{
					ID:          7,
					Username:    "alice",
					Email:       "alice@example.com",
					IsActive:    true,
					IsSuperuser: true,
					DateJoined:  time.Date(2026, 3, 18, 8, 0, 0, 0, time.UTC),
					LastLogin:   &lastLogin,
				},
			},
			total: 1,
		}
		handler := newUserHandlerForTest(store)

		recorder := performOrganizationRequest(
			t,
			http.MethodGet,
			"/v1/users",
			"/v1/users?pageSize=0",
			handler.List,
			"",
		)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body httpdto.PaginatedResponse[dto.UserResponse]
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal list users response: %v", err)
		}
		if body.TotalSize != 1 || body.NextPageToken != "" {
			t.Fatalf("unexpected pagination metadata: %+v", body)
		}
		if len(body.Results) != 1 || body.Results[0].ID != 7 || body.Results[0].Username != "alice" || body.Results[0].Email != "alice@example.com" || !body.Results[0].IsSuperuser {
			t.Fatalf("unexpected paginated results: %+v", body.Results)
		}
		if body.Results[0].LastLogin == nil || !body.Results[0].LastLogin.Equal(lastLogin) {
			t.Fatalf("expected last login preserved, got %+v", body.Results[0].LastLogin)
		}
		if store.lastListPage != 1 || store.lastListPageSize != 20 {
			t.Fatalf("expected default pagination sent to facade, got page=%d pageSize=%d", store.lastListPage, store.lastListPageSize)
		}
	})

	t.Run("unexpected facade error maps to internal error", func(t *testing.T) {
		handler := newUserHandlerForTest(&userHandlerStoreStub{
			listErr: errors.New("query failed"),
		})

		recorder := performOrganizationRequest(
			t,
			http.MethodGet,
			"/v1/users",
			"/v1/users?pageToken=cGFnZToy&pageSize=5",
			handler.List,
			"",
		)

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"INTERNAL_ERROR"`) || !strings.Contains(recorder.Body.String(), `"message":"Failed to list users"`) {
			t.Fatalf("unexpected internal-error response: %s", recorder.Body.String())
		}
	})
}

func TestUserHandlerUpdateCurrentUserPassword(t *testing.T) {
	t.Run("missing claims maps to unauthorized", func(t *testing.T) {
		handler := NewUserHandler(nil)

		recorder := performUpdatePasswordRequest(
			t,
			handler.UpdateCurrentUserPassword,
			`{"oldPassword":"old-pass","newPassword":"new-pass"}`,
			nil,
		)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"UNAUTHORIZED"`) || !strings.Contains(recorder.Body.String(), `"message":"Not authenticated"`) {
			t.Fatalf("unexpected unauthorized response: %s", recorder.Body.String())
		}
	})

	t.Run("validation fast-fails before service access", func(t *testing.T) {
		handler := NewUserHandler(nil)

		recorder := performUpdatePasswordRequest(
			t,
			handler.UpdateCurrentUserPassword,
			`{"oldPassword":"old-pass"}`,
			&auth.Claims{UserID: 9, Username: "alice"},
		)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"VALIDATION_ERROR"`) || !strings.Contains(recorder.Body.String(), `"field":"NewPassword"`) {
			t.Fatalf("unexpected validation response: %s", recorder.Body.String())
		}
	})

	t.Run("success and public error mappings remain stable", func(t *testing.T) {
		validHash, err := auth.HashPassword("old-pass")
		if err != nil {
			t.Fatalf("hash old password: %v", err)
		}

		runRequest := func(store *userHandlerStoreStub, body string) *httptest.ResponseRecorder {
			handler := newUserHandlerForTest(store)
			return performUpdatePasswordRequest(
				t,
				handler.UpdateCurrentUserPassword,
				body,
				&auth.Claims{UserID: 7, Username: "alice"},
			)
		}

		successStore := &userHandlerStoreStub{
			userByID: map[int]*identitydomain.User{
				7: {ID: 7, Username: "alice", Password: validHash, IsActive: true},
			},
		}
		recorder := runRequest(successStore, `{"oldPassword":"old-pass","newPassword":"new-pass"}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"message":"Password updated"`) {
			t.Fatalf("unexpected success response: %s", recorder.Body.String())
		}
		if successStore.updatedUser == nil || !auth.VerifyPassword("new-pass", successStore.updatedUser.Password) {
			t.Fatalf("expected updated password hash stored, got %+v", successStore.updatedUser)
		}

		recorder = runRequest(&userHandlerStoreStub{
			userByID: map[int]*identitydomain.User{},
		}, `{"oldPassword":"old-pass","newPassword":"new-pass"}`)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"NOT_FOUND"`) || !strings.Contains(recorder.Body.String(), `"message":"User not found"`) {
			t.Fatalf("unexpected not-found response: %s", recorder.Body.String())
		}

		recorder = runRequest(&userHandlerStoreStub{
			userByID: map[int]*identitydomain.User{
				7: {ID: 7, Username: "alice", Password: validHash, IsActive: true},
			},
		}, `{"oldPassword":"wrong-pass","newPassword":"new-pass"}`)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"BAD_REQUEST"`) || !strings.Contains(recorder.Body.String(), `"message":"Invalid old password"`) {
			t.Fatalf("unexpected invalid-password response: %s", recorder.Body.String())
		}

		recorder = runRequest(&userHandlerStoreStub{
			userByID: map[int]*identitydomain.User{
				7: {ID: 7, Username: "alice", Password: validHash, IsActive: true},
			},
			updateErr: errors.New("write failed"),
		}, `{"oldPassword":"old-pass","newPassword":"new-pass"}`)
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"INTERNAL_ERROR"`) || !strings.Contains(recorder.Body.String(), `"message":"Failed to update password"`) {
			t.Fatalf("unexpected internal-error response: %s", recorder.Body.String())
		}
	})
}
