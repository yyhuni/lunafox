package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"google.golang.org/protobuf/proto"
)

type plannedScanCreateStoreStub struct {
	committed           *CreateScan
	failAfterFinalizers int
}

func (stub *plannedScanCreateStoreStub) CreateWithScanTasksAndPlans(_ context.Context, scan *CreateScan, finalize ScanCreateTaskFinalizer) error {
	if scan == nil || finalize == nil {
		return fmt.Errorf("planned create arguments are required")
	}
	scan.ID = 23
	for index := range scan.ScanTasks {
		task := &scan.ScanTasks[index]
		task.ID = 31 + index
		if err := finalize(scan.ID, task.ID, task); err != nil {
			return err
		}
		if stub.failAfterFinalizers > 0 && index+1 == stub.failAfterFinalizers {
			return fmt.Errorf("save failed")
		}
	}
	clone := *scan
	clone.ScanTasks = append([]CreateScanTask(nil), scan.ScanTasks...)
	stub.committed = &clone
	return nil
}

func TestScanCreatePlanTaskPersistsExecutablePlanForBlockedTasks(t *testing.T) {
	definition := testPlanTaskDefinition([]string{engineexecution.InputSubdomains}, nil, false)
	definition.EngineID = "engine.lunafox.port_scan"
	definition.Execution.SupportedTargetTypes = []string{engineexecution.TargetTypeDomain, engineexecution.TargetTypeIP, engineexecution.TargetTypeCIDR}
	definition.Execution.ConfigSections[0].Params = append(definition.Execution.ConfigSections[0].Params, engineexecution.ParamDefinition{
		Key: "max-execution-duration", Type: engineexecution.ParamTypeInteger, Default: 60, Minimum: intPointer(1),
	})
	exact := planTaskExactPackage(definition)
	packages := &planTaskPackageReaderStub{packageValue: exact}
	compiler, _ := NewPlanTaskCompiler(packages, nil)
	store := &plannedScanCreateStoreStub{}
	manifest := ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{
		{StageID: "ports", Steps: []ScanCreateWorkflowStep{{StepID: "port_a", EngineID: "engine.lunafox.port_scan"}}},
		{StageID: "later", Steps: []ScanCreateWorkflowStep{{StepID: "port_b", EngineID: "engine.lunafox.port_scan"}}},
	}}
	reader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		"engine.lunafox.port_scan": scanCreatePackageFromPlanPackage(exact),
	}}
	service := mustConfigurePlanTaskForTest(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		return &TargetRef{ID: 17, Name: "example.com", Type: "domain", CreatedAt: time.Now().UTC()}, nil
	}, nil, scanCreateWorkflowReaderStub{manifest: manifest}, reader), compiler, 3*time.Second)

	configuration := completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{
		"port_a": definition.Execution,
		"port_b": definition.Execution,
	})
	configuration["steps"].(map[string]any)["port_a"].(map[string]any)["engineConfig"].(map[string]any)["scan"].(map[string]any)["max-execution-duration"] = 1
	result, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(17, "scanWorkflows/default", configuration))
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if result.CreatedCount != 1 || store.committed == nil {
		t.Fatalf("planned create was not committed: result=%#v store=%#v", result, store)
	}
	if len(store.committed.ScanTasks) != 2 || store.committed.ScanTasks[0].Status != CreateTaskStatusPending || store.committed.ScanTasks[1].Status != CreateTaskStatusBlocked {
		t.Fatalf("unexpected workflow task states: %#v", store.committed.ScanTasks)
	}
	if len(packages.requests) != len(store.committed.ScanTasks) {
		t.Fatalf("PlanTask package reads = %d, want one per workflow step (%d)", len(packages.requests), len(store.committed.ScanTasks))
	}
	for index := range store.committed.ScanTasks {
		task := store.committed.ScanTasks[index]
		if len(task.ResolvedExecutionPlan) == 0 {
			t.Fatalf("task %d has no saved plan", index)
		}
		plan := &agentexecutionv1.ResolvedEngineExecutionPlan{}
		if err := proto.Unmarshal(task.ResolvedExecutionPlan, plan); err != nil {
			t.Fatalf("decode task %d plan: %v", index, err)
		}
		if plan.GetTask() != fmt.Sprintf("scans/23/tasks/%d", 31+index) || plan.GetTarget().GetValue() != "example.com" {
			t.Fatalf("unexpected task %d plan: %#v", index, plan)
		}
		if plan.GetLimits().GetMaxExecutionDuration().AsDuration() != 3*time.Second {
			t.Fatalf("task %d max execution duration = %s, want injected 3s", index, plan.GetLimits().GetMaxExecutionDuration().AsDuration())
		}
		if index == 0 {
			foundConfigValue := false
			for _, section := range plan.GetConfig().GetSections() {
				for _, param := range section.GetParams() {
					if section.GetSectionId() == "scan" && param.GetKey() == "max-execution-duration" && param.GetValue().GetIntegerValue() == 1 {
						foundConfigValue = true
					}
				}
			}
			if !foundConfigValue {
				t.Fatalf("task config override did not reach Engine config independently of plan limits: %#v", plan.GetConfig().GetSections())
			}
		}
	}
}

