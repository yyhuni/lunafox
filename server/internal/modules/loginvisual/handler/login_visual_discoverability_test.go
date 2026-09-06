package handler_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/handler"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/router"
)

func TestDiscoverabilityRoutesAreAuthenticatedAndIdempotent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &discoverabilityService{}
	engine := gin.New()
	api := engine.Group("/v1")
	protected := api.Group("")
	manager := auth.NewJWTManager("test-login-visual-secret-32-chars", time.Minute, time.Hour)
	protected.Use(middleware.AuthMiddleware(manager, discoverabilityTokenVersionReader{}))
	router.RegisterLoginVisualRoutes(api, protected, handler.NewLoginVisualHandler(service))
	token, _, err := manager.GenerateAccessToken(7, "operator", 1)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	request := func(method, path string, token string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		httpRequest := httptest.NewRequest(method, path, nil)
		if token != "" {
			httpRequest.Header.Set("Authorization", "Bearer "+token)
		}
		engine.ServeHTTP(recorder, httpRequest)
		return recorder
	}

	if response := request(http.MethodGet, "/v1/settings/loginVisual:checkDiscoverability", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated check status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if response := request(http.MethodGet, "/v1/settings/loginVisual:checkDiscoverability", token); response.Code != http.StatusOK || response.Body.String() != "{\"unlocked\":false}" {
		t.Fatalf("initial check = status %d body %s", response.Code, response.Body.String())
	}
	for range 2 {
		if response := request(http.MethodPost, "/v1/settings/loginVisual:unlockDiscoverability", token); response.Code != http.StatusOK || response.Body.String() != "{\"unlocked\":true}" {
			t.Fatalf("unlock = status %d body %s", response.Code, response.Body.String())
		}
	}
	if response := request(http.MethodGet, "/v1/settings/loginVisual:checkDiscoverability", token); response.Code != http.StatusOK || response.Body.String() != "{\"unlocked\":true}" {
		t.Fatalf("persisted check = status %d body %s", response.Code, response.Body.String())
	}
	if service.checkUserID != 7 || service.unlockCalls != 2 || service.unlockUserID != 7 {
		t.Fatalf("service calls = %#v", service)
	}
}

type discoverabilityTokenVersionReader struct{}

func (discoverabilityTokenVersionReader) GetTokenVersion(context.Context, int) (int, error) {
	return 1, nil
}

type discoverabilityService struct {
	publicOnlyService
	unlocked     bool
	checkUserID  int
	unlockUserID int
	unlockCalls  int
}

func (service *discoverabilityService) IsDiscoverabilityUnlocked(_ context.Context, userID int) (bool, error) {
	service.checkUserID = userID
	return service.unlocked, nil
}

func (service *discoverabilityService) UnlockDiscoverability(_ context.Context, userID int) error {
	service.unlockUserID = userID
	service.unlockCalls++
	service.unlocked = true
	return nil
}

func (discoverabilityService) OpenPublished(context.Context, bool) (domain.Media, io.ReadCloser, error) {
	return domain.Media{}, nil, nil
}
