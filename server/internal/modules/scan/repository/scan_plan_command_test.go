package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"google.golang.org/protobuf/types/known/durationpb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateWithScanTasksAndPlansIsAtomicAndPersistsSkippedWithoutPlan(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-plan-command?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(createWithScanTasksScanDDL).Error; err != nil {
		t.Fatalf("create scan table: %v", err)
	}
	setupActiveTargetsForScanCreate(t, db)
	if err := db.Exec(createWithScanTasksScanTaskDDL).Error; err != nil {
		t.Fatalf("create scan_task table: %v", err)
	}
	repo := &ScanRepository{db: db}
	scan := &scandomain.CreateScan{TargetID: 1, ScanWorkflowID: "default", Configuration: map[string]any{}, InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending", ScanTasks: []scandomain.CreateScanTask{
		{StageOrder: 1, StageID: "discover", StepOrder: 1, StepID: "skip", EngineID: "engine.lunafox.subdomain_discovery", Status: "pending"},
		{StageOrder: 2, StageID: "ports", StepOrder: 1, StepID: "run", EngineID: "engine.lunafox.port_scan", Status: "pending"},
	}}
	plan := &agentexecutionv1.ResolvedEngineExecutionPlan{
		Execution: "executions/scan-1-task-2", Task: "scans/1/tasks/2",
		Target:        &agentexecutionv1.CanonicalTarget{Resource: "targets/1", Type: agentexecutionv1.TargetType_TARGET_TYPE_IP, Value: "192.0.2.1"},
		WorkflowStep:  &agentexecutionv1.WorkflowStepScope{Scan: "scans/1", Workflow: "scanWorkflows/default", StageId: "ports", StepId: "run"},
		EngineRelease: &agentexecutionv1.EngineRelease{Engine: "engine.lunafox.port_scan", PackageDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", EngineApiMajor: 2, CompatibilityRevision: "engine-execution-diagnostics-r1"},
		RuntimeImage:  &agentexecutionv1.RuntimeImage{Refs: []string{"docker.io/lunafox/lunafox-engine-runtime-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},
		Config:        &agentexecutionv1.FinalEngineConfig{},
		Limits:        &agentexecutionv1.ExecutionLimits{MaxExecutionDuration: durationpb.New(time.Second), ProgressMessageMaxBytes: 1, ResultBatchMaxItems: 1, ResultBatchMaxBytes: 1},
	}
	encoded, err := agentexecution.MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		t.Fatalf("encode plan: %v", err)
	}
	err = repo.CreateWithScanTasksAndPlans(context.Background(), scan, resolveEmptyEffectiveBlacklistPatterns, func(_ int, taskID int, task *scandomain.CreateScanTask) error {
		if taskID == 1 {
			task.Status = "skipped"
			task.SkipReason = "target_not_applicable"
			return nil
		}
		task.Status = "pending"
		task.ResolvedExecutionPlan = encoded
		return nil
	})
	if err != nil {
		t.Fatalf("CreateWithScanTasksAndPlans failed: %v", err)
	}
	var rows []struct {
		ID                  int
		Status              string
		SkipReason          string
		EngineConfig        string `gorm:"column:engine_config"`
		TaskExecutionConfig string `gorm:"column:task_execution_config"`
		Plan                []byte `gorm:"column:resolved_execution_plan"`
	}
	if err := db.Table("scan_task").Order("id").Find(&rows).Error; err != nil {
		t.Fatalf("read tasks: %v", err)
	}
	if len(rows) != 2 || rows[0].Status != "skipped" || rows[0].SkipReason == "" || len(rows[0].Plan) != 0 || rows[1].Status != "pending" || string(rows[1].Plan) != string(encoded) {
		t.Fatalf("unexpected persisted plan rows: %#v", rows)
	}
	for _, row := range rows {
		if row.EngineConfig != "{}" || row.TaskExecutionConfig != "{}" {
			t.Fatalf("task %d empty configs = engine %q, execution %q; want JSON objects", row.ID, row.EngineConfig, row.TaskExecutionConfig)
		}
	}

	failed := &scandomain.CreateScan{TargetID: 1, ScanWorkflowID: "default", Configuration: map[string]any{}, InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending", ScanTasks: []scandomain.CreateScanTask{{StageOrder: 1, StageID: "ports", StepOrder: 1, StepID: "broken", EngineID: "engine.lunafox.port_scan", Status: "pending"}}}
	if err := repo.CreateWithScanTasksAndPlans(context.Background(), failed, resolveEmptyEffectiveBlacklistPatterns, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return assertPlanFailure{} }); err == nil {
		t.Fatal("expected finalizer failure")
	}
	var scanCount, taskCount, planCount int64
	if err := db.Table("scan").Count(&scanCount).Error; err != nil {
		t.Fatalf("count scans: %v", err)
	}
	if err := db.Table("scan_task").Count(&taskCount).Error; err != nil {
		t.Fatalf("count scan tasks: %v", err)
	}
	if err := db.Table("scan_task").Where("length(resolved_execution_plan) > 0").Count(&planCount).Error; err != nil {
		t.Fatalf("count saved plans: %v", err)
	}
	if scanCount != 1 || taskCount != 2 || planCount != 1 {
		t.Fatalf("finalizer failure changed committed rows: scans=%d tasks=%d plans=%d", scanCount, taskCount, planCount)
	}
}

func TestCreateWithScanTasksAndPlansRejectsCorruptOrWrongScopePlanAtomically(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-plan-invalid?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(createWithScanTasksScanDDL).Error; err != nil {
		t.Fatalf("create scan table: %v", err)
	}
	setupActiveTargetsForScanCreate(t, db)
	if err := db.Exec(createWithScanTasksScanTaskDDL).Error; err != nil {
		t.Fatalf("create scan_task table: %v", err)
	}
	repo := &ScanRepository{db: db}

	for _, test := range []struct {
		name       string
		status     string
		skipReason string
		plan       []byte
	}{
		{name: "corrupt", status: "pending", plan: []byte("not protobuf")},
		{name: "wrong scope", status: "pending", plan: mustSavedPlanForTask(t, "scans/99/tasks/99", "scans/99", "engine.lunafox.port_scan", "ports", "run")},
		{name: "wrong target", status: "pending", plan: mustSavedPlanForTaskWithParentScope(t, "scans/1/tasks/1", "scans/1", "engine.lunafox.port_scan", "ports", "run", "targets/99", "scanWorkflows/default")},
		{name: "wrong scan workflow", status: "pending", plan: mustSavedPlanForTaskWithParentScope(t, "scans/1/tasks/1", "scans/1", "engine.lunafox.port_scan", "ports", "run", "targets/1", "scanWorkflows/other")},
		{name: "wrong engine", status: "pending", plan: mustSavedPlanForTask(t, "scans/1/tasks/1", "scans/1", "engine.lunafox.website_discovery", "ports", "run")},
		{name: "wrong workflow scope", status: "pending", plan: mustSavedPlanForTask(t, "scans/1/tasks/1", "scans/1", "engine.lunafox.port_scan", "other", "run")},
		{name: "invalid executable status", status: "running", plan: mustSavedPlanForTask(t, "scans/1/tasks/1", "scans/1", "engine.lunafox.port_scan", "ports", "run")},
		{name: "non-canonical skipped status", status: " skipped ", skipReason: "not applicable"},
		{name: "non-canonical skipped reason", status: "skipped", skipReason: " not applicable "},
		{name: "executable skip reason", status: "pending", skipReason: "stale reason", plan: mustSavedPlanForTask(t, "scans/1/tasks/1", "scans/1", "engine.lunafox.port_scan", "ports", "run")},
	} {
		t.Run(test.name, func(t *testing.T) {
			scan := &scandomain.CreateScan{TargetID: 1, ScanWorkflowID: "default", Configuration: map[string]any{}, InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending", ScanTasks: []scandomain.CreateScanTask{{StageOrder: 1, StageID: "ports", StepOrder: 1, StepID: "run", EngineID: "engine.lunafox.port_scan", Status: "pending"}}}
			err := repo.CreateWithScanTasksAndPlans(context.Background(), scan, resolveEmptyEffectiveBlacklistPatterns, func(_ int, _ int, task *scandomain.CreateScanTask) error {
				task.Status = test.status
				task.SkipReason = test.skipReason
				task.ResolvedExecutionPlan = test.plan
				return nil
			})
			if err == nil {
				t.Fatal("expected invalid plan transaction to fail")
			}
		})
	}

	var count int64
	if err := db.Table("scan").Count(&count).Error; err != nil {
		t.Fatalf("count scans: %v", err)
	}
	if count != 0 {
		t.Fatalf("invalid plans left persisted scan rows: %d", count)
	}
}

func TestCreateWithScanTasksAndPlansRollsBackPlanSaveFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-plan-save-failure?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(createWithScanTasksScanDDL).Error; err != nil {
		t.Fatalf("create scan table: %v", err)
	}
	setupActiveTargetsForScanCreate(t, db)
	// The missing plan column lets task identity allocation and compilation
	// succeed, then forces the plan persistence update itself to fail.
	scanTaskDDLWithoutPlan := strings.Replace(createWithScanTasksScanTaskDDL, "\tresolved_execution_plan BLOB NOT NULL DEFAULT '',\n", "", 1)
	if err := db.Exec(scanTaskDDLWithoutPlan).Error; err != nil {
		t.Fatalf("create scan_task table: %v", err)
	}
	repo := &ScanRepository{db: db}
	scan := &scandomain.CreateScan{TargetID: 1, ScanWorkflowID: "default", Configuration: map[string]any{}, InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending", ScanTasks: []scandomain.CreateScanTask{{StageOrder: 1, StageID: "ports", StepOrder: 1, StepID: "run", EngineID: "engine.lunafox.port_scan", Status: "pending"}}}

	err = repo.CreateWithScanTasksAndPlans(context.Background(), scan, resolveEmptyEffectiveBlacklistPatterns, func(scanID, taskID int, task *scandomain.CreateScanTask) error {
		task.Status = "pending"
		task.ResolvedExecutionPlan = mustSavedPlanForTask(t, resourcenames.Task(scanID, taskID), resourcenames.Scan(scanID), task.EngineID, task.StageID, task.StepID)
		return nil
	})
	if err == nil {
		t.Fatal("expected plan save failure")
	}
	var scanCount, taskCount int64
	if err := db.Table("scan").Count(&scanCount).Error; err != nil {
		t.Fatalf("count scans: %v", err)
	}
	if err := db.Table("scan_task").Count(&taskCount).Error; err != nil {
		t.Fatalf("count scan tasks: %v", err)
	}
	if scanCount != 0 || taskCount != 0 {
		t.Fatalf("plan save failure left rows: scans=%d tasks=%d", scanCount, taskCount)
	}
}

func mustSavedPlanForTask(t *testing.T, task, scan, engine, stage, step string) []byte {
	return mustSavedPlanForTaskWithParentScope(t, task, scan, engine, stage, step, "targets/1", "scanWorkflows/default")
}

func mustSavedPlanForTaskWithParentScope(t *testing.T, task, scan, engine, stage, step, target, workflow string) []byte {
	t.Helper()
	plan := &agentexecutionv1.ResolvedEngineExecutionPlan{
		Execution: "executions/run-1", Task: task,
		Target:        &agentexecutionv1.CanonicalTarget{Resource: target, Type: agentexecutionv1.TargetType_TARGET_TYPE_IP, Value: "192.0.2.1"},
		WorkflowStep:  &agentexecutionv1.WorkflowStepScope{Scan: scan, Workflow: workflow, StageId: stage, StepId: step},
		EngineRelease: &agentexecutionv1.EngineRelease{Engine: engine, PackageDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", EngineApiMajor: 2, CompatibilityRevision: "engine-execution-diagnostics-r1"},
		RuntimeImage:  &agentexecutionv1.RuntimeImage{Refs: []string{"docker.io/lunafox/lunafox-engine-runtime-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},
		Config:        &agentexecutionv1.FinalEngineConfig{},
		Limits:        &agentexecutionv1.ExecutionLimits{MaxExecutionDuration: durationpb.New(time.Second), ProgressMessageMaxBytes: 1, ResultBatchMaxItems: 1, ResultBatchMaxBytes: 1},
	}
	encoded, err := agentexecution.MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		t.Fatalf("encode saved plan: %v", err)
	}
	return encoded
}

type assertPlanFailure struct{}

func (assertPlanFailure) Error() string { return "plan failure" }
