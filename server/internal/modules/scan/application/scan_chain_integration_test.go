package application

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/contracts/agentexecution"
	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	contractresults "github.com/yyhuni/lunafox/contracts/results"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	resultingestapp "github.com/yyhuni/lunafox/server/internal/modules/resultingest/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

// integrationStore is an in-memory store that implements all ports required by
// ScanCreateService and ScanTaskBridgeService. It simulates the database
// layer so we can exercise the full scan execution chain without a real DB.
//
// This is the only test that wires ScanCreateService → ScanTaskBridgeService
// together, verifying the cross-layer contract from scan creation through task
// task claim, terminal result reporting, and scan status recalculation.
type integrationStore struct {
	scanIDCounter int
	taskIDCounter int
	scans         map[int]*integrationScan
	tasks         map[int]*integrationTask
}

type integrationScan struct {
	id             int
	targetID       int
	scanWorkflowID string
	configuration  map[string]any
	status         string
	agentID        *int
	failure        *FailureDetail
	targetName     string
	targetType     string
}

type integrationTask struct {
	id                     int
	scanID                 int
	scanWorkflowStageOrder int
	scanWorkflowID         string
	scanWorkflowStageID    string
	scanWorkflowStepID     string
	engineID               string
	status                 string
	resolvedExecutionPlan  []byte
	assignedAgentID        *int
	assignedSessionID      *string
	assignedSessionEpoch   *int64
	assignedRequestID      *string
	taskExecutionConfig    map[string]any
	failure                *FailureDetail
	completedAt            *time.Time
}

func newIntegrationStore() *integrationStore {
	return &integrationStore{
		scans: make(map[int]*integrationScan),
		tasks: make(map[int]*integrationTask),
	}
}

// Status constants matching the domain values. Using string literals to avoid
// importing the domain package, which is the same convention used by the
// repository package for its internal constants.
const (
	intStatusPending   = "pending"
	intStatusRunning   = "running"
	intStatusSucceeded = "succeeded"
	intStatusSkipped   = "skipped"
	intStatusFailed    = "failed"
	intStatusCancelled = "cancelled"
	intStatusBlocked   = "blocked"
)

func multiStageScanTasks(firstStatus string, secondStatus string) []CreateScanTask {
	return []CreateScanTask{
		{
			StageOrder:          1,
			StageID:             "stage1",
			StepOrder:           1,
			StepID:              "step1",
			EngineID:            "engine.lunafox.step1",
			TaskExecutionConfig: workflowStepExecutionConfigForIntegrationTest("stage1", "step1", "engine.lunafox.step1"),
			Status:              firstStatus,
		},
		{
			StageOrder:          2,
			StageID:             "stage2",
			StepOrder:           1,
			StepID:              "step2",
			EngineID:            "engine.lunafox.step2",
			TaskExecutionConfig: workflowStepExecutionConfigForIntegrationTest("stage2", "step2", "engine.lunafox.step2"),
			Status:              secondStatus,
		},
	}
}

func workflowStepExecutionConfigForIntegrationTest(stageID, stepID, engineID string) map[string]any {
	return map[string]any{
		"workflowStepExecution": map[string]any{
			"schemaVersion":  1,
			"scanWorkflowId": "integration_workflow",
			"step": map[string]any{
				"stageId":      stageID,
				"stepId":       stepID,
				"engineId":     engineID,
				"engineConfig": map[string]any{},
			},
		},
	}
}

// --- ScanCreateCommandStore ---

func (s *integrationStore) CreateWithScanTasks(scan *CreateScan) error {
	s.scanIDCounter++
	scan.ID = s.scanIDCounter
	scan.CreatedAt = time.Now().UTC()

	s.scans[scan.ID] = &integrationScan{
		id:             scan.ID,
		targetID:       scan.TargetID,
		scanWorkflowID: scan.ScanWorkflowID,
		configuration:  scan.Configuration,
		status:         scan.Status,
	}

	for _, task := range scan.ScanTasks {
		s.taskIDCounter++
		s.tasks[s.taskIDCounter] = &integrationTask{
			id:                     s.taskIDCounter,
			scanID:                 scan.ID,
			scanWorkflowStageOrder: task.StageOrder,
			scanWorkflowID:         scan.ScanWorkflowID,
			scanWorkflowStageID:    task.StageID,
			scanWorkflowStepID:     task.StepID,
			engineID:               task.EngineID,
			status:                 task.Status,
			resolvedExecutionPlan:  append([]byte(nil), task.ResolvedExecutionPlan...),
			taskExecutionConfig:    task.TaskExecutionConfig,
		}
	}
	return nil
}

func (s *integrationStore) CreateWithScanTasksAndPlans(_ context.Context, scan *CreateScan, finalize ScanCreateTaskFinalizer) error {
	if scan == nil || finalize == nil {
		return fmt.Errorf("scan and PlanTask finalizer are required")
	}
	scanID := s.scanIDCounter + 1
	for index := range scan.ScanTasks {
		if err := finalize(scanID, s.taskIDCounter+index+1, &scan.ScanTasks[index]); err != nil {
			return err
		}
	}
	return s.CreateWithScanTasks(scan)
}

// --- ScanTaskStore (ScanTaskQueryStore + ScanTaskCommandStore) ---

func (s *integrationStore) GetByID(_ context.Context, id int) (*ScanTaskRecord, error) {
	t, ok := s.tasks[id]
	if !ok {
		return nil, nil
	}
	return s.toTaskRecord(t), nil
}

func (s *integrationStore) ClaimNextCompatibleSavedExecutionPlan(
	_ context.Context,
	agentID int,
	sessionID string,
	sessionEpoch int64,
	requestID string,
	supportedEngineAPIMajors []uint32,
) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	for _, task := range s.tasks {
		if task.assignedRequestID != nil && *task.assignedRequestID == requestID {
			return s.integrationPlan(task)
		}
	}
	var selected *integrationTask
	for _, task := range s.tasks {
		scan := s.scans[task.scanID]
		if scan == nil || task.status != intStatusPending || (scan.status != intStatusPending && scan.status != intStatusRunning) || (scan.agentID != nil && *scan.agentID != agentID) {
			continue
		}
		plan, err := s.integrationPlan(task)
		if err != nil {
			return nil, err
		}
		if !integrationSupportsMajor(supportedEngineAPIMajors, plan.GetEngineRelease().GetEngineApiMajor()) {
			continue
		}
		if selected == nil || task.scanWorkflowStageOrder > selected.scanWorkflowStageOrder || (task.scanWorkflowStageOrder == selected.scanWorkflowStageOrder && task.id < selected.id) {
			selected = task
		}
	}
	if selected == nil {
		return nil, nil
	}
	scan := s.scans[selected.scanID]
	scan.agentID = &agentID
	scan.status = intStatusRunning
	selected.status = intStatusRunning
	selected.assignedAgentID = &agentID
	selected.assignedSessionID = &sessionID
	selected.assignedSessionEpoch = &sessionEpoch
	selected.assignedRequestID = &requestID
	return s.integrationPlan(selected)
}

func (s *integrationStore) integrationPlan(task *integrationTask) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	if len(task.resolvedExecutionPlan) != 0 {
		return agentexecution.UnmarshalResolvedEngineExecutionPlan(task.resolvedExecutionPlan)
	}
	return &agentexecutionv1.ResolvedEngineExecutionPlan{
		Execution: "executions/integration-" + fmt.Sprint(task.id),
		Task:      resourcenames.Task(task.scanID, task.id),
		EngineRelease: &agentexecutionv1.EngineRelease{
			Engine:                task.engineID,
			EngineApiMajor:        2,
			CompatibilityRevision: "engine-execution-diagnostics-r1",
		},
	}, nil
}

