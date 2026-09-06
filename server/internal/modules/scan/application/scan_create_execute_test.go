package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
	"gorm.io/gorm"
)

type scanCreateStoreStub struct{}

func (scanCreateStoreStub) CreateWithScanTasksAndPlans(_ context.Context, scan *CreateScan, finalize ScanCreateTaskFinalizer) error {
	if scan == nil || finalize == nil {
		return errors.New("planned scan create arguments are required")
	}
	scan.ID = 1
	for index := range scan.ScanTasks {
		task := &scan.ScanTasks[index]
		task.ID = index + 1
		if err := finalize(scan.ID, task.ID, task); err != nil {
			return err
		}
	}
	return nil
}

type scanCreateStoreCaptureStub struct {
	lastScan *CreateScan
	scans    []CreateScan
}

func (stub *scanCreateStoreCaptureStub) CreateWithScanTasksAndPlans(_ context.Context, scan *CreateScan, finalize ScanCreateTaskFinalizer) error {
	if scan == nil || finalize == nil {
		return errors.New("planned scan create arguments are required")
	}
	scan.ID = len(stub.scans) + 1
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
	stub.scans = append(stub.scans, clone)
	return nil
}

type scanCreateStoreFailOnCallStub struct {
	calls      int
	failOnCall int
	err        error
	scans      []CreateScan
}

func (stub *scanCreateStoreFailOnCallStub) CreateWithScanTasksAndPlans(_ context.Context, scan *CreateScan, finalize ScanCreateTaskFinalizer) error {
	stub.calls++
	if stub.calls == stub.failOnCall {
		return stub.err
	}
	scan.ID = stub.calls
	for index := range scan.ScanTasks {
		task := &scan.ScanTasks[index]
		task.ID = scan.ID*100 + index + 1
		if err := finalize(scan.ID, task.ID, task); err != nil {
			return err
		}
	}
	clone := *scan
	clone.ScanTasks = append([]CreateScanTask(nil), scan.ScanTasks...)
	stub.scans = append(stub.scans, clone)
	return nil
}

type scanCreateWorkflowReaderStub struct {
	manifest ScanCreateWorkflowManifest
	err      error
}

type cancelAwareScanCreateWorkflowReader struct{ called bool }

func (reader *cancelAwareScanCreateWorkflowReader) GetScanWorkflowManifest(ctx context.Context, _ string) (ScanCreateWorkflowManifest, error) {
	reader.called = true
	return ScanCreateWorkflowManifest{}, ctx.Err()
}

type triggerTimeDependencyState struct {
	unavailable string
}

type triggerTimeWorkflowReader struct {
	state    *triggerTimeDependencyState
	manifest ScanCreateWorkflowManifest
}

func (reader triggerTimeWorkflowReader) GetScanWorkflowManifest(context.Context, string) (ScanCreateWorkflowManifest, error) {
	if reader.state.unavailable == "workflow" {
		return ScanCreateWorkflowManifest{}, errors.New("current workflow unavailable")
	}
	return reader.manifest, nil
}

type triggerTimeEnginePackageReader struct {
	state         *triggerTimeDependencyState
	enginePackage ScanCreateEnginePackage
}

func (reader triggerTimeEnginePackageReader) LoadEnginePackage(context.Context, string) (ScanCreateEnginePackage, error) {
	if reader.state.unavailable == "engine" {
		return ScanCreateEnginePackage{}, errors.New("current engine package unavailable")
	}
	return reader.enginePackage, nil
}

type triggerTimeResourceResolver struct {
	state *triggerTimeDependencyState
}

func (resolver triggerTimeResourceResolver) ResolveConfigResource(_ context.Context, request ConfigResourceResolveRequest) (PlanTaskWordlist, error) {
	if resolver.state.unavailable == "configuration resource" {
		return PlanTaskWordlist{}, NewConfigResourceUnavailableError(request.Field, request.ResourceKind, request.ResourceName, errors.New("current resource unavailable"))
	}
	return PlanTaskWordlist{
		Resource: request.ResourceName, Basename: "dns.txt",
		SizeBytes: 8, LineCount: 2, SHA256: planTaskWordlistDigest,
	}, nil
}

type triggerTimeAgentLookup struct {
	state *triggerTimeDependencyState
}

func (lookup triggerTimeAgentLookup) AgentExists(context.Context, int) (bool, error) {
	return lookup.state.unavailable != "pinned agent", nil
}

func (stub scanCreateWorkflowReaderStub) GetScanWorkflowManifest(context.Context, string) (ScanCreateWorkflowManifest, error) {
	if stub.err != nil {
		return ScanCreateWorkflowManifest{}, stub.err
	}
	return stub.manifest, nil
}

type scanCreateEnginePackageReaderStub struct {
	enginePackages map[string]ScanCreateEnginePackage
	err            error
}

func (stub scanCreateEnginePackageReaderStub) ListEnginePackagesByID() (map[string]ScanCreateEnginePackage, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return stub.enginePackages, nil
}

func (stub scanCreateEnginePackageReaderStub) LoadExactPackage(_ context.Context, identity PlanTaskPackageIdentity) (PlanTaskPackage, error) {
	if stub.err != nil {
		return PlanTaskPackage{}, stub.err
	}
	entry, ok := stub.enginePackages[identity.EngineID]
	if !ok {
		return PlanTaskPackage{}, fmt.Errorf("package %q not found", identity.EngineID)
	}
	if entry.Package.Identity != identity {
		return PlanTaskPackage{}, fmt.Errorf("package identity mismatch for %q", identity.EngineID)
	}
	return entry.Package, nil
}

