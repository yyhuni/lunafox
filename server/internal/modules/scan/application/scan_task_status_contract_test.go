package application

import (
	"context"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
)

func TestScanCreatePlanningSkipLeavesLaterApplicableTaskPending(t *testing.T) {
	const (
		subdomainEngineID = "engine.lunafox.subdomain_discovery"
		portScanEngineID  = "engine.lunafox.port_scan"
	)

	subdomainDefinition := scanCreateTestEngineDefinition(subdomainEngineID, []string{engineexecution.TargetTypeDomain})
	portScanDefinition := scanCreateTestEngineDefinition(portScanEngineID, []string{
		engineexecution.TargetTypeDomain,
		engineexecution.TargetTypeIP,
		engineexecution.TargetTypeCIDR,
	})

	packages := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		subdomainEngineID: {Package: scanCreateTestPlanTaskPackage(subdomainDefinition)},
		portScanEngineID:  {Package: scanCreateTestPlanTaskPackage(portScanDefinition)},
	}}
	manifest := ScanCreateWorkflowManifest{
		ScanWorkflowID: "default",
		Stages: []ScanCreateWorkflowStage{
			{StageID: "subdomains", Steps: []ScanCreateWorkflowStep{{StepID: "subdomain_discovery", EngineID: subdomainEngineID}}},
			{StageID: "ports", Steps: []ScanCreateWorkflowStep{{StepID: "port_scan", EngineID: portScanEngineID}}},
		},
	}
	store := &plannedScanCreateStoreStub{}
	service := mustConfigureScanCreatePlanTask(t, NewScanCreateService(
		store,
		func(context.Context, int) (*TargetRef, error) {
			return &TargetRef{ID: 17, Name: "192.0.2.10", Type: engineexecution.TargetTypeIP, CreatedAt: time.Now().UTC()}, nil
		},
		nil,
		scanCreateWorkflowReaderStub{manifest: manifest},
		packages,
	), packages)

	result, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(17, "scanWorkflows/default", completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{
		"subdomain_discovery": subdomainDefinition.Execution,
		"port_scan":           portScanDefinition.Execution,
	})))
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if result.CreatedCount != 1 || store.committed == nil {
		t.Fatalf("planned scan was not committed: result=%#v scan=%#v", result, store.committed)
	}
	if store.committed.Status != CreateScanStatusPending {
		t.Fatalf("scan status = %q, want pending because a later task is executable", store.committed.Status)
	}
	if len(store.committed.ScanTasks) != 2 {
		t.Fatalf("scan tasks = %d, want 2", len(store.committed.ScanTasks))
	}

	skipped := store.committed.ScanTasks[0]
	if skipped.Status != CreateTaskStatusSkipped || skipped.SkipReason == "" || len(skipped.ResolvedExecutionPlan) != 0 {
		t.Fatalf("planning-time unsupported task = %#v, want skipped with reason and no plan", skipped)
	}
	executable := store.committed.ScanTasks[1]
	if executable.Status != CreateTaskStatusPending || executable.SkipReason != "" || len(executable.ResolvedExecutionPlan) == 0 {
		t.Fatalf("later applicable task = %#v, want pending with saved plan", executable)
	}
}

func TestScanChainCancellationCancelsUnstartedDownstreamAndLeavesItUnclaimable(t *testing.T) {
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
	if err := store.CreateWithScanTasks(scan); err != nil {
		t.Fatalf("CreateWithScanTasks failed: %v", err)
	}
	store.scans[scan.ID].targetName = "example.com"
	store.scans[scan.ID].targetType = engineexecution.TargetTypeDomain

	bridge := newScanTaskBridgeServiceForTest(store, store)
	assignment, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 1)
	if err != nil || assignment == nil {
		t.Fatalf("stage 1 claim failed: assignment=%+v err=%v", assignment, err)
	}
	if err := bridge.ReportTerminalTaskResult(
		context.Background(),
		1,
		assignment.SessionID,
		1,
		assignment.TaskID,
		intStatusCancelled,
		nil,
	); err != nil {
		t.Fatalf("stage 1 cancellation update failed: %v", err)
	}

	stage1Task := integrationTaskByStageOrder(store, scan.ID, 1)
	if stage1Task == nil || stage1Task.status != intStatusCancelled {
		t.Fatalf("claimed task = %+v, want cancelled", stage1Task)
	}
	stage2Task := integrationTaskByStageOrder(store, scan.ID, 2)
	if stage2Task == nil || stage2Task.status != intStatusCancelled || stage2Task.completedAt == nil {
		t.Fatalf("unstarted downstream task = %+v, want terminal cancelled", stage2Task)
	}
	scanRecord, err := store.GetScanForScanTask(scan.ID)
	if err != nil {
		t.Fatalf("GetScanForScanTask failed: %v", err)
	}
	if scanRecord.Status != intStatusCancelled {
		t.Fatalf("scan status = %q, want cancelled", scanRecord.Status)
	}

	next, err := claimNextIntegrationExecution(context.Background(), bridge, 1, 2)
	if err != nil {
		t.Fatalf("claim after cancellation returned error: %v", err)
	}
	if next != nil {
		t.Fatalf("cancelled downstream task was claimable: %+v", next)
	}
}
