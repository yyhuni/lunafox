package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/contracts/agentexecution"
	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	smokeServerEvidenceAgentID      = 9001
	smokeServerEvidenceSessionID    = "engine-cut-server-evidence"
	smokeServerEvidenceSessionEpoch = int64(1)
)

var (
	errSmokeStaleAgentExecutionSession = errors.New("smoke Agent execution session is stale")
	// Smoke fixtures keep submitted Scan configuration beside their pure
	// orchestration manifest. This mirrors the production Profile-to-Scan
	// request boundary without reintroducing configuration on workflow steps.
	smokeWorkflowConfigurations = map[string]map[string]any{}
)

// The smoke store intentionally uses the production repositories over a
// run-scoped SQLite database. This schema is the smallest projection those
// repositories need; repository validation, not a testsupport persistence
// substitute, remains responsible for saved-plan and skipped invariants.
const smokeScanStoreDDL = `
CREATE TABLE target (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	created_at DATETIME,
	last_scanned_at DATETIME,
	deleted_at DATETIME
);
CREATE TABLE agent (
	id INTEGER PRIMARY KEY
);
CREATE TABLE scan (
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
);
CREATE TABLE scan_blacklist_snapshot (
	scan_id INTEGER PRIMARY KEY,
	patterns TEXT NOT NULL
);
CREATE TABLE scan_task (
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
);
CREATE TABLE agent_runtime_status (
	agent_id INTEGER PRIMARY KEY,
	session_id TEXT NOT NULL,
	session_epoch INTEGER NOT NULL
);
CREATE UNIQUE INDEX smoke_scan_task_claim_request
	ON scan_task(assigned_agent_id, assigned_session_epoch, assigned_request_id)
	WHERE assigned_request_id IS NOT NULL;
`

type smokeScanStore struct {
	root       string
	db         *gorm.DB
	scans      *scanrepo.ScanRepository
	tasks      scanrepo.ScanTaskRepository
	savedPlans scanrepo.SavedExecutionPlanReader
	nextTarget int
}

// smokeScanCreateCommandStore keeps the smoke on the production Scan
// repository transaction while supplying its intentionally valid empty policy
// snapshot. The artifact smoke later reads that persisted Scan-owned row.
type smokeScanCreateCommandStore struct {
	repository *scanrepo.ScanRepository
}

func (store smokeScanCreateCommandStore) CreateWithScanTasksAndPlans(ctx context.Context, scan *scanapp.CreateScan, finalize scanapp.ScanCreateTaskFinalizer) error {
	if store.repository == nil {
		return errors.New("smoke scan repository is required")
	}
	return store.repository.CreateWithScanTasksAndPlans(ctx, scan, func(context.Context, int) ([]string, error) {
		return []string{}, nil
	}, finalize)
}

var _ scanapp.ScanCreateCommandStore = smokeScanCreateCommandStore{}

type smokePersistedTask struct {
	ID                    int    `gorm:"column:id"`
	ScanID                int    `gorm:"column:scan_id"`
	StageOrder            int    `gorm:"column:stage_order"`
	EngineID              string `gorm:"column:engine_id"`
	Status                string `gorm:"column:status"`
	SkipReason            string `gorm:"column:skip_reason"`
	ResolvedExecutionPlan []byte `gorm:"column:resolved_execution_plan"`
	AssignedAgentID       *int   `gorm:"column:assigned_agent_id"`
}

type smokeCreatedScan struct {
	ID       int
	TargetID int
	Target   scanapp.TargetRef
	Tasks    map[string]smokePersistedTask
}

type smokeScanCreateEvidence struct {
	Passed                   bool `json:"passed"`
	CreatedScans             int  `json:"createdScans"`
	PersistedTaskRows        int  `json:"persistedTaskRows"`
	ExecutableTasks          int  `json:"executableTasks"`
	SavedPlanRepositoryReads int  `json:"savedPlanRepositoryReads"`
	PlanningSkippedTasks     int  `json:"planningSkippedTasks"`
	SkippedWithoutPlanReads  int  `json:"skippedWithoutPlanReads"`
}

type smokeInputSemanticsEvidence struct {
	Passed             bool     `json:"passed"`
	DomainSubdomains   []string `json:"domainSubdomains"`
	IPSubdomains       []string `json:"ipSubdomains"`
	CIDRSubdomains     []string `json:"cidrSubdomains"`
	DomainHostPorts    []string `json:"domainHostPorts"`
	IPHostPorts        []string `json:"ipHostPorts"`
	CIDRHostPorts      []string `json:"cidrHostPorts"`
	NoCartesianProduct bool     `json:"noCartesianProduct"`
}

type smokeWorkflowCaseEvidence struct {
	Passed                  bool     `json:"passed"`
	ScanStatus              string   `json:"scanStatus"`
	UpstreamStatus          string   `json:"upstreamStatus"`
	DownstreamStatuses      []string `json:"downstreamStatuses"`
	DownstreamNeverAssigned bool     `json:"downstreamNeverAssigned"`
}