func (stub scanCreateEnginePackageReaderStub) LoadEnginePackage(_ context.Context, engineID string) (ScanCreateEnginePackage, error) {
	entry, ok := stub.enginePackages[engineID]
	if !ok {
		return ScanCreateEnginePackage{}, fmt.Errorf("%w: engine %q", ErrCreateScanWorkflowEngineUnavailable, engineID)
	}
	return entry, nil
}

type selectiveScanCreatePackageReaderStub struct {
	enginePackages map[string]ScanCreateEnginePackage
	loadCalls      []string
}

func (stub *selectiveScanCreatePackageReaderStub) LoadEnginePackage(_ context.Context, engineID string) (ScanCreateEnginePackage, error) {
	stub.loadCalls = append(stub.loadCalls, engineID)
	entry, ok := stub.enginePackages[engineID]
	if !ok {
		return ScanCreateEnginePackage{}, fmt.Errorf("package %q not found", engineID)
	}
	return entry, nil
}

type scanCreateRegistrationReaderStub struct {
	exists map[string]bool
	calls  []string
}

func (stub *scanCreateRegistrationReaderStub) EngineRegistrationExists(_ context.Context, engineID string) (bool, error) {
	stub.calls = append(stub.calls, engineID)
	return stub.exists[engineID], nil
}

func scanCreateReaderStubs() (ScanCreateWorkflowReader, ScanCreateEnginePackageReader) {
	manifest := ScanCreateWorkflowManifest{
		ScanWorkflowID: "subdomain_discovery",
		Stages: []ScanCreateWorkflowStage{{
			StageID: "discovery",
			Steps: []ScanCreateWorkflowStep{{
				StepID:   "subdomain_discovery",
				EngineID: "engine.lunafox.subdomain_discovery",
			}},
		}},
	}
	definition := scanCreateTestEngineDefinition("engine.lunafox.subdomain_discovery", []string{engineexecution.TargetTypeDomain})
	enginePackages := map[string]ScanCreateEnginePackage{
		"engine.lunafox.subdomain_discovery": {
			Package: scanCreateTestPlanTaskPackage(definition),
		},
	}
	return scanCreateWorkflowReaderStub{manifest: manifest}, scanCreateEnginePackageReaderStub{enginePackages: enginePackages}
}

func scanCreateTestEngineDefinition(engineID string, targetTypes []string) enginecontract.EngineDefinition {
	return enginecontract.EngineDefinition{
		ManifestVersion: enginecontract.SupportedRootManifestVersion,
		EngineID:        engineID,
		Publisher:       "lunafox",
		Execution: engineexecution.ExecutionDefinition{
			EngineAPIMajor:       2,
			SupportedTargetTypes: append([]string(nil), targetTypes...),
			ConfigSections: []engineexecution.ConfigSectionDefinition{{
				ID:             "recon",
				DefaultEnabled: true,
				Params: []engineexecution.ParamDefinition{
					{Key: "timeout", Type: engineexecution.ParamTypeInteger, Default: 3600},
					{Key: "threads", Type: engineexecution.ParamTypeInteger, Default: 10},
				},
			}},
		},
	}
}

func scanCreateTestPlanTaskPackage(definition enginecontract.EngineDefinition) PlanTaskPackage {
	return PlanTaskPackage{
		Identity:       PlanTaskPackageIdentity{EngineID: definition.EngineID, PackageDigest: planTaskPackageDigest},
		PackageVersion: "1.0.0",
		Definition:     definition,
		RuntimeImageRefs: []string{
			"docker.io/lunafox/lunafox-engine-runtime-subdomain-discovery@" + planTaskImageDigest,
		},
	}
}

func mustConfigureScanCreatePlanTask(t *testing.T, service *ScanCreateService, reader ScanCreateEnginePackageReader) *ScanCreateService {
	t.Helper()
	packageReader, ok := reader.(PlanTaskPackageReader)
	if !ok {
		t.Fatalf("scan create package reader must expose exact v2 packages")
	}
	compiler, err := NewPlanTaskCompiler(packageReader, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler failed: %v", err)
	}
	if err := service.ConfigurePlanTask(compiler, FixedPlanTaskLimitsProvider(time.Hour)); err != nil {
		t.Fatalf("ConfigurePlanTask failed: %v", err)
	}
	return service
}

func singleTargetBatchCreateInput(targetID int, scanWorkflow string, configuration map[string]any) *CreateBatchInput {
	return &CreateBatchInput{
		Requests:      []CreateBatchItem{{TargetID: targetID}},
		ScanWorkflow:  scanWorkflow,
		Configuration: configuration,
		InputSource:   InputSourceScanSnapshot,
		TriggerType:   ScanTriggerTypeManual,
	}
}

func TestScanCreateServiceRejectsInvalidTriggerBeforePersistence(t *testing.T) {
	store := &scanCreateStoreCaptureStub{}
	service := NewScanCreateService(store, nil, nil, nil, nil)

	for _, test := range []struct {
		name  string
		input *CreateBatchInput
	}{
		{name: "missing", input: &CreateBatchInput{TriggerType: ""}},
		{name: "unknown", input: &CreateBatchInput{TriggerType: ScanTriggerType("legacy")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.CreateBatch(context.Background(), test.input)
			if !errors.Is(err, ErrCreateInvalidTriggerType) {
				t.Fatalf("CreateBatch() error = %v, want invalid trigger type", err)
			}
			if store.lastScan != nil || len(store.scans) != 0 {
				t.Fatalf("invalid trigger type persisted Scan: %+v", store.scans)
			}
		})
	}
}

