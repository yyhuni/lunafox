package application

import (
	"context"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Integration test DDL
// ---------------------------------------------------------------------------

const integrationTargetDDL = `
CREATE TABLE IF NOT EXISTS target (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	type TEXT DEFAULT 'domain',
	created_at DATETIME,
	last_scanned_at DATETIME,
	deleted_at DATETIME
);`

const integrationScanDDL = `
CREATE TABLE IF NOT EXISTS scan (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	target_id INTEGER NOT NULL,
	scan_workflow_id TEXT NOT NULL,
	configuration TEXT,
	input_source TEXT NOT NULL CHECK (input_source IN ('scan_snapshot', 'target_inventory')),
	trigger_type TEXT NOT NULL CHECK (trigger_type IN ('manual', 'scheduled', 'ai')),
	status TEXT DEFAULT 'pending',
	results_dir TEXT DEFAULT '',
	container_ids TEXT DEFAULT '[]',
	agent_id INTEGER,
	assignment_mode TEXT NOT NULL DEFAULT 'automatic',
	error_message TEXT DEFAULT '',
	failure_kind TEXT DEFAULT '',
	progress INTEGER DEFAULT 0,
	current_stage TEXT DEFAULT '',
	stage_progress TEXT DEFAULT '{}',
	created_at DATETIME,
	stopped_at DATETIME,
	deleted_at DATETIME,
	cached_subdomains_count INTEGER DEFAULT 0,
	cached_websites_count INTEGER DEFAULT 0,
	cached_endpoints_count INTEGER DEFAULT 0,
	cached_ips_count INTEGER DEFAULT 0,
	cached_directories_count INTEGER DEFAULT 0,
	cached_screenshots_count INTEGER DEFAULT 0,
	cached_vulns_total INTEGER DEFAULT 0,
	cached_vulns_critical INTEGER DEFAULT 0,
	cached_vulns_high INTEGER DEFAULT 0,
	cached_vulns_medium INTEGER DEFAULT 0,
	cached_vulns_low INTEGER DEFAULT 0,
	stats_updated_at DATETIME
);`

const integrationScanBlacklistSnapshotDDL = `
CREATE TABLE IF NOT EXISTS scan_blacklist_snapshot (
	scan_id INTEGER PRIMARY KEY,
	patterns TEXT NOT NULL
);`

const integrationScanTaskDDL = `
CREATE TABLE IF NOT EXISTS scan_task (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	scan_id INTEGER NOT NULL,
	stage_order INTEGER NOT NULL DEFAULT 0,
	stage_id TEXT NOT NULL,
	step_order INTEGER NOT NULL DEFAULT 0,
	step_id TEXT NOT NULL,
	engine_id TEXT NOT NULL,
	engine_config TEXT,
	task_execution_config TEXT,
	resolved_execution_plan BLOB NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending',
	assigned_agent_id INTEGER,
	assigned_session_id TEXT,
	assigned_session_epoch INTEGER,
	assigned_request_id TEXT,
	terminal_reconciliation_pending BOOLEAN NOT NULL DEFAULT FALSE,
	error_message TEXT DEFAULT '',
	failure_kind TEXT DEFAULT '',
	failure_detail TEXT DEFAULT '',
	engine_diagnostics TEXT,
	skip_reason TEXT DEFAULT '',
	created_at DATETIME,
	started_at DATETIME,
	completed_at DATETIME
	);`

const integrationAgentRuntimeStatusDDL = `
	CREATE TABLE IF NOT EXISTS agent_runtime_status (
		agent_id INTEGER PRIMARY KEY,
		session_id TEXT NOT NULL,
		session_epoch INTEGER NOT NULL
	);`

const integrationClaimRequestIndexDDL = `
	CREATE UNIQUE INDEX IF NOT EXISTS integration_scan_task_claim_request
	ON scan_task(assigned_agent_id, assigned_session_epoch, assigned_request_id)
	WHERE assigned_request_id IS NOT NULL;`

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:integration_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	for _, ddl := range []string{integrationTargetDDL, integrationScanDDL, integrationScanBlacklistSnapshotDDL, integrationScanTaskDDL, integrationAgentRuntimeStatusDDL, integrationClaimRequestIndexDDL} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("setup DDL failed: %v", err)
		}
	}
	// Insert a target row so Preload("Target") in GetScanForScanTask works.
	if err := db.Exec(`INSERT INTO target (id, name, type) VALUES (1, 'example.com', 'domain')`).Error; err != nil {
		t.Fatalf("insert target failed: %v", err)
	}
	return db
}

// integrationRepositoryScanCreateStore keeps this pre-blacklist integration
// fixture explicit about the immutable empty policy it is exercising.
type integrationRepositoryScanCreateStore struct {
	repo *repository.ScanRepository
}

func (store integrationRepositoryScanCreateStore) CreateWithScanTasksAndPlans(ctx context.Context, scan *CreateScan, finalize ScanCreateTaskFinalizer) error {
	return store.repo.CreateWithScanTasksAndPlans(ctx, scan, func(context.Context, int) ([]string, error) {
		return []string{}, nil
	}, finalize)
}