type smokeWorkflowEvidence struct {
	Passed    bool                      `json:"passed"`
	Failure   smokeWorkflowCaseEvidence `json:"failure"`
	Cancelled smokeWorkflowCaseEvidence `json:"cancelled"`
}

func newSmokeScanStore() (*smokeScanStore, error) {
	root, err := os.MkdirTemp("", "lunafox-engine-cut-scan-store-")
	if err != nil {
		return nil, fmt.Errorf("create smoke scan store root: %w", err)
	}
	db, err := gorm.Open(sqlite.Open(filepath.Join(root, "scan-store.db")), &gorm.Config{})
	if err != nil {
		_ = os.RemoveAll(root)
		return nil, fmt.Errorf("open smoke scan store: %w", err)
	}
	if err := db.Exec(smokeScanStoreDDL).Error; err != nil {
		closeGormDB(db)
		_ = os.RemoveAll(root)
		return nil, fmt.Errorf("initialize smoke scan store: %w", err)
	}
	if err := db.Exec(`INSERT INTO agent (id) VALUES (?)`, smokeServerEvidenceAgentID).Error; err != nil {
		closeGormDB(db)
		_ = os.RemoveAll(root)
		return nil, fmt.Errorf("initialize smoke Agent: %w", err)
	}
	if err := db.Exec(`INSERT INTO agent_runtime_status (agent_id, session_id, session_epoch) VALUES (?, ?, ?)`,
		smokeServerEvidenceAgentID, smokeServerEvidenceSessionID, smokeServerEvidenceSessionEpoch).Error; err != nil {
		closeGormDB(db)
		_ = os.RemoveAll(root)
		return nil, fmt.Errorf("initialize smoke Agent session: %w", err)
	}
	tasks := scanrepo.NewScanTaskRepository(db)
	savedPlans, ok := tasks.(scanrepo.SavedExecutionPlanReader)
	if !ok {
		closeGormDB(db)
		_ = os.RemoveAll(root)
		return nil, fmt.Errorf("scan task repository does not expose the saved-plan reader boundary")
	}
	return &smokeScanStore{
		root: root, db: db, scans: scanrepo.NewScanRepository(db),
		tasks: tasks, savedPlans: savedPlans, nextTarget: 1000,
	}, nil
}