// completeScanCreateConfigurationForTest models the Profile round trip that
// production clients perform before submitting a Scan create request.
func completeScanCreateConfigurationForTest(t *testing.T, definitions map[string]engineexecution.ExecutionDefinition) map[string]any {
	t.Helper()
	steps := make(map[string]any, len(definitions))
	for stepID, definition := range definitions {
		config, err := engineexecution.NormalizeAndValidateConfig(map[string]any{}, definition)
		if err != nil {
			t.Fatalf("materialize Profile config for %s: %v", stepID, err)
		}
		for _, section := range definition.ConfigSections {
			sectionValue, ok := config[section.ID].(map[string]any)
			if !ok {
				continue
			}
			for _, param := range section.Params {
				if param.Resource != nil && param.Resource.Kind == engineexecution.ConfigResourceKindWordlist {
					// Engine Definition defaults are deployment-independent filenames;
					// a persisted Profile round trip must contain a canonical Catalog name.
					sectionValue[param.Key] = testCanonicalWordlistResource
				}
			}
		}
		steps[stepID] = map[string]any{"enabled": true, "engineConfig": config}
	}
	return map[string]any{"steps": steps}
}

func TestCreateBatchRejectsTypedRefWorkflowBoundary(t *testing.T) {
	workflowReader, enginePackageReader := scanCreateReaderStubs()
	service := NewScanCreateService(scanCreateStoreStub{}, func(context.Context, int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
	}, nil, workflowReader, enginePackageReader)

	_, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "engine.lunafox.subdomain_discovery", map[string]any{"steps": map[string]any{}}))
	if !errors.Is(err, ErrCreateInvalidScanWorkflow) {
		t.Fatalf("expected invalid scan workflow error, got %v", err)
	}
}

func TestCreateBatchPropagatesCanceledOperationContext(t *testing.T) {
	workflowReader := &cancelAwareScanCreateWorkflowReader{}
	service := NewScanCreateService(
		scanCreateStoreStub{},
		func(context.Context, int) (*TargetRef, error) { return nil, errors.New("target lookup must not run") },
		nil,
		workflowReader,
		scanCreateEnginePackageReaderStub{},
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.CreateBatch(ctx, singleTargetBatchCreateInput(1, "scanWorkflows/default", map[string]any{}))
	if !errors.Is(err, context.Canceled) || !workflowReader.called {
		t.Fatalf("CreateBatch() error = %v called=%t, want propagated cancellation", err, workflowReader.called)
	}
}

func TestCreateBatchPersistsResolvedExecutionPlan(t *testing.T) {
	store := &scanCreateStoreCaptureStub{}
	workflowReader, enginePackageReader := scanCreateReaderStubs()
	service := mustConfigureScanCreatePlanTask(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
	}, nil, workflowReader, enginePackageReader), enginePackageReader)

	result, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "scanWorkflows/default", map[string]any{
		"steps": map[string]any{
			"subdomain_discovery": map[string]any{
				"enabled":      true,
				"engineConfig": map[string]any{"recon": map[string]any{"enabled": true, "timeout": 3600, "threads": 10}},
			},
		},
	}))
	if err != nil {
		t.Fatalf("CreateBatch returned error: %v", err)
	}
	if result == nil || len(result.Scans) != 1 || store.lastScan == nil {
		t.Fatal("expected scan to be persisted")
	}
	if store.lastScan.ScanWorkflowID != "subdomain_discovery" {
		t.Fatalf("expected single scan workflow id persisted, got %+v", store.lastScan)
	}
	if len(store.lastScan.ScanTasks) != 1 {
		t.Fatalf("expected one scan task execution for one workflow step, got %+v", store.lastScan.ScanTasks)
	}
	scanTask := store.lastScan.ScanTasks[0]
	if scanTask.StageID != "discovery" || scanTask.StepID != "subdomain_discovery" || scanTask.EngineID != "engine.lunafox.subdomain_discovery" {
		t.Fatalf("unexpected scan task execution model: %+v", scanTask)
	}
	if scanTask.TaskExecutionConfig != nil || scanTask.EngineConfig != nil {
		t.Fatalf("v2 task must not persist legacy execution/config projections: %+v", scanTask)
	}
	plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(scanTask.ResolvedExecutionPlan)
	if err != nil {
		t.Fatalf("expected saved ResolvedExecutionPlan: %v", err)
	}
	if plan.GetExecution() != "executions/scan-1-task-101" || plan.GetTask() != "scans/1/tasks/101" {
		t.Fatalf("unexpected saved plan identity: %#v", plan)
	}
	if plan.GetTarget().GetResource() != "targets/1" || plan.GetTarget().GetType().String() != "TARGET_TYPE_DOMAIN" || plan.GetTarget().GetValue() != "example.com" {
		t.Fatalf("unexpected saved plan target: %#v", plan.GetTarget())
	}
	if got := plan.GetWorkflowStep(); got.GetScan() != "scans/1" || got.GetWorkflow() != "scanWorkflows/subdomain_discovery" || got.GetStageId() != "discovery" || got.GetStepId() != "subdomain_discovery" {
		t.Fatalf("unexpected saved workflow scope: %#v", got)
	}
	sections := plan.GetConfig().GetSections()
	if len(sections) != 1 || sections[0].GetSectionId() != "recon" || !sections[0].GetEnabled() || len(sections[0].GetParams()) != 2 {
		t.Fatalf("unexpected saved default config sections: %#v", sections)
	}
	values := map[string]int{}
	for _, param := range sections[0].GetParams() {
		values[param.GetKey()] = int(param.GetValue().GetIntegerValue())
	}
	if values["timeout"] != 3600 || values["threads"] != 10 {
		t.Fatalf("unexpected saved default config values: %#v", values)
	}
}

