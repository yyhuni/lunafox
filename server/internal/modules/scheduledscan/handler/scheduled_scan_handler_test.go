package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/dto"
)

type scheduledScanHandlerStore struct {
	created        *scheduledapp.ScheduledScanCreate
	updated        *scheduledapp.ScheduledScanUpdate
	batchUpdates   []scheduledapp.ScheduledScanStatusUpdate
	batchErr       error
	items          []scheduledapp.ScheduledScan
	total          int64
	overviewQuery  *scheduledapp.ScheduledScanOverviewQuery
	overviewResult *scheduledapp.ScheduledScanOverviewProjection
}

type scheduledScanWorkflowStoreStub struct{}

type scheduledScanConfigResourceValidatorStub struct{ err error }

func (stub scheduledScanConfigResourceValidatorStub) Validate(context.Context, scanapp.WorkflowConfigResourceValidationRequest) error {
	return stub.err
}

func newScheduledScanServiceWithResourceFailureForHandlerTest(store *scheduledScanHandlerStore, err error) *scheduledapp.ScheduledScanService {
	return scheduledapp.NewScheduledScanService(store, scheduledScanWorkflowStoreStub{}).
		WithConfigResourceValidator(scheduledScanConfigResourceValidatorStub{err: err})
}

func (scheduledScanWorkflowStoreStub) GetScanWorkflowByID(id string) (*catalogdomain.ManagedScanWorkflow, error) {
	if id != "default" {
		return nil, scheduledapp.ErrScheduledScanNotFound
	}
	return &catalogdomain.ManagedScanWorkflow{
		ScanWorkflowID: "default",
		Stages: []scanworkflow.Stage{{
			StageID: "discovery",
			Steps:   []scanworkflow.Step{{StepID: "subdomain_discovery", EngineID: "engine.lunafox.subdomain_discovery"}},
		}},
	}, nil
}

func newScheduledScanServiceForHandlerTest(store *scheduledScanHandlerStore) *scheduledapp.ScheduledScanService {
	return scheduledapp.NewScheduledScanService(store, scheduledScanWorkflowStoreStub{}).
		WithConfigResourceValidator(scheduledScanConfigResourceValidatorStub{})
}

func (store *scheduledScanHandlerStore) Create(_ context.Context, scan *scheduledapp.ScheduledScanCreate) (*scheduledapp.ScheduledScan, error) {
	store.created = scan
	return scheduledScanRecord(1, scan.Name, scan.IsEnabled, scan.InputSource), nil
}

func (store *scheduledScanHandlerStore) Update(_ context.Context, id int, scan *scheduledapp.ScheduledScanUpdate) (*scheduledapp.ScheduledScan, error) {
	store.updated = scan
	name := "daily"
	if scan.Name != nil {
		name = *scan.Name
	}
	enabled := true
	if scan.IsEnabled != nil {
		enabled = *scan.IsEnabled
	}
	inputSource := scanapp.InputSourceScanSnapshot
	if scan.InputSource != nil {
		inputSource = *scan.InputSource
	}
	return scheduledScanRecord(id, name, enabled, inputSource), nil
}

func (store *scheduledScanHandlerStore) BatchUpdateStatus(_ context.Context, updates []scheduledapp.ScheduledScanStatusUpdate) (int, error) {
	store.batchUpdates = append([]scheduledapp.ScheduledScanStatusUpdate(nil), updates...)
	if store.batchErr != nil {
		return 0, store.batchErr
	}
	return len(updates), nil
}

func (store *scheduledScanHandlerStore) List(context.Context, scheduledapp.ScheduledScanListQuery) ([]scheduledapp.ScheduledScan, int64, error) {
	return store.items, store.total, nil
}

func (store *scheduledScanHandlerStore) GetOverviewSummary(_ context.Context, query scheduledapp.ScheduledScanOverviewQuery) (*scheduledapp.ScheduledScanOverviewProjection, error) {
	store.overviewQuery = &query
	if store.overviewResult == nil {
		return &scheduledapp.ScheduledScanOverviewProjection{UpcomingScheduledScans: []scheduledapp.ScheduledScanOverviewUpcoming{}}, nil
	}
	return store.overviewResult, nil
}

func (store *scheduledScanHandlerStore) GetByID(_ context.Context, id int) (*scheduledapp.ScheduledScan, error) {
	return scheduledScanRecord(id, "daily", true, scanapp.InputSourceScanSnapshot), nil
}