func TestScanCreateRejectsLegacySubdomainDNSConfigBeforeTaskPersistence(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, false)
	definition.EngineID = "engine.lunafox.subdomain_discovery"
	definition.Execution.ConfigSections = []engineexecution.ConfigSectionDefinition{{
		ID:              "resolve",
		DefaultEnabled:  true,
		RequiredEnabled: true,
		Params:          definition.Execution.ConfigSections[0].Params,
	}}
	exact := planTaskExactPackage(definition)
	exact.Identity.EngineID = definition.EngineID
	compilerPackages := &planTaskPackageReaderStub{packageValue: exact}
	compiler, err := NewPlanTaskCompiler(compilerPackages, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}
	store := &plannedScanCreateStoreStub{}
	manifest := ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{
		StageID: "discovery",
		Steps:   []ScanCreateWorkflowStep{{StepID: "subdomain_discovery", EngineID: definition.EngineID}},
	}}}
	reader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		definition.EngineID: scanCreatePackageFromPlanPackage(exact),
	}}
	service := mustConfigurePlanTaskForTest(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		return &TargetRef{ID: 17, Name: "example.com", Type: "domain", CreatedAt: time.Now().UTC()}, nil
	}, nil, scanCreateWorkflowReaderStub{manifest: manifest}, reader), compiler, 3*time.Second)

	configuration := completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{
		"subdomain_discovery": definition.Execution,
	})
	engineConfig := configuration["steps"].(map[string]any)["subdomain_discovery"].(map[string]any)["engineConfig"].(map[string]any)
	engineConfig["dns"] = map[string]any{"enabled": false, "resolvers": "resolvers.txt"}

	_, err = service.CreateBatch(context.Background(), singleTargetBatchCreateInput(17, "scanWorkflows/default", configuration))
	if err == nil || !strings.Contains(err.Error(), `unknown config section "dns"`) {
		t.Fatalf("CreateBatch() error = %v, want legacy dns rejection", err)
	}
	if store.committed != nil {
		t.Fatalf("legacy dns configuration persisted Scan or task: %#v", store.committed)
	}
}

