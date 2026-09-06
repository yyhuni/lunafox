package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/application"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
)

type handlerServiceStub struct {
	updateCalls        int
	activationCalls    int
	activationEnabled  bool
	activationErr      error
	getPOC             *domain.POC
	filterOptions      []domain.FilterOption
	filterOptionsErr   error
	filterOptionsField string
}

func (stub *handlerServiceStub) CurrentSource(context.Context) (*domain.Source, error) {
	return nil, domain.ErrSourceNotFound
}
func (stub *handlerServiceStub) CreateSync(context.Context, app.CreateSyncInput) (*domain.SyncTask, error) {
	return &domain.SyncTask{
		ID:          uuid.MustParse("00000000-0000-4000-8000-000000000010"),
		RequestID:   uuid.MustParse("00000000-0000-4000-8000-000000000001"),
		SourceType:  domain.SourceTypeGit,
		State:       domain.SyncTaskValidatingSource,
		Phase:       domain.SyncTaskValidatingSource,
		Diagnostics: domain.Diagnostics{Samples: []domain.DiagnosticSample{}},
	}, nil
}
func (stub *handlerServiceStub) GetSyncTask(context.Context, uuid.UUID) (*domain.SyncTask, error) {
	return nil, domain.ErrSyncTaskNotFound
}
func (stub *handlerServiceStub) List(context.Context, app.POCListQuery) (*app.POCListResult, error) {
	return &app.POCListResult{Results: []domain.POC{}}, nil
}
func (stub *handlerServiceStub) ListFilterOptions(_ context.Context, field string) ([]domain.FilterOption, error) {
	stub.filterOptionsField = field
	if stub.filterOptionsErr != nil {
		return nil, stub.filterOptionsErr
	}
	return stub.filterOptions, nil
}
func (stub *handlerServiceStub) Get(context.Context, string) (*domain.POC, error) {
	if stub.getPOC != nil {
		return stub.getPOC, nil
	}
	return nil, domain.ErrPOCNotFound
}
func (stub *handlerServiceStub) UpdateEnabled(context.Context, string, bool) (*domain.POC, error) {
	stub.updateCalls++
	return &domain.POC{TemplateID: "example", IsEnabled: true}, nil
}
func (stub *handlerServiceStub) SetActivation(_ context.Context, input app.SetPOCActivationInput) (*app.SetPOCActivationResult, error) {
	stub.activationCalls++
	stub.activationEnabled = input.Enabled
	if stub.activationErr != nil {
		return nil, stub.activationErr
	}
	return &app.SetPOCActivationResult{Enabled: input.Enabled, AffectedCount: 2}, nil
}

func newHandlerContext(method, path, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	context.Request.Header.Set("Content-Type", "application/json")
	return context, recorder
}

func TestUpdateRejectsDuplicateUpdateMaskEntries(t *testing.T) {
	stub := &handlerServiceStub{}
	handler := NewNucleiPOCHandler(stub)
	context, recorder := newHandlerContext(http.MethodPatch, "/v1/nucleiPocs/example", `{"name":"nucleiPocs/example","isEnabled":false,"updateMask":["isEnabled","isEnabled"]}`)
	context.Params = gin.Params{{Key: "nucleiPoc", Value: "example"}}
	handler.Update(context)
	if recorder.Code != http.StatusBadRequest || stub.updateCalls != 0 {
		t.Fatalf("status=%d calls=%d body=%s", recorder.Code, stub.updateCalls, recorder.Body.String())
	}
}

func TestGetReturnsEmptyArraysForMissingPOCMetadata(t *testing.T) {
	now := time.Now().UTC()
	stub := &handlerServiceStub{getPOC: &domain.POC{
		TemplateID:    "example",
		DisplayName:   "Example",
		Severity:      "info",
		RelativePath:  "http/example.yaml",
		ContentSHA256: "digest",
		Content:       "id: example",
		CreatedAt:     now,
		UpdatedAt:     now,
	}}
	handler := NewNucleiPOCHandler(stub)
	context, recorder := newHandlerContext(http.MethodGet, "/v1/nucleiPocs/example", "")
	context.Params = gin.Params{{Key: "nucleiPoc", Value: "example"}}

	handler.Get(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"tags", "cve", "cwe", "references"} {
		if string(body[field]) != "[]" {
			t.Errorf("%s=%s, want []", field, body[field])
		}
	}
}