func integrationSupportsMajor(supported []uint32, required uint32) bool {
	for _, candidate := range supported {
		if candidate == required {
			return true
		}
	}
	return false
}

func (s *integrationStore) ListFailedByScanID(_ context.Context, scanID int) ([]ScanTaskRecord, error) {
	var results []ScanTaskRecord
	for _, t := range s.tasks {
		if t.scanID == scanID && t.status == intStatusFailed {
			results = append(results, *s.toTaskRecord(t))
		}
	}
	return results, nil
}

func (s *integrationStore) CountByStatusForScanID(_ context.Context, scanID int) (pending, running, completed, failed, cancelled, skipped int, err error) {
	for _, t := range s.tasks {
		if t.scanID != scanID {
			continue
		}
		switch t.status {
		case intStatusPending:
			pending++
		case intStatusRunning:
			running++
		case intStatusSucceeded:
			completed++
		case intStatusSkipped:
			skipped++
		case intStatusFailed:
			failed++
		case intStatusCancelled:
			cancelled++
		}
	}
	return
}

func (s *integrationStore) CountActiveByScanAndStageOrder(_ context.Context, scanID, stageOrder int) (int, error) {
	count := 0
	for _, t := range s.tasks {
		if t.scanID == scanID && t.scanWorkflowStageOrder == stageOrder &&
			(t.status == intStatusPending || t.status == intStatusRunning) {
			count++
		}
	}
	return count, nil
}

func (s *integrationStore) CommitScanTaskTerminalStatusForSession(_ context.Context, id, agentID int, sessionID string, sessionEpoch int64, status string, failure *FailureDetail) (bool, error) {
	t, ok := s.tasks[id]
	if !ok || t.status != intStatusRunning ||
		t.assignedAgentID == nil || *t.assignedAgentID != agentID ||
		t.assignedSessionID == nil || *t.assignedSessionID != sessionID ||
		t.assignedSessionEpoch == nil || *t.assignedSessionEpoch != sessionEpoch {
		return false, nil
	}
	scan := s.scans[t.scanID]
	if scan == nil || scan.agentID == nil || *scan.agentID != agentID {
		return false, nil
	}
	t.status = status
	if failure != nil {
		cloned := *failure
		t.failure = &cloned
	}
	now := time.Now().UTC()
	t.completedAt = &now
	return true, nil
}

func (s *integrationStore) RequireCurrentAgentExecutionSession(_ context.Context, agentID int, sessionID string, sessionEpoch int64) error {
	for _, task := range s.tasks {
		if task.assignedAgentID != nil && *task.assignedAgentID == agentID &&
			task.assignedSessionID != nil && *task.assignedSessionID == sessionID &&
			task.assignedSessionEpoch != nil && *task.assignedSessionEpoch == sessionEpoch {
			return nil
		}
	}
	return scandomain.ErrAgentExecutionSessionFenced
}

func (s *integrationStore) ListTerminalTasksPendingReconciliation(context.Context, int, int) ([]ScanTaskRecord, error) {
	return nil, nil
}

func (s *integrationStore) ListSupersededSessionTerminalTasksPendingReconciliation(context.Context, int, int64, int) ([]ScanTaskRecord, error) {
	return nil, nil
}

func (s *integrationStore) ClearTerminalTaskReconciliationPending(context.Context, int) error {
	return nil
}

func (s *integrationStore) FailClaimedTask(_ context.Context, id int, failure *FailureDetail) error {
	t, ok := s.tasks[id]
	if !ok {
		return nil
	}
	t.status = intStatusFailed
	if failure != nil {
		cloned := *failure
		t.failure = &cloned
	}
	now := time.Now().UTC()
	t.completedAt = &now
	return nil
}

func (s *integrationStore) FailTasksForSupersededAgentSession(_ context.Context, agentID int, currentSessionEpoch int64) ([]int, error) {
	seen := make(map[int]struct{})
	for _, task := range s.tasks {
		scan := s.scans[task.scanID]
		if scan == nil || scan.agentID == nil || *scan.agentID != agentID || task.status != intStatusRunning || task.assignedSessionEpoch == nil || *task.assignedSessionEpoch >= currentSessionEpoch {
			continue
		}
		task.status = intStatusFailed
		task.failure = &FailureDetail{Kind: "agent_disconnected", Message: "Agent disconnected"}
		now := time.Now().UTC()
		task.completedAt = &now
		seen[task.scanID] = struct{}{}
	}
	result := make([]int, 0, len(seen))
	for scanID := range seen {
		result = append(result, scanID)
	}
	return result, nil
}

func (s *integrationStore) SkipUnstartedTasksByScanID(_ context.Context, scanID int, _ string) (int64, error) {
	var affected int64
	now := time.Now().UTC()
	for _, t := range s.tasks {
		if t.scanID != scanID || (t.status != intStatusPending && t.status != intStatusBlocked) {
			continue
		}
		t.status = intStatusSkipped
		t.completedAt = &now
		affected++
	}
	return affected, nil
}

func (s *integrationStore) CancelUnstartedTasksByScanID(_ context.Context, scanID int) (int64, error) {
	var affected int64
	now := time.Now().UTC()
	for _, t := range s.tasks {
		if t.scanID != scanID || (t.status != intStatusPending && t.status != intStatusBlocked) {
			continue
		}
		t.status = intStatusCancelled
		t.completedAt = &now
		affected++
	}
	return affected, nil
}

func (s *integrationStore) UnlockNextStageOrder(_ context.Context, scanID, stageOrder int) (int64, error) {
	nextOrder := -1
	for _, t := range s.tasks {
		if t.scanID == scanID && t.status == intStatusBlocked && t.scanWorkflowStageOrder > stageOrder {
			if s.hasUnlockBlockerBetween(scanID, stageOrder, t.scanWorkflowStageOrder) {
				continue
			}
			if nextOrder == -1 || t.scanWorkflowStageOrder < nextOrder {
				nextOrder = t.scanWorkflowStageOrder
			}
		}
	}
	if nextOrder == -1 {
		return 0, nil
	}
	var affected int64
	for _, t := range s.tasks {
		if t.scanID == scanID && t.status == intStatusBlocked && t.scanWorkflowStageOrder == nextOrder {
			t.status = intStatusPending
			affected++
		}
	}
	return affected, nil
}

func (s *integrationStore) hasUnlockBlockerBetween(scanID, afterStageOrder, beforeStageOrder int) bool {
	for _, t := range s.tasks {
		if t.scanID != scanID || t.scanWorkflowStageOrder <= afterStageOrder || t.scanWorkflowStageOrder >= beforeStageOrder {
			continue
		}
		if t.status == intStatusPending || t.status == intStatusRunning || t.status == intStatusFailed || t.status == intStatusCancelled {
			return true
		}
	}
	return false
}

// --- ScanTaskRuntimeScanStore (ScanTaskRuntimeScanQueryStore + ScanTaskRuntimeScanCommandStore) ---