func TestCreateBatchFreezesSavedPlanAgainstLaterConfigurationMutation(t *testing.T) {
	store := &scanCreateStoreCaptureStub{}
	workflowReader, enginePackageReader := scanCreateReaderStubs()
	service := mustConfigureScanCreatePlanTask(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
	}, nil, workflowReader, enginePackageReader), enginePackageReader)
	configuration := map[string]any{"steps": map[string]any{
		"subdomain_discovery": map[string]any{
			"enabled":      true,
			"engineConfig": map[string]any{"recon": map[string]any{"enabled": true, "timeout": 3600, "threads": 10}},
		},
	}}
	if _, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "scanWorkflows/default", configuration)); err != nil {
		t.Fatal(err)
	}
	configuration["steps"].(map[string]any)["subdomain_discovery"].(map[string]any)["engineConfig"].(map[string]any)["recon"].(map[string]any)["timeout"] = 1
	plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(store.lastScan.ScanTasks[0].ResolvedExecutionPlan)
	if err != nil {
		t.Fatal(err)
	}
	for _, parameter := range plan.GetConfig().GetSections()[0].GetParams() {
		if parameter.GetKey() == "timeout" && parameter.GetValue().GetIntegerValue() != 3600 {
			t.Fatalf("saved plan configuration changed after create: %#v", plan.GetConfig())
		}
	}
}

func TestCreateBatchKeepsEarlierSavedPlanWhenWorkflowChanges(t *testing.T) {
	store := &scanCreateStoreCaptureStub{}
	definition := scanCreateTestEngineDefinition("engine.lunafox.subdomain_discovery", []string{engineexecution.TargetTypeDomain})
	workflowReader := &scanCreateWorkflowReaderStub{manifest: ScanCreateWorkflowManifest{
		ScanWorkflowID: "default",
		Stages: []ScanCreateWorkflowStage{{StageID: "discovery", Steps: []ScanCreateWorkflowStep{{
			StepID: "discover", EngineID: definition.EngineID,
		}}}},
	}}
	engineReader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		definition.EngineID: {Package: scanCreateTestPlanTaskPackage(definition)},
	}}
	service := mustConfigureScanCreatePlanTask(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
	}, nil, workflowReader, engineReader), engineReader)
	firstConfig := completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{"discover": definition.Execution})
	if _, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "scanWorkflows/default", firstConfig)); err != nil {
		t.Fatal(err)
	}
	workflowReader.manifest.Stages[0].StageID = "updated"
	workflowReader.manifest.Stages[0].Steps[0].StepID = "updated_step"
	secondConfig := completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{"updated_step": definition.Execution})
	if _, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "scanWorkflows/default", secondConfig)); err != nil {
		t.Fatal(err)
	}
	firstPlan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(store.scans[0].ScanTasks[0].ResolvedExecutionPlan)
	if err != nil {
		t.Fatal(err)
	}
	secondPlan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(store.scans[1].ScanTasks[0].ResolvedExecutionPlan)
	if err != nil {
		t.Fatal(err)
	}
	if firstPlan.GetWorkflowStep().GetStageId() != "discovery" || firstPlan.GetWorkflowStep().GetStepId() != "discover" {
		t.Fatalf("earlier plan changed with workflow: %#v", firstPlan.GetWorkflowStep())
	}
	if secondPlan.GetWorkflowStep().GetStageId() != "updated" || secondPlan.GetWorkflowStep().GetStepId() != "updated_step" {
		t.Fatalf("later plan did not use current workflow: %#v", secondPlan.GetWorkflowStep())
	}
}

func TestCreateBatchKeepsEarlierSavedPlanWhenEnginePackageChanges(t *testing.T) {
	store := &scanCreateStoreCaptureStub{}
	workflowReader, _ := scanCreateReaderStubs()
	definition := scanCreateTestEngineDefinition("engine.lunafox.subdomain_discovery", []string{engineexecution.TargetTypeDomain})
	engineReader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		definition.EngineID: {Package: scanCreateTestPlanTaskPackage(definition)},
	}}
	service := mustConfigureScanCreatePlanTask(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
	}, nil, workflowReader, engineReader), engineReader)
	configuration := completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{"subdomain_discovery": definition.Execution})
	if _, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "scanWorkflows/default", configuration)); err != nil {
		t.Fatal(err)
	}
	upgraded := scanCreateTestPlanTaskPackage(definition)
	upgraded.Identity.PackageDigest = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	engineReader.enginePackages[definition.EngineID] = ScanCreateEnginePackage{Package: upgraded}
	if _, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "scanWorkflows/default", configuration)); err != nil {
		t.Fatal(err)
	}
	firstPlan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(store.scans[0].ScanTasks[0].ResolvedExecutionPlan)
	if err != nil {
		t.Fatal(err)
	}
	secondPlan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(store.scans[1].ScanTasks[0].ResolvedExecutionPlan)
	if err != nil {
		t.Fatal(err)
	}
	if firstPlan.GetEngineRelease().GetPackageDigest() != planTaskPackageDigest {
		t.Fatalf("earlier plan package changed: %q", firstPlan.GetEngineRelease().GetPackageDigest())
	}
	if secondPlan.GetEngineRelease().GetPackageDigest() != upgraded.Identity.PackageDigest {
		t.Fatalf("later plan package digest = %q", secondPlan.GetEngineRelease().GetPackageDigest())
	}
}