func TestScanCreatePlansFingerprintDetectionWhenItIsTheOnlyEnabledStep(t *testing.T) {
	fingerprint := fingerprintDetectionPlanDefinition()
	urlCollection := urlCollectionPlanDefinition()
	manifest := ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{
		StageID: "url_collection",
		Steps: []ScanCreateWorkflowStep{
			{StepID: "fingerprint_detection", EngineID: fingerprint.EngineID},
			{StepID: "url_collection", EngineID: urlCollection.EngineID},
		},
	}}}
	reader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		fingerprint.EngineID:   {Package: scanCreateTestPlanTaskPackage(fingerprint)},
		urlCollection.EngineID: {Package: scanCreateTestPlanTaskPackage(urlCollection)},
	}}
	registration := &scanCreateRegistrationReaderStub{exists: map[string]bool{fingerprint.EngineID: true, urlCollection.EngineID: true}}
	store := &plannedScanCreateStoreStub{}
	compiler, err := NewPlanTaskCompiler(reader, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}
	service := mustConfigurePlanTaskForTest(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		return &TargetRef{ID: 17, Name: "example.com", Type: "domain", CreatedAt: time.Now().UTC()}, nil
	}, nil, scanCreateWorkflowReaderStub{manifest: manifest}, reader), compiler, 3*time.Second).WithEngineRegistrationReader(registration)

	configuration := map[string]any{"steps": map[string]any{
		"fingerprint_detection": map[string]any{"enabled": true, "engineConfig": fingerprintDetectionConfig()},
		"url_collection":        map[string]any{"enabled": false},
	}}
	result, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(17, "scanWorkflows/default", configuration))
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if result.CreatedCount != 1 || store.committed == nil || len(store.committed.ScanTasks) != 2 {
		t.Fatalf("expected one planned scan with both topology tasks: result=%#v scan=%#v", result, store.committed)
	}
	fingerprintTask := store.committed.ScanTasks[0]
	if fingerprintTask.Status != CreateTaskStatusPending || fingerprintTask.StageID != "url_collection" || fingerprintTask.StepID != "fingerprint_detection" {
		t.Fatalf("fingerprint task = %#v", fingerprintTask)
	}
	plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(fingerprintTask.ResolvedExecutionPlan)
	if err != nil {
		t.Fatalf("decode fingerprint plan: %v", err)
	}
	if len(plan.GetPlatformResourceBindings()) != 1 || plan.GetPlatformResourceBindings()[0].GetResourceId() != engineexecution.PlatformResourceFingerprintLibraryFingerPrintHub {
		t.Fatalf("fingerprint plan did not preserve its FingerprintHub resource: %#v", plan)
	}
	disabled := store.committed.ScanTasks[1]
	if disabled.Status != CreateTaskStatusSkipped || disabled.SkipReason != "user_disabled" || len(disabled.ResolvedExecutionPlan) != 0 {
		t.Fatalf("disabled URL Collection task = %#v", disabled)
	}
}

func TestScanCreatePlansFingerprintDetectionAndURLCollectionFromIndependentWebsiteURLInputs(t *testing.T) {
	fingerprint := fingerprintDetectionPlanDefinition()
	urlCollection := urlCollectionPlanDefinition()
	manifest := ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{
		StageID: "url_collection",
		Steps: []ScanCreateWorkflowStep{
			{StepID: "fingerprint_detection", EngineID: fingerprint.EngineID},
			{StepID: "url_collection", EngineID: urlCollection.EngineID},
		},
	}}}
	reader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		fingerprint.EngineID:   {Package: scanCreateTestPlanTaskPackage(fingerprint)},
		urlCollection.EngineID: {Package: scanCreateTestPlanTaskPackage(urlCollection)},
	}}
	store := &plannedScanCreateStoreStub{}
	compiler, err := NewPlanTaskCompiler(reader, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}
	service := mustConfigurePlanTaskForTest(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		return &TargetRef{ID: 17, Name: "example.com", Type: "domain", CreatedAt: time.Now().UTC()}, nil
	}, nil, scanCreateWorkflowReaderStub{manifest: manifest}, reader), compiler, 3*time.Second)

	configuration := map[string]any{"steps": map[string]any{
		"fingerprint_detection": map[string]any{"enabled": true, "engineConfig": fingerprintDetectionConfig()},
		"url_collection":        map[string]any{"enabled": true, "engineConfig": urlCollectionConfig()},
	}}
	if _, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(17, "scanWorkflows/default", configuration)); err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if store.committed == nil || len(store.committed.ScanTasks) != 2 {
		t.Fatalf("expected independent same-stage tasks: %#v", store.committed)
	}
	first, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(store.committed.ScanTasks[0].ResolvedExecutionPlan)
	if err != nil {
		t.Fatalf("decode fingerprint plan: %v", err)
	}
	second, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(store.committed.ScanTasks[1].ResolvedExecutionPlan)
	if err != nil {
		t.Fatalf("decode URL Collection plan: %v", err)
	}
	if first.GetWorkflowStep().GetStageId() != "url_collection" || second.GetWorkflowStep().GetStageId() != "url_collection" || first.GetWorkflowStep().GetStepId() != "fingerprint_detection" || second.GetWorkflowStep().GetStepId() != "url_collection" {
		t.Fatalf("same-stage workflow projections = %#v %#v", first.GetWorkflowStep(), second.GetWorkflowStep())
	}
	for _, plan := range []*agentexecutionv1.ResolvedEngineExecutionPlan{first, second} {
		if plan.ProtoReflect().Descriptor().Fields().ByName("inputs") != nil {
			t.Fatal("saved plan unexpectedly exposes an Engine-specific input allow-set")
		}
	}
}