func (s *integrationStore) GetScanForScanTask(scanID int) (*ScanTaskRuntimeScanRecord, error) {
	scan, ok := s.scans[scanID]
	if !ok {
		return nil, nil
	}
	record := &ScanTaskRuntimeScanRecord{
		ID:       scan.id,
		TargetID: scan.targetID,
		Status:   scan.status,
		Failure:  scan.failure,
	}
	if scan.targetName != "" {
		record.Target = &ScanTaskTargetRef{
			ID:   scan.targetID,
			Name: scan.targetName,
			Type: scan.targetType,
		}
	}
	return record, nil
}

func (s *integrationStore) UpdateScanStatus(id int, status string, failure *FailureDetail) error {
	scan, ok := s.scans[id]
	if !ok {
		return nil
	}
	scan.status = status
	if failure != nil {
		cloned := *failure
		scan.failure = &cloned
	}
	return nil
}

// --- helper ---

func (s *integrationStore) toTaskRecord(t *integrationTask) *ScanTaskRecord {
	return &ScanTaskRecord{
		ID:                       t.id,
		ScanID:                   t.scanID,
		ScanWorkflowStageOrder:   t.scanWorkflowStageOrder,
		ScanWorkflowID:           t.scanWorkflowID,
		ScanWorkflowStageID:      t.scanWorkflowStageID,
		ScanWorkflowStepID:       t.scanWorkflowStepID,
		EngineID:                 t.engineID,
		Status:                   t.status,
		AgentID:                  s.scans[t.scanID].agentID,
		AssignedAgentID:          t.assignedAgentID,
		AssignedSessionID:        t.assignedSessionID,
		AssignedSessionEpoch:     t.assignedSessionEpoch,
		AssignedRequestID:        t.assignedRequestID,
		HasResolvedExecutionPlan: len(t.resolvedExecutionPlan) > 0,
		TaskExecutionConfig:      t.taskExecutionConfig,
		Failure:                  t.failure,
		CompletedAt:              t.completedAt,
	}
}

// --- Target lookup stub ---

func integrationTargetLookup(_ context.Context, id int) (*TargetRef, error) {
	now := time.Now().UTC()
	return &TargetRef{ID: id, Name: "example.com", Type: "domain", CreatedAt: now}, nil
}

// --- Workflow/engine stubs (shared with multi-stage test) ---

type integrationWorkflowReaderStub struct{}

func (integrationWorkflowReaderStub) GetScanWorkflowManifest(_ context.Context, id string) (ScanCreateWorkflowManifest, error) {
	if id == "multi_stage" {
		return ScanCreateWorkflowManifest{
			ScanWorkflowID: "multi_stage",
			Stages: []ScanCreateWorkflowStage{
				{
					StageID: "stage1",
					Steps: []ScanCreateWorkflowStep{
						{StepID: "step1", EngineID: "engine.lunafox.step1"},
					},
				},
				{
					StageID: "stage2",
					Steps: []ScanCreateWorkflowStep{
						{StepID: "step2", EngineID: "engine.lunafox.step2"},
					},
				},
			},
		}, nil
	}
	return ScanCreateWorkflowManifest{
		ScanWorkflowID: "subdomain_discovery",
		Stages: []ScanCreateWorkflowStage{{
			StageID: "discovery",
			Steps: []ScanCreateWorkflowStep{{
				StepID:   "subdomain_discovery",
				EngineID: "engine.lunafox.subdomain_discovery",
			}},
		}},
	}, nil
}

type integrationEnginePackageReaderStub struct{}

func (integrationEnginePackageReaderStub) LoadEnginePackage(_ context.Context, engineID string) (ScanCreateEnginePackage, error) {
	return ScanCreateEnginePackage{Package: integrationPlanTaskPackage(engineID)}, nil
}

func (integrationEnginePackageReaderStub) ListEnginePackagesByID() (map[string]ScanCreateEnginePackage, error) {
	packages := map[string]ScanCreateEnginePackage{}
	for _, engineID := range []string{
		"engine.lunafox.subdomain_discovery",
		"engine.lunafox.step1",
		"engine.lunafox.step2",
	} {
		packages[engineID] = ScanCreateEnginePackage{Package: integrationPlanTaskPackage(engineID)}
	}
	return packages, nil
}

func (integrationEnginePackageReaderStub) LoadExactPackage(_ context.Context, identity PlanTaskPackageIdentity) (PlanTaskPackage, error) {
	return integrationPlanTaskPackage(identity.EngineID), nil
}

func integrationPlanTaskPackage(engineID string) PlanTaskPackage {
	definition := testPlanTaskDefinition(nil, nil, false)
	definition.EngineID = engineID
	loaded := planTaskExactPackage(definition)
	loaded.Identity.EngineID = engineID
	slug := strings.ReplaceAll(strings.TrimPrefix(engineID, "engine.lunafox."), "_", "-")
	repository := "lunafox-engine-runtime-" + slug
	loaded.RuntimeImageRefs = []string{
		"docker.io/lunafox/" + repository + "@" + planTaskImageDigest,
		"ghcr.io/lunafox/" + repository + "@" + planTaskImageDigest,
	}
	return loaded
}

func newIntegrationScanCreateService(t *testing.T, store *integrationStore) *ScanCreateService {
	t.Helper()
	packageReader := integrationEnginePackageReaderStub{}
	compiler, err := NewPlanTaskCompiler(packageReader, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler failed: %v", err)
	}
	service := NewScanCreateService(store, integrationTargetLookup, nil, integrationWorkflowReaderStub{}, packageReader)
	if err := service.ConfigurePlanTask(compiler, FixedPlanTaskLimitsProvider(time.Hour)); err != nil {
		t.Fatalf("ConfigurePlanTask failed: %v", err)
	}
	return service
}

// --- ResultTaskScopeDataPlane ---

type integrationResultTaskScope struct {
	TaskID   int
	ScanID   int
	TargetID int
}

func (s *integrationStore) GetResultTaskScopeByTaskID(_ context.Context, taskID int) (*integrationResultTaskScope, error) {
	t, ok := s.tasks[taskID]
	if !ok {
		return nil, nil
	}
	scan := s.scans[t.scanID]
	if scan == nil {
		return nil, nil
	}
	return &integrationResultTaskScope{
		TaskID:   t.id,
		ScanID:   t.scanID,
		TargetID: scan.targetID,
	}, nil
}

// --- SubdomainResultMaterializer stub ---

type integrationMaterializerCall struct {
	ScanID   int
	TargetID int
	Items    []snapshotapp.SubdomainSnapshotItem
}

type integrationSubdomainMaterializerStub struct {
	calls []integrationMaterializerCall
}

type integrationResultSummaryUpdaterStub struct{}

func (integrationResultSummaryUpdaterStub) RefreshScanResultSummary(context.Context, int, int) error {
	return nil
}

// The integration store has no database or concurrent lifecycle state. Its
// coordinator preserves the production Facade contract while the real
// Target/Scan/Task fence is covered by repository-backed wiring tests.
type integrationResultMaterializationCoordinator struct{}

func (integrationResultMaterializationCoordinator) Materialize(ctx context.Context, _ resultingestapp.ResultMaterializationScope, persist func(context.Context) error) error {
	return persist(ctx)
}

func (stub *integrationSubdomainMaterializerStub) SaveAndSyncContext(_ context.Context, scanID int, targetID int, items []snapshotapp.SubdomainSnapshotItem) (snapshotapp.MaterializationSummary, error) {
	stub.calls = append(stub.calls, integrationMaterializerCall{
		ScanID:   scanID,
		TargetID: targetID,
		Items:    items,
	})
	count := int64(len(items))
	return snapshotapp.MaterializationSummary{ReceivedItems: len(items), SnapshotCount: count, AssetCount: count}, nil
}