func (store *scheduledScanHandlerStore) Delete(context.Context, int) error {
	return nil
}

func TestScheduledScanListUsesResourceSpecificCollection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &scheduledScanHandlerStore{
		items: []scheduledapp.ScheduledScan{*scheduledScanRecord(1, "daily", true, scanapp.InputSourceScanSnapshot)},
		total: 1,
	}
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(store))
	recorder := performScheduledScanRequest(handler.List, http.MethodGet, "/v1/scheduledScans", nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := body["scheduledScans"]; !ok {
		t.Fatalf("expected scheduledScans collection, got %s", recorder.Body.String())
	}
	if _, ok := body["results"]; ok {
		t.Fatalf("results must not be used for scheduled scan list: %s", recorder.Body.String())
	}
}

func TestScheduledScanOverviewReturnsCompactCustomMethodResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &scheduledScanHandlerStore{overviewResult: &scheduledapp.ScheduledScanOverviewProjection{
		EnabledScheduledScanCount: 3,
		PausedScheduledScanCount:  2,
		UpcomingScheduledScans: []scheduledapp.ScheduledScanOverviewUpcoming{{
			ID: 7, DisplayName: "nightly", NextRunTime: time.Date(2026, 6, 15, 2, 0, 0, 0, time.UTC),
		}},
	}}
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(store).WithClock(func() time.Time {
		return time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC)
	}))
	recorder := performScheduledScanRequest(handler.Summarize, http.MethodGet, "/v1/scheduledScans:summarize", nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		AsOfTime                  time.Time `json:"asOfTime"`
		EnabledScheduledScanCount int64     `json:"enabledScheduledScanCount"`
		UpcomingScheduledScans    []struct {
			Name        string    `json:"name"`
			DisplayName string    `json:"displayName"`
			NextRunTime time.Time `json:"nextRunTime"`
		} `json:"upcomingScheduledScans"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode overview: %v", err)
	}
	if body.EnabledScheduledScanCount != 3 || len(body.UpcomingScheduledScans) != 1 {
		t.Fatalf("unexpected overview response: %+v", body)
	}
	if body.UpcomingScheduledScans[0].Name != "scheduledScans/7" || body.UpcomingScheduledScans[0].DisplayName != "nightly" {
		t.Fatalf("unexpected upcoming projection: %+v", body.UpcomingScheduledScans[0])
	}
}

func TestScheduledScanOverviewRejectsRetiredTimeZone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(&scheduledScanHandlerStore{}))
	recorder := performScheduledScanRequest(handler.Summarize, http.MethodGet, "/v1/scheduledScans:summarize?timeZone=Local", nil)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "INVALID_ARGUMENT") {
		t.Fatalf("expected INVALID_ARGUMENT, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestScheduledScanCreateReturnsResourceDirectly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &scheduledScanHandlerStore{}
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(store))
	body := `{"displayName":"daily","scanWorkflow":"scanWorkflows/default","inputSource":"targetInventory","configuration":{"steps":{"subdomain_discovery":{"enabled":true,"engineConfig":{"recon":{"enabled":true,"timeout":3600}}}}},"target":"targets/7","cronExpression":"0 2 * * *"}`
	recorder := performScheduledScanRequest(handler.Create, http.MethodPost, "/v1/scheduledScans", bytes.NewBufferString(body))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded["name"] != "scheduledScans/1" || decoded["displayName"] != "daily" {
		t.Fatalf("expected direct scheduled scan resource, got %+v", decoded)
	}
	if decoded["inputSource"] != "targetInventory" || store.created == nil || store.created.InputSource != scanapp.InputSourceTargetInventory {
		t.Fatalf("expected persisted targetInventory source, got response=%+v input=%+v", decoded, store.created)
	}
	if _, ok := decoded["message"]; ok {
		t.Fatalf("mutation response must not include message envelope: %+v", decoded)
	}
}

func TestScheduledScanCreateRejectsRetiredTimeZone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &scheduledScanHandlerStore{}
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(store))
	body := `{"displayName":"daily","scanWorkflow":"scanWorkflows/default","inputSource":"scanSnapshot","configuration":{"steps":{"subdomain_discovery":{"enabled":true,"engineConfig":{"recon":{"enabled":true,"timeout":3600}}}}},"target":"targets/7","timeZone":"UTC","cronExpression":"0 2 * * *"}`
	recorder := performScheduledScanRequest(handler.Create, http.MethodPost, "/v1/scheduledScans", bytes.NewBufferString(body))

	if recorder.Code != http.StatusBadRequest || store.created != nil {
		t.Fatalf("retired timeZone response = %d body=%s store=%+v", recorder.Code, recorder.Body.String(), store.created)
	}
}

func TestScheduledScanCreateMapsMissingStepEnablementToWorkflowDiagnostic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(&scheduledScanHandlerStore{}))
	body := `{"displayName":"daily","scanWorkflow":"scanWorkflows/default","inputSource":"scanSnapshot","configuration":{"steps":{"subdomain_discovery":{"engineConfig":{}}}},"target":"targets/7","cronExpression":"0 2 * * *"}`
	recorder := performScheduledScanRequest(handler.Create, http.MethodPost, "/v1/scheduledScans", bytes.NewBufferString(body))

	assertWorkflowConfigurationDiagnostic(t, recorder, `configuration.steps["subdomain_discovery"].enabled`)
}

func TestScheduledScanCreateMissingConfigurationUsesWorkflowDiagnostic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(&scheduledScanHandlerStore{}))
	body := `{"displayName":"daily","scanWorkflow":"scanWorkflows/default","inputSource":"scanSnapshot","target":"targets/7","cronExpression":"0 2 * * *"}`
	recorder := performScheduledScanRequest(handler.Create, http.MethodPost, "/v1/scheduledScans", bytes.NewBufferString(body))

	assertWorkflowConfigurationDiagnostic(t, recorder, "configuration.steps")
}

func TestScheduledScanConfigurationPatchUsesWorkflowDiagnostic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(&scheduledScanHandlerStore{}))
	body := `{"name":"scheduledScans/1","scanWorkflow":"scanWorkflows/default","configuration":{"steps":{"subdomain_discovery":{"enabled":true}}},"updateMask":"scanWorkflow,configuration"}`
	recorder := performScheduledScanRequest(handler.Update, http.MethodPatch, "/v1/scheduledScans/1", bytes.NewBufferString(body))

	assertWorkflowConfigurationDiagnostic(t, recorder, `configuration.steps["subdomain_discovery"].engineConfig`)
}

func TestScheduledScanUpdateRequiresUpdateMask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(&scheduledScanHandlerStore{}))
	body := `{"name":"scheduledScans/1","displayName":"daily"}`
	recorder := performScheduledScanRequest(handler.Update, http.MethodPatch, "/v1/scheduledScans/1", bytes.NewBufferString(body))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing updateMask, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestScheduledScanPatchIsEnabledUsesStandardUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &scheduledScanHandlerStore{}
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(store))
	body := `{"name":"scheduledScans/1","isEnabled":false,"updateMask":"isEnabled"}`
	recorder := performScheduledScanRequest(handler.Update, http.MethodPatch, "/v1/scheduledScans/1", bytes.NewBufferString(body))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.updated == nil || store.updated.IsEnabled == nil || *store.updated.IsEnabled {
		t.Fatalf("expected isEnabled update through standard update, got %+v", store.updated)
	}
	var decoded map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded["isEnabled"] != false {
		t.Fatalf("expected updated resource, got %+v", decoded)
	}
	if _, ok := decoded["message"]; ok {
		t.Fatalf("update response must not include message envelope: %+v", decoded)
	}
}

func TestScheduledScanBatchUpdateAssignsExplicitStatusToEveryRequestedSchedule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &scheduledScanHandlerStore{}
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(store))
	body := `{"requests":[{"name":"scheduledScans/2","isEnabled":true,"updateMask":"isEnabled"},{"name":"scheduledScans/1","isEnabled":true,"updateMask":"isEnabled"}]}`
	recorder := performScheduledScanRequest(handler.BatchUpdate, http.MethodPost, "/v1/scheduledScans:batchUpdate", bytes.NewBufferString(body))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(store.batchUpdates) != 2 || store.batchUpdates[0].ID != 1 || store.batchUpdates[1].ID != 2 {
		t.Fatalf("expected stable ID order, got %+v", store.batchUpdates)
	}
	for _, update := range store.batchUpdates {
		if !update.IsEnabled {
			t.Fatalf("expected explicit enabled state, got %+v", store.batchUpdates)
		}
	}
	var response dto.BatchUpdateScheduledScansResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.UpdatedCount != 2 {
		t.Fatalf("updatedCount = %d, want 2", response.UpdatedCount)
	}
}

func TestScheduledScanBatchUpdateRejectsInvalidItemsBeforePersistence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		body string
	}{
		{name: "empty requests", body: `{"requests":[]}`},
		{name: "too many requests", body: `{"requests":[` + strings.Repeat(`{"name":"scheduledScans/1","isEnabled":true,"updateMask":"isEnabled"},`, 100) + `{"name":"scheduledScans/101","isEnabled":true,"updateMask":"isEnabled"}]}`},
		{name: "unknown field", body: `{"requests":[{"name":"scheduledScans/1","isEnabled":true,"updateMask":"isEnabled","displayName":"invalid"}]}`},
		{name: "non-canonical name", body: `{"requests":[{"name":"scheduledScans/01","isEnabled":true,"updateMask":"isEnabled"}]}`},
		{name: "duplicate name", body: `{"requests":[{"name":"scheduledScans/1","isEnabled":true,"updateMask":"isEnabled"},{"name":"scheduledScans/1","isEnabled":false,"updateMask":"isEnabled"}]}`},
		{name: "missing enabled state", body: `{"requests":[{"name":"scheduledScans/1","updateMask":"isEnabled"}]}`},
		{name: "non-exact update mask", body: `{"requests":[{"name":"scheduledScans/1","isEnabled":true,"updateMask":"isEnabled,displayName"}]}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &scheduledScanHandlerStore{}
			handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(store))
			recorder := performScheduledScanRequest(handler.BatchUpdate, http.MethodPost, "/v1/scheduledScans:batchUpdate", bytes.NewBufferString(test.body))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
			}
			if len(store.batchUpdates) != 0 {
				t.Fatalf("invalid request reached persistence: %+v", store.batchUpdates)
			}
		})
	}
}