func fingerprintDetectionPlanDefinition() enginecontract.EngineDefinition {
	return enginecontract.EngineDefinition{
		ManifestVersion: enginecontract.SupportedRootManifestVersion,
		EngineID:        "engine.lunafox.fingerprint_detection",
		Publisher:       "lunafox",
		Execution: engineexecution.ExecutionDefinition{
			EngineAPIMajor:       2,
			SupportedTargetTypes: []string{engineexecution.TargetTypeDomain, engineexecution.TargetTypeIP, engineexecution.TargetTypeCIDR},
			ExecutionResources:   []string{engineexecution.PlatformResourceFingerprintLibraryFingerPrintHub},
			ConfigSections: []engineexecution.ConfigSectionDefinition{{
				ID: "observer_ward", DefaultEnabled: true,
				Params: []engineexecution.ParamDefinition{
					{Key: "timeout", Type: engineexecution.ParamTypeInteger, Default: 3600, Minimum: intPointer(60), Maximum: intPointer(604800)},
					{Key: "request-timeout", Type: engineexecution.ParamTypeInteger, Default: 10, Minimum: intPointer(1), Maximum: intPointer(120)},
					{Key: "threads", Type: engineexecution.ParamTypeInteger, Default: 25, Minimum: intPointer(1), Maximum: intPointer(200)},
				},
			}},
		},
	}
}

func urlCollectionPlanDefinition() enginecontract.EngineDefinition {
	return enginecontract.EngineDefinition{
		ManifestVersion: enginecontract.SupportedRootManifestVersion,
		EngineID:        "engine.lunafox.url_collection",
		Publisher:       "lunafox",
		Execution: engineexecution.ExecutionDefinition{
			EngineAPIMajor:       2,
			SupportedTargetTypes: []string{engineexecution.TargetTypeDomain, engineexecution.TargetTypeIP, engineexecution.TargetTypeCIDR},
			ConfigSections: []engineexecution.ConfigSectionDefinition{{
				ID: "collect", DefaultEnabled: true,
				Params: []engineexecution.ParamDefinition{{Key: "timeout", Type: engineexecution.ParamTypeInteger, Default: 60, Minimum: intPointer(1), Maximum: intPointer(3600)}},
			}},
		},
	}
}

func fingerprintDetectionConfig() map[string]any {
	return map[string]any{"observer_ward": map[string]any{"enabled": true, "timeout": 3600, "request-timeout": 10, "threads": 25}}
}

func urlCollectionConfig() map[string]any {
	return map[string]any{"collect": map[string]any{"enabled": true, "timeout": 60}}
}