func integrationResultCommand(taskID int, scope *integrationResultTaskScope, resultType string, itemsJSON []string) resultingestapp.ResultIngestCommand {
	items := make([][]byte, len(itemsJSON))
	for index := range itemsJSON {
		items[index] = []byte(itemsJSON[index])
	}
	return resultingestapp.ResultIngestCommand{
		TaskID: taskID, ScanID: scope.ScanID, TargetID: scope.TargetID,
		ResultType: resultType, Items: items,
	}
}

func (stub *integrationSubdomainMaterializerStub) totalItemCount() int {
	count := 0
	for _, call := range stub.calls {
		count += len(call.Items)
	}
	return count
}

func (stub *integrationSubdomainMaterializerStub) allDNSNames() []string {
	var names []string
	for _, call := range stub.calls {
		for _, item := range call.Items {
			names = append(names, item.DNSName)
		}
	}
	return names
}

func createSingleTargetBatchScanForIntegrationTest(t *testing.T, service *ScanCreateService) *CreateScan {
	t.Helper()
	result, err := service.CreateBatch(context.Background(), &CreateBatchInput{
		Requests:     []CreateBatchItem{{TargetID: 1}},
		ScanWorkflow: "scanWorkflows/default",
		TriggerType:  ScanTriggerTypeManual,
		InputSource:  InputSourceScanSnapshot,
		Configuration: completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{
			"subdomain_discovery": testPlanTaskDefinition(nil, nil, false).Execution,
		}),
	})
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if result == nil || len(result.Scans) != 1 {
		t.Fatalf("expected one created scan, got %+v", result)
	}
	return &result.Scans[0]
}

type integrationExecutionClaim struct {
	Plan      *agentexecutionv1.ResolvedEngineExecutionPlan
	TaskID    int
	ScanID    int
	SessionID string
}

func claimNextIntegrationExecution(ctx context.Context, bridge *ScanTaskBridgeService, agentID int, sessionEpoch int64) (*integrationExecutionClaim, error) {
	sessionID := fmt.Sprintf("session-%d-%d", agentID, sessionEpoch)
	plan, err := bridge.ClaimNextExecutionPlan(ctx, agentID, sessionID, sessionEpoch, uuid.NewString(), agentdomain.AgentExecutionCapabilitySnapshot{
		ContainerRuntimeReady:    true,
		SupportedEngineAPIMajors: []uint32{2},
	})
	if err != nil || plan == nil {
		return nil, err
	}
	scanID, taskID, err := resourcenames.ParseTask(plan.GetTask())
	if err != nil {
		return nil, err
	}
	return &integrationExecutionClaim{Plan: plan, TaskID: taskID, ScanID: scanID, SessionID: sessionID}, nil
}

// --- Tests ---

// TestScanChain_CreateClaimSucceed_RecalculatesStatus verifies the full chain:
// 1. CreateBatch persists scan + tasks
// 2. ClaimNextExecutionPlan claims a pending saved plan and promotes scan to running
// 3. ReportTerminalTaskResult reports success for the exact execution session
// 4. Scan status is recalculated to "succeeded"
func TestScanChain_CreateClaimSucceed_RecalculatesStatus(t *testing.T) {
	store := newIntegrationStore()

	// Step 1: Create a scan with tasks.
	createService := newIntegrationScanCreateService(t, store)

	scan := createSingleTargetBatchScanForIntegrationTest(t, createService)
	if scan.ID == 0 {
		t.Fatal("expected scan to be persisted with an ID")
	}
	if scan.Status != intStatusPending {
		t.Fatalf("expected initial scan status 'pending', got %q", scan.Status)
	}

	// Verify tasks were created.
	taskCount := 0
	for _, task := range store.tasks {
		if task.scanID == scan.ID {
			taskCount++
		}
	}
	if taskCount == 0 {
		t.Fatal("expected at least one task to be created")
	}

	// Step 2: Claim the task (simulates agent requesting work).
	bridge := newScanTaskBridgeServiceForTest(store, store)

	assignment, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 1)
	if err != nil {
		t.Fatalf("ClaimNextExecutionPlan failed: %v", err)
	}
	if assignment == nil {
		t.Fatal("expected task assignment, got nil")
	}
	if assignment.ScanID != scan.ID {
		t.Fatalf("expected assignment scanID %d, got %d", scan.ID, assignment.ScanID)
	}
	if assignment.Plan.GetEngineRelease().GetEngine() != "engine.lunafox.subdomain_discovery" {
		t.Fatalf("expected canonical engine ID, got %q", assignment.Plan.GetEngineRelease().GetEngine())
	}
	if assignment.Plan.GetEngineRelease().GetEngineApiMajor() != 2 {
		t.Fatalf("expected Engine API major 2, got %d", assignment.Plan.GetEngineRelease().GetEngineApiMajor())
	}
	if assignment.Plan.GetTask() != resourcenames.Task(scan.ID, assignment.TaskID) {
		t.Fatalf("expected canonical task resource, got %q", assignment.Plan.GetTask())
	}

	// Verify scan was promoted to running.
	scanRecord, err := store.GetScanForScanTask(scan.ID)
	if err != nil {
		t.Fatalf("GetScanForScanTask failed: %v", err)
	}
	if scanRecord.Status != intStatusRunning {
		t.Fatalf("expected scan status 'running' after claim, got %q", scanRecord.Status)
	}

	// Verify task was claimed.
	taskRecord, err := store.GetByID(context.Background(), assignment.TaskID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if taskRecord.Status != intStatusRunning {
		t.Fatalf("expected task status 'running', got %q", taskRecord.Status)
	}
	if taskRecord.AgentID == nil || *taskRecord.AgentID != 1 {
		t.Fatalf("expected task assigned to agent 1, got %v", taskRecord.AgentID)
	}

	// Step 3: Report task success for the claimed execution session.
	err = bridge.ReportTerminalTaskResult(
		context.Background(),
		1, assignment.SessionID, 1,
		assignment.TaskID,
		intStatusSucceeded,
		nil,
	)
	if err != nil {
		t.Fatalf("ReportTerminalTaskResult failed: %v", err)
	}

	// Step 4: Verify scan status was recalculated to succeeded.
	scanRecord, err = store.GetScanForScanTask(scan.ID)
	if err != nil {
		t.Fatalf("GetScanForScanTask failed: %v", err)
	}
	if scanRecord.Status != intStatusSucceeded {
		t.Fatalf("expected scan status 'succeeded' after all tasks complete, got %q", scanRecord.Status)
	}

	// Verify task terminal state.
	taskRecord, err = store.GetByID(context.Background(), assignment.TaskID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if taskRecord.Status != intStatusSucceeded {
		t.Fatalf("expected task status 'succeeded', got %q", taskRecord.Status)
	}
	if taskRecord.CompletedAt == nil {
		t.Fatal("expected task completed_at to be set")
	}
}

// TestScanChain_CreateClaimFail_RecalculatesToFailed verifies:
// 1. CreateBatch persists scan + tasks
// 2. ClaimNextExecutionPlan claims a pending saved plan
// 3. ReportTerminalTaskResult reports failure for the exact execution session
// 4. Scan status is recalculated to "failed" with the failure detail
func TestScanChain_CreateClaimFail_RecalculatesToFailed(t *testing.T) {
	store := newIntegrationStore()

	createService := newIntegrationScanCreateService(t, store)

	scan := createSingleTargetBatchScanForIntegrationTest(t, createService)

	bridge := newScanTaskBridgeServiceForTest(store, store)

	assignment, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 1)
	if err != nil {
		t.Fatalf("ClaimNextExecutionPlan failed: %v", err)
	}
	if assignment == nil {
		t.Fatal("expected task assignment")
	}

	// Report task failure.
	failure := &FailureDetail{Kind: "runtime_error", Message: "engine crashed"}
	err = bridge.ReportTerminalTaskResult(
		context.Background(),
		1, assignment.SessionID, 1,
		assignment.TaskID,
		intStatusFailed,
		failure,
	)
	if err != nil {
		t.Fatalf("ReportTerminalTaskResult failed: %v", err)
	}

	// Verify scan status was recalculated to failed.
	scanRecord, err := store.GetScanForScanTask(scan.ID)
	if err != nil {
		t.Fatalf("GetScanForScanTask failed: %v", err)
	}
	if scanRecord.Status != intStatusFailed {
		t.Fatalf("expected scan status 'failed', got %q", scanRecord.Status)
	}

	// Verify the failure detail was propagated to the scan.
	if scanRecord.Failure == nil {
		t.Fatal("expected scan failure detail to be set")
	}
	if scanRecord.Failure.Kind != "runtime_error" {
		t.Fatalf("expected failure kind 'runtime_error', got %q", scanRecord.Failure.Kind)
	}
	if scanRecord.Failure.Message != "engine crashed" {
		t.Fatalf("expected failure message 'engine crashed', got %q", scanRecord.Failure.Message)
	}
}

