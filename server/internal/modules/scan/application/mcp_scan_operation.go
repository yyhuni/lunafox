package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	"github.com/yyhuni/lunafox/server/internal/mcp/idempotency"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

const (
	mcpScanStartAction = "start_scan"
)

var (
	// ErrMCPRequestIDConflict prevents one business replay key from being
	// silently reused for different Scan input or another mutation action.
	ErrMCPRequestIDConflict = errors.New("MCP request_id conflicts with an existing request")
	// ErrMCPOperationNotFound is returned only after canonical operation name
	// parsing succeeds and no retained operation remains.
	ErrMCPOperationNotFound = errors.New("MCP operation not found")
)

// MCPScanStartRequest is the application-level one-target immediate Scan
// request. Transport parses resource names before reaching this boundary.
type MCPScanStartRequest struct {
	TargetID      int
	ScanWorkflow  string
	Configuration map[string]any
	AgentID       *int
	RequestID     string
}

// MCPScanStartResult is the bounded polling reference returned after commit.
type MCPScanStartResult struct {
	Operation string
	Scan      string
}

// MCPRequestReplay is the application projection of the shared replay ledger.
type MCPRequestReplay struct {
	RequestID          string
	Action             string
	RequestFingerprint string
	Response           json.RawMessage
	CreatedAt          time.Time
	ExpiresAt          time.Time
}

// MCPScanOperationCreate carries the immutable operation and replay facts that
// must commit with the Scan, Tasks, and saved plans.
type MCPScanOperationCreate struct {
	ID                 string
	RequestID          string
	RequestFingerprint string
	ReplayExpiresAt    time.Time
}

// MCPScanOperation is the query projection for a durable Scan operation.
// Its lifecycle fields are derived from the linked Scan and Task rows.
type MCPScanOperation struct {
	ID          string
	ScanID      int
	TargetID    int
	Status      string
	Phase       string
	Progress    int
	CurrentTask string
	FailureKind string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MCPScanOperationStore is an additive Scan-create boundary. Existing create
// callers keep using ScanCreateCommandStore and cannot accidentally create an
// MCP operation or request replay record.
type MCPScanOperationStore interface {
	CreateWithScanTasksAndPlansAndMCPOperation(context.Context, *CreateScan, *MCPScanOperationCreate, ScanCreateTaskFinalizer) error
	FindMCPRequestReplay(context.Context, string) (*MCPRequestReplay, error)
	GetMCPScanOperation(context.Context, string) (*MCPScanOperation, error)
}

// StartMCPScan starts exactly one new Scan after strict canonical Scan-create
// validation. A request_id only replays an already committed result; it never
// causes the Server to retry a side effect automatically.
func (service *ScanFacade) StartMCPScan(ctx context.Context, request MCPScanStartRequest) (*MCPScanStartResult, error) {
	if service == nil || service.createService == nil {
		return nil, ErrScanInvalidConfig
	}
	return service.createService.StartMCPScan(ctx, request)
}

// GetMCPOperation returns the retained Scan-derived operation projection.
func (service *ScanFacade) GetMCPOperation(ctx context.Context, operationID string) (*MCPScanOperation, error) {
	if service == nil || service.createService == nil {
		return nil, ErrScanInvalidConfig
	}
	store, ok := service.createService.scanStore.(MCPScanOperationStore)
	if !ok || store == nil {
		return nil, ErrScanInvalidConfig
	}
	operation, err := store.GetMCPScanOperation(ctx, strings.TrimSpace(operationID))
	if dberrors.IsRecordNotFound(err) {
		return nil, ErrMCPOperationNotFound
	}
	return operation, err
}

func (service *ScanCreateService) StartMCPScan(ctx context.Context, request MCPScanStartRequest) (*MCPScanStartResult, error) {
	if service == nil || ctx == nil {
		return nil, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if request.TargetID <= 0 {
		return nil, ErrCreateTargetNotFound
	}
	requestID, err := idempotency.NormalizeRequestID(request.RequestID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreateInvalidConfig, err)
	}
	store, ok := service.scanStore.(MCPScanOperationStore)
	if !ok || store == nil {
		return nil, ErrScanInvalidConfig
	}
	fingerprint, err := idempotency.Fingerprint(mcpScanStartAction, struct {
		TargetID      int            `json:"target"`
		ScanWorkflow  string         `json:"scanWorkflow"`
		Configuration map[string]any `json:"configuration"`
		AgentID       *int           `json:"agentId,omitempty"`
	}{TargetID: request.TargetID, ScanWorkflow: strings.TrimSpace(request.ScanWorkflow), Configuration: request.Configuration, AgentID: cloneIntPtr(request.AgentID)})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreateInvalidConfig, err)
	}
	if requestID != "" {
		if replay, err := store.FindMCPRequestReplay(ctx, requestID); err != nil {
			return nil, err
		} else if replay != nil {
			return decodeMCPScanStartReplay(replay, fingerprint)
		}
	}

	plan, err := service.prepareScanCreatePlan(ctx, request.ScanWorkflow, request.Configuration)
	if err != nil {
		return nil, err
	}
	if err := service.validateSelectedAgent(ctx, request.AgentID); err != nil {
		return nil, err
	}
	if service.targetLookup == nil {
		return nil, ErrCreateTargetLookupNotReady
	}
	target, err := service.targetLookup(ctx, request.TargetID)
	if err != nil || target == nil {
		if err != nil {
			return nil, err
		}
		return nil, ErrCreateTargetNotFound
	}

	operation := &MCPScanOperationCreate{
		ID:                 uuid.NewString(),
		RequestID:          requestID,
		RequestFingerprint: fingerprint,
		ReplayExpiresAt:    time.Now().UTC().Add(idempotency.ReplayRetention),
	}
	result, err := service.createPreparedMCPScan(ctx, target, plan.manifest, plan.normalizedConfig, request.AgentID, operation, store)
	if err == nil {
		return result, nil
	}
	if requestID == "" {
		return nil, err
	}
	// A competing request can win the unique replay key after the initial read.
	// Re-read once without submitting another write; this is replay resolution,
	// never an automatic side-effect retry.
	replay, replayErr := store.FindMCPRequestReplay(ctx, requestID)
	if replayErr != nil || replay == nil {
		return nil, err
	}
	return decodeMCPScanStartReplay(replay, fingerprint)
}

