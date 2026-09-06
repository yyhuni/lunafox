package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type workflowManagementHandlerStore struct {
	workflows map[string]catalogdomain.ManagedScanWorkflow
}

func (store *workflowManagementHandlerStore) GetScanWorkflowByID(id string) (*catalogdomain.ManagedScanWorkflow, error) {
	workflow, ok := store.workflows[id]
	if !ok {
		return nil, catalogdomain.ErrScanWorkflowNotFound
	}
	return &workflow, nil
}

func (store *workflowManagementHandlerStore) ListScanWorkflows(_ catalogdomain.ScanWorkflowListFilter) ([]catalogdomain.ManagedScanWorkflow, int64, error) {
	workflows := make([]catalogdomain.ManagedScanWorkflow, 0, len(store.workflows))
	for _, workflow := range store.workflows {
		workflows = append(workflows, workflow)
	}
	return workflows, int64(len(workflows)), nil
}

func (store *workflowManagementHandlerStore) CreateScanWorkflow(workflow *catalogdomain.ManagedScanWorkflow) error {
	store.workflows[workflow.ScanWorkflowID] = *workflow
	return nil
}

func (store *workflowManagementHandlerStore) FindScanWorkflowByRequestID(string) (*catalogdomain.ManagedScanWorkflow, error) {
	return nil, nil
}

func (store *workflowManagementHandlerStore) UpdateUserScanWorkflow(*catalogdomain.ManagedScanWorkflow, int64) (bool, error) {
	return false, nil
}

type workflowManagementHandlerEngines map[string]bool

func (engines workflowManagementHandlerEngines) HasEngine(_ context.Context, engineID string) (bool, error) {
	return engines[engineID], nil
}

func TestScanWorkflowManagementHandlerListUsesCanonicalResponseAndRejectsInvalidPagination(t *testing.T) {
	handler := newWorkflowManagementHandlerForTest(t)
	router := gin.New()
	router.GET("/v1/scanWorkflows", handler.List)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/scanWorkflows?pageSize=1", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	results, ok := payload["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("list results = %#v", payload["results"])
	}
	workflow, ok := results[0].(map[string]any)
	if !ok || workflow["name"] != "scanWorkflows/default" || workflow["isBuiltin"] != true || workflow["isExecutable"] != true {
		t.Fatalf("unexpected workflow response: %#v", results[0])
	}
	if _, exists := workflow["configuration"]; exists {
		t.Fatalf("management workflow must not expose configuration: %#v", workflow)
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/scanWorkflows?pageSize=invalid", nil))
	assertWorkflowHandlerErrorReason(t, response, http.StatusBadRequest, "INVALID_ARGUMENT")
}

func TestScanWorkflowManagementHandlerRejectsBuiltinUpdateWithStructuredReason(t *testing.T) {
	handler := newWorkflowManagementHandlerForTest(t)
	router := gin.New()
	router.PATCH("/v1/scanWorkflows/:scanWorkflow", handler.Update)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/v1/scanWorkflows/default", http.NoBody)
	request.Header.Set("Content-Type", "application/json")
	request.Body = http.NoBody
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("empty body status = %d, want binding error", response.Code)
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPatch, "/v1/scanWorkflows/default", strings.NewReader(`{"scanWorkflow":{"name":"scanWorkflows/default","etag":"ignored"},"updateMask":["description"]}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)
	assertWorkflowHandlerErrorReason(t, response, http.StatusBadRequest, "BUILTIN_SCAN_WORKFLOW_IMMUTABLE")
}

func TestScanWorkflowManagementHandlerRequiresProfileDefaultEnabledOnCreate(t *testing.T) {
	store := &workflowManagementHandlerStore{workflows: map[string]catalogdomain.ManagedScanWorkflow{}}
	service, err := catalogapp.NewScanWorkflowManagementService(store, workflowManagementHandlerEngines{
		"engine.lunafox.discovery": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewScanWorkflowManagementHandler(service)
	router := gin.New()
	router.POST("/v1/scanWorkflows", handler.Create)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/scanWorkflows", strings.NewReader(`{"requestId":"00000000-0000-4000-8000-000000000001","scanWorkflow":{"displayName":"Discovery","description":"","stages":[{"stageId":"discovery","steps":[{"stepId":"discover","engineId":"engine.lunafox.discovery"}]}]}}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)
	assertWorkflowHandlerErrorReason(t, response, http.StatusBadRequest, "BAD_REQUEST")

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/v1/scanWorkflows", strings.NewReader(`{"requestId":"00000000-0000-4000-8000-000000000001","scanWorkflow":{"displayName":"Discovery","description":"","stages":[{"stageId":"discovery","steps":[{"stepId":"discover","engineId":"engine.lunafox.discovery","profileDefaultEnabled":false}]}]}}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("valid explicit false Create status = %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Stages []scanworkflow.Stage `json:"stages"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Stages) != 1 || payload.Stages[0].Steps[0].ProfileDefaultEnabled {
		t.Fatalf("Create response lost explicit false Profile default: %+v", payload.Stages)
	}
}

func newWorkflowManagementHandlerForTest(t *testing.T) *ScanWorkflowManagementHandler {
	t.Helper()
	stages := []scanworkflow.Stage{{StageID: "discovery", Steps: []scanworkflow.Step{{StepID: "discover", EngineID: "engine.lunafox.discovery", ProfileDefaultEnabled: true}}}}
	digest, err := scanworkflow.CanonicalWorkflowDigest("default", "Default", "", stages)
	if err != nil {
		t.Fatal(err)
	}
	service, err := catalogapp.NewScanWorkflowManagementService(&workflowManagementHandlerStore{workflows: map[string]catalogdomain.ManagedScanWorkflow{
		"default": {ScanWorkflowID: "default", DisplayName: "Default", Stages: stages, IsBuiltin: true, DefinitionDigest: digest, Version: 1},
	}}, workflowManagementHandlerEngines{"engine.lunafox.discovery": true})
	if err != nil {
		t.Fatal(err)
	}
	return NewScanWorkflowManagementHandler(service)
}

func assertWorkflowHandlerErrorReason(t *testing.T, response *httptest.ResponseRecorder, status int, reason string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d: %s", response.Code, status, response.Body.String())
	}
	var payload struct {
		Error struct {
			Details []struct {
				Reason string `json:"reason"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Error.Details) == 0 || payload.Error.Details[0].Reason != reason {
		t.Fatalf("error details = %#v, want reason %q", payload.Error.Details, reason)
	}
}