// TestScanChain_MultiTask_StageUnlock verifies:
// 1. Two tasks in different stages (stage 1 pending, stage 2 blocked)
// 2. Claim + succeed on stage 1 task
// 3. Stage 2 task is unlocked to pending
// 4. Claim + succeed on stage 2 task
// 5. Scan status = succeeded
func TestScanChain_MultiTask_StageUnlock(t *testing.T) {
	store := newIntegrationStore()

	scan := &CreateScan{
		TargetID:       1,
		ScanWorkflowID: "multi_stage",
		Status:         intStatusPending,
		ScanTasks: multiStageScanTasks(
			CreateTaskStatusPending,
			CreateTaskStatusBlocked,
		),
	}
	store.CreateWithScanTasks(scan)

	store.scans[scan.ID].targetName = "example.com"
	store.scans[scan.ID].targetType = "domain"

	bridge := newScanTaskBridgeServiceForTest(store, store)

	// Claim stage 1 task.
	assignment1, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 1)
	if err != nil {
		t.Fatalf("ClaimNextExecutionPlan (stage 1) failed: %v", err)
	}
	if assignment1 == nil {
		t.Fatal("expected stage 1 assignment")
	}

	// Succeed stage 1.
	err = bridge.ReportTerminalTaskResult(context.Background(), 1, assignment1.SessionID, 1, assignment1.TaskID, intStatusSucceeded, nil)
	if err != nil {
		t.Fatalf("ReportTerminalTaskResult (stage 1) failed: %v", err)
	}

	// Verify stage 2 task was unlocked to pending.
	var stage2Task *integrationTask
	for _, task := range store.tasks {
		if task.scanWorkflowStageOrder == 2 {
			stage2Task = task
			break
		}
	}
	if stage2Task == nil {
		t.Fatal("stage 2 task not found")
	}
	if stage2Task.status != intStatusPending {
		t.Fatalf("expected stage 2 task to be unlocked to 'pending', got %q", stage2Task.status)
	}

	// Scan should still be running (stage 2 has a pending task).
	scanRecord, _ := store.GetScanForScanTask(scan.ID)
	if scanRecord.Status != intStatusRunning {
		t.Fatalf("expected scan 'running' while stage 2 pending, got %q", scanRecord.Status)
	}

	// Claim stage 2 task.
	assignment2, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 2)
	if err != nil {
		t.Fatalf("ClaimNextExecutionPlan (stage 2) failed: %v", err)
	}
	if assignment2 == nil {
		t.Fatal("expected stage 2 assignment")
	}

	// Succeed stage 2.
	err = bridge.ReportTerminalTaskResult(context.Background(), 1, assignment2.SessionID, 2, assignment2.TaskID, intStatusSucceeded, nil)
	if err != nil {
		t.Fatalf("ReportTerminalTaskResult (stage 2) failed: %v", err)
	}

	// Scan should now be succeeded.
	scanRecord, _ = store.GetScanForScanTask(scan.ID)
	if scanRecord.Status != intStatusSucceeded {
		t.Fatalf("expected scan 'succeeded', got %q", scanRecord.Status)
	}
}

func TestScanChain_MultiTaskStageUnlock_WaitsForFanIn(t *testing.T) {
	store := newIntegrationStore()

	scan := &CreateScan{
		TargetID:       1,
		ScanWorkflowID: "multi_stage_fan_in",
		Status:         intStatusPending,
		ScanTasks: []CreateScanTask{
			{
				StageOrder:          1,
				StageID:             "stage1",
				StepOrder:           1,
				StepID:              "step1a",
				EngineID:            "engine.lunafox.step1",
				TaskExecutionConfig: workflowStepExecutionConfigForIntegrationTest("stage1", "step1a", "engine.lunafox.step1"),
				Status:              CreateTaskStatusPending,
			},
			{
				StageOrder:          1,
				StageID:             "stage1",
				StepOrder:           2,
				StepID:              "step1b",
				EngineID:            "engine.lunafox.step1",
				TaskExecutionConfig: workflowStepExecutionConfigForIntegrationTest("stage1", "step1b", "engine.lunafox.step1"),
				Status:              CreateTaskStatusPending,
			},
			{
				StageOrder:          2,
				StageID:             "stage2",
				StepOrder:           1,
				StepID:              "step2",
				EngineID:            "engine.lunafox.step2",
				TaskExecutionConfig: workflowStepExecutionConfigForIntegrationTest("stage2", "step2", "engine.lunafox.step2"),
				Status:              CreateTaskStatusBlocked,
			},
		},
	}
	store.CreateWithScanTasks(scan)
	store.scans[scan.ID].targetName = "example.com"
	store.scans[scan.ID].targetType = "domain"

	bridge := newScanTaskBridgeServiceForTest(store, store)

	first, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 1)
	if err != nil || first == nil {
		t.Fatalf("first stage claim failed: assignment=%+v err=%v", first, err)
	}
	if err := bridge.ReportTerminalTaskResult(context.Background(), 1, first.SessionID, 1, first.TaskID, intStatusSucceeded, nil); err != nil {
		t.Fatalf("first stage task success failed: %v", err)
	}

	stage2Task := integrationTaskByStageOrder(store, scan.ID, 2)
	if stage2Task == nil || stage2Task.status != intStatusBlocked {
		t.Fatalf("expected stage 2 to remain blocked while stage 1 still has active tasks, got %+v", stage2Task)
	}

	second, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 2)
	if err != nil || second == nil {
		t.Fatalf("second stage-1 claim failed: assignment=%+v err=%v", second, err)
	}
	if secondTask := store.tasks[second.TaskID]; secondTask == nil || secondTask.scanWorkflowStageOrder != 1 {
		t.Fatalf("expected second assignment to stay in stage 1, got task %+v", secondTask)
	}
	if err := bridge.ReportTerminalTaskResult(context.Background(), 1, second.SessionID, 2, second.TaskID, intStatusSucceeded, nil); err != nil {
		t.Fatalf("second stage task success failed: %v", err)
	}

	if stage2Task.status != intStatusPending {
		t.Fatalf("expected stage 2 to unlock after all stage 1 tasks finished, got %q", stage2Task.status)
	}
}