func closeGormDB(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

func (store *smokeScanStore) cleanup() {
	if store == nil {
		return
	}
	closeGormDB(store.db)
	if store.root != "" {
		_ = os.RemoveAll(store.root)
	}
}

func (store *smokeScanStore) setAgentExecutionSession(agentID int, sessionID string, sessionEpoch int64) error {
	if store == nil || store.db == nil || agentID <= 0 || strings.TrimSpace(sessionID) == "" || sessionEpoch <= 0 {
		return fmt.Errorf("smoke Agent execution session is invalid")
	}
	if err := store.db.Exec(`INSERT OR IGNORE INTO agent (id) VALUES (?)`, agentID).Error; err != nil {
		return err
	}
	result := store.db.Exec(
		`INSERT INTO agent_runtime_status (agent_id, session_id, session_epoch) VALUES (?, ?, ?)
		 ON CONFLICT(agent_id) DO UPDATE SET session_id = excluded.session_id, session_epoch = excluded.session_epoch
		 WHERE excluded.session_epoch > agent_runtime_status.session_epoch
		    OR (excluded.session_epoch = agent_runtime_status.session_epoch AND excluded.session_id = agent_runtime_status.session_id)`,
		agentID,
		strings.TrimSpace(sessionID),
		sessionEpoch,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errSmokeStaleAgentExecutionSession
	}
	return nil
}

type smokeWorkflowReader struct {
	manifest scanapp.ScanCreateWorkflowManifest
}

func (reader smokeWorkflowReader) GetScanWorkflowManifest(_ context.Context, id string) (scanapp.ScanCreateWorkflowManifest, error) {
	if id != reader.manifest.ScanWorkflowID {
		return scanapp.ScanCreateWorkflowManifest{}, fmt.Errorf("scan workflow %q not found", id)
	}
	return reader.manifest, nil
}

func (store *smokeScanStore) createScan(
	compiler *scanapp.PlanTaskCompiler,
	packages *packageMapReader,
	manifest scanapp.ScanCreateWorkflowManifest,
	targetType string,
	targetValue string,
	maxExecutionDuration time.Duration,
) (*smokeCreatedScan, error) {
	if store == nil || store.db == nil || store.scans == nil || store.tasks == nil {
		return nil, fmt.Errorf("smoke scan store is not initialized")
	}
	if compiler == nil || packages == nil {
		return nil, fmt.Errorf("smoke scan-create dependencies are required")
	}
	store.nextTarget++
	target := scanapp.TargetRef{ID: store.nextTarget, Name: targetValue, Type: targetType, CreatedAt: time.Now().UTC()}
	if err := store.db.Exec(`INSERT INTO target (id, name, type, created_at) VALUES (?, ?, ?, ?)`, target.ID, target.Name, target.Type, target.CreatedAt).Error; err != nil {
		return nil, fmt.Errorf("persist smoke target: %w", err)
	}
	service := scanapp.NewScanCreateService(
		smokeScanCreateCommandStore{repository: store.scans},
		func(_ context.Context, id int) (*scanapp.TargetRef, error) {
			if id != target.ID {
				return nil, fmt.Errorf("target %d not found", id)
			}
			copy := target
			return &copy, nil
		},
		nil,
		smokeWorkflowReader{manifest: manifest},
		packages,
	)
	if err := service.ConfigurePlanTask(compiler, scanapp.FixedPlanTaskLimitsProvider(maxExecutionDuration)); err != nil {
		return nil, fmt.Errorf("configure smoke ScanCreateService: %w", err)
	}
	configuration, err := materializeSmokeProfileConfiguration(manifest, packages, smokeWorkflowConfigurations[manifest.ScanWorkflowID])
	if err != nil {
		return nil, fmt.Errorf("materialize smoke workflow Profile: %w", err)
	}
	result, err := service.CreateBatch(context.Background(), &scanapp.CreateBatchInput{
		Requests:      []scanapp.CreateBatchItem{{TargetID: target.ID}},
		ScanWorkflow:  resourcenames.ScanWorkflow(manifest.ScanWorkflowID),
		Configuration: configuration,
		InputSource:   scanapp.InputSourceScanSnapshot,
		TriggerType:   scanapp.ScanTriggerTypeManual,
	})
	if err != nil {
		return nil, fmt.Errorf("create smoke scan %q: %w", manifest.ScanWorkflowID, err)
	}
	if result == nil || result.CreatedCount != 1 || len(result.Scans) != 1 || len(result.Failed) != 0 || len(result.Skipped) != 0 {
		return nil, fmt.Errorf("scan-create returned an unexpected result for %q: %#v", manifest.ScanWorkflowID, result)
	}
	scanID := result.Scans[0].ID
	var rows []smokePersistedTask
	if err := store.db.Table("scan_task").
		Select("id, scan_id, stage_order, engine_id, status, skip_reason, resolved_execution_plan, assigned_agent_id").
		Where("scan_id = ?", scanID).
		Order("stage_order ASC, step_order ASC, id ASC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("read persisted smoke tasks: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("scan-create persisted no tasks for %q", manifest.ScanWorkflowID)
	}
	created := &smokeCreatedScan{ID: scanID, TargetID: target.ID, Target: target, Tasks: make(map[string]smokePersistedTask, len(rows))}
	for _, row := range rows {
		if _, exists := created.Tasks[row.EngineID]; exists {
			return nil, fmt.Errorf("smoke workflow %q repeats engine %q", manifest.ScanWorkflowID, row.EngineID)
		}
		created.Tasks[row.EngineID] = row
	}
	return created, nil
}

func materializeSmokeProfileConfiguration(manifest scanapp.ScanCreateWorkflowManifest, packages *packageMapReader, submitted map[string]any) (map[string]any, error) {
	if packages == nil {
		return nil, fmt.Errorf("package reader is required")
	}
	steps, ok := submitted["steps"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("smoke submitted configuration has no steps")
	}
	complete := make(map[string]any, len(steps))
	for _, stage := range manifest.Stages {
		for _, step := range stage.Steps {
			entry, ok := steps[step.StepID].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("smoke submitted configuration has no step %q", step.StepID)
			}
			enabled, ok := entry["enabled"].(bool)
			if !ok || !enabled {
				return nil, fmt.Errorf("smoke submitted configuration must enable step %q", step.StepID)
			}
			raw, ok := entry["engineConfig"].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("smoke submitted configuration has no Engine config for step %q", step.StepID)
			}
			pkg, ok := packages.packageForEngine(step.EngineID)
			if !ok {
				return nil, fmt.Errorf("smoke package %q is unavailable", step.EngineID)
			}
			resolved, err := engineexecution.NormalizeAndValidateConfig(raw, pkg.Definition.Execution)
			if err != nil {
				return nil, fmt.Errorf("step %q: %w", step.StepID, err)
			}
			complete[step.StepID] = map[string]any{"enabled": true, "engineConfig": resolved}
		}
	}
	return map[string]any{"steps": complete}, nil
}

func smokeOneStepManifest(id, stageID, stepID, engineID string, engineConfig map[string]any) scanapp.ScanCreateWorkflowManifest {
	smokeWorkflowConfigurations[id] = map[string]any{"steps": map[string]any{
		stepID: map[string]any{"enabled": true, "engineConfig": engineConfig},
	}}
	return scanapp.ScanCreateWorkflowManifest{
		ScanWorkflowID: id,
		Stages: []scanapp.ScanCreateWorkflowStage{{
			StageID: stageID,
			Steps:   []scanapp.ScanCreateWorkflowStep{{StepID: stepID, EngineID: engineID}},
		}},
	}
}