func TestScanCreatePlanTaskPersistsUnsupportedAsSkippedWithoutPlan(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, false)
	definition.EngineID = "engine.lunafox.subdomain_discovery"
	exact := planTaskExactPackage(definition)
	exact.Identity.EngineID = "engine.lunafox.subdomain_discovery"
	exact.Definition.EngineID = "engine.lunafox.subdomain_discovery"
	exact.RuntimeImageRefs = []string{"docker.io/lunafox/lunafox-engine-runtime-subdomain-discovery@" + planTaskImageDigest}
	packages := &planTaskPackageReaderStub{packageValue: exact}
	compiler, _ := NewPlanTaskCompiler(packages, nil)
	store := &plannedScanCreateStoreStub{}
	manifest := ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{StageID: "discover", Steps: []ScanCreateWorkflowStep{{StepID: "subdomain_discovery", EngineID: exact.Identity.EngineID}}}}}
	reader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{exact.Identity.EngineID: scanCreatePackageFromPlanPackage(exact)}}
	service := mustConfigurePlanTaskForTest(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		return &TargetRef{ID: 17, Name: "192.0.2.10", Type: "ip", CreatedAt: time.Now().UTC()}, nil
	}, nil, scanCreateWorkflowReaderStub{manifest: manifest}, reader), compiler, 3*time.Second)

	result, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(17, "scanWorkflows/default", completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{
		"subdomain_discovery": definition.Execution,
	})))
	if err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if result.CreatedCount != 1 || store.committed == nil || store.committed.Status != CreateScanStatusSucceeded {
		t.Fatalf("all-skipped scan did not converge: %#v %#v", result, store.committed)
	}
	task := store.committed.ScanTasks[0]
	if task.Status != CreateTaskStatusSkipped || task.SkipReason == "" || len(task.ResolvedExecutionPlan) != 0 {
		t.Fatalf("unexpected skipped task: %#v", task)
	}
}

func TestScanCreatePlanTaskPinsResourceBindingInSavedPlan(t *testing.T) {
	definition := testPlanTaskDefinition([]string{engineexecution.InputSubdomains}, nil, true)
	exact := planTaskExactPackage(definition)
	packages := &planTaskPackageReaderStub{packageValue: exact}
	wordlists := &planTaskWordlistReaderStub{byResource: map[string]PlanTaskWordlist{
		testCanonicalWordlistResource: {Resource: testCanonicalWordlistResource, Basename: "dns.txt", SizeBytes: 8, LineCount: 2, SHA256: planTaskWordlistDigest},
	}}
	compiler, _ := NewPlanTaskCompiler(packages, wordlists)
	store := &plannedScanCreateStoreStub{}
	manifest := ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{
		StageID: "ports", Steps: []ScanCreateWorkflowStep{{StepID: "port_scan", EngineID: exact.Identity.EngineID}},
	}}}
	reader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		exact.Identity.EngineID: scanCreatePackageFromPlanPackage(exact),
	}}
	service := mustConfigurePlanTaskForTest(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		return &TargetRef{ID: 17, Name: "example.com", Type: "domain", CreatedAt: time.Now().UTC()}, nil
	}, nil, scanCreateWorkflowReaderStub{manifest: manifest}, reader), compiler, 3*time.Second)

	if _, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(17, "scanWorkflows/default", completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{
		"port_scan": definition.Execution,
	}))); err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if store.committed == nil || len(store.committed.ScanTasks) != 1 {
		t.Fatalf("planned scan was not committed: %#v", store.committed)
	}
	task := store.committed.ScanTasks[0]
	plan := &agentexecutionv1.ResolvedEngineExecutionPlan{}
	if err := proto.Unmarshal(task.ResolvedExecutionPlan, plan); err != nil {
		t.Fatalf("decode saved plan: %v", err)
	}
	bindings := plan.GetConfigResourceBindings()
	if len(bindings) != 1 || bindings[0].GetWordlist().GetResource() != testCanonicalWordlistResource {
		t.Fatalf("wordlist binding was not pinned in the saved plan: %#v", bindings)
	}
	if plan.ProtoReflect().Descriptor().Fields().ByName("inputs") != nil {
		t.Fatal("saved plan unexpectedly exposes an Engine-specific input allow-set")
	}
	if len(wordlists.requests) != 1 || wordlists.requests[0] != testCanonicalWordlistResource {
		t.Fatalf("PlanTask did not resolve the declared resource exactly once: %#v", wordlists.requests)
	}
}

