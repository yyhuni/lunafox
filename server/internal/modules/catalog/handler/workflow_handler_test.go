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

type workflowQueryStoreForHandlerStub struct {
	workflows []catalogapp.Workflow
	listErr   error
	getErr    error
}

func (stub *workflowQueryStoreForHandlerStub) ListWorkflows(_ context.Context) ([]catalogapp.Workflow, error) {
	if stub.listErr != nil {
		return nil, stub.listErr
	}
	out := make([]catalogapp.Workflow, len(stub.workflows))
	copy(out, stub.workflows)
	return out, nil
}

func (stub *workflowQueryStoreForHandlerStub) GetWorkflowByName(_ context.Context, name string) (*catalogapp.Workflow, error) {
	if stub.getErr != nil {
		return nil, stub.getErr
	}
	for i := range stub.workflows {
		if stub.workflows[i].Name == name {
			workflow := stub.workflows[i]
			return &workflow, nil
		}
	}
	return nil, catalogapp.ErrWorkflowNotFound
}

type workflowProfileQueryStoreForHandlerStub struct{}

func (stub *workflowProfileQueryStoreForHandlerStub) ListWorkflowProfiles(_ context.Context) ([]catalogapp.WorkflowProfile, error) {
	_ = stub
	return nil, nil
}

func (stub *workflowProfileQueryStoreForHandlerStub) GetWorkflowProfileByID(_ context.Context, _ string) (*catalogapp.WorkflowProfile, error) {
	_ = stub
	return nil, catalogapp.ErrWorkflowProfileNotFound
}

func newWorkflowHandlerForTest(store catalogapp.WorkflowQueryStore) *WorkflowHandler {
	facade := catalogapp.NewWorkflowCatalogFacade(store, &workflowProfileQueryStoreForHandlerStub{})
	return NewWorkflowHandler(facade)
}

func TestWorkflowHandlerListSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newWorkflowHandlerForTest(&workflowQueryStoreForHandlerStub{
		workflows: []catalogapp.Workflow{
			{
				Name:        "subdomain_discovery",
				Title:       "Subdomain Discovery",
				Description: "Discover subdomains",
			},
			{
				Name:        "port_scan",
				Title:       "Port Scan",
				Description: "Scan open ports",
			},
		},
	})

	router := gin.New()
	router.GET("/api/workflows", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var payload []catalogdto.WorkflowResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("expected 2 workflows, got %d", len(payload))
	}
	if payload[0].Name != "subdomain_discovery" || payload[1].Name != "port_scan" {
		t.Fatalf("unexpected workflows payload: %+v", payload)
	}
	if payload[0].Title != "Subdomain Discovery" || payload[0].Description != "Discover subdomains" {
		t.Fatalf("unexpected first workflow metadata: %+v", payload[0])
	}
	if payload[1].Title != "Port Scan" || payload[1].Description != "Scan open ports" {
		t.Fatalf("unexpected second workflow metadata: %+v", payload[1])
	}
}

func TestWorkflowHandlerListInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newWorkflowHandlerForTest(&workflowQueryStoreForHandlerStub{
		listErr: errors.New("boom"),
	})

	router := gin.New()
	router.GET("/api/workflows", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
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

func TestWorkflowHandlerGetByWorkflowNameSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newWorkflowHandlerForTest(&workflowQueryStoreForHandlerStub{
		workflows: []catalogapp.Workflow{
			{
				Name:        "subdomain_discovery",
				Title:       "Subdomain Discovery",
				Description: "Discover subdomains",
			},
		},
	})

	router := gin.New()
	router.GET("/api/workflows/:name", handler.GetByWorkflowName)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/subdomain_discovery", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var payload catalogdto.WorkflowResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if payload.Name != "subdomain_discovery" {
		t.Fatalf("unexpected workflow payload: %+v", payload)
	}
	if payload.Title != "Subdomain Discovery" || payload.Description != "Discover subdomains" {
		t.Fatalf("unexpected workflow metadata payload: %+v", payload)
	}
}

func TestWorkflowHandlerGetByWorkflowNameNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newWorkflowHandlerForTest(&workflowQueryStoreForHandlerStub{
		workflows: []catalogapp.Workflow{
			{Name: "subdomain_discovery"},
		},
	})

	router := gin.New()
	router.GET("/api/workflows/:name", handler.GetByWorkflowName)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/unknown_workflow", nil)
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

func TestWorkflowHandlerGetByWorkflowNameInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newWorkflowHandlerForTest(&workflowQueryStoreForHandlerStub{
		getErr: errors.New("db unavailable"),
	})

	router := gin.New()
	router.GET("/api/workflows/:name", handler.GetByWorkflowName)

	req := httptest.NewRequest(http.MethodGet, "/api/workflows/subdomain_discovery", nil)
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