func smokeFullWorkflowManifest(id string) scanapp.ScanCreateWorkflowManifest {
	return smokeFullWorkflowManifestWithPortConfig(id, portTaskConfig("80,443,8080", 100))
}

func smokeFullWorkflowManifestWithPortConfig(id string, portConfig map[string]any) scanapp.ScanCreateWorkflowManifest {
	smokeWorkflowConfigurations[id] = map[string]any{"steps": map[string]any{
		"subdomain_discovery": map[string]any{"enabled": true, "engineConfig": subdomainTaskConfig()},
		"port_scan":           map[string]any{"enabled": true, "engineConfig": portConfig},
		"website_discovery":   map[string]any{"enabled": true, "engineConfig": websiteTaskConfig()},
	}}
	return scanapp.ScanCreateWorkflowManifest{
		ScanWorkflowID: id,
		Stages: []scanapp.ScanCreateWorkflowStage{
			{StageID: "discovery", Steps: []scanapp.ScanCreateWorkflowStep{{StepID: "subdomain_discovery", EngineID: engineSubdomainID}}},
			{StageID: "ports", Steps: []scanapp.ScanCreateWorkflowStep{{StepID: "port_scan", EngineID: enginePortID}}},
			{StageID: "websites", Steps: []scanapp.ScanCreateWorkflowStep{{StepID: "website_discovery", EngineID: engineWebsiteID}}},
		},
	}
}

func smokePortWebsiteManifest(id string, portConfig map[string]any) scanapp.ScanCreateWorkflowManifest {
	// Runtime chain evidence intentionally contains only the two applicable
	// stages, so IP/CIDR planning skips cannot masquerade as chain execution.
	smokeWorkflowConfigurations[id] = map[string]any{"steps": map[string]any{
		"port_scan":         map[string]any{"enabled": true, "engineConfig": portConfig},
		"website_discovery": map[string]any{"enabled": true, "engineConfig": websiteTaskConfig()},
	}}
	return scanapp.ScanCreateWorkflowManifest{
		ScanWorkflowID: id,
		Stages: []scanapp.ScanCreateWorkflowStage{
			{StageID: "ports", Steps: []scanapp.ScanCreateWorkflowStep{{StepID: "port_scan", EngineID: enginePortID}}},
			{StageID: "websites", Steps: []scanapp.ScanCreateWorkflowStep{{StepID: "website_discovery", EngineID: engineWebsiteID}}},
		},
	}
}

func subdomainTaskConfig() map[string]any {
	return map[string]any{
		"recon":      map[string]any{"enabled": false},
		"bruteforce": map[string]any{"enabled": false},
		"resolve":    map[string]any{"enabled": true, "resolvers": "wordlists/2"},
	}
}

func subdomainFailureTaskConfig() map[string]any {
	return subdomainResourceTaskConfig()
}

func subdomainResourceTaskConfig() map[string]any {
	return map[string]any{
		"recon": map[string]any{"enabled": false},
		"bruteforce": map[string]any{
			"enabled":   true,
			"wordlist":  "wordlists/1",
			"resolvers": "wordlists/2",
		},
		"resolve": map[string]any{"enabled": true, "resolvers": "wordlists/2"},
	}
}

func websiteTaskConfig() map[string]any {
	return map[string]any{"httpx": map[string]any{
		"enabled": true, "timeout": 60, "threads": 1, "rate-limit": 100,
		"request-timeout": 1, "retries": 0,
	}}
}

func (store *smokeScanStore) savedPlan(ctx context.Context, task smokePersistedTask) (*agentexecutionv1.ResolvedEngineExecutionPlan, []byte, error) {
	plan, err := store.savedPlans.GetSavedExecutionPlan(ctx, task.ID)
	if err != nil {
		return nil, nil, err
	}
	lease, err := store.tasks.GetSavedExecutionPlanLease(ctx, task.ID)
	if err != nil {
		return nil, nil, err
	}
	if plan == nil || lease == nil || len(lease.ResolvedExecutionPlan) == 0 {
		return nil, nil, fmt.Errorf("task %d has no persisted executable plan", task.ID)
	}
	return plan, append([]byte(nil), lease.ResolvedExecutionPlan...), nil
}