func (service *ScanCreateService) createPreparedMCPScan(
	ctx context.Context,
	target *TargetRef,
	manifest ScanCreateWorkflowManifest,
	normalizedConfig dynamicConfiguration,
	agentID *int,
	operation *MCPScanOperationCreate,
	store MCPScanOperationStore,
) (*MCPScanStartResult, error) {
	if target == nil {
		return nil, ErrCreateTargetNotFound
	}
	if operation == nil || strings.TrimSpace(operation.ID) == "" || strings.TrimSpace(operation.RequestFingerprint) == "" {
		return nil, ErrCreateInvalidConfig
	}
	scan := &CreateScan{
		TargetID:       target.ID,
		ScanWorkflowID: manifest.ScanWorkflowID,
		Configuration:  encodeDynamicConfiguration(normalizedConfig),
		// MCP follows the current interactive creation default. The caller may
		// edit public workflow configuration but cannot override this
		// Server-owned provenance/input-source policy through a hidden field.
		InputSource:    InputSourceScanSnapshot,
		TriggerType:    ScanTriggerTypeAI,
		AssignmentMode: assignmentModeForAgent(agentID),
		AgentID:        cloneIntPtr(agentID),
		Status:         CreateScanStatusPending,
	}
	scanTasks, err := buildPlanTaskScanTasks(manifest)
	if err != nil {
		return nil, err
	}
	scan.ScanTasks = scanTasks
	if err := store.CreateWithScanTasksAndPlansAndMCPOperation(ctx, scan, operation, service.planTaskFinalizer(ctx, manifest, normalizedConfig, target, len(scanTasks), scan)); err != nil {
		return nil, err
	}
	return &MCPScanStartResult{Operation: "operations/" + operation.ID, Scan: resourcenames.Scan(scan.ID)}, nil
}

func decodeMCPScanStartReplay(replay *MCPRequestReplay, fingerprint string) (*MCPScanStartResult, error) {
	if replay == nil || replay.Action != mcpScanStartAction || replay.RequestFingerprint != fingerprint {
		return nil, ErrMCPRequestIDConflict
	}
	var result MCPScanStartResult
	if err := json.Unmarshal(replay.Response, &result); err != nil || strings.TrimSpace(result.Operation) == "" || strings.TrimSpace(result.Scan) == "" {
		return nil, ErrScanInvalidConfig
	}
	return &result, nil
}