func TestCreateBatchRejectsUnavailableWorkflowEngineBeforePersistence(t *testing.T) {
	store := &scanCreateStoreCaptureStub{}
	workflowReader, _ := scanCreateReaderStubs()
	engineReader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{}}
	service := mustConfigureScanCreatePlanTask(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
	}, nil, workflowReader, engineReader), engineReader)

	_, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "scanWorkflows/default", map[string]any{
		"steps": map[string]any{
			"subdomain_discovery": map[string]any{"enabled": true, "engineConfig": map[string]any{}},
		},
	}))
	if !errors.Is(err, ErrCreateScanWorkflowEngineUnavailable) {
		t.Fatalf("expected unavailable workflow Engine error, got %v", err)
	}
	if store.lastScan != nil || len(store.scans) != 0 {
		t.Fatalf("unavailable Engine must not persist a Scan: %+v", store.scans)
	}
}

func TestCreateBatchRejectsAllUserDisabledWorkflowBeforePlanning(t *testing.T) {
	workflowReader, _ := scanCreateReaderStubs()
	service := NewScanCreateService(&scanCreateStoreCaptureStub{}, nil, nil, workflowReader, nil)

	_, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "scanWorkflows/default", map[string]any{
		"steps": map[string]any{
			"subdomain_discovery": map[string]any{"enabled": false},
		},
	}))
	if !errors.Is(err, ErrCreateNoScanWorkflows) {
		t.Fatalf("expected all-disabled workflow rejection, got %v", err)
	}
}

func TestCreateBatchDisabledStepUsesRegistrationOnlyAndPersistsSkippedTask(t *testing.T) {
	store := &scanCreateStoreCaptureStub{}
	definition := scanCreateTestEngineDefinition("engine.lunafox.enabled", []string{engineexecution.TargetTypeDomain})
	manifest := ScanCreateWorkflowManifest{
		ScanWorkflowID: "default",
		Stages: []ScanCreateWorkflowStage{
			{StageID: "discovery", Steps: []ScanCreateWorkflowStep{{StepID: "disabled", EngineID: "engine.lunafox.disabled"}}},
			{StageID: "execution", Steps: []ScanCreateWorkflowStep{{StepID: "enabled", EngineID: definition.EngineID}}},
		},
	}
	packageReader := &selectiveScanCreatePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		definition.EngineID: {Package: scanCreateTestPlanTaskPackage(definition)},
	}}
	compilerPackages := &planTaskPackageReaderStub{packageValue: scanCreateTestPlanTaskPackage(definition)}
	compiler, err := NewPlanTaskCompiler(compilerPackages, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler failed: %v", err)
	}
	registrationReader := &scanCreateRegistrationReaderStub{exists: map[string]bool{
		"engine.lunafox.disabled": true,
		definition.EngineID:       true,
	}}
	service := NewScanCreateService(
		store,
		func(context.Context, int) (*TargetRef, error) {
			now := time.Now().UTC()
			return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
		},
		nil,
		scanCreateWorkflowReaderStub{manifest: manifest},
		packageReader,
	).WithEngineRegistrationReader(registrationReader)
	if err := service.ConfigurePlanTask(compiler, FixedPlanTaskLimitsProvider(time.Hour)); err != nil {
		t.Fatalf("ConfigurePlanTask failed: %v", err)
	}

	result, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "scanWorkflows/default", map[string]any{
		"steps": map[string]any{
			"disabled": map[string]any{"enabled": false},
			"enabled":  map[string]any{"enabled": true, "engineConfig": map[string]any{"recon": map[string]any{"enabled": true, "timeout": 3600, "threads": 10}}},
		},
	}))
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if result == nil || len(result.Scans) != 1 || store.lastScan == nil || len(store.lastScan.ScanTasks) != 2 {
		t.Fatalf("expected one scan with two frozen Step tasks: result=%#v scan=%#v", result, store.lastScan)
	}
	if len(registrationReader.calls) != 2 || registrationReader.calls[0] != "engine.lunafox.disabled" || registrationReader.calls[1] != definition.EngineID {
		t.Fatalf("registration lookup calls = %#v, want every topology Step", registrationReader.calls)
	}
	if len(packageReader.loadCalls) != 1 || packageReader.loadCalls[0] != definition.EngineID {
		t.Fatalf("exact package calls = %#v, want enabled engine only", packageReader.loadCalls)
	}
	if len(compilerPackages.requests) != 1 {
		t.Fatalf("PlanTask exact package calls = %d, want one enabled Step call", len(compilerPackages.requests))
	}
	disabled := store.lastScan.ScanTasks[0]
	if disabled.Status != CreateTaskStatusSkipped || disabled.SkipReason != "user_disabled" || len(disabled.ResolvedExecutionPlan) != 0 || disabled.EngineConfig != nil || disabled.TaskExecutionConfig != nil {
		t.Fatalf("unexpected disabled task projection: %+v", disabled)
	}
	if store.lastScan.ScanTasks[1].Status != CreateTaskStatusPending || len(store.lastScan.ScanTasks[1].ResolvedExecutionPlan) == 0 {
		t.Fatalf("enabled downstream task was not executable: %+v", store.lastScan.ScanTasks[1])
	}
}