// forceSubdomainResolveDisabledPlan constructs an Agent-visible plan that has
// bypassed the Server's manifest-aware requiredEnabled validation. The generic
// saved-plan protocol does not own that Engine-specific invariant, so this
// tests the handler's separate fail-closed defense without teaching normal
// scan creation to accept a disabled Resolve section.
func (store *smokeScanStore) forceSubdomainResolveDisabledPlan(task *smokePersistedTask) error {
	if store == nil || store.db == nil || task == nil || task.ID <= 0 {
		return errors.New("smoke persisted task is required")
	}
	plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(task.ResolvedExecutionPlan)
	if err != nil {
		return fmt.Errorf("decode persisted plan: %w", err)
	}
	resolveFound := false
	for _, section := range plan.GetConfig().GetSections() {
		if section.GetSectionId() != "resolve" {
			continue
		}
		section.Enabled = false
		section.Params = nil
		resolveFound = true
	}
	if !resolveFound {
		return errors.New("persisted Subdomain Discovery plan has no Resolve section")
	}
	bindings := plan.GetConfigResourceBindings()
	plan.ConfigResourceBindings = bindings[:0]
	for _, binding := range bindings {
		if binding.GetSectionId() != "resolve" {
			plan.ConfigResourceBindings = append(plan.ConfigResourceBindings, binding)
		}
	}
	encoded, err := agentexecution.MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		return fmt.Errorf("encode forced defense-in-depth plan: %w", err)
	}
	if err := store.db.Exec(`UPDATE scan_task SET resolved_execution_plan = ? WHERE id = ?`, encoded, task.ID).Error; err != nil {
		return fmt.Errorf("persist forced defense-in-depth plan: %w", err)
	}
	task.ResolvedExecutionPlan = append(task.ResolvedExecutionPlan[:0], encoded...)
	return nil
}

// forceNucleiAcknowledgedBatchPlan narrows only the Server-owned batch limit
// in this dedicated smoke plan. It proves that an in-scope finding is durably
// acknowledged before the later delayed request reaches the task timeout;
// normal plans retain the production batch limits.
func (store *smokeScanStore) forceNucleiAcknowledgedBatchPlan(task *smokePersistedTask) error {
	if store == nil || store.db == nil || task == nil || task.ID <= 0 {
		return errors.New("smoke persisted task is required")
	}
	plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(task.ResolvedExecutionPlan)
	if err != nil {
		return fmt.Errorf("decode persisted plan: %w", err)
	}
	if plan.GetEngineRelease().GetEngine() != engineNucleiID || plan.GetLimits() == nil {
		return errors.New("persisted Nuclei plan has no execution limits")
	}
	plan.Limits.ResultBatchMaxItems = 1
	encoded, err := agentexecution.MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		return fmt.Errorf("encode acknowledged-result plan: %w", err)
	}
	if err := store.db.Exec(`UPDATE scan_task SET resolved_execution_plan = ? WHERE id = ?`, encoded, task.ID).Error; err != nil {
		return fmt.Errorf("persist acknowledged-result plan: %w", err)
	}
	task.ResolvedExecutionPlan = append(task.ResolvedExecutionPlan[:0], encoded...)
	return nil
}

func (store *smokeScanStore) claimAndReport(ctx context.Context, scanID int, expectedEngineID, status string, failure *scanapp.FailureDetail) (int, error) {
	plan, err := store.tasks.ClaimNextCompatibleSavedExecutionPlan(
		ctx,
		smokeServerEvidenceAgentID,
		smokeServerEvidenceSessionID,
		smokeServerEvidenceSessionEpoch,
		uuid.NewString(),
		[]uint32{2},
	)
	if err != nil {
		return 0, fmt.Errorf("claim persisted smoke plan: %w", err)
	}
	if plan == nil || plan.GetEngineRelease().GetEngine() != expectedEngineID {
		return 0, fmt.Errorf("claimed engine = %q, want %q", plan.GetEngineRelease().GetEngine(), expectedEngineID)
	}
	claimedScanID, taskID, err := resourcenames.ParseTask(plan.GetTask())
	if err != nil || claimedScanID != scanID {
		return 0, fmt.Errorf("claimed task %q is outside scan %d", plan.GetTask(), scanID)
	}
	bridge := scanapp.NewScanTaskBridgeService(store.tasks, store.scans)
	if err := bridge.ReportTerminalTaskResult(
		ctx,
		smokeServerEvidenceAgentID,
		smokeServerEvidenceSessionID,
		smokeServerEvidenceSessionEpoch,
		taskID,
		status,
		failure,
	); err != nil {
		return 0, fmt.Errorf("report persisted smoke terminal result: %w", err)
	}
	return taskID, nil
}

func (store *smokeScanStore) retireInputFixtureScan(ctx context.Context, scanID int) error {
	if _, err := store.tasks.CancelUnstartedTasksByScanID(ctx, scanID); err != nil {
		return err
	}
	return store.scans.UpdateScanStatus(scanID, "cancelled", nil)
}