func TestScanChain_MixedSucceededAndSkippedStageUnlocksNextStage(t *testing.T) {
	store := newIntegrationStore()
	scan := &CreateScan{
		TargetID:       1,
		ScanWorkflowID: "mixed_skipped_stage",
		Status:         intStatusPending,
		ScanTasks: []CreateScanTask{
			{
				StageOrder: 1, StageID: "stage1", StepOrder: 1, StepID: "disabled_step",
				EngineID: "engine.lunafox.disabled", Status: CreateTaskStatusSkipped,
			},
			{
				StageOrder: 1, StageID: "stage1", StepOrder: 2, StepID: "active_step",
				EngineID:            "engine.lunafox.active",
				TaskExecutionConfig: workflowStepExecutionConfigForIntegrationTest("stage1", "active_step", "engine.lunafox.active"),
				Status:              CreateTaskStatusPending,
			},
			{
				StageOrder: 2, StageID: "stage2", StepOrder: 1, StepID: "downstream_step",
				EngineID:            "engine.lunafox.downstream",
				TaskExecutionConfig: workflowStepExecutionConfigForIntegrationTest("stage2", "downstream_step", "engine.lunafox.downstream"),
				Status:              CreateTaskStatusBlocked,
			},
		},
	}
	store.CreateWithScanTasks(scan)
	store.scans[scan.ID].targetName = "example.com"
	store.scans[scan.ID].targetType = "domain"
	bridge := newScanTaskBridgeServiceForTest(store, store)

	first, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 1)
	if err != nil || first == nil {
		t.Fatalf("active Step claim failed: assignment=%+v err=%v", first, err)
	}
	if err := bridge.ReportTerminalTaskResult(context.Background(), 1, first.SessionID, 1, first.TaskID, intStatusSucceeded, nil); err != nil {
		t.Fatalf("active Step success failed: %v", err)
	}

	downstream := integrationTaskByStageOrder(store, scan.ID, 2)
	if downstream == nil || downstream.status != intStatusPending {
		t.Fatalf("mixed succeeded/skipped Stage did not unlock downstream Stage: %+v", downstream)
	}
	scanRecord, _ := store.GetScanForScanTask(scan.ID)
	if scanRecord.Status != intStatusRunning {
		t.Fatalf("expected scan to remain running with downstream pending, got %q", scanRecord.Status)
	}

	second, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 2)
	if err != nil || second == nil {
		t.Fatalf("downstream Step claim failed: assignment=%+v err=%v", second, err)
	}
	if err := bridge.ReportTerminalTaskResult(context.Background(), 1, second.SessionID, 2, second.TaskID, intStatusSucceeded, nil); err != nil {
		t.Fatalf("downstream Step success failed: %v", err)
	}
	scanRecord, _ = store.GetScanForScanTask(scan.ID)
	if scanRecord.Status != intStatusSucceeded {
		t.Fatalf("expected mixed succeeded/skipped workflow to succeed, got %q", scanRecord.Status)
	}
}

func TestScanChain_AllSkippedStageUnlocksNextStage(t *testing.T) {
	store := newIntegrationStore()
	scan := &CreateScan{
		TargetID:       1,
		ScanWorkflowID: "all_skipped_stage",
		Status:         intStatusPending,
		ScanTasks: []CreateScanTask{
			{StageOrder: 1, StageID: "stage1", StepOrder: 1, StepID: "disabled_a", EngineID: "engine.lunafox.disabled-a", Status: CreateTaskStatusSkipped},
			{StageOrder: 1, StageID: "stage1", StepOrder: 2, StepID: "disabled_b", EngineID: "engine.lunafox.disabled-b", Status: CreateTaskStatusSkipped},
			{
				StageOrder: 2, StageID: "stage2", StepOrder: 1, StepID: "downstream_step",
				EngineID: "engine.lunafox.downstream",
				Status:   CreateTaskStatusBlocked,
			},
		},
	}
	store.CreateWithScanTasks(scan)
	bridge := newScanTaskBridgeServiceForTest(store, store)

	if err := bridge.unlockNextStageIfReady(context.Background(), scan.ID, 1); err != nil {
		t.Fatalf("all-skipped Stage reconciliation failed: %v", err)
	}
	downstream := integrationTaskByStageOrder(store, scan.ID, 2)
	if downstream == nil || downstream.status != intStatusPending {
		t.Fatalf("all-skipped Stage did not unlock downstream Stage: %+v", downstream)
	}
}

func TestScanChain_MultiStageFailureCancelsLaterStageAndLeavesItUnclaimable(t *testing.T) {
	store := newIntegrationStore()
	scan := &CreateScan{
		TargetID:       1,
		ScanWorkflowID: "multi_stage",
		Status:         intStatusPending,
		ScanTasks: multiStageScanTasks(
			CreateTaskStatusPending,
			CreateTaskStatusBlocked,
		),
	}
	store.CreateWithScanTasks(scan)
	store.scans[scan.ID].targetName = "example.com"
	store.scans[scan.ID].targetType = "domain"

	bridge := newScanTaskBridgeServiceForTest(store, store)

	assignment, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 1)
	if err != nil || assignment == nil {
		t.Fatalf("stage 1 claim failed: assignment=%+v err=%v", assignment, err)
	}
	failure := &FailureDetail{Kind: "runtime_error", Message: "stage 1 failed"}
	if err := bridge.ReportTerminalTaskResult(context.Background(), 1, assignment.SessionID, 1, assignment.TaskID, intStatusFailed, failure); err != nil {
		t.Fatalf("stage 1 failure update failed: %v", err)
	}

	stage2Task := integrationTaskByStageOrder(store, scan.ID, 2)
	if stage2Task == nil || stage2Task.status != intStatusCancelled {
		t.Fatalf("expected later stage to be cancelled after failure, got %+v", stage2Task)
	}
	if stage2Task.completedAt == nil {
		t.Fatalf("expected cancelled later stage to have completed_at set")
	}
	next, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 2)
	if err != nil {
		t.Fatalf("claim after failure returned error: %v", err)
	}
	if next != nil {
		t.Fatalf("expected no assignment after failed dependency, got %+v", next)
	}
	scanRecord, _ := store.GetScanForScanTask(scan.ID)
	if scanRecord.Status != intStatusFailed {
		t.Fatalf("expected scan failed after stage 1 failure, got %q", scanRecord.Status)
	}
}

func integrationTaskByStageOrder(store *integrationStore, scanID, stageOrder int) *integrationTask {
	for _, task := range store.tasks {
		if task.scanID == scanID && task.scanWorkflowStageOrder == stageOrder {
			return task
		}
	}
	return nil
}

