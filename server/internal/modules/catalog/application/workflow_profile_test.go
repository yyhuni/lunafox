package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	installedenginetest "github.com/yyhuni/lunafox/server/internal/installedengines/testsupport"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type workflowProfileEngineResolverStub struct {
	definitions map[string]engineexecution.ExecutionDefinition
}

func (stub workflowProfileEngineResolverStub) ExecutionDefinition(_ context.Context, engineID string) (engineexecution.ExecutionDefinition, bool, error) {
	definition, ok := stub.definitions[engineID]
	return definition, ok, nil
}

func TestScanWorkflowProfileMaterializesCompleteDefaultsForParentWorkflow(t *testing.T) {
	workflow := builtinWorkflowForApplicationTest(t)
	store := &workflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{"default": workflow}}
	definition := engineexecution.ExecutionDefinition{
		EngineAPIMajor:       2,
		SupportedTargetTypes: []string{engineexecution.TargetTypeDomain},
		ConfigSections: []engineexecution.ConfigSectionDefinition{{
			ID: "recon", DefaultEnabled: true, RequiredEnabled: true,
			Params: []engineexecution.ParamDefinition{
				{Key: "timeout", Type: engineexecution.ParamTypeInteger, Default: 60},
				{
					Key: "wordlist", Type: engineexecution.ParamTypeString, Default: "preferred.txt",
					MinLength: workflowProfileIntPointer(1),
					Resource:  &engineexecution.ParamResourceBinding{Kind: engineexecution.ConfigResourceKindWordlist},
				},
			},
		}},
	}
	service, err := NewScanWorkflowProfileService(store, workflowProfileEngineResolverStub{definitions: map[string]engineexecution.ExecutionDefinition{
		"engine.lunafox.discovery": definition,
	}})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := service.GetScanWorkflowProfile(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	steps := profile.Configuration["steps"].(map[string]any)
	step := steps["discover"].(map[string]any)
	if enabled, ok := step["enabled"].(bool); !ok || !enabled {
		t.Fatalf("Profile must explicitly enable each Step: %+v", step)
	}
	config := step["engineConfig"].(map[string]any)
	recon := config["recon"].(map[string]any)
	if profile.ScanWorkflowID != "default" || recon["enabled"] != true || recon["timeout"] != 60 || recon["wordlist"] != "preferred.txt" {
		t.Fatalf("unexpected Profile: %+v", profile)
	}
}

func TestScanWorkflowProfileKeepsFingerprintDetectionStepEnabledWithCompleteDefaults(t *testing.T) {
	workflow := builtinWorkflowForApplicationTest(t)
	workflow.Stages = []scanworkflow.Stage{{
		StageID: "url_collection",
		Steps: []scanworkflow.Step{
			{StepID: "fingerprint_detection", EngineID: "engine.lunafox.fingerprint_detection", ProfileDefaultEnabled: true},
			{StepID: "url_collection", EngineID: "engine.lunafox.url_collection", ProfileDefaultEnabled: true},
		},
	}}
	store := &workflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{"default": workflow}}
	fingerprintDefinition := engineexecution.ExecutionDefinition{
		EngineAPIMajor:       2,
		SupportedTargetTypes: []string{engineexecution.TargetTypeDomain, engineexecution.TargetTypeIP, engineexecution.TargetTypeCIDR},
		ConfigSections: []engineexecution.ConfigSectionDefinition{{
			ID: "observer_ward", DefaultEnabled: true,
			Params: []engineexecution.ParamDefinition{
				{Key: "timeout", Type: engineexecution.ParamTypeInteger, Default: 3600, Minimum: workflowProfileIntPointer(60), Maximum: workflowProfileIntPointer(604800)},
				{Key: "request-timeout", Type: engineexecution.ParamTypeInteger, Default: 10, Minimum: workflowProfileIntPointer(1), Maximum: workflowProfileIntPointer(120)},
				{Key: "threads", Type: engineexecution.ParamTypeInteger, Default: 25, Minimum: workflowProfileIntPointer(1), Maximum: workflowProfileIntPointer(200)},
			},
		}},
	}
	urlCollectionDefinition := engineexecution.ExecutionDefinition{
		EngineAPIMajor:       2,
		SupportedTargetTypes: []string{engineexecution.TargetTypeDomain},
		ConfigSections: []engineexecution.ConfigSectionDefinition{{
			ID: "collect", DefaultEnabled: true,
			Params: []engineexecution.ParamDefinition{{Key: "timeout", Type: engineexecution.ParamTypeInteger, Default: 60}},
		}},
	}
	service, err := NewScanWorkflowProfileService(store, workflowProfileEngineResolverStub{definitions: map[string]engineexecution.ExecutionDefinition{
		"engine.lunafox.fingerprint_detection": fingerprintDefinition,
		"engine.lunafox.url_collection":        urlCollectionDefinition,
	}})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := service.GetScanWorkflowProfile(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	step := profile.Configuration["steps"].(map[string]any)["fingerprint_detection"].(map[string]any)
	if enabled, ok := step["enabled"].(bool); !ok || !enabled {
		t.Fatalf("Fingerprint Detection Profile Step must be explicitly enabled: %#v", step)
	}
	config, ok := step["engineConfig"].(map[string]any)
	if !ok || len(config) != 1 {
		t.Fatalf("Fingerprint Detection Profile config = %#v", step["engineConfig"])
	}
	observerWard, ok := config["observer_ward"].(map[string]any)
	if !ok || len(observerWard) != 4 || observerWard["enabled"] != true || observerWard["timeout"] != 3600 || observerWard["request-timeout"] != 10 || observerWard["threads"] != 25 {
		t.Fatalf("unexpected Observer Ward defaults: %#v", observerWard)
	}
}

