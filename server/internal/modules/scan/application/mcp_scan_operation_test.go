package application

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

// mcpScanCreateStoreStub models the one transaction owned by the repository.
// It only exposes the MCP extension when StartMCPScan is exercised, so normal
// Scan create callers cannot accidentally gain operation persistence.
type mcpScanCreateStoreStub struct {
	creationCalls int
	nextScanID    int
	lastScan      *CreateScan
	replays       map[string]*MCPRequestReplay
	createErr     error
	operation     *MCPScanOperation
	operationErr  error
}

func (stub *mcpScanCreateStoreStub) CreateWithScanTasksAndPlans(_ context.Context, _ *CreateScan, _ ScanCreateTaskFinalizer) error {
	return errors.New("ordinary Scan create boundary must not be used for MCP")
}

func (stub *mcpScanCreateStoreStub) CreateWithScanTasksAndPlansAndMCPOperation(
	ctx context.Context,
	scan *CreateScan,
	operation *MCPScanOperationCreate,
	finalize ScanCreateTaskFinalizer,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	stub.creationCalls++
	if stub.createErr != nil {
		return stub.createErr
	}
	stub.nextScanID++
	scan.ID = stub.nextScanID
	scan.CreatedAt = time.Now().UTC()
	for index := range scan.ScanTasks {
		task := &scan.ScanTasks[index]
		task.ID = scan.ID*100 + index + 1
		if err := finalize(scan.ID, task.ID, task); err != nil {
			return err
		}
	}
	clone := *scan
	clone.ScanTasks = append([]CreateScanTask(nil), scan.ScanTasks...)
	stub.lastScan = &clone
	if operation != nil && operation.RequestID != "" {
		response, err := json.Marshal(MCPScanStartResult{Operation: "operations/" + operation.ID, Scan: "scans/" + strconv.Itoa(scan.ID)})
		if err != nil {
			return err
		}
		if stub.replays == nil {
			stub.replays = map[string]*MCPRequestReplay{}
		}
		stub.replays[operation.RequestID] = &MCPRequestReplay{
			RequestID: operation.RequestID, Action: mcpScanStartAction, RequestFingerprint: operation.RequestFingerprint,
			Response: response, ExpiresAt: operation.ReplayExpiresAt,
		}
	}
	return nil
}

func (stub *mcpScanCreateStoreStub) FindMCPRequestReplay(_ context.Context, requestID string) (*MCPRequestReplay, error) {
	replay := stub.replays[requestID]
	if replay == nil {
		return nil, nil
	}
	clone := *replay
	clone.Response = append([]byte(nil), replay.Response...)
	return &clone, nil
}

func (stub *mcpScanCreateStoreStub) GetMCPScanOperation(_ context.Context, _ string) (*MCPScanOperation, error) {
	if stub.operationErr != nil {
		return nil, stub.operationErr
	}
	return stub.operation, nil
}

func newMCPScanCreateService(t *testing.T, store *mcpScanCreateStoreStub) *ScanCreateService {
	t.Helper()
	workflowReader, enginePackageReader := scanCreateReaderStubs()
	service := NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{ID: 1, Name: "example.test", Type: "domain", CreatedAt: now}, nil
	}, nil, workflowReader, enginePackageReader)
	return mustConfigureScanCreatePlanTask(t, service, enginePackageReader)
}

func completeMCPScanConfiguration() map[string]any {
	return map[string]any{"steps": map[string]any{
		"subdomain_discovery": map[string]any{
			"enabled":      true,
			"engineConfig": map[string]any{"recon": map[string]any{"enabled": true, "timeout": 3600, "threads": 10}},
		},
	}}
}

