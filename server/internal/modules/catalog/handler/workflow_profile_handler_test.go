package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdto "github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

type workflowProfileQueryStoreForProfileHandlerStub struct {
	profiles []catalogapp.WorkflowProfile
	listErr  error
	getErr   error
}

const workflowProfileConfigFixture = "subdomain_discovery:\n" +
	"  recon:\n" +
	"    enabled: false\n" +
	"    tools:\n" +
	"      subfinder:\n" +
	"        enabled: false\n" +
	"  bruteforce:\n" +
	"    enabled: false\n" +
	"    tools:\n" +
	"      subdomain-bruteforce:\n" +
	"        enabled: false\n" +
	"  permutation:\n" +
	"    enabled: false\n" +
	"    tools:\n" +
	"      subdomain-permutation-resolve:\n" +
	"        enabled: false\n" +
	"  resolve:\n" +
	"    enabled: false\n" +
	"    tools:\n" +
	"      subdomain-resolve:\n" +
	"        enabled: false\n"

func (stub *workflowProfileQueryStoreForProfileHandlerStub) ListWorkflowProfiles(_ context.Context) ([]catalogapp.WorkflowProfile, error) {
	if stub.listErr != nil {
		return nil, stub.listErr
	}
	out := make([]catalogapp.WorkflowProfile, len(stub.profiles))
	copy(out, stub.profiles)
	return out, nil
}

func (stub *workflowProfileQueryStoreForProfileHandlerStub) GetWorkflowProfileByID(_ context.Context, id string) (*catalogapp.WorkflowProfile, error) {
	if stub.getErr != nil {
		return nil, stub.getErr
	}
	for i := range stub.profiles {
		if stub.profiles[i].ID == id {
			profile := stub.profiles[i]
			return &profile, nil
		}
	}
	return nil, catalogapp.ErrWorkflowProfileNotFound
}

func newWorkflowProfileHandlerForTest(store catalogapp.WorkflowProfileQueryStore) *WorkflowProfileHandler {
	facade := catalogapp.NewWorkflowCatalogFacade(&workflowQueryStoreForHandlerStub{}, store)
	return NewWorkflowProfileHandler(facade)
}

func TestWorkflowProfileHandlerListSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newWorkflowProfileHandlerForTest(&workflowProfileQueryStoreForProfileHandlerStub{
		profiles: []catalogapp.WorkflowProfile{
			{
				ID:            "subdomain_default",
				Name:          "Subdomain Discovery Default",
				Description:   "Default subdomain workflow profile",
				WorkflowIDs:   []string{"subdomain_discovery"},
				Configuration: workflowProfileConfigFixture,
			},
			{
				ID:            "subdomain_fast",
				Name:          "Subdomain Discovery Fast",
				Description:   "Fast profile",
				WorkflowIDs:   []string{"subdomain_discovery"},
				Configuration: workflowProfileConfigFixture,
			},
		},
	})

	router := gin.New()
	router.GET("/api/workflows/profiles", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/profiles", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var payload []catalogdto.ProfileResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(payload))
	}
	if payload[0].ID != "subdomain_default" || payload[1].ID != "subdomain_fast" {
		t.Fatalf("unexpected profiles payload: %+v", payload)
	}
	if len(payload[0].WorkflowIDs) != 1 || payload[0].WorkflowIDs[0] != "subdomain_discovery" {
		t.Fatalf("unexpected workflowIds in first profile payload: %+v", payload[0])
	}
}

func TestWorkflowProfileHandlerListInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newWorkflowProfileHandlerForTest(&workflowProfileQueryStoreForProfileHandlerStub{
		listErr: errors.New("load failed"),
	})

	router := gin.New()
	router.GET("/api/workflows/profiles", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/profiles", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var payload catalogdto.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if payload.Error.Code != "INTERNAL_ERROR" {
		t.Fatalf("expected INTERNAL_ERROR, got %s", payload.Error.Code)
	}
}

func TestWorkflowProfileHandlerGetByIDSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newWorkflowProfileHandlerForTest(&workflowProfileQueryStoreForProfileHandlerStub{
		profiles: []catalogapp.WorkflowProfile{
			{
				ID:            "subdomain_default",
				Name:          "Subdomain Discovery Default",
				Description:   "Default subdomain workflow profile",
				WorkflowIDs:   []string{"subdomain_discovery"},
				Configuration: workflowProfileConfigFixture,
			},
		},
	})

	router := gin.New()
	router.GET("/api/workflows/profiles/:id", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/profiles/subdomain_default", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var payload catalogdto.ProfileResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if payload.ID != "subdomain_default" || payload.Name == "" {
		t.Fatalf("unexpected profile payload: %+v", payload)
	}
	if len(payload.WorkflowIDs) != 1 || payload.WorkflowIDs[0] != "subdomain_discovery" {
		t.Fatalf("unexpected workflowIds in profile payload: %+v", payload)
	}
}

func TestWorkflowProfileHandlerGetByIDNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newWorkflowProfileHandlerForTest(&workflowProfileQueryStoreForProfileHandlerStub{
		profiles: []catalogapp.WorkflowProfile{
			{
				ID:            "subdomain_default",
				Name:          "Subdomain Discovery Default",
				Description:   "Default subdomain workflow profile",
				WorkflowIDs:   []string{"subdomain_discovery"},
				Configuration: workflowProfileConfigFixture,
			},
		},
	})

	router := gin.New()
	router.GET("/api/workflows/profiles/:id", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/profiles/not_exist", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var payload catalogdto.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if payload.Error.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %s", payload.Error.Code)
	}
}

func TestWorkflowProfileHandlerGetByIDInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newWorkflowProfileHandlerForTest(&workflowProfileQueryStoreForProfileHandlerStub{
		getErr: errors.New("storage unavailable"),
	})

	router := gin.New()
	router.GET("/api/workflows/profiles/:id", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/profiles/subdomain_default", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var payload catalogdto.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if payload.Error.Code != "INTERNAL_ERROR" {
		t.Fatalf("expected INTERNAL_ERROR, got %s", payload.Error.Code)
	}
}