func (store *smokeScanStore) observePropagation(scanID, upstreamTaskID, expectedDownstream int) (smokeWorkflowCaseEvidence, error) {
	var scanRow struct {
		Status string `gorm:"column:status"`
	}
	if err := store.db.Table("scan").Select("status").Where("id = ?", scanID).Take(&scanRow).Error; err != nil {
		return smokeWorkflowCaseEvidence{}, err
	}
	var upstream smokePersistedTask
	if err := store.db.Table("scan_task").Select("id, scan_id, stage_order, status").Where("id = ?", upstreamTaskID).Take(&upstream).Error; err != nil {
		return smokeWorkflowCaseEvidence{}, err
	}
	var downstream []smokePersistedTask
	if err := store.db.Table("scan_task").
		Select("id, scan_id, stage_order, status, assigned_agent_id").
		Where("scan_id = ? AND stage_order > ?", scanID, upstream.StageOrder).
		Order("stage_order ASC, id ASC").
		Scan(&downstream).Error; err != nil {
		return smokeWorkflowCaseEvidence{}, err
	}
	evidence := smokeWorkflowCaseEvidence{
		ScanStatus: scanRow.Status, UpstreamStatus: upstream.Status,
		DownstreamStatuses: make([]string, 0, len(downstream)), DownstreamNeverAssigned: true,
	}
	for _, task := range downstream {
		evidence.DownstreamStatuses = append(evidence.DownstreamStatuses, task.Status)
		if task.AssignedAgentID != nil {
			evidence.DownstreamNeverAssigned = false
		}
	}
	evidence.Passed = len(downstream) == expectedDownstream && evidence.DownstreamNeverAssigned && allStringsEqual(evidence.DownstreamStatuses, "cancelled")
	return evidence, nil
}

func runSmokeWorkflowEvidence(ctx context.Context, store *smokeScanStore, compiler *scanapp.PlanTaskCompiler, packages *packageMapReader) (smokeWorkflowEvidence, error) {
	failureScan, err := store.createScan(
		compiler, packages, smokeFullWorkflowManifest("smoke_linked_failure"),
		scanapp.ExecutionTargetTypeDomain, "failure.example.com", 30*time.Second,
	)
	if err != nil {
		return smokeWorkflowEvidence{}, err
	}
	failureTaskID, err := store.claimAndReport(ctx, failureScan.ID, engineSubdomainID, "failed", &scanapp.FailureDetail{
		Kind: "engine_exit_failed", Message: "The Engine exited unsuccessfully.",
	})
	if err != nil {
		return smokeWorkflowEvidence{}, err
	}
	failureEvidence, err := store.observePropagation(failureScan.ID, failureTaskID, 2)
	if err != nil {
		return smokeWorkflowEvidence{}, err
	}
	failureEvidence.Passed = failureEvidence.Passed && failureEvidence.ScanStatus == "failed" && failureEvidence.UpstreamStatus == "failed"

	cancelScan, err := store.createScan(
		compiler, packages, smokeFullWorkflowManifest("smoke_linked_cancel"),
		scanapp.ExecutionTargetTypeIP, "192.0.2.20", 30*time.Second,
	)
	if err != nil {
		return smokeWorkflowEvidence{}, err
	}
	cancelTaskID, err := store.claimAndReport(ctx, cancelScan.ID, enginePortID, "cancelled", nil)
	if err != nil {
		return smokeWorkflowEvidence{}, err
	}
	cancelEvidence, err := store.observePropagation(cancelScan.ID, cancelTaskID, 1)
	if err != nil {
		return smokeWorkflowEvidence{}, err
	}
	cancelEvidence.Passed = cancelEvidence.Passed && cancelEvidence.ScanStatus == "cancelled" && cancelEvidence.UpstreamStatus == "cancelled"

	evidence := smokeWorkflowEvidence{Failure: failureEvidence, Cancelled: cancelEvidence}
	evidence.Passed = failureEvidence.Passed && cancelEvidence.Passed
	if !evidence.Passed {
		return smokeWorkflowEvidence{}, fmt.Errorf("workflow propagation evidence did not converge: %#v", evidence)
	}
	return evidence, nil
}

type smokeDNSCursor struct{ byScan map[int][]string }

func (cursor smokeDNSCursor) ForEachDNSNameByScanID(ctx context.Context, scanID int, visit func(string) error) error {
	if ctx == nil || visit == nil {
		return fmt.Errorf("DNS cursor context and visitor are required")
	}
	for _, value := range cursor.byScan[scanID] {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := visit(value); err != nil {
			return err
		}
	}
	return nil
}

type smokeHostPortCursor struct {
	byScan map[int][]scanapp.HostPortEvidence
}

func (cursor smokeHostPortCursor) ForEachHostPortByScanID(ctx context.Context, scanID int, visit func(scanapp.HostPortEvidence) error) error {
	if ctx == nil || visit == nil {
		return fmt.Errorf("HostPort cursor context and visitor are required")
	}
	for _, value := range cursor.byScan[scanID] {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := visit(value); err != nil {
			return err
		}
	}
	return nil
}

func collectSmokeRecords(producer func(func(string) error) error) ([]string, error) {
	if producer == nil {
		return nil, fmt.Errorf("smoke producer is required")
	}
	records := make([]string, 0)
	if err := producer(func(value string) error {
		records = append(records, value)
		return nil
	}); err != nil {
		return nil, err
	}
	return records, nil
}

