package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/application"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/handler"
)

type wireServiceStub struct{}

func (wireServiceStub) CurrentSource(context.Context) (*domain.Source, error) { return nil, nil }
func (wireServiceStub) CreateSync(context.Context, app.CreateSyncInput) (*domain.SyncTask, error) {
	return nil, nil
}
func (wireServiceStub) GetSyncTask(context.Context, uuid.UUID) (*domain.SyncTask, error) {
	return nil, nil
}
func (wireServiceStub) CancelSync(_ context.Context, id uuid.UUID) (*domain.SyncTask, error) {
	return &domain.SyncTask{ID: id, RequestID: uuid.New(), SourceType: domain.SourceTypeGit, State: domain.SyncTaskCancelling, Phase: domain.SyncTaskCancelling, Diagnostics: domain.Diagnostics{Samples: []domain.DiagnosticSample{}}}, nil
}
func (wireServiceStub) List(context.Context, app.POCListQuery) (*app.POCListResult, error) {
	return &app.POCListResult{}, nil
}
func (wireServiceStub) ListFilterOptions(context.Context, string) ([]domain.FilterOption, error) {
	return nil, nil
}
func (wireServiceStub) Get(context.Context, string) (*domain.POC, error) { return nil, nil }
func (wireServiceStub) UpdateEnabled(context.Context, string, bool) (*domain.POC, error) {
	return nil, nil
}
func (wireServiceStub) SetActivation(context.Context, app.SetPOCActivationInput) (*app.SetPOCActivationResult, error) {
	return &app.SetPOCActivationResult{}, nil
}

func TestRegisterNucleiPOCRoutesExposesOnlyReplacementBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")
	RegisterNucleiPOCRoutes(protected, handler.NewNucleiPOCHandler(nil))
	routes := engine.Routes()
	seen := make(map[string]string, len(routes))
	for _, route := range routes {
		seen[route.Method+" "+route.Path] = route.Handler
	}
	for _, expected := range []string{
		http.MethodPost + " /v1/nucleiPocSources:sync",
		http.MethodGet + " /v1/nucleiPocSources/current",
		http.MethodGet + " /v1/nucleiPocSyncTasks/:task",
		http.MethodPost + " /v1/nucleiPocSyncTasks/:task",
		http.MethodGet + " /v1/nucleiPocs",
		http.MethodGet + " /v1/nucleiPocs/filterOptions",
		http.MethodPost + " /v1/nucleiPocs:setActivation",
		http.MethodGet + " /v1/nucleiPocs/:nucleiPoc",
		http.MethodPatch + " /v1/nucleiPocs/:nucleiPoc",
	} {
		if _, ok := seen[expected]; !ok {
			t.Fatalf("missing route %s", expected)
		}
	}
	for route := range seen {
		if len(route) >= len("/nuclei/repos") && route[len(route)-len("/nuclei/repos"):] == "/nuclei/repos" {
			t.Fatalf("legacy route registered: %s", route)
		}
	}
}

func TestRegisterNucleiPOCRoutesKeepsCollectionActivationUnderProtectedGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")
	protected.Use(func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	})
	RegisterNucleiPOCRoutes(protected, handler.NewNucleiPOCHandler(nil))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/nucleiPocs:setActivation", strings.NewReader(`{"enabled":true}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated collection activation status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRegisterNucleiPOCRoutesHitsCancelHandlerForColonAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")
	RegisterNucleiPOCRoutes(protected, handler.NewNucleiPOCHandler(wireServiceStub{}))
	taskID := "00000000-0000-4000-8000-000000000001"
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/nucleiPocSyncTasks/"+taskID+":cancel", nil)
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"state":"CANCELLING"`) {
		t.Fatalf("status=%d body=%s, want cancel handler response", recorder.Code, recorder.Body.String())
	}

	// The same wildcard route must not silently treat a plain POST task path as
	// a cancellation command.
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/v1/nucleiPocSyncTasks/"+taskID, nil)
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "task action must be :cancel") {
		t.Fatalf("plain POST status=%d body=%s, want explicit action validation", recorder.Code, recorder.Body.String())
	}
}