func TestStartMCPScanCreatesOneScanOperationAndStrictlyReplaysRequestID(t *testing.T) {
	store := &mcpScanCreateStoreStub{replays: map[string]*MCPRequestReplay{}}
	service := newMCPScanCreateService(t, store)
	request := MCPScanStartRequest{
		TargetID: 1, ScanWorkflow: "scanWorkflows/default", Configuration: completeMCPScanConfiguration(), RequestID: "start-once",
	}
	first, err := service.StartMCPScan(context.Background(), request)
	if err != nil || first == nil || !strings.HasPrefix(first.Operation, "operations/") || first.Scan != "scans/1" {
		t.Fatalf("first MCP start = %#v, %v", first, err)
	}
	if store.creationCalls != 1 || store.lastScan == nil || store.lastScan.TriggerType != ScanTriggerTypeAI || store.lastScan.InputSource != InputSourceScanSnapshot {
		t.Fatalf("MCP start did not use existing interactive defaults: calls=%d scan=%+v", store.creationCalls, store.lastScan)
	}
	if len(store.lastScan.ScanTasks) != 1 || len(store.lastScan.ScanTasks[0].ResolvedExecutionPlan) == 0 {
		t.Fatalf("MCP start did not freeze a task plan: %+v", store.lastScan)
	}

	replayed, err := service.StartMCPScan(context.Background(), request)
	if err != nil || replayed == nil || *replayed != *first || store.creationCalls != 1 {
		t.Fatalf("same request replay = %#v, %v, calls=%d", replayed, err, store.creationCalls)
	}

	changed := request
	changed.Configuration = completeMCPScanConfiguration()
	changed.Configuration["steps"].(map[string]any)["subdomain_discovery"].(map[string]any)["engineConfig"].(map[string]any)["recon"].(map[string]any)["threads"] = 20
	if _, err := service.StartMCPScan(context.Background(), changed); !errors.Is(err, ErrMCPRequestIDConflict) {
		t.Fatalf("changed request-id payload error = %v", err)
	}
	if store.creationCalls != 1 {
		t.Fatalf("conflicting replay reached create boundary: %d", store.creationCalls)
	}
}

func TestStartMCPScanAllowsIndependentStartsAndHonorsCancellation(t *testing.T) {
	store := &mcpScanCreateStoreStub{replays: map[string]*MCPRequestReplay{}}
	service := newMCPScanCreateService(t, store)
	request := MCPScanStartRequest{TargetID: 1, ScanWorkflow: "scanWorkflows/default", Configuration: completeMCPScanConfiguration()}
	first, err := service.StartMCPScan(context.Background(), request)
	if err != nil {
		t.Fatalf("first independent start: %v", err)
	}
	second, err := service.StartMCPScan(context.Background(), request)
	if err != nil || second == nil || second.Scan == first.Scan || store.creationCalls != 2 {
		t.Fatalf("independent same-target start = %#v, %v, calls=%d", second, err, store.creationCalls)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.StartMCPScan(ctx, request); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-commit cancellation = %v", err)
	}
	if store.creationCalls != 2 {
		t.Fatalf("cancelled start reached create boundary: %d", store.creationCalls)
	}

	// Once StartMCPScan returned after the in-boundary commit, later caller
	// cancellation cannot roll back the durable Scan/operation result.
	postCommit, cancelPostCommit := context.WithCancel(context.Background())
	committed, err := service.StartMCPScan(postCommit, MCPScanStartRequest{
		TargetID: 1, ScanWorkflow: "scanWorkflows/default", Configuration: completeMCPScanConfiguration(), RequestID: "committed-before-disconnect",
	})
	if err != nil || committed == nil {
		t.Fatalf("committed start = %#v, %v", committed, err)
	}
	cancelPostCommit()
	if replay, err := store.FindMCPRequestReplay(context.Background(), "committed-before-disconnect"); err != nil || replay == nil {
		t.Fatalf("post-commit disconnect removed replay result: %+v, %v", replay, err)
	}
}

func TestStartMCPScanDoesNotRetryAFailedSideEffect(t *testing.T) {
	store := &mcpScanCreateStoreStub{
		replays:   map[string]*MCPRequestReplay{},
		createErr: errors.New("transient database failure"),
	}
	service := newMCPScanCreateService(t, store)
	_, err := service.StartMCPScan(context.Background(), MCPScanStartRequest{
		TargetID: 1, ScanWorkflow: "scanWorkflows/default", Configuration: completeMCPScanConfiguration(), RequestID: "failed-once",
	})
	if err == nil || store.creationCalls != 1 {
		t.Fatalf("failed MCP start retries=%d err=%v", store.creationCalls, err)
	}
}