func collectSmokeHostPortRecords(producer scanapp.HostPortsProducer) ([]string, error) {
	if producer == nil {
		return nil, fmt.Errorf("smoke HostPort producer is required")
	}
	records := make([]string, 0)
	if err := producer(func(value scanapp.HostPortEvidence) error {
		encoded, err := json.Marshal(struct {
			Host string `json:"host"`
			IP   string `json:"ip"`
			Port int    `json:"port"`
		}{Host: value.Host, IP: value.IP, Port: value.Port})
		if err != nil {
			return err
		}
		records = append(records, string(encoded))
		return nil
	}); err != nil {
		return nil, err
	}
	return records, nil
}

func (store *smokeScanStore) subdomainsEvidence(ctx context.Context, scan *smokeCreatedScan, cursor scanapp.DNSNameCursor) ([]string, error) {
	_, ok := scan.Tasks[enginePortID]
	if !ok {
		return nil, fmt.Errorf("scan %d has no port task", scan.ID)
	}
	blacklist, err := newSmokeEmptyExecutionInputBlacklistFilter()
	if err != nil {
		return nil, err
	}
	producer, err := scanapp.NewSubdomainsProducer(ctx, scan.ID, cursor, blacklist)
	if err != nil {
		return nil, err
	}
	return collectSmokeRecords(producer)
}

func (store *smokeScanStore) hostPortsEvidence(ctx context.Context, scan *smokeCreatedScan, cursor scanapp.HostPortCursor) ([]string, error) {
	_, ok := scan.Tasks[engineWebsiteID]
	if !ok {
		return nil, fmt.Errorf("scan %d has no website task", scan.ID)
	}
	blacklist, err := newSmokeEmptyExecutionInputBlacklistFilter()
	if err != nil {
		return nil, err
	}
	producer, err := scanapp.NewHostPortsProducer(ctx, scan.ID, cursor, blacklist)
	if err != nil {
		return nil, err
	}
	return collectSmokeHostPortRecords(producer)
}

func runSmokeInputSemanticsEvidence(ctx context.Context, store *smokeScanStore, compiler *scanapp.PlanTaskCompiler, packages *packageMapReader, fixtureIP string) (smokeInputSemanticsEvidence, error) {
	domainScan, err := store.createScan(
		compiler, packages, smokeFullWorkflowManifest("smoke_inputs_domain"),
		scanapp.ExecutionTargetTypeDomain, "example.com", 30*time.Second,
	)
	if err != nil {
		return smokeInputSemanticsEvidence{}, err
	}
	if _, err := store.claimAndReport(ctx, domainScan.ID, engineSubdomainID, "succeeded", nil); err != nil {
		return smokeInputSemanticsEvidence{}, err
	}
	domainHosts, err := store.subdomainsEvidence(ctx, domainScan, smokeDNSCursor{byScan: map[int][]string{
		domainScan.ID: {"api.example.com", "www.example.com"},
	}})
	if err != nil {
		return smokeInputSemanticsEvidence{}, err
	}
	if _, err := store.claimAndReport(ctx, domainScan.ID, enginePortID, "succeeded", nil); err != nil {
		return smokeInputSemanticsEvidence{}, err
	}
	domainURLs, err := store.hostPortsEvidence(ctx, domainScan, smokeHostPortCursor{byScan: map[int][]scanapp.HostPortEvidence{
		domainScan.ID: {
			{Host: "api.example.com", IP: "192.0.2.10", Port: 8080},
			{Host: "b.example.com", IP: "192.0.2.11", Port: 443},
			{Host: "example.com", IP: "192.0.2.12", Port: 80},
		},
	}})
	if err != nil {
		return smokeInputSemanticsEvidence{}, err
	}
	if err := store.retireInputFixtureScan(ctx, domainScan.ID); err != nil {
		return smokeInputSemanticsEvidence{}, err
	}

	ipScan, err := store.createScan(
		compiler, packages, smokeFullWorkflowManifest("smoke_inputs_ip"),
		scanapp.ExecutionTargetTypeIP, fixtureIP, 30*time.Second,
	)
	if err != nil {
		return smokeInputSemanticsEvidence{}, err
	}
	ipHosts, err := store.subdomainsEvidence(ctx, ipScan, smokeDNSCursor{})
	if err != nil {
		return smokeInputSemanticsEvidence{}, err
	}
	if _, err := store.claimAndReport(ctx, ipScan.ID, enginePortID, "succeeded", nil); err != nil {
		return smokeInputSemanticsEvidence{}, err
	}
	ipURLs, err := store.hostPortsEvidence(ctx, ipScan, smokeHostPortCursor{byScan: map[int][]scanapp.HostPortEvidence{
		ipScan.ID: {{IP: fixtureIP, Port: 80}},
	}})
	if err != nil {
		return smokeInputSemanticsEvidence{}, err
	}
	if err := store.retireInputFixtureScan(ctx, ipScan.ID); err != nil {
		return smokeInputSemanticsEvidence{}, err
	}

	cidrScan, err := store.createScan(
		compiler, packages, smokeFullWorkflowManifest("smoke_inputs_cidr"),
		scanapp.ExecutionTargetTypeCIDR, "192.0.2.0/30", 30*time.Second,
	)
	if err != nil {
		return smokeInputSemanticsEvidence{}, err
	}
	cidrHosts, err := store.subdomainsEvidence(ctx, cidrScan, smokeDNSCursor{})
	if err != nil {
		return smokeInputSemanticsEvidence{}, err
	}
	if _, err := store.claimAndReport(ctx, cidrScan.ID, enginePortID, "succeeded", nil); err != nil {
		return smokeInputSemanticsEvidence{}, err
	}
	cidrURLs, err := store.hostPortsEvidence(ctx, cidrScan, smokeHostPortCursor{})
	if err != nil {
		return smokeInputSemanticsEvidence{}, err
	}
	if err := store.retireInputFixtureScan(ctx, cidrScan.ID); err != nil {
		return smokeInputSemanticsEvidence{}, err
	}

	evidence := smokeInputSemanticsEvidence{
		DomainSubdomains: domainHosts,
		IPSubdomains:     ipHosts,
		CIDRSubdomains:   cidrHosts,
		DomainHostPorts:  domainURLs,
		IPHostPorts:      ipURLs,
		CIDRHostPorts:    cidrURLs,
	}
	evidence.NoCartesianProduct = true
	evidence.Passed = equalStrings(domainHosts, []string{"api.example.com", "www.example.com"}) &&
		equalStrings(ipHosts, nil) &&
		equalStrings(cidrHosts, nil) &&
		equalStrings(domainURLs, []string{
			`{"host":"api.example.com","ip":"192.0.2.10","port":8080}`,
			`{"host":"b.example.com","ip":"192.0.2.11","port":443}`,
			`{"host":"example.com","ip":"192.0.2.12","port":80}`,
		}) &&
		equalStrings(ipURLs, []string{`{"host":"","ip":"` + fixtureIP + `","port":80}`}) &&
		equalStrings(cidrURLs, nil) && evidence.NoCartesianProduct
	if !evidence.Passed {
		return smokeInputSemanticsEvidence{}, fmt.Errorf("input producer evidence did not match canonical records: %#v", evidence)
	}
	return evidence, nil
}