func TestSyncRejectsNonCanonicalUUIDBeforeServiceCall(t *testing.T) {
	stub := &handlerServiceStub{}
	handler := NewNucleiPOCHandler(stub)
	context, recorder := newHandlerContext(http.MethodPost, "/v1/nucleiPocSources:sync", `{"requestId":"00000000-0000-4000-8000-000000000001","sourceType":"git","repoUrl":"https://example.com/templates.git"}`)
	handler.Sync(context)
	if recorder.Code != http.StatusCreated {
		// The UUID above is canonical; this assertion also verifies the handler
		// reached the service boundary without requiring a configured runner.
		t.Fatalf("canonical UUID was rejected: status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	context, recorder = newHandlerContext(http.MethodPost, "/v1/nucleiPocSources:sync", `{"requestId":"00000000-0000-4000-8000-000000000001 ","sourceType":"git","repoUrl":"https://example.com/templates.git"}`)
	handler.Sync(context)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("non-canonical UUID status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	context, recorder = newHandlerContext(http.MethodPost, "/v1/nucleiPocSources:sync", `{"requestId":"00000000-0000-0000-0000-000000000000","sourceType":"git","repoUrl":"https://example.com/templates.git"}`)
	handler.Sync(context)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("nil UUID status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSetActivationRequiresExplicitEnabledAndRejectsUnknownFields(t *testing.T) {
	stub := &handlerServiceStub{}
	handler := NewNucleiPOCHandler(stub)
	for _, body := range []string{
		`{}`,
		`{"enabled":null}`,
		`{"enabled":false,"pageSize":50}`,
	} {
		context, recorder := newHandlerContext(http.MethodPost, "/v1/nucleiPocs:setActivation", body)
		handler.SetActivation(context)
		if recorder.Code != http.StatusBadRequest || stub.activationCalls != 0 {
			t.Fatalf("body=%s status=%d calls=%d response=%s", body, recorder.Code, stub.activationCalls, recorder.Body.String())
		}
	}

	context, recorder := newHandlerContext(http.MethodPost, "/v1/nucleiPocs:setActivation", `{"enabled":false}`)
	handler.SetActivation(context)
	if recorder.Code != http.StatusOK || stub.activationCalls != 1 || stub.activationEnabled {
		t.Fatalf("status=%d calls=%d enabled=%t response=%s", recorder.Code, stub.activationCalls, stub.activationEnabled, recorder.Body.String())
	}
}

func TestSetActivationMapsActiveSyncToCanonicalConflict(t *testing.T) {
	stub := &handlerServiceStub{activationErr: &app.SyncAlreadyRunningError{TaskID: uuid.MustParse("00000000-0000-4000-8000-000000000020")}}
	handler := NewNucleiPOCHandler(stub)
	context, recorder := newHandlerContext(http.MethodPost, "/v1/nucleiPocs:setActivation", `{"enabled":true}`)
	handler.SetActivation(context)
	if recorder.Code != http.StatusConflict || !bytes.Contains(recorder.Body.Bytes(), []byte(`"SYNC_ALREADY_RUNNING"`)) {
		t.Fatalf("status=%d response=%s", recorder.Code, recorder.Body.String())
	}
}

func TestFilterOptionsRequiresTagsAndUsesSharedResponse(t *testing.T) {
	stub := &handlerServiceStub{filterOptions: []domain.FilterOption{{Value: "cve", Label: "cve", Count: 2}}}
	handler := NewNucleiPOCHandler(stub)

	for _, path := range []string{"/v1/nucleiPocs/filterOptions", "/v1/nucleiPocs/filterOptions?field=tags&filter=x"} {
		context, recorder := newHandlerContext(http.MethodGet, path, "")
		handler.FilterOptions(context)
		if recorder.Code != http.StatusBadRequest || stub.filterOptionsField != "" {
			t.Fatalf("path=%s status=%d field=%q body=%s", path, recorder.Code, stub.filterOptionsField, recorder.Body.String())
		}
	}

	stub.filterOptionsErr = app.ErrInvalidArgument
	context, recorder := newHandlerContext(http.MethodGet, "/v1/nucleiPocs/filterOptions?field=severity", "")
	handler.FilterOptions(context)
	if recorder.Code != http.StatusBadRequest || stub.filterOptionsField != "severity" {
		t.Fatalf("status=%d field=%q body=%s", recorder.Code, stub.filterOptionsField, recorder.Body.String())
	}

	stub.filterOptionsErr = nil
	stub.filterOptionsField = ""

	context, recorder = newHandlerContext(http.MethodGet, "/v1/nucleiPocs/filterOptions?field=tags", "")
	handler.FilterOptions(context)
	if recorder.Code != http.StatusOK || stub.filterOptionsField != "tags" || !bytes.Contains(recorder.Body.Bytes(), []byte(`"count":2`)) {
		t.Fatalf("status=%d field=%q body=%s", recorder.Code, stub.filterOptionsField, recorder.Body.String())
	}
}