func TestScanWorkflowProfileUsesPersistedDefaultForAnyEngineID(t *testing.T) {
	workflow := builtinWorkflowForApplicationTest(t)
	workflow.Stages = []scanworkflow.Stage{{
		StageID: "custom_stage",
		Steps: []scanworkflow.Step{{
			StepID: "custom_step", EngineID: "engine.example.custom", ProfileDefaultEnabled: false,
		}},
	}}
	store := &workflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{"default": workflow}}
	definition := engineexecution.ExecutionDefinition{
		EngineAPIMajor:       2,
		SupportedTargetTypes: []string{engineexecution.TargetTypeDomain},
		ConfigSections: []engineexecution.ConfigSectionDefinition{{
			ID: "scan", DefaultEnabled: true,
			Params: []engineexecution.ParamDefinition{{Key: "timeout", Type: engineexecution.ParamTypeInteger, Default: 60}},
		}},
	}
	service, err := NewScanWorkflowProfileService(store, workflowProfileEngineResolverStub{definitions: map[string]engineexecution.ExecutionDefinition{
		"engine.example.custom": definition,
	}})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := service.GetScanWorkflowProfile(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	step := profile.Configuration["steps"].(map[string]any)["custom_step"].(map[string]any)
	if enabled, ok := step["enabled"].(bool); !ok || enabled {
		t.Fatalf("Profile enabled value did not come from persisted Step metadata: %#v", step)
	}
	config := step["engineConfig"].(map[string]any)["scan"].(map[string]any)
	if config["enabled"] != true || config["timeout"] != 60 {
		t.Fatalf("inner Engine defaults were not materialized independently: %#v", config)
	}
}

func TestBuiltinProfileMaterializesEnabledStepsFromCurrentDefinition(t *testing.T) {
	repoRoot := workflowProfileRepositoryRoot(t)
	workflowPath := filepath.Join(repoRoot, "extensions", "workflows", "default.scan-workflow.json")
	payload, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read builtin Workflow: %v", err)
	}
	definition, err := scanworkflow.DecodeDefinition(payload, workflowPath)
	if err != nil {
		t.Fatalf("decode builtin Workflow: %v", err)
	}
	packages, err := installedenginetest.LoadBuiltinSourcePackages(repoRoot)
	if err != nil {
		t.Fatalf("load builtin Engine definitions: %v", err)
	}
	definitions := make(map[string]engineexecution.ExecutionDefinition, len(packages))
	for _, pkg := range packages {
		definitions[pkg.Registration.EngineID] = pkg.Layout.Definition.EngineDefinition.Execution
	}

	workflow := catalogdomain.ManagedScanWorkflow{
		ScanWorkflowID: definition.ScanWorkflowID,
		DisplayName:    definition.DisplayName,
		Description:    definition.Description,
		Stages:         definition.Stages,
		IsBuiltin:      true,
	}
	store := &workflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{"default": workflow}}
	service, err := NewScanWorkflowProfileService(store, workflowProfileEngineResolverStub{definitions: definitions})
	if err != nil {
		t.Fatalf("NewScanWorkflowProfileService() error = %v", err)
	}
	profile, err := service.GetScanWorkflowProfile(context.Background(), "default")
	if err != nil {
		t.Fatalf("GetScanWorkflowProfile() error = %v", err)
	}

	steps, ok := profile.Configuration["steps"].(map[string]any)
	if !ok {
		t.Fatalf("Profile steps = %#v", profile.Configuration["steps"])
	}
	wantEnabled := make(map[string]bool)
	for _, stage := range definition.Stages {
		for _, step := range stage.Steps {
			wantEnabled[step.StepID] = step.ProfileDefaultEnabled
		}
	}
	for stepID, rawStep := range steps {
		step, ok := rawStep.(map[string]any)
		if !ok {
			t.Fatalf("Profile Step %q = %#v", stepID, rawStep)
		}
		enabled, ok := step["enabled"].(bool)
		if !ok || enabled != wantEnabled[stepID] {
			t.Fatalf("Profile Step %q outer enabled = %#v, want persisted default %v", stepID, step["enabled"], wantEnabled[stepID])
		}
	}
	directoryStep, ok := steps["directory_scan"].(map[string]any)
	if !ok {
		t.Fatalf("Directory Profile Step = %#v", steps["directory_scan"])
	}
	if enabled, ok := directoryStep["enabled"].(bool); !ok || !enabled {
		t.Fatalf("Directory Profile outer enabled = %#v, want true", directoryStep["enabled"])
	}
	config, ok := directoryStep["engineConfig"].(map[string]any)
	if !ok || len(config) != 1 {
		t.Fatalf("Directory Profile config = %#v", directoryStep["engineConfig"])
	}
	wantFFUF := map[string]any{
		"enabled":               true,
		"wordlist":              "dir_default.txt",
		"recursion":             false,
		"recursion-depth":       1,
		"recursion-strategy":    "default",
		"auto-calibration":      true,
		"auto-calibration-mode": "ac",
		"match-codes":           "200-299,301,302,307,401,403,405,500",
		"concurrency":           5,
		"threads":               10,
		"rate":                  0,
		"delay":                 "0.1-2.0",
		"request-timeout":       10,
		"timeout":               86400,
		"follow-redirects":      false,
		"http2":                 false,
	}
	if got := config["ffuf"]; !reflect.DeepEqual(got, wantFFUF) {
		t.Fatalf("Directory ffuf defaults = %#v, want current Definition defaults %#v", got, wantFFUF)
	}

	urlCollectionStep, ok := steps["url_collection"].(map[string]any)
	if !ok {
		t.Fatalf("URL Collection Profile Step = %#v", steps["url_collection"])
	}
	uro, ok := urlCollectionStep["engineConfig"].(map[string]any)["uro"].(map[string]any)
	if !ok {
		t.Fatalf("URL Collection uro defaults = %#v", urlCollectionStep["engineConfig"])
	}
	for _, key := range []string{"whitelist", "blacklist", "filters"} {
		values, ok := uro[key].([]string)
		if !ok || values == nil || len(values) != 0 {
			t.Fatalf("URL Collection uro.%s must be a non-nil empty string array, got %#v", key, uro[key])
		}
	}
}

func workflowProfileRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", ".."))
}

func workflowProfileIntPointer(value int) *int { return &value }

func TestScanWorkflowProfileRejectsUnavailableEngine(t *testing.T) {
	workflow := builtinWorkflowForApplicationTest(t)
	store := &workflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{"default": workflow}}
	service, err := NewScanWorkflowProfileService(store, workflowProfileEngineResolverStub{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetScanWorkflowProfile(context.Background(), "default"); !errors.Is(err, ErrScanWorkflowEngineUnavailable) {
		t.Fatalf("unavailable Profile error = %v", err)
	}
}

func TestScanWorkflowProfileIsAllOrNothingAcrossWorkflowSteps(t *testing.T) {
	workflow := builtinWorkflowForApplicationTest(t)
	workflow.Stages = []scanworkflow.Stage{{
		StageID: "discovery",
		Steps: []scanworkflow.Step{
			{StepID: "discover", EngineID: "engine.lunafox.discovery", ProfileDefaultEnabled: true},
			{StepID: "missing", EngineID: "engine.lunafox.missing", ProfileDefaultEnabled: true},
		},
	}}
	store := &workflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{"default": workflow}}
	definition := engineexecution.ExecutionDefinition{
		EngineAPIMajor:       2,
		SupportedTargetTypes: []string{engineexecution.TargetTypeDomain},
		ConfigSections: []engineexecution.ConfigSectionDefinition{{
			ID: "scan", DefaultEnabled: true,
			Params: []engineexecution.ParamDefinition{{Key: "timeout", Type: engineexecution.ParamTypeInteger, Default: 60}},
		}},
	}
	service, err := NewScanWorkflowProfileService(store, workflowProfileEngineResolverStub{definitions: map[string]engineexecution.ExecutionDefinition{
		"engine.lunafox.discovery": definition,
	}})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := service.GetScanWorkflowProfile(context.Background(), "default")
	if profile != nil {
		t.Fatalf("failed Profile must not return a partial draft: %+v", profile)
	}
	if !errors.Is(err, ErrScanWorkflowEngineUnavailable) {
		t.Fatalf("expected unavailable Engine error, got %v", err)
	}
}

func TestScanWorkflowProfileRejectsInvalidParentTopologyWithoutPartialDraft(t *testing.T) {
	workflow := builtinWorkflowForApplicationTest(t)
	workflow.Stages = nil
	store := &workflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{"default": workflow}}
	service, err := NewScanWorkflowProfileService(store, workflowProfileEngineResolverStub{})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := service.GetScanWorkflowProfile(context.Background(), "default")
	if profile != nil || err == nil {
		t.Fatalf("invalid Profile parent must fail atomically: profile=%+v err=%v", profile, err)
	}
}