func (store *smokeScanStore) scanCreateEvidence(ctx context.Context) (smokeScanCreateEvidence, error) {
	var evidence smokeScanCreateEvidence
	var createdScans int64
	if err := store.db.Table("scan").Count(&createdScans).Error; err != nil {
		return smokeScanCreateEvidence{}, err
	}
	evidence.CreatedScans = int(createdScans)
	var tasks []smokePersistedTask
	if err := store.db.Table("scan_task").
		Select("id, scan_id, stage_order, engine_id, status, skip_reason, resolved_execution_plan").
		Order("id ASC").Scan(&tasks).Error; err != nil {
		return smokeScanCreateEvidence{}, err
	}
	evidence.PersistedTaskRows = len(tasks)
	for _, task := range tasks {
		plan, err := store.savedPlans.GetSavedExecutionPlan(ctx, task.ID)
		if err != nil {
			return smokeScanCreateEvidence{}, fmt.Errorf("read saved plan for task %d: %w", task.ID, err)
		}
		if task.Status == scanapp.CreateTaskStatusSkipped {
			evidence.PlanningSkippedTasks++
			if plan != nil || len(task.ResolvedExecutionPlan) != 0 || strings.TrimSpace(task.SkipReason) == "" {
				return smokeScanCreateEvidence{}, fmt.Errorf("planning-time skipped task %d retained a plan or lost its reason", task.ID)
			}
			evidence.SkippedWithoutPlanReads++
			continue
		}
		evidence.ExecutableTasks++
		if plan == nil || len(task.ResolvedExecutionPlan) == 0 {
			return smokeScanCreateEvidence{}, fmt.Errorf("executable task %d has no saved-plan repository result", task.ID)
		}
		evidence.SavedPlanRepositoryReads++
	}
	evidence.Passed = evidence.CreatedScans > 0 && evidence.PersistedTaskRows > 0 &&
		evidence.PlanningSkippedTasks > 0 &&
		evidence.ExecutableTasks == evidence.SavedPlanRepositoryReads &&
		evidence.PlanningSkippedTasks == evidence.SkippedWithoutPlanReads &&
		evidence.PersistedTaskRows == evidence.ExecutableTasks+evidence.PlanningSkippedTasks
	if !evidence.Passed {
		return smokeScanCreateEvidence{}, fmt.Errorf("scan-create repository evidence is incomplete: %#v", evidence)
	}
	return evidence, nil
}

func allStringsEqual(values []string, wanted string) bool {
	for _, value := range values {
		if value != wanted {
			return false
		}
	}
	return true
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