func TestScheduledScanBatchUpdateMapsMissingScheduleToNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &scheduledScanHandlerStore{batchErr: scheduledapp.ErrScheduledScanNotFound}
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(store))
	body := `{"requests":[{"name":"scheduledScans/1","isEnabled":false,"updateMask":"isEnabled"}]}`
	recorder := performScheduledScanRequest(handler.BatchUpdate, http.MethodPost, "/v1/scheduledScans:batchUpdate", bytes.NewBufferString(body))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestScheduledScanCreateAndConfigurationUpdateMapConfigResourceFailures(t *testing.T) {
	field := `configuration.steps["subdomain_discovery"].engineConfig.recon.wordlist`
	private := errors.New("open /srv/private/wordlists/dns.txt: permission denied")
	tests := []struct {
		name       string
		err        error
		statusCode int
		rpcStatus  string
		reason     string
	}{
		{
			name: "unavailable", err: scanapp.NewConfigResourceUnavailableError(field, "wordlist", "dns.txt", private),
			statusCode: http.StatusBadRequest, rpcStatus: "FAILED_PRECONDITION", reason: "ENGINE_CONFIG_RESOURCE_UNAVAILABLE",
		},
		{
			name: "validation unavailable", err: scanapp.NewConfigResourceValidationUnavailableError(field, "wordlist", "dns.txt", private),
			statusCode: http.StatusServiceUnavailable, rpcStatus: "UNAVAILABLE", reason: "ENGINE_CONFIG_RESOURCE_VALIDATION_UNAVAILABLE",
		},
		{
			name: "internal", err: scanapp.NewConfigResourceInternalError(field, "wordlist", "dns.txt", private),
			statusCode: http.StatusInternalServerError, rpcStatus: "INTERNAL", reason: "INTERNAL_ERROR",
		},
	}
	operations := []struct {
		name   string
		method string
		path   string
		body   string
		call   func(*ScheduledScanHandler) gin.HandlerFunc
	}{
		{
			name: "create", method: http.MethodPost, path: "/v1/scheduledScans",
			body: `{"displayName":"daily","scanWorkflow":"scanWorkflows/default","inputSource":"scanSnapshot","configuration":{"steps":{"subdomain_discovery":{"enabled":true,"engineConfig":{"recon":{"enabled":true,"timeout":3600}}}}},"target":"targets/7","cronExpression":"0 2 * * *"}`,
			call: func(handler *ScheduledScanHandler) gin.HandlerFunc { return handler.Create },
		},
		{
			name: "configuration update", method: http.MethodPatch, path: "/v1/scheduledScans/1",
			body: `{"name":"scheduledScans/1","scanWorkflow":"scanWorkflows/default","configuration":{"steps":{"subdomain_discovery":{"enabled":true,"engineConfig":{"recon":{"enabled":true,"timeout":3600}}}}},"updateMask":"scanWorkflow,configuration"}`,
			call: func(handler *ScheduledScanHandler) gin.HandlerFunc { return handler.Update },
		},
	}
	for _, operation := range operations {
		for _, test := range tests {
			t.Run(operation.name+"/"+test.name, func(t *testing.T) {
				gin.SetMode(gin.TestMode)
				store := &scheduledScanHandlerStore{}
				handler := NewScheduledScanHandler(newScheduledScanServiceWithResourceFailureForHandlerTest(store, test.err))
				recorder := performScheduledScanRequest(operation.call(handler), operation.method, operation.path, bytes.NewBufferString(operation.body))
				var response struct {
					Error struct {
						Status  string `json:"status"`
						Details []struct {
							Reason   string            `json:"reason"`
							Metadata map[string]string `json:"metadata"`
						} `json:"details"`
					} `json:"error"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if recorder.Code != test.statusCode || response.Error.Status != test.rpcStatus || len(response.Error.Details) != 1 || response.Error.Details[0].Reason != test.reason {
					t.Fatalf("unexpected response: status=%d body=%s", recorder.Code, recorder.Body.String())
				}
				if strings.Contains(recorder.Body.String(), "/srv/private") || strings.Contains(recorder.Body.String(), "permission denied") {
					t.Fatalf("response leaked private diagnostics: %s", recorder.Body.String())
				}
				if test.statusCode == http.StatusInternalServerError {
					if len(response.Error.Details[0].Metadata) != 0 {
						t.Fatalf("internal response exposed metadata: %s", recorder.Body.String())
					}
				} else if response.Error.Details[0].Metadata["field"] != field || response.Error.Details[0].Metadata["resourceKind"] != "wordlist" || response.Error.Details[0].Metadata["resourceName"] != "dns.txt" {
					t.Fatalf("unexpected safe metadata: %#v", response.Error.Details[0].Metadata)
				}
				if store.created != nil || store.updated != nil {
					t.Fatalf("failed validation reached persistence: created=%#v updated=%#v", store.created, store.updated)
				}
			})
		}
	}
}

func TestScheduledScanDeleteReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewScheduledScanHandler(newScheduledScanServiceForHandlerTest(&scheduledScanHandlerStore{}))
	recorder := performScheduledScanRequest(handler.Delete, http.MethodDelete, "/v1/scheduledScans/1", nil)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty 204 body, got %s", recorder.Body.String())
	}
}

func performScheduledScanRequest(handler gin.HandlerFunc, method, target string, body *bytes.Buffer) *httptest.ResponseRecorder {
	if body == nil {
		body = bytes.NewBuffer(nil)
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = gin.Params{{Key: "scheduled_scan", Value: "1"}}
	handler(c)
	return recorder
}

func assertWorkflowConfigurationDiagnostic(t *testing.T, recorder *httptest.ResponseRecorder, field string) {
	t.Helper()
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Error struct {
			Status  string            `json:"status"`
			Details []json.RawMessage `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error.Status != "INVALID_ARGUMENT" || len(response.Error.Details) != 2 {
		t.Fatalf("unexpected workflow error envelope: %+v", response.Error)
	}
	var info struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(response.Error.Details[0], &info); err != nil || info.Reason != "WORKFLOW_CONFIGURATION_INVALID" {
		t.Fatalf("unexpected workflow ErrorInfo: %s", response.Error.Details[0])
	}
	var badRequest struct {
		FieldViolations []struct {
			Field string `json:"field"`
		} `json:"fieldViolations"`
	}
	if err := json.Unmarshal(response.Error.Details[1], &badRequest); err != nil {
		t.Fatalf("decode BadRequest detail: %v", err)
	}
	if len(badRequest.FieldViolations) != 1 || badRequest.FieldViolations[0].Field != field {
		t.Fatalf("unexpected field violations: %+v", badRequest.FieldViolations)
	}
}

func scheduledScanRecord(id int, displayName string, enabled bool, inputSource scanapp.InputSource) *scheduledapp.ScheduledScan {
	return &scheduledapp.ScheduledScan{
		ID:             id,
		Name:           displayName,
		ScanWorkflowID: "default",
		Configuration:  map[string]any{"steps": map[string]any{}},
		InputSource:    inputSource,
		CronExpression: "0 2 * * *",
		IsEnabled:      enabled,
		CreatedAt:      time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
	}
}