func TestIntegrationScanCreateUsesPlanTaskAsOnlyDeepPlanner(t *testing.T) {
	tests := []struct {
		name       string
		definition func() PlanTaskPackage
		assertPlan func(*testing.T, []byte)
	}{
		{
			name: "host and web URL inputs reach PlanTask",
			definition: func() PlanTaskPackage {
				return planTaskExactPackage(testPlanTaskDefinition(
					[]string{engineexecution.InputSubdomains, engineexecution.InputHostPorts},
					nil,
					false,
				))
			},
			assertPlan: func(t *testing.T, encoded []byte) {
				t.Helper()
				plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(encoded)
				if err != nil {
					t.Fatalf("decode saved plan: %v", err)
				}
				if plan.ProtoReflect().Descriptor().Fields().ByName("inputs") != nil {
					t.Fatal("saved plan unexpectedly exposes an Engine-specific input allow-set")
				}
			},
		},
		{
			name: "disabled scalar default reaches PlanTask",
			definition: func() PlanTaskPackage {
				definition := testPlanTaskDefinition(nil, nil, false)
				definition.Execution.ConfigSections = []engineexecution.ConfigSectionDefinition{{
					ID:             "scan",
					DefaultEnabled: false,
					Params: []engineexecution.ParamDefinition{{
						Key: "threads", Type: engineexecution.ParamTypeInteger, Default: 1,
					}},
				}, {
					ID:             "alpha",
					DefaultEnabled: true,
					Params: []engineexecution.ParamDefinition{{
						Key: "label", Type: engineexecution.ParamTypeString, Default: "active",
					}},
				}}
				return planTaskExactPackage(definition)
			},
			assertPlan: func(t *testing.T, encoded []byte) {
				t.Helper()
				plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(encoded)
				if err != nil {
					t.Fatalf("decode saved plan: %v", err)
				}
				sections := plan.GetConfig().GetSections()
				if len(sections) != 2 || sections[0].GetSectionId() != "alpha" || !sections[0].GetEnabled() || len(sections[0].GetParams()) != 1 || sections[0].GetParams()[0].GetValue().GetStringValue() != "active" || sections[1].GetSectionId() != "scan" || sections[1].GetEnabled() || len(sections[1].GetParams()) != 0 {
					t.Fatalf("complete disabled config was not compiled canonically: %#v", sections)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupIntegrationDB(t)
			scanRepo := repository.NewScanRepository(db)
			exact := test.definition()
			packages := &planTaskPackageReaderStub{packageValue: exact}
			compiler, err := NewPlanTaskCompiler(packages, nil)
			if err != nil {
				t.Fatalf("NewPlanTaskCompiler failed: %v", err)
			}
			manifest := ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{
				StageID: "stage", Steps: []ScanCreateWorkflowStep{{StepID: "step", EngineID: exact.Identity.EngineID}},
			}}}
			catalog := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
				exact.Identity.EngineID: scanCreatePackageFromPlanPackage(exact),
			}}
			service := mustConfigurePlanTaskForTest(t, NewScanCreateService(
				integrationRepositoryScanCreateStore{repo: scanRepo},
				func(context.Context, int) (*TargetRef, error) {
					return &TargetRef{ID: 1, Name: "example.com", Type: "domain"}, nil
				},
				nil,
				scanCreateWorkflowReaderStub{manifest: manifest},
				catalog,
			), compiler, 3*time.Second)

			result, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "scanWorkflows/default", completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{
				"step": exact.Definition.Execution,
			})))
			if err != nil {
				t.Fatalf("CreateBatch failed: %v", err)
			}
			if result.CreatedCount != 1 || len(packages.requests) != 1 {
				t.Fatalf("scan transaction did not invoke PlanTask exactly once: result=%#v requests=%#v", result, packages.requests)
			}

			var row struct {
				Status string `gorm:"column:status"`
				Plan   []byte `gorm:"column:resolved_execution_plan"`
			}
			if err := db.Table("scan_task").Select("status, resolved_execution_plan").Where("scan_id = ?", result.Scans[0].ID).Take(&row).Error; err != nil {
				t.Fatalf("read committed scan task: %v", err)
			}
			if row.Status != CreateTaskStatusPending || len(row.Plan) == 0 {
				t.Fatalf("scan transaction did not commit executable plan: %#v", row)
			}
			test.assertPlan(t, row.Plan)
		})
	}
}