func TestCreateBatchPersistsSkippedStepTaskForUnsupportedTargetType(t *testing.T) {
	store := &scanCreateStoreCaptureStub{}
	workflowReader, enginePackageReader := scanCreateReaderStubs()
	service := mustConfigureScanCreatePlanTask(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{ID: 1, Name: "192.0.2.10", Type: "ip", CreatedAt: now}, nil
	}, nil, workflowReader, enginePackageReader), enginePackageReader)

	result, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "scanWorkflows/default", map[string]any{
		"steps": map[string]any{
			"subdomain_discovery": map[string]any{
				"enabled":      true,
				"engineConfig": map[string]any{"recon": map[string]any{"enabled": true, "timeout": 3600, "threads": 10}},
			},
		},
	}))
	if err != nil {
		t.Fatalf("CreateBatch returned error: %v", err)
	}
	if result == nil || len(result.Scans) != 1 || store.lastScan == nil {
		t.Fatal("expected scan to be persisted")
	}
	if store.lastScan.Status != CreateScanStatusSucceeded {
		t.Fatalf("expected all-skipped scan to be persisted as succeeded, got %+v", store.lastScan)
	}
	if len(store.lastScan.ScanTasks) != 1 {
		t.Fatalf("expected one skipped step task, got %+v", store.lastScan.ScanTasks)
	}
	task := store.lastScan.ScanTasks[0]
	if task.Status != CreateTaskStatusSkipped {
		t.Fatalf("expected skipped step task, got %+v", task)
	}
	if len(task.ResolvedExecutionPlan) != 0 || task.TaskExecutionConfig != nil || task.EngineConfig != nil {
		t.Fatalf("planning-time skipped task must not carry a v2 plan or legacy projections: %+v", task)
	}
	if task.SkipReason != "target_not_applicable" {
		t.Fatalf("unexpected skip reason: %+v", task)
	}
}