// TestScanChain_CreateClaimResultIngestSucceed verifies the full data-plane chain:
// 1. CreateBatch persists scan + tasks
// 2. ClaimNextExecutionPlan claims a pending saved plan
// 3. BatchIngestTaskResults submits subdomain results
// 4. Results are persisted via SubdomainResultMaterializer
// 5. ReportTerminalTaskResult reports success
// 6. Scan status is recalculated to "succeeded"
func TestScanChain_CreateClaimResultIngestSucceed(t *testing.T) {
	store := newIntegrationStore()
	materializer := &integrationSubdomainMaterializerStub{}

	// Step 1: Create a scan.
	createService := newIntegrationScanCreateService(t, store)

	scan := createSingleTargetBatchScanForIntegrationTest(t, createService)

	// Step 2: Claim the task.
	bridge := newScanTaskBridgeServiceForTest(store, store)
	assignment, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 1)
	if err != nil {
		t.Fatalf("ClaimNextExecutionPlan failed: %v", err)
	}
	if assignment == nil {
		t.Fatal("expected task assignment")
	}

	// Step 3: Submit results via ResultIngestFacade (data plane path).
	ingestFacade := resultingestapp.NewResultIngestFacade(resultingestapp.ResultIngestFacadeDependencies{Subdomains: materializer, ScanSummary: integrationResultSummaryUpdaterStub{}, Materialization: integrationResultMaterializationCoordinator{}})

	scope, err := store.GetResultTaskScopeByTaskID(context.Background(), assignment.TaskID)
	if err != nil {
		t.Fatalf("GetResultTaskScopeByTaskID failed: %v", err)
	}
	if scope == nil {
		t.Fatal("expected task scope, got nil")
	}

	itemsJSON := []string{
		`{"dnsName":"api.example.com"}`,
		`{"dnsName":"admin.example.com"}`,
		`{"dnsName":"cdn.example.com"}`,
	}
	output, err := ingestFacade.Ingest(context.Background(), integrationResultCommand(assignment.TaskID, scope, contractresults.ResultKindAssetSubdomain, itemsJSON))
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}
	if output.ReceivedItems != 3 {
		t.Fatalf("expected 3 received items, got %d", output.ReceivedItems)
	}

	// Verify materializer received the correct data.
	if len(materializer.calls) != 1 {
		t.Fatalf("expected 1 materializer call, got %d", len(materializer.calls))
	}
	call := materializer.calls[0]
	if call.ScanID != scan.ID {
		t.Fatalf("materializer received scanID %d, expected %d", call.ScanID, scan.ID)
	}
	if call.TargetID != 1 {
		t.Fatalf("materializer received targetID %d, expected 1", call.TargetID)
	}
	if len(call.Items) != 3 {
		t.Fatalf("materializer received %d items, expected 3", len(call.Items))
	}
	expectedDNS := map[string]bool{"api.example.com": true, "admin.example.com": true, "cdn.example.com": true}
	for _, item := range call.Items {
		if !expectedDNS[item.DNSName] {
			t.Fatalf("unexpected DNS name in materializer: %q", item.DNSName)
		}
		delete(expectedDNS, item.DNSName)
	}
	if len(expectedDNS) > 0 {
		t.Fatalf("materializer missing expected DNS names: %v", expectedDNS)
	}

	// Step 4: Report task success.
	err = bridge.ReportTerminalTaskResult(
		context.Background(),
		1, assignment.SessionID, 1,
		assignment.TaskID,
		intStatusSucceeded,
		nil,
	)
	if err != nil {
		t.Fatalf("ReportTerminalTaskResult failed: %v", err)
	}

	// Step 5: Verify scan status = succeeded.
	scanRecord, err := store.GetScanForScanTask(scan.ID)
	if err != nil {
		t.Fatalf("GetScanForScanTask failed: %v", err)
	}
	if scanRecord.Status != intStatusSucceeded {
		t.Fatalf("expected scan 'succeeded', got %q", scanRecord.Status)
	}
}

// TestScanChain_ResultIngestThenTaskFail_ResultsPersistedScanFailed verifies:
// Results are submitted and persisted, but the task subsequently fails.
// The scan should be "failed" — results and status are independent.
func TestScanChain_ResultIngestThenTaskFail_ResultsPersistedScanFailed(t *testing.T) {
	store := newIntegrationStore()
	materializer := &integrationSubdomainMaterializerStub{}

	createService := newIntegrationScanCreateService(t, store)
	scan := createSingleTargetBatchScanForIntegrationTest(t, createService)

	bridge := newScanTaskBridgeServiceForTest(store, store)
	assignment, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 1)
	if err != nil || assignment == nil {
		t.Fatalf("ClaimNextExecutionPlan failed: err=%v, assignment=%v", err, assignment)
	}

	// Submit results.
	ingestFacade := resultingestapp.NewResultIngestFacade(resultingestapp.ResultIngestFacadeDependencies{Subdomains: materializer, ScanSummary: integrationResultSummaryUpdaterStub{}, Materialization: integrationResultMaterializationCoordinator{}})
	scope, _ := store.GetResultTaskScopeByTaskID(context.Background(), assignment.TaskID)
	_, err = ingestFacade.Ingest(context.Background(), integrationResultCommand(assignment.TaskID, scope, contractresults.ResultKindAssetSubdomain, []string{`{"dnsName":"api.example.com"}`}))
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}

	// Results should be persisted.
	if materializer.totalItemCount() != 1 {
		t.Fatalf("expected 1 result persisted, got %d", materializer.totalItemCount())
	}

	// Task fails after results were submitted.
	failure := &FailureDetail{Kind: "runtime_error", Message: "engine crashed after partial results"}
	err = bridge.ReportTerminalTaskResult(context.Background(), 1, assignment.SessionID, 1, assignment.TaskID, intStatusFailed, failure)
	if err != nil {
		t.Fatalf("ReportTerminalTaskResult failed: %v", err)
	}

	// Scan should be failed.
	scanRecord, _ := store.GetScanForScanTask(scan.ID)
	if scanRecord.Status != intStatusFailed {
		t.Fatalf("expected scan 'failed', got %q", scanRecord.Status)
	}
	if scanRecord.Failure == nil || scanRecord.Failure.Kind != "runtime_error" {
		t.Fatalf("expected failure detail on scan, got %+v", scanRecord.Failure)
	}
}

