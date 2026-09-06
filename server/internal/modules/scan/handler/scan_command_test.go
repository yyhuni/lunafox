package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
	service "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

const (
	handlerTestPackageDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	handlerTestImageDigest   = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func TestParseScanStopSegmentRequiresStopCustomMethod(t *testing.T) {
	id, err := parseScanStopSegment("7:stop")
	if err != nil {
		t.Fatalf("expected scan stop segment to parse: %v", err)
	}
	if id != 7 {
		t.Fatalf("expected scan id 7, got %d", id)
	}

	for _, value := range []string{"7", "7:cancel", "bad:stop"} {
		if _, err := parseScanStopSegment(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

type handlerStopStoreStub struct {
	outcome      *service.ScanStopOutcome
	batchOutcome *service.BatchScanStopOutcome
	err          error
	batchErr     error
	ctx          context.Context
	scanID       int
	batchIDs     []int
}

func (stub *handlerStopStoreStub) StopActiveScan(ctx context.Context, scanID int, _ time.Time) (*service.ScanStopOutcome, error) {
	stub.ctx = ctx
	stub.scanID = scanID
	return stub.outcome, stub.err
}

func (stub *handlerStopStoreStub) BatchStopActiveScans(ctx context.Context, scanIDs []int, _ time.Time) (*service.BatchScanStopOutcome, error) {
	stub.ctx = ctx
	stub.batchIDs = append([]int(nil), scanIDs...)
	return stub.batchOutcome, stub.batchErr
}

func TestStopHandlerPropagatesRequestContextAndReturnsCommittedOutcome(t *testing.T) {
	store := &handlerStopStoreStub{outcome: &service.ScanStopOutcome{ScanID: 8, CancelledTaskCount: 3}}
	facade := service.NewScanFacade(nil, nil, nil, nil, service.NewLifecycleService(nil, store, nil), nil)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handler := NewScanHandler(facade, ScanHistoryRetentionPolicy{})
	engine.POST("/v1/scans/:scan", handler.Stop)

	contextKey := struct{}{}
	request := httptest.NewRequest(http.MethodPost, "/v1/scans/8:stop", nil)
	request = request.WithContext(context.WithValue(request.Context(), contextKey, "request-bound"))
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Stop status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if store.scanID != 8 || store.ctx.Value(contextKey) != "request-bound" {
		t.Fatalf("Stop store scope = scan:%d context:%v", store.scanID, store.ctx.Value(contextKey))
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode Stop response: %v", err)
	}
	if got, _ := response["revokedTaskCount"].(float64); got != 3 {
		t.Fatalf("revoked task count = %v, want 3", response["revokedTaskCount"])
	}
}

func TestStopHandlerMapsWrappedCallerCancellationBeforeInternalFallback(t *testing.T) {
	store := &handlerStopStoreStub{err: fmt.Errorf("store wait: %w", context.Canceled)}
	facade := service.NewScanFacade(nil, nil, nil, nil, service.NewLifecycleService(nil, store, nil), nil)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handler := NewScanHandler(facade, ScanHistoryRetentionPolicy{})
	engine.POST("/v1/scans/:scan", handler.Stop)

	recorder := doJSONRequest(engine, http.MethodPost, "/v1/scans/8:stop", "", "")
	if recorder.Code != 499 {
		t.Fatalf("wrapped cancellation status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Error struct {
			Status string `json:"status"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode cancellation response: %v", err)
	}
	if response.Error.Status != "CANCELLED" {
		t.Fatalf("cancellation status = %q, want CANCELLED", response.Error.Status)
	}
}

func TestBatchStopHandlerReturnsExplicitCountsAndPreservesRequestContext(t *testing.T) {
	store := &handlerStopStoreStub{batchOutcome: &service.BatchScanStopOutcome{
		StoppedCount: 2, SkippedCount: 1, RevokedTaskCount: 3,
	}}
	facade := service.NewScanFacade(nil, nil, nil, nil, service.NewLifecycleService(nil, store, nil), nil)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	h := NewScanHandler(facade, ScanHistoryRetentionPolicy{})
	engine.POST("/v1/scans:batchStop", h.BatchStop)

	contextKey := struct{}{}
	request := httptest.NewRequest(http.MethodPost, "/v1/scans:batchStop", strings.NewReader(`{"names":["scans/8","scans/9","scans/10"]}`))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(context.WithValue(request.Context(), contextKey, "batch-request"))
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("BatchStop status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if store.ctx.Value(contextKey) != "batch-request" || fmt.Sprint(store.batchIDs) != "[8 9 10]" {
		t.Fatalf("batch store input = ids:%v context:%v", store.batchIDs, store.ctx.Value(contextKey))
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode BatchStop response: %v", err)
	}
	for key, want := range map[string]float64{"stoppedCount": 2, "skippedCount": 1, "revokedTaskCount": 3} {
		if got := response[key]; got != want {
			t.Fatalf("%s = %v, want %v", key, got, want)
		}
	}
}

func TestBatchStopHandlerRejectsDuplicateNamesBeforeService(t *testing.T) {
	store := &handlerStopStoreStub{batchOutcome: &service.BatchScanStopOutcome{StoppedCount: 1}}
	facade := service.NewScanFacade(nil, nil, nil, nil, service.NewLifecycleService(nil, store, nil), nil)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	h := NewScanHandler(facade, ScanHistoryRetentionPolicy{})
	engine.POST("/v1/scans:batchStop", h.BatchStop)

	recorder := doJSONRequest(engine, http.MethodPost, "/v1/scans:batchStop", `{"names":["scans/8","scans/8"]}`, "application/json")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("duplicate BatchStop status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if len(store.batchIDs) != 0 {
		t.Fatalf("duplicate request reached lifecycle service: %v", store.batchIDs)
	}
}

func TestBatchStopHandlerMapsWrappedCallerCancellation(t *testing.T) {
	store := &handlerStopStoreStub{batchErr: fmt.Errorf("batch wait: %w", context.Canceled)}
	facade := service.NewScanFacade(nil, nil, nil, nil, service.NewLifecycleService(nil, store, nil), nil)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	h := NewScanHandler(facade, ScanHistoryRetentionPolicy{})
	engine.POST("/v1/scans:batchStop", h.BatchStop)

	recorder := doJSONRequest(engine, http.MethodPost, "/v1/scans:batchStop", `{"names":["scans/8"]}`, "application/json")
	if recorder.Code != 499 {
		t.Fatalf("wrapped batch cancellation status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
}

// --- Scan command handler tests ---

type handlerCreateStoreStub struct{}

func (handlerCreateStoreStub) CreateWithScanTasksAndPlans(context.Context, *service.CreateScan, service.ScanCreateTaskFinalizer) error {
	return fmt.Errorf("scan persistence must not be reached")
}

type handlerCreateStoreCaptureStub struct {
	scan   *service.CreateScan
	nextID int
	err    error
}

func (stub *handlerCreateStoreCaptureStub) CreateWithScanTasksAndPlans(_ context.Context, scan *service.CreateScan, finalize service.ScanCreateTaskFinalizer) error {
	if scan == nil || finalize == nil {
		return fmt.Errorf("planned scan create arguments are required")
	}
	if stub.nextID > 0 {
		scan.ID = stub.nextID
		stub.nextID++
	} else if scan.ID <= 0 {
		scan.ID = 1
	}
	for index := range scan.ScanTasks {
		task := &scan.ScanTasks[index]
		task.ID = scan.ID*100 + index + 1
		if err := finalize(scan.ID, task.ID, task); err != nil {
			stub.err = err
			return err
		}
	}
	stub.scan = scan
	return nil
}

type handlerWorkflowReaderStub struct {
	manifest service.ScanCreateWorkflowManifest
}

func (stub handlerWorkflowReaderStub) GetScanWorkflowManifest(context.Context, string) (service.ScanCreateWorkflowManifest, error) {
	return stub.manifest, nil
}

type handlerEnginePackageReaderStub struct {
	packages map[string]service.ScanCreateEnginePackage
}

type handlerConfigResourceResolverStub struct{ err error }

func (stub handlerConfigResourceResolverStub) ResolveConfigResource(_ context.Context, request service.ConfigResourceResolveRequest) (service.PlanTaskWordlist, error) {
	if stub.err != nil {
		return service.PlanTaskWordlist{}, stub.err
	}
	return service.PlanTaskWordlist{
		Resource: request.ResourceName, Basename: "dns.txt", SizeBytes: 8, LineCount: 2,
		SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}, nil
}

func (stub handlerEnginePackageReaderStub) ListEnginePackagesByID() (map[string]service.ScanCreateEnginePackage, error) {
	return stub.packages, nil
}

func (stub handlerEnginePackageReaderStub) LoadExactPackage(_ context.Context, identity service.PlanTaskPackageIdentity) (service.PlanTaskPackage, error) {
	entry, ok := stub.packages[identity.EngineID]
	if !ok {
		return service.PlanTaskPackage{}, fmt.Errorf("package %q not found", identity.EngineID)
	}
	if entry.Package.Identity != identity {
		return service.PlanTaskPackage{}, fmt.Errorf("package identity mismatch for %q", identity.EngineID)
	}
	return entry.Package, nil
}

func (stub handlerEnginePackageReaderStub) LoadEnginePackage(_ context.Context, engineID string) (service.ScanCreateEnginePackage, error) {
	entry, ok := stub.packages[engineID]
	if !ok {
		return service.ScanCreateEnginePackage{}, fmt.Errorf("package %q not found", engineID)
	}
	return entry, nil
}

func handlerTestEnginePackage(engineID string) service.ScanCreateEnginePackage {
	definition := enginecontract.EngineDefinition{
		ManifestVersion: enginecontract.SupportedRootManifestVersion,
		EngineID:        engineID,
		Publisher:       "lunafox",
		Execution: engineexecution.ExecutionDefinition{
			EngineAPIMajor:       2,
			SupportedTargetTypes: []string{engineexecution.TargetTypeDomain},
			ConfigSections: []engineexecution.ConfigSectionDefinition{{
				ID:             "recon",
				DefaultEnabled: true,
				Params: []engineexecution.ParamDefinition{{
					Key: "timeout", Type: engineexecution.ParamTypeInteger, Default: 3600,
				}},
			}},
		},
	}
	return service.ScanCreateEnginePackage{Package: service.PlanTaskPackage{
		Identity:         service.PlanTaskPackageIdentity{EngineID: engineID, PackageDigest: handlerTestPackageDigest},
		PackageVersion:   "1.0.0",
		Definition:       definition,
		RuntimeImageRefs: []string{"docker.io/lunafox/lunafox-engine-runtime-subdomain-discovery@" + handlerTestImageDigest},
	}}
}

func handlerTestResourceEnginePackage(engineID string) service.ScanCreateEnginePackage {
	result := handlerTestEnginePackage(engineID)
	minimum := 1
	result.Package.Definition.Execution.ConfigSections[0].Params = append(
		result.Package.Definition.Execution.ConfigSections[0].Params,
		engineexecution.ParamDefinition{
			Key: "wordlist", Type: engineexecution.ParamTypeString, Default: "dns.txt", MinLength: &minimum,
			Resource: &engineexecution.ParamResourceBinding{Kind: engineexecution.ConfigResourceKindWordlist},
		},
	)
	return result
}

func newTestResourceScanFacade(t *testing.T, resolver service.ConfigResourceResolver) (*service.ScanFacade, *handlerCreateStoreCaptureStub) {
	t.Helper()
	store := &handlerCreateStoreCaptureStub{nextID: 41}
	manifest := service.ScanCreateWorkflowManifest{
		ScanWorkflowID: "default",
		Stages: []service.ScanCreateWorkflowStage{{
			StageID: "discovery",
			Steps:   []service.ScanCreateWorkflowStep{{StepID: "subdomain_discovery", EngineID: "engine.lunafox.subdomain_discovery"}},
		}},
	}
	packageReader := handlerEnginePackageReaderStub{packages: map[string]service.ScanCreateEnginePackage{
		"engine.lunafox.subdomain_discovery": handlerTestResourceEnginePackage("engine.lunafox.subdomain_discovery"),
	}}
	createService := service.NewScanCreateService(
		store,
		func(_ context.Context, id int) (*service.TargetRef, error) {
			return &service.TargetRef{ID: id, Name: "example.com", Type: "domain", CreatedAt: time.Now().UTC()}, nil
		},
		func(context.Context, []string) (*service.QuickTargetResolution, error) {
			return &service.QuickTargetResolution{
				Targets:     []service.TargetRef{{ID: 7, Name: "example.com", Type: "domain", CreatedAt: time.Now().UTC()}},
				TargetStats: service.QuickTargetStats{Created: 1},
			}, nil
		},
		handlerWorkflowReaderStub{manifest: manifest},
		packageReader,
	)
	compiler, err := service.NewPlanTaskCompiler(packageReader, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if err := createService.ConfigurePlanTask(compiler, service.FixedPlanTaskLimitsProvider(time.Hour)); err != nil {
		t.Fatal(err)
	}
	return newTestScanFacade(createService), store
}

func mustConfigureHandlerPlanTask(t *testing.T, createService *service.ScanCreateService, reader handlerEnginePackageReaderStub) *service.ScanCreateService {
	t.Helper()
	compiler, err := service.NewPlanTaskCompiler(reader, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler failed: %v", err)
	}
	if err := createService.ConfigurePlanTask(compiler, service.FixedPlanTaskLimitsProvider(time.Hour)); err != nil {
		t.Fatalf("ConfigurePlanTask failed: %v", err)
	}
	return createService
}

func newTestScanFacade(createService *service.ScanCreateService) *service.ScanFacade {
	return service.NewScanFacade(nil, nil, nil, nil, nil, createService)
}

func newTestCreateService(t *testing.T, store service.ScanCreateCommandStore) *service.ScanCreateService {
	t.Helper()
	manifest := service.ScanCreateWorkflowManifest{
		ScanWorkflowID: "subdomain_discovery",
		Stages: []service.ScanCreateWorkflowStage{{
			StageID: "discovery",
			Steps: []service.ScanCreateWorkflowStep{{
				StepID:   "subdomain_discovery",
				EngineID: "engine.lunafox.subdomain_discovery",
			}},
		}},
	}
	enginePackages := map[string]service.ScanCreateEnginePackage{
		"engine.lunafox.subdomain_discovery": handlerTestEnginePackage("engine.lunafox.subdomain_discovery"),
	}
	packageReader := handlerEnginePackageReaderStub{packages: enginePackages}
	return mustConfigureHandlerPlanTask(t, service.NewScanCreateService(
		store,
		func(_ context.Context, id int) (*service.TargetRef, error) {
			now := time.Now().UTC()
			return &service.TargetRef{ID: id, Name: "example.com", Type: "domain", CreatedAt: now}, nil
		},
		nil,
		handlerWorkflowReaderStub{manifest: manifest},
		packageReader,
		func(_ context.Context, organizationID int) ([]service.TargetRef, error) {
			now := time.Now().UTC()
			if organizationID != 5 {
				return []service.TargetRef{}, nil
			}
			return []service.TargetRef{{ID: 8, Name: "org.example.com", Type: "domain", CreatedAt: now}}, nil
		},
	), packageReader)
}

func doJSONRequest(engine *gin.Engine, method, path, body string, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func setupCreateBatchHandler(facade *service.ScanFacade) (*gin.Engine, *ScanHandler) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	h := NewScanHandler(facade, ScanHistoryRetentionPolicy{})
	engine.POST("/v1/scans:batchCreate", h.BatchCreate)
	return engine, h
}

func TestCreateQuickHandler_SuccessCreatesPendingScan(t *testing.T) {
	now := time.Now().UTC()
	store := &handlerCreateStoreCaptureStub{nextID: 41}
	packageReader := handlerEnginePackageReaderStub{packages: map[string]service.ScanCreateEnginePackage{
		"engine.lunafox.subdomain_discovery": handlerTestEnginePackage("engine.lunafox.subdomain_discovery"),
	}}
	createService := mustConfigureHandlerPlanTask(t, service.NewScanCreateService(
		store,
		func(context.Context, int) (*service.TargetRef, error) { return nil, nil },
		func(context.Context, []string) (*service.QuickTargetResolution, error) {
			return &service.QuickTargetResolution{
				Targets: []service.TargetRef{{ID: 7, Name: "example.com", Type: "domain", CreatedAt: now}},
				TargetStats: service.QuickTargetStats{
					Created: 1,
					Skipped: 0,
					Failed:  0,
				},
			}, nil
		},
		handlerWorkflowReaderStub{manifest: service.ScanCreateWorkflowManifest{
			ScanWorkflowID: "subdomain_discovery",
			Stages: []service.ScanCreateWorkflowStage{{
				StageID: "discovery",
				Steps:   []service.ScanCreateWorkflowStep{{StepID: "subdomain_discovery", EngineID: "engine.lunafox.subdomain_discovery"}},
			}},
		}},
		packageReader,
	), packageReader)
	facade := newTestScanFacade(createService)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	h := NewScanHandler(facade, ScanHistoryRetentionPolicy{})
	engine.POST("/v1/scans:quickCreate", h.CreateQuick)

	body := `{"targets":["example.com"],"scanWorkflow":"scanWorkflows/default","inputSource":"scanSnapshot","configuration":{"steps":{"subdomain_discovery":{"enabled":true,"engineConfig":{"recon":{"enabled":true,"timeout":3600}}}}}}`
	rec := doJSONRequest(engine, http.MethodPost, "/v1/scans:quickCreate", body, "application/json")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s (store error: %v)", rec.Code, rec.Body.String(), store.err)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if count, _ := resp["count"].(float64); count != 1 {
		t.Fatalf("expected count 1, got %+v", resp)
	}
	if stats, ok := resp["targetStats"].(map[string]any); !ok || stats["created"].(float64) != 1 {
		t.Fatalf("expected created target stats, got %+v", resp["targetStats"])
	}
	scans, ok := resp["scans"].([]any)
	if !ok || len(scans) != 1 {
		t.Fatalf("expected one scan in quick response, got %+v", resp["scans"])
	}
	scan, ok := scans[0].(map[string]any)
	if !ok {
		t.Fatalf("expected scan object, got %+v", scans[0])
	}
	if status, _ := scan["status"].(string); status != "pending" {
		t.Fatalf("expected pending scan, got %+v", scan)
	}
	if targetID, _ := scan["targetId"].(float64); targetID != 7 {
		t.Fatalf("expected target 7, got %+v", scan)
	}
	if inputSource, _ := scan["inputSource"].(string); inputSource != "scanSnapshot" {
		t.Fatalf("expected scanSnapshot input source, got %+v", scan)
	}

	if store.scan == nil {
		t.Fatal("expected quick scan to persist a scan")
	}
	if store.scan.TriggerType != service.ScanTriggerTypeManual {
		t.Fatalf("expected manual trigger type, got %q", store.scan.TriggerType)
	}
	if store.scan.ScanWorkflowID != "subdomain_discovery" {
		t.Fatalf("expected workflow default, got %q", store.scan.ScanWorkflowID)
	}
	if store.scan.InputSource != service.InputSourceScanSnapshot {
		t.Fatalf("expected scanSnapshot input source, got %q", store.scan.InputSource)
	}
	if len(store.scan.ScanTasks) != 1 {
		t.Fatalf("expected 1 scan task execution, got %+v", store.scan.ScanTasks)
	}
}

func TestCreateQuickHandlerRejectsMoreThanFiveThousandTargetsBeforeService(t *testing.T) {
	now := time.Now().UTC()
	store := &handlerCreateStoreCaptureStub{nextID: 41}
	serviceCalled := false
	packageReader := handlerEnginePackageReaderStub{packages: map[string]service.ScanCreateEnginePackage{
		"engine.lunafox.subdomain_discovery": handlerTestEnginePackage("engine.lunafox.subdomain_discovery"),
	}}
	createService := mustConfigureHandlerPlanTask(t, service.NewScanCreateService(
		store,
		func(context.Context, int) (*service.TargetRef, error) { return nil, nil },
		func(context.Context, []string) (*service.QuickTargetResolution, error) {
			serviceCalled = true
			return &service.QuickTargetResolution{
				Targets:     []service.TargetRef{{ID: 7, Name: "example.com", Type: "domain", CreatedAt: now}},
				TargetStats: service.QuickTargetStats{Created: 1},
			}, nil
		},
		handlerWorkflowReaderStub{manifest: service.ScanCreateWorkflowManifest{
			ScanWorkflowID: "subdomain_discovery",
			Stages: []service.ScanCreateWorkflowStage{{
				StageID: "discovery",
				Steps:   []service.ScanCreateWorkflowStep{{StepID: "subdomain_discovery", EngineID: "engine.lunafox.subdomain_discovery"}},
			}},
		}},
		packageReader,
	), packageReader)
	facade := newTestScanFacade(createService)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	h := NewScanHandler(facade, ScanHistoryRetentionPolicy{})
	engine.POST("/v1/scans:quickCreate", h.CreateQuick)

	targets := make([]string, 5001)
	for index := range targets {
		targets[index] = "example.com"
	}
	bodyBytes, err := json.Marshal(map[string]any{
		"targets":      targets,
		"scanWorkflow": "scanWorkflows/default",
		"inputSource":  "scanSnapshot",
		"configuration": map[string]any{
			"steps": map[string]any{
				"subdomain_discovery": map[string]any{
					"enabled":      true,
					"engineConfig": map[string]any{"recon": map[string]any{"enabled": true, "timeout": 3600}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	rec := doJSONRequest(engine, http.MethodPost, "/v1/scans:quickCreate", string(bodyBytes), "application/json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if serviceCalled {
		t.Fatal("expected oversized quick target request to fail during binding before scan creation service")
	}
}

func TestBatchCreateHandlerCreatesTargetsAndOrganizationTargets(t *testing.T) {
	store := &handlerCreateStoreCaptureStub{nextID: 101}
	facade := newTestScanFacade(newTestCreateService(t, store))
	engine, _ := setupCreateBatchHandler(facade)
	body := `{
		"requests": [
			{"target":"targets/7"},
			{"organization":"organizations/5"},
			{"organization":"organizations/6"}
		],
		"scanWorkflow":"scanWorkflows/subdomain_discovery",
		"inputSource":"scanSnapshot",
		"configuration":{"steps":{"subdomain_discovery":{"enabled":true,"engineConfig":{"recon":{"enabled":true,"timeout":3600}}}}}
	}`

	rec := doJSONRequest(engine, http.MethodPost, "/v1/scans:batchCreate", body, "application/json")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s (store error: %v)", rec.Code, rec.Body.String(), store.err)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["createdCount"] != float64(2) || resp["count"] != float64(2) {
		t.Fatalf("expected two created scans, got %s", rec.Body.String())
	}
	scans, ok := resp["scans"].([]any)
	if !ok || len(scans) != 2 {
		t.Fatalf("expected two scan outputs, got %s", rec.Body.String())
	}
	skipped, ok := resp["skipped"].([]any)
	if !ok || len(skipped) != 1 {
		t.Fatalf("expected one skipped output for empty organization, got %s", rec.Body.String())
	}
}

func TestBatchCreateHandlerRejectsIncompleteEngineConfigWithStableReason(t *testing.T) {
	store := &handlerCreateStoreCaptureStub{nextID: 101}
	facade := newTestScanFacade(newTestCreateService(t, store))
	engine, _ := setupCreateBatchHandler(facade)
	rec := doJSONRequest(engine, http.MethodPost, "/v1/scans:batchCreate", `{
		"requests":[{"target":"targets/7"}],
		"scanWorkflow":"scanWorkflows/default",
		"inputSource":"scanSnapshot",
		"configuration":{"steps":{"subdomain_discovery":{"enabled":true,"engineConfig":{"recon":{"enabled":true}}}}}
	}`, "application/json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Error struct {
			Details []struct {
				Reason string `json:"reason"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if len(response.Error.Details) == 0 || response.Error.Details[0].Reason != "ENGINE_CONFIG_INVALID" {
		t.Fatalf("expected ENGINE_CONFIG_INVALID ErrorInfo, got %s", rec.Body.String())
	}
	if store.scan != nil {
		t.Fatalf("invalid configuration reached persistence: %#v", store.scan)
	}
}

func TestQuickAndBatchCreateHandlersMapConfigResourceFailures(t *testing.T) {
	field := `configuration.steps["subdomain_discovery"].engineConfig.recon.wordlist`
	privatePath := "/srv/private/wordlists/dns.txt"
	tests := []struct {
		name       string
		resolver   service.ConfigResourceResolver
		statusCode int
		rpcStatus  string
		reason     string
	}{
		{
			name:       "missing Catalog row",
			resolver:   handlerConfigResourceResolverStub{err: service.NewConfigResourceUnavailableError(field, "wordlist", "dns.txt", fmt.Errorf("missing %s", privatePath))},
			statusCode: http.StatusBadRequest, rpcStatus: "FAILED_PRECONDITION", reason: "ENGINE_CONFIG_RESOURCE_UNAVAILABLE",
		},
		{
			name:       "unreadable file",
			resolver:   handlerConfigResourceResolverStub{err: service.NewConfigResourceUnavailableError(field, "wordlist", "dns.txt", fmt.Errorf("open %s: permission denied", privatePath))},
			statusCode: http.StatusBadRequest, rpcStatus: "FAILED_PRECONDITION", reason: "ENGINE_CONFIG_RESOURCE_UNAVAILABLE",
		},
		{
			name:       "metadata drift",
			resolver:   handlerConfigResourceResolverStub{err: service.NewConfigResourceUnavailableError(field, "wordlist", "dns.txt", errors.New("digest mismatch"))},
			statusCode: http.StatusBadRequest, rpcStatus: "FAILED_PRECONDITION", reason: "ENGINE_CONFIG_RESOURCE_UNAVAILABLE",
		},
		{
			name:       "Catalog database unavailable",
			resolver:   handlerConfigResourceResolverStub{err: service.NewConfigResourceValidationUnavailableError(field, "wordlist", "dns.txt", errors.New("database connection refused"))},
			statusCode: http.StatusServiceUnavailable, rpcStatus: "UNAVAILABLE", reason: "ENGINE_CONFIG_RESOURCE_VALIDATION_UNAVAILABLE",
		},
		{
			name: "missing resolver dependency", resolver: nil,
			statusCode: http.StatusInternalServerError, rpcStatus: "INTERNAL", reason: "INTERNAL_ERROR",
		},
	}
	operations := []struct {
		name   string
		method string
		path   string
		body   string
		route  func(*gin.Engine, *ScanHandler)
	}{
		{
			name: "quick", method: http.MethodPost, path: "/v1/scans:quickCreate",
			body: `{"targets":["example.com"],"scanWorkflow":"scanWorkflows/default","inputSource":"scanSnapshot","configuration":{"steps":{"subdomain_discovery":{"enabled":true,"engineConfig":{"recon":{"enabled":true,"timeout":3600,"wordlist":"dns.txt"}}}}}}`,
			route: func(engine *gin.Engine, handler *ScanHandler) {
				engine.POST("/v1/scans:quickCreate", handler.CreateQuick)
			},
		},
		{
			name: "batch", method: http.MethodPost, path: "/v1/scans:batchCreate",
			body: `{"requests":[{"target":"targets/7"}],"scanWorkflow":"scanWorkflows/default","inputSource":"scanSnapshot","configuration":{"steps":{"subdomain_discovery":{"enabled":true,"engineConfig":{"recon":{"enabled":true,"timeout":3600,"wordlist":"dns.txt"}}}}}}`,
			route: func(engine *gin.Engine, handler *ScanHandler) {
				engine.POST("/v1/scans:batchCreate", handler.BatchCreate)
			},
		},
	}
	for _, operation := range operations {
		for _, test := range tests {
			t.Run(operation.name+"/"+test.name, func(t *testing.T) {
				facade, store := newTestResourceScanFacade(t, test.resolver)
				gin.SetMode(gin.TestMode)
				engine := gin.New()
				handler := NewScanHandler(facade, ScanHistoryRetentionPolicy{})
				operation.route(engine, handler)
				recorder := doJSONRequest(engine, operation.method, operation.path, operation.body, "application/json")
				var response struct {
					Error struct {
						Status  string `json:"status"`
						Message string `json:"message"`
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
					t.Fatalf("unexpected response: status=%d body=%s storeError=%v", recorder.Code, recorder.Body.String(), store.err)
				}
				if strings.Contains(recorder.Body.String(), privatePath) || strings.Contains(recorder.Body.String(), "permission denied") || strings.Contains(recorder.Body.String(), "digest mismatch") {
					t.Fatalf("response leaked diagnostic: %s", recorder.Body.String())
				}
				if test.statusCode == http.StatusInternalServerError {
					if len(response.Error.Details[0].Metadata) != 0 {
						t.Fatalf("internal response exposed metadata: %s", recorder.Body.String())
					}
				} else if response.Error.Details[0].Metadata["field"] != field || response.Error.Details[0].Metadata["resourceKind"] != "wordlist" || response.Error.Details[0].Metadata["resourceName"] != "dns.txt" {
					t.Fatalf("unexpected metadata: %#v", response.Error.Details[0].Metadata)
				}
				if test.statusCode == http.StatusServiceUnavailable && strings.Contains(strings.ToLower(response.Error.Message), "another") {
					t.Fatalf("transient error recommends replacement: %q", response.Error.Message)
				}
				if store.scan != nil {
					t.Fatalf("resource failure committed Scan/Task/plan: %#v", store.scan)
				}
			})
		}
	}
}

func TestBatchCreateHandlerMissingConfigurationUsesWorkflowDiagnostic(t *testing.T) {
	store := &handlerCreateStoreCaptureStub{nextID: 101}
	facade := newTestScanFacade(newTestCreateService(t, store))
	engine, _ := setupCreateBatchHandler(facade)
	rec := doJSONRequest(engine, http.MethodPost, "/v1/scans:batchCreate", `{
		"requests":[{"target":"targets/7"}],
		"scanWorkflow":"scanWorkflows/default",
		"inputSource":"scanSnapshot"
	}`, "application/json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Error struct {
			Status  string            `json:"status"`
			Details []json.RawMessage `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error.Status != "INVALID_ARGUMENT" || len(response.Error.Details) != 2 {
		t.Fatalf("expected typed workflow diagnostic, got %s", rec.Body.String())
	}
	var info struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(response.Error.Details[0], &info); err != nil || info.Reason != "WORKFLOW_CONFIGURATION_INVALID" {
		t.Fatalf("unexpected ErrorInfo: %s", response.Error.Details[0])
	}
	if store.scan != nil {
		t.Fatalf("missing configuration reached persistence: %#v", store.scan)
	}
}