func TestIntegrationSavedPlanClaimAndTerminalResultUseCurrentAgentSession(t *testing.T) {
	ctx := context.Background()
	db := setupIntegrationDB(t)
	scanRepo := repository.NewScanRepository(db)
	taskRepo := repository.NewScanTaskRepository(db)
	exact := planTaskExactPackage(testPlanTaskDefinition(nil, nil, false))
	packages := &planTaskPackageReaderStub{packageValue: exact}
	compiler, err := NewPlanTaskCompiler(packages, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler failed: %v", err)
	}
	manifest := ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{
		StageID: "stage", Steps: []ScanCreateWorkflowStep{{StepID: "step", EngineID: exact.Identity.EngineID}},
	}}}
	catalog := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		exact.Identity.EngineID: scanCreatePackageFromPlanPackage(exact),
	}}
	createService := mustConfigurePlanTaskForTest(t, NewScanCreateService(
		integrationRepositoryScanCreateStore{repo: scanRepo},
		func(context.Context, int) (*TargetRef, error) {
			return &TargetRef{ID: 1, Name: "example.com", Type: "domain"}, nil
		},
		nil,
		scanCreateWorkflowReaderStub{manifest: manifest},
		catalog,
	), compiler, 3*time.Second)

	result, err := createService.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "scanWorkflows/default", completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{
		"step": exact.Definition.Execution,
	})))
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if result.CreatedCount != 1 || len(result.Scans) != 1 {
		t.Fatalf("expected one created scan, got %#v", result)
	}

	var persisted struct {
		ID     int    `gorm:"column:id"`
		Status string `gorm:"column:status"`
		Plan   []byte `gorm:"column:resolved_execution_plan"`
	}
	if err := db.Table("scan_task").Select("id, status, resolved_execution_plan").Where("scan_id = ?", result.Scans[0].ID).Take(&persisted).Error; err != nil {
		t.Fatalf("read persisted task plan: %v", err)
	}
	wantPlan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(persisted.Plan)
	if err != nil {
		t.Fatalf("decode persisted task plan: %v", err)
	}
	if persisted.Status != CreateTaskStatusPending {
		t.Fatalf("task status before claim = %q, want pending", persisted.Status)
	}

	const (
		agentID      = 42
		sessionID    = "session-42"
		sessionEpoch = int64(9)
		requestID    = "018f6f22-0d23-7b4a-a508-8f4c8abf16d1"
	)
	if err := db.Exec(
		`INSERT INTO agent_runtime_status (agent_id, session_id, session_epoch) VALUES (?, ?, ?)`,
		agentID, sessionID, sessionEpoch,
	).Error; err != nil {
		t.Fatalf("insert active Agent session: %v", err)
	}
	bridge := NewScanTaskBridgeService(taskRepo, scanRepo).WithEngineExecutionClaimStore(taskRepo)
	claimed, err := bridge.ClaimNextExecutionPlan(ctx, agentID, sessionID, sessionEpoch, requestID, agentdomain.AgentExecutionCapabilitySnapshot{
		ContainerRuntimeReady:    true,
		SupportedEngineAPIMajors: []uint32{2},
	})
	if err != nil {
		t.Fatalf("ClaimNextExecutionPlan failed: %v", err)
	}
	if !proto.Equal(claimed, wantPlan) {
		t.Fatalf("claimed plan differs from persisted plan: got=%#v want=%#v", claimed, wantPlan)
	}

	var claimedState struct {
		Status               string  `gorm:"column:status"`
		AssignedAgentID      *int    `gorm:"column:assigned_agent_id"`
		AssignedSessionID    *string `gorm:"column:assigned_session_id"`
		AssignedSessionEpoch *int64  `gorm:"column:assigned_session_epoch"`
		AssignedRequestID    *string `gorm:"column:assigned_request_id"`
	}
	if err := db.Table("scan_task").Select("status, assigned_agent_id, assigned_session_id, assigned_session_epoch, assigned_request_id").Where("id = ?", persisted.ID).Take(&claimedState).Error; err != nil {
		t.Fatalf("read claimed task state: %v", err)
	}
	if claimedState.Status != "running" || claimedState.AssignedAgentID == nil || *claimedState.AssignedAgentID != agentID || claimedState.AssignedSessionID == nil || *claimedState.AssignedSessionID != sessionID || claimedState.AssignedSessionEpoch == nil || *claimedState.AssignedSessionEpoch != sessionEpoch || claimedState.AssignedRequestID == nil || *claimedState.AssignedRequestID != requestID {
		t.Fatalf("claim was not bound to the active Agent session: %#v", claimedState)
	}

	if err := bridge.ReportTerminalTaskResult(ctx, agentID, sessionID, sessionEpoch, persisted.ID, "succeeded", nil); err != nil {
		t.Fatalf("ReportTerminalTaskResult failed: %v", err)
	}
	var terminalState struct {
		TaskStatus                    string     `gorm:"column:task_status"`
		TerminalReconciliationPending bool       `gorm:"column:terminal_reconciliation_pending"`
		ScanStatus                    string     `gorm:"column:scan_status"`
		StoppedAt                     *time.Time `gorm:"column:stopped_at"`
	}
	if err := db.Table("scan_task AS st").
		Joins("JOIN scan AS s ON s.id = st.scan_id").
		Select("st.status AS task_status, st.terminal_reconciliation_pending, s.status AS scan_status, s.stopped_at").
		Where("st.id = ?", persisted.ID).
		Take(&terminalState).Error; err != nil {
		t.Fatalf("read terminal execution state: %v", err)
	}
	if terminalState.TaskStatus != "succeeded" || terminalState.TerminalReconciliationPending || terminalState.ScanStatus != "succeeded" || terminalState.StoppedAt == nil {
		t.Fatalf("terminal result was not fully reconciled: %#v", terminalState)
	}
}