// TestScanChain_MultiStage_Stage1SucceedStage2Fail verifies:
// Stage 1 completes successfully with results, stage 2 fails.
// Scan should be "failed" with stage 2's failure detail.
func TestScanChain_MultiStage_Stage1SucceedStage2Fail(t *testing.T) {
	store := newIntegrationStore()
	materializer := &integrationSubdomainMaterializerStub{}

	scan := &CreateScan{
		TargetID:       1,
		ScanWorkflowID: "multi_stage",
		Status:         intStatusPending,
		ScanTasks: multiStageScanTasks(
			CreateTaskStatusPending,
			CreateTaskStatusBlocked,
		),
	}
	store.CreateWithScanTasks(scan)
	store.scans[scan.ID].targetName = "example.com"
	store.scans[scan.ID].targetType = "domain"

	bridge := newScanTaskBridgeServiceForTest(store, store)
	ingestFacade := resultingestapp.NewResultIngestFacade(resultingestapp.ResultIngestFacadeDependencies{Subdomains: materializer, ScanSummary: integrationResultSummaryUpdaterStub{}, Materialization: integrationResultMaterializationCoordinator{}})

	// Stage 1: claim -> submit results → succeed.
	a1, _ := claimNextIntegrationExecution(context.Background(), bridge, 1, 1)
	scope1, _ := store.GetResultTaskScopeByTaskID(context.Background(), a1.TaskID)
	_, err := ingestFacade.Ingest(context.Background(), integrationResultCommand(a1.TaskID, scope1, contractresults.ResultKindAssetSubdomain, []string{`{"dnsName":"stage1.example.com"}`}))
	if err != nil {
		t.Fatalf("stage 1 Ingest failed: %v", err)
	}
	err = bridge.ReportTerminalTaskResult(context.Background(), 1, a1.SessionID, 1, a1.TaskID, intStatusSucceeded, nil)
	if err != nil {
		t.Fatalf("stage 1 ReportTerminalTaskResult failed: %v", err)
	}

	// Stage 2 should be unlocked.
	var stage2Task *integrationTask
	for _, task := range store.tasks {
		if task.scanWorkflowStageOrder == 2 {
			stage2Task = task
			break
		}
	}
	if stage2Task.status != intStatusPending {
		t.Fatalf("expected stage 2 'pending', got %q", stage2Task.status)
	}

	// Stage 2: claim -> submit partial results → fail.
	a2, _ := claimNextIntegrationExecution(context.Background(), bridge, 1, 2)
	scope2, _ := store.GetResultTaskScopeByTaskID(context.Background(), a2.TaskID)
	_, err = ingestFacade.Ingest(context.Background(), integrationResultCommand(a2.TaskID, scope2, contractresults.ResultKindAssetSubdomain, []string{`{"dnsName":"stage2.example.com"}`}))
	if err != nil {
		t.Fatalf("stage 2 Ingest failed: %v", err)
	}

	failure := &FailureDetail{Kind: "runtime_error", Message: "stage 2 engine crashed"}
	err = bridge.ReportTerminalTaskResult(context.Background(), 1, a2.SessionID, 2, a2.TaskID, intStatusFailed, failure)
	if err != nil {
		t.Fatalf("stage 2 ReportTerminalTaskResult failed: %v", err)
	}

	// Scan should be failed with stage 2's failure.
	scanRecord, _ := store.GetScanForScanTask(scan.ID)
	if scanRecord.Status != intStatusFailed {
		t.Fatalf("expected scan 'failed', got %q", scanRecord.Status)
	}
	if scanRecord.Failure == nil || scanRecord.Failure.Message != "stage 2 engine crashed" {
		t.Fatalf("expected stage 2 failure on scan, got %+v", scanRecord.Failure)
	}

	// Both stages' results should be persisted (2 calls total).
	if materializer.totalItemCount() != 2 {
		t.Fatalf("expected 2 results persisted across both stages, got %d", materializer.totalItemCount())
	}
}

// TestScanChain_TaskFailNoResults_ScanFailed verifies:
// Task fails without any results being submitted. Scan should still be "failed".
func TestScanChain_TaskFailNoResults_ScanFailed(t *testing.T) {
	store := newIntegrationStore()
	materializer := &integrationSubdomainMaterializerStub{}

	createService := newIntegrationScanCreateService(t, store)
	scan := createSingleTargetBatchScanForIntegrationTest(t, createService)

	bridge := newScanTaskBridgeServiceForTest(store, store)
	assignment, _ := claimNextIntegrationExecution(context.Background(), bridge, 1, 1)

	// Task fails immediately — no results submitted.
	failure := &FailureDetail{Kind: "container_start_failed", Message: "Engine container failed to start"}
	err := bridge.ReportTerminalTaskResult(context.Background(), 1, assignment.SessionID, 1, assignment.TaskID, intStatusFailed, failure)
	if err != nil {
		t.Fatalf("ReportTerminalTaskResult failed: %v", err)
	}

	scanRecord, _ := store.GetScanForScanTask(scan.ID)
	if scanRecord.Status != intStatusFailed {
		t.Fatalf("expected scan 'failed', got %q", scanRecord.Status)
	}
	if scanRecord.Failure == nil || scanRecord.Failure.Kind != "container_start_failed" {
		t.Fatalf("expected container_start_failed on scan, got %+v", scanRecord.Failure)
	}

	// No results should have been submitted.
	if materializer.totalItemCount() != 0 {
		t.Fatalf("expected 0 results, got %d", materializer.totalItemCount())
	}
}

// TestScanChain_MultipleResultBatches_ThenSucceed verifies:
// Agent submits results in multiple batches, then reports task success.
// All batches should be persisted.
func TestScanChain_MultipleResultBatches_ThenSucceed(t *testing.T) {
	store := newIntegrationStore()
	materializer := &integrationSubdomainMaterializerStub{}

	createService := newIntegrationScanCreateService(t, store)
	scan := createSingleTargetBatchScanForIntegrationTest(t, createService)

	bridge := newScanTaskBridgeServiceForTest(store, store)
	assignment, _ := claimNextIntegrationExecution(context.Background(), bridge, 1, 1)

	ingestFacade := resultingestapp.NewResultIngestFacade(resultingestapp.ResultIngestFacadeDependencies{Subdomains: materializer, ScanSummary: integrationResultSummaryUpdaterStub{}, Materialization: integrationResultMaterializationCoordinator{}})
	scope, _ := store.GetResultTaskScopeByTaskID(context.Background(), assignment.TaskID)
	// Batch 1.
	_, err := ingestFacade.Ingest(context.Background(), integrationResultCommand(assignment.TaskID, scope, contractresults.ResultKindAssetSubdomain, []string{`{"dnsName":"batch1-a.example.com"}`, `{"dnsName":"batch1-b.example.com"}`}))
	if err != nil {
		t.Fatalf("batch 1 failed: %v", err)
	}

	// Batch 2.
	_, err = ingestFacade.Ingest(context.Background(), integrationResultCommand(assignment.TaskID, scope, contractresults.ResultKindAssetSubdomain, []string{`{"dnsName":"batch2.example.com"}`}))
	if err != nil {
		t.Fatalf("batch 2 failed: %v", err)
	}

	// Batch 3.
	_, err = ingestFacade.Ingest(context.Background(), integrationResultCommand(assignment.TaskID, scope, contractresults.ResultKindAssetSubdomain, []string{`{"dnsName":"batch3-a.example.com"}`, `{"dnsName":"batch3-b.example.com"}`, `{"dnsName":"batch3-c.example.com"}`}))
	if err != nil {
		t.Fatalf("batch 3 failed: %v", err)
	}

	// Verify all batches were persisted.
	if len(materializer.calls) != 3 {
		t.Fatalf("expected 3 materializer calls, got %d", len(materializer.calls))
	}
	if materializer.totalItemCount() != 6 {
		t.Fatalf("expected 6 total results, got %d", materializer.totalItemCount())
	}

	// Task succeeds.
	err = bridge.ReportTerminalTaskResult(context.Background(), 1, assignment.SessionID, 1, assignment.TaskID, intStatusSucceeded, nil)
	if err != nil {
		t.Fatalf("ReportTerminalTaskResult failed: %v", err)
	}

	scanRecord, _ := store.GetScanForScanTask(scan.ID)
	if scanRecord.Status != intStatusSucceeded {
		t.Fatalf("expected scan 'succeeded', got %q", scanRecord.Status)
	}

	// Verify all DNS names across batches.
	allNames := materializer.allDNSNames()
	expectedCount := map[string]int{
		"batch1-a.example.com": 1, "batch1-b.example.com": 1,
		"batch2.example.com":   1,
		"batch3-a.example.com": 1, "batch3-b.example.com": 1, "batch3-c.example.com": 1,
	}
	for _, name := range allNames {
		expectedCount[name]--
	}
	for name, count := range expectedCount {
		if count != 0 {
			t.Fatalf("DNS name %q count mismatch: expected 1, got %d", name, 1-count)
		}
	}
}