func TestCreateQuickPersistsOnePendingQuickScanPerResolvedTargetAndReturnsResolutionErrors(t *testing.T) {
	store := &scanCreateStoreCaptureStub{}
	workflowReader, enginePackageReader := scanCreateReaderStubs()
	now := time.Date(2026, 6, 18, 10, 30, 0, 0, time.UTC)
	var resolvedNames []string
	service := mustConfigureScanCreatePlanTask(t, NewScanCreateService(
		store,
		nil,
		func(_ context.Context, names []string) (*QuickTargetResolution, error) {
			resolvedNames = append([]string(nil), names...)
			return &QuickTargetResolution{
				Targets: []TargetRef{
					{ID: 7, Name: "example.com", Type: "domain", CreatedAt: now},
					{ID: 8, Name: "api.example.com", Type: "domain", CreatedAt: now},
				},
				TargetStats: QuickTargetStats{Created: 2, Skipped: 1, Failed: 1},
				Errors:      []QuickTargetError{{Input: "bad target", Error: "invalid target"}},
			}, nil
		},
		workflowReader,
		enginePackageReader,
	), enginePackageReader)

	result, err := service.CreateQuick(context.Background(), &CreateQuickInput{
		Targets:      []string{"example.com", "api.example.com", "bad target"},
		ScanWorkflow: "scanWorkflows/default",
		Configuration: map[string]any{
			"steps": map[string]any{
				"subdomain_discovery": map[string]any{
					"enabled":      true,
					"engineConfig": map[string]any{"recon": map[string]any{"enabled": true, "timeout": 3600, "threads": 10}},
				},
			},
		},
		InputSource: InputSourceScanSnapshot,
		TriggerType: ScanTriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("CreateQuick returned error: %v", err)
	}
	if len(resolvedNames) != 3 || resolvedNames[0] != "example.com" || resolvedNames[1] != "api.example.com" || resolvedNames[2] != "bad target" {
		t.Fatalf("expected raw quick-scan targets to be resolved, got %+v", resolvedNames)
	}
	if result.TargetStats != (QuickTargetStats{Created: 2, Skipped: 1, Failed: 1}) {
		t.Fatalf("unexpected quick target stats: %+v", result.TargetStats)
	}
	if len(result.Errors) != 1 || result.Errors[0].Input != "bad target" {
		t.Fatalf("expected resolution error to be preserved, got %+v", result.Errors)
	}
	if len(result.Scans) != 2 || len(store.scans) != 2 {
		t.Fatalf("expected two persisted scans, got result=%d scans=%d", len(result.Scans), len(store.scans))
	}
	for index, scan := range result.Scans {
		if scan.Status != CreateScanStatusPending || scan.TriggerType != ScanTriggerTypeManual {
			t.Fatalf("expected pending manual scan at index %d, got %+v", index, scan)
		}
		if scan.ScanWorkflowID != "subdomain_discovery" {
			t.Fatalf("expected workflow default at index %d, got %q", index, scan.ScanWorkflowID)
		}
		if scan.Target == nil || scan.Target.ID != 7+index {
			t.Fatalf("expected result scan target to be populated at index %d, got %+v", index, scan.Target)
		}
		persisted := store.scans[index]
		if persisted.TargetID != 7+index || persisted.Status != CreateScanStatusPending || persisted.TriggerType != ScanTriggerTypeManual {
			t.Fatalf("unexpected persisted scan at index %d: %+v", index, persisted)
		}
		if len(persisted.ScanTasks) != 1 {
			t.Fatalf("expected quick sca scan task at index %d, got %+v", index, persisted.ScanTasks)
		}
		scanTask := persisted.ScanTasks[0]
		if scanTask.Status != CreateTaskStatusPending {
			t.Fatalf("expected one pending scan task at index %d, got %+v", index, scanTask)
		}
	}
}

func TestCreateBatchExpandsOrganizationTargetsAndReportsEmptyOrganizations(t *testing.T) {
	store := &scanCreateStoreCaptureStub{}
	workflowReader, enginePackageReader := scanCreateReaderStubs()
	now := time.Now().UTC()
	service := mustConfigureScanCreatePlanTask(t, NewScanCreateService(
		store,
		func(_ context.Context, id int) (*TargetRef, error) {
			return &TargetRef{ID: id, Name: "target.example.com", Type: "domain", CreatedAt: now}, nil
		},
		nil,
		workflowReader,
		enginePackageReader,
		func(_ context.Context, organizationID int) ([]TargetRef, error) {
			if organizationID == 5 {
				return []TargetRef{
					{ID: 8, Name: "org-one.example.com", Type: "domain", CreatedAt: now},
					{ID: 9, Name: "org-two.example.com", Type: "domain", CreatedAt: now},
				}, nil
			}
			return []TargetRef{}, nil
		},
	), enginePackageReader)

	result, err := service.CreateBatch(context.Background(), &CreateBatchInput{
		Requests: []CreateBatchItem{
			{TargetID: 7},
			{OrganizationID: 5},
			{OrganizationID: 6},
		},
		ScanWorkflow: "scanWorkflows/subdomain_discovery",
		Configuration: map[string]any{"steps": map[string]any{
			"subdomain_discovery": map[string]any{
				"enabled":      true,
				"engineConfig": map[string]any{"recon": map[string]any{"enabled": true, "timeout": 3600, "threads": 10}},
			},
		}},
		InputSource: InputSourceScanSnapshot,
		TriggerType: ScanTriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("CreateBatch returned error: %v", err)
	}
	if result.CreatedCount != 3 || len(result.Scans) != 3 || len(store.scans) != 3 {
		t.Fatalf("expected three created scans, result=%+v stored=%d", result, len(store.scans))
	}
	if got := []int{store.scans[0].TargetID, store.scans[1].TargetID, store.scans[2].TargetID}; got[0] != 7 || got[1] != 8 || got[2] != 9 {
		t.Fatalf("unexpected created target order: %+v", got)
	}
	if len(result.Skipped) != 1 || result.Skipped[0].Index != 2 || result.Skipped[0].OrganizationID != 6 {
		t.Fatalf("expected empty organization scope to be skipped, got %+v", result.Skipped)
	}
}

func TestCreateBatchPreservesCommittedOrganizationChildWhenLaterChildFails(t *testing.T) {
	persistenceErr := errors.New("second child persistence unavailable")
	store := &scanCreateStoreFailOnCallStub{failOnCall: 2, err: persistenceErr}
	workflowReader, enginePackageReader := scanCreateReaderStubs()
	now := time.Now().UTC()
	createService := mustConfigureScanCreatePlanTask(t, NewScanCreateService(
		store,
		func(_ context.Context, id int) (*TargetRef, error) {
			return &TargetRef{ID: id, Name: "target.example.com", Type: "domain", CreatedAt: now}, nil
		},
		nil,
		workflowReader,
		enginePackageReader,
		func(context.Context, int) ([]TargetRef, error) {
			return []TargetRef{
				{ID: 8, Name: "first.example.com", Type: "domain", CreatedAt: now},
				{ID: 9, Name: "second.example.com", Type: "domain", CreatedAt: now},
			}, nil
		},
	), enginePackageReader)
	facade := NewScanFacade(nil, nil, nil, nil, nil, createService)

	result, err := facade.CreateBatch(context.Background(), &CreateBatchRequest{
		Requests:     []CreateBatchItem{{OrganizationID: 5}},
		ScanWorkflow: "scanWorkflows/subdomain_discovery",
		Configuration: map[string]any{"steps": map[string]any{
			"subdomain_discovery": map[string]any{
				"enabled":      true,
				"engineConfig": map[string]any{"recon": map[string]any{"enabled": true, "timeout": 3600, "threads": 10}},
			},
		}},
		InputSource: InputSourceScanSnapshot,
		TriggerType: ScanTriggerTypeManual,
	})
	if !errors.Is(err, persistenceErr) {
		t.Fatalf("CreateBatch() error = %v, want committed-child failure", err)
	}
	if result == nil || result.CreatedCount != 1 || len(result.Scans) != 1 || result.Scans[0].TargetID != 8 {
		t.Fatalf("CreateBatch() partial result = %+v, want first committed child", result)
	}
	if len(store.scans) != 1 || store.scans[0].TargetID != 8 || store.calls != 2 {
		t.Fatalf("persisted organization children = %+v calls=%d", store.scans, store.calls)
	}
}

func TestCreateBatchContinuesAfterFinalTargetFenceRejectsOneItem(t *testing.T) {
	store := &scanCreateStoreFailOnCallStub{failOnCall: 1, err: gorm.ErrRecordNotFound}
	workflowReader, enginePackageReader := scanCreateReaderStubs()
	now := time.Now().UTC()
	service := mustConfigureScanCreatePlanTask(t, NewScanCreateService(
		store,
		func(_ context.Context, id int) (*TargetRef, error) {
			return &TargetRef{ID: id, Name: "target.example", Type: "domain", CreatedAt: now}, nil
		},
		nil,
		workflowReader,
		enginePackageReader,
	), enginePackageReader)

	result, err := service.CreateBatch(context.Background(), &CreateBatchInput{
		Requests:     []CreateBatchItem{{TargetID: 7}, {TargetID: 8}},
		ScanWorkflow: "scanWorkflows/subdomain_discovery",
		Configuration: map[string]any{"steps": map[string]any{
			"subdomain_discovery": map[string]any{
				"enabled":      true,
				"engineConfig": map[string]any{"recon": map[string]any{"enabled": true, "timeout": 3600, "threads": 10}},
			},
		}},
		InputSource: InputSourceScanSnapshot,
		TriggerType: ScanTriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("CreateBatch() error = %v, want per-item target failure", err)
	}
	if result.CreatedCount != 1 || len(result.Scans) != 1 || result.Scans[0].TargetID != 8 {
		t.Fatalf("created scans = %#v, want only Target 8", result.Scans)
	}
	if len(result.Failed) != 1 || result.Failed[0].TargetID != 7 || result.Failed[0].Reason != "TARGET_NOT_FOUND" {
		t.Fatalf("failed outcomes = %#v, want Target 7 unavailable", result.Failed)
	}
	if len(store.scans) != 1 || store.scans[0].TargetID != 8 {
		t.Fatalf("persisted scans = %#v, want only Target 8", store.scans)
	}
}

func TestCreateBatchReevaluatesTriggerTimeDependenciesOnEveryAttempt(t *testing.T) {
	tests := []struct {
		name              string
		organizationScope bool
		pinnedAgent       bool
	}{
		{name: "workflow"},
		{name: "engine"},
		{name: "target"},
		{name: "organization", organizationScope: true},
		{name: "configuration resource"},
		{name: "pinned agent", pinnedAgent: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &triggerTimeDependencyState{unavailable: test.name}
			definition := testPlanTaskDefinition(nil, nil, true)
			exactPackage := planTaskExactPackage(definition)
			manifest := ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{
				StageID: "scan", Steps: []ScanCreateWorkflowStep{{StepID: "port_scan", EngineID: definition.EngineID}},
			}}}
			store := &scanCreateStoreCaptureStub{}
			engineReader := triggerTimeEnginePackageReader{state: state, enginePackage: scanCreatePackageFromPlanPackage(exactPackage)}
			service := NewScanCreateService(
				store,
				func(_ context.Context, id int) (*TargetRef, error) {
					if state.unavailable == "target" {
						return nil, errors.New("current target unavailable")
					}
					return &TargetRef{ID: id, Name: "target.example.com", Type: "domain", CreatedAt: time.Now().UTC()}, nil
				},
				nil,
				triggerTimeWorkflowReader{state: state, manifest: manifest},
				engineReader,
				func(context.Context, int) ([]TargetRef, error) {
					if state.unavailable == "organization" {
						return nil, errors.New("current organization unavailable")
					}
					return []TargetRef{{ID: 18, Name: "organization.example.com", Type: "domain", CreatedAt: time.Now().UTC()}}, nil
				},
			).WithAgentLookup(triggerTimeAgentLookup{state: state})
			compiler, err := NewPlanTaskCompiler(
				&planTaskPackageReaderStub{packageValue: exactPackage},
				triggerTimeResourceResolver{state: state},
			)
			if err != nil {
				t.Fatalf("NewPlanTaskCompiler() error = %v", err)
			}
			mustConfigurePlanTaskForTest(t, service, compiler, time.Minute)
			facade := NewScanFacade(nil, nil, nil, nil, nil, service)
			request := &CreateBatchRequest{
				Requests:      []CreateBatchItem{{TargetID: 17}},
				ScanWorkflow:  "scanWorkflows/default",
				Configuration: completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{"port_scan": definition.Execution}),
				InputSource:   InputSourceScanSnapshot,
				TriggerType:   ScanTriggerTypeManual,
			}
			if test.organizationScope {
				request.Requests = []CreateBatchItem{{OrganizationID: 5}}
			}
			if test.pinnedAgent {
				agentID := 42
				request.AgentID = &agentID
			}

			failedResult, failedErr := facade.CreateBatch(context.Background(), request)
			if failedErr == nil && (failedResult == nil || len(failedResult.Failed) == 0) {
				t.Fatalf("unavailable %s did not fail closed: result=%+v err=%v", test.name, failedResult, failedErr)
			}
			if failedResult != nil && failedResult.CreatedCount != 0 {
				t.Fatalf("unavailable %s used a fallback: %+v", test.name, failedResult)
			}
			if len(store.scans) != 0 {
				t.Fatalf("unavailable %s committed a Scan: %+v", test.name, store.scans)
			}

			state.unavailable = ""
			recovered, err := facade.CreateBatch(context.Background(), request)
			if err != nil || recovered == nil || recovered.CreatedCount != 1 || len(store.scans) != 1 {
				t.Fatalf("repaired %s did not recover on a later attempt: result=%+v err=%v scans=%+v", test.name, recovered, err, store.scans)
			}
			wantTargetID := 17
			if test.organizationScope {
				wantTargetID = 18
			}
			if recovered.Scans[0].TargetID != wantTargetID {
				t.Fatalf("repaired %s changed scope: %+v", test.name, recovered.Scans[0])
			}
			if test.pinnedAgent && (recovered.Scans[0].AgentID == nil || *recovered.Scans[0].AgentID != 42 || recovered.Scans[0].AssignmentMode != "pinned") {
				t.Fatalf("repaired pinned Agent used fallback assignment: %+v", recovered.Scans[0])
			}
		})
	}
}

func TestCreateBatchRejectsInvalidDynamicEngineConfigAgainstDefinition(t *testing.T) {
	store := &scanCreateStoreCaptureStub{}
	workflowReader, _ := scanCreateReaderStubs()
	definition := scanCreateTestEngineDefinition("engine.lunafox.subdomain_discovery", []string{engineexecution.TargetTypeDomain})
	enginePackageReader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		"engine.lunafox.subdomain_discovery": {
			Package: scanCreateTestPlanTaskPackage(definition),
		},
	}}
	service := mustConfigureScanCreatePlanTask(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		now := time.Now().UTC()
		return &TargetRef{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}, nil
	}, nil, workflowReader, enginePackageReader), enginePackageReader)

	_, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(1, "scanWorkflows/default", map[string]any{
		"steps": map[string]any{
			"subdomain_discovery": map[string]any{
				"enabled": true,
				"engineConfig": map[string]any{
					"recon": map[string]any{"enabled": true, "timeout": 3600, "threads": "fast"},
				},
			},
		},
	}))
	if !isPlanTaskErrorKind(err, PlanTaskConfigError) {
		t.Fatalf("expected PlanTask config validation error, got %v", err)
	}
}
