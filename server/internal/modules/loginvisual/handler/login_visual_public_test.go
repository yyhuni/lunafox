package handler_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/handler"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/router"
)

func TestPublicCurrentReturnsBuiltInWhenNoPublishedVisualExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/v1")
	protected := api.Group("")
	router.RegisterLoginVisualRoutes(api, protected, handler.NewLoginVisualHandler(publicOnlyService{}))

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/loginVisual/current", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "{\"kind\":\"builtIn\"}" {
		t.Fatalf("body = %s", response.Body.String())
	}
}

type publicOnlyService struct{}

func (publicOnlyService) IsDiscoverabilityUnlocked(context.Context, int) (bool, error) {
	return false, nil
}
func (publicOnlyService) UnlockDiscoverability(context.Context, int) error { return nil }
func (publicOnlyService) GetSettings(context.Context, int) (domain.Settings, error) {
	return domain.Settings{}, nil
}
func (publicOnlyService) Upload(context.Context, int, []byte) (domain.Settings, error) {
	return domain.Settings{}, nil
}
func (publicOnlyService) Publish(context.Context, int) (domain.Settings, error) {
	return domain.Settings{}, nil
}
func (publicOnlyService) RestoreDefault(context.Context, int) (domain.Settings, error) {
	return domain.Settings{}, nil
}
func (publicOnlyService) Public(context.Context) (domain.PublicVisual, error) {
	return domain.PublicVisual{}, nil
}
func (publicOnlyService) OpenPublished(context.Context, bool) (domain.Media, io.ReadCloser, error) {
	return domain.Media{}, nil, nil
}
func (publicOnlyService) OpenPreview(context.Context, int) (domain.Media, io.ReadCloser, error) {
	return domain.Media{}, nil, nil
}