func TestScanCreateConfigResourceFailureAbortsAtomicPersistence(t *testing.T) {
	tests := []struct {
		name  string
		cause ConfigResourceValidationCause
		new   func(error) *ConfigResourceValidationError
	}{
		{name: "unavailable", cause: ConfigResourceUnavailable, new: func(err error) *ConfigResourceValidationError {
			return NewConfigResourceUnavailableError("field", "wordlist", testCanonicalWordlistResource, err)
		}},
		{name: "validation unavailable", cause: ConfigResourceValidationUnavailable, new: func(err error) *ConfigResourceValidationError {
			return NewConfigResourceValidationUnavailableError("field", "wordlist", testCanonicalWordlistResource, err)
		}},
		{name: "internal", cause: ConfigResourceInternal, new: func(err error) *ConfigResourceValidationError {
			return NewConfigResourceInternalError("field", "wordlist", testCanonicalWordlistResource, err)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := testPlanTaskDefinition([]string{engineexecution.InputSubdomains}, nil, true)
			exact := planTaskExactPackage(definition)
			packages := &planTaskPackageReaderStub{packageValue: exact}
			diagnostic := errors.New("resource validation diagnostic")
			resources := &planTaskWordlistReaderStub{err: test.new(diagnostic)}
			compiler, _ := NewPlanTaskCompiler(packages, resources)
			store := &plannedScanCreateStoreStub{}
			manifest := ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{
				StageID: "ports", Steps: []ScanCreateWorkflowStep{{StepID: "port_scan", EngineID: exact.Identity.EngineID}},
			}}}
			reader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
				exact.Identity.EngineID: scanCreatePackageFromPlanPackage(exact),
			}}
			service := mustConfigurePlanTaskForTest(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
				return &TargetRef{ID: 17, Name: "example.com", Type: "domain", CreatedAt: time.Now().UTC()}, nil
			}, nil, scanCreateWorkflowReaderStub{manifest: manifest}, reader), compiler, 3*time.Second)

			_, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(17, "scanWorkflows/default", completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{
				"port_scan": definition.Execution,
			})))
			var typed *ConfigResourceValidationError
			if !errors.As(err, &typed) || typed.ConfigResourceValidationCause() != string(test.cause) || !errors.Is(err, diagnostic) {
				t.Fatalf("CreateBatch() error = %v, want typed cause %q", err, test.cause)
			}
			if store.committed != nil {
				t.Fatalf("resource failure committed Scan/Tasks/plans: %#v", store.committed)
			}
		})
	}
}

func TestScanCreatePlanTaskRejectsNonCanonicalPersistedTargetWithoutRepair(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, false)
	exact := planTaskExactPackage(definition)
	packages := &planTaskPackageReaderStub{packageValue: exact}
	compiler, _ := NewPlanTaskCompiler(packages, nil)
	store := &plannedScanCreateStoreStub{}
	manifest := ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{
		StageID: "ports", Steps: []ScanCreateWorkflowStep{{StepID: "port_scan", EngineID: exact.Identity.EngineID}},
	}}}
	reader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		exact.Identity.EngineID: scanCreatePackageFromPlanPackage(exact),
	}}
	service := mustConfigurePlanTaskForTest(t, NewScanCreateService(store, func(context.Context, int) (*TargetRef, error) {
		return &TargetRef{ID: 17, Name: "example.com ", Type: " DOMAIN ", CreatedAt: time.Now().UTC()}, nil
	}, nil, scanCreateWorkflowReaderStub{manifest: manifest}, reader), compiler, 3*time.Second)

	if _, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(17, "scanWorkflows/default", completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{
		"port_scan": definition.Execution,
	}))); !isPlanTaskErrorKind(err, PlanTaskInvalidRequest) {
		t.Fatalf("CreateBatch error = %v, want non-canonical Target rejection", err)
	}
	if store.committed != nil {
		t.Fatalf("non-canonical Target committed a scan: %#v", store.committed)
	}
}

func scanCreatePackageFromPlanPackage(exact PlanTaskPackage) ScanCreateEnginePackage {
	return ScanCreateEnginePackage{Package: exact}
}
