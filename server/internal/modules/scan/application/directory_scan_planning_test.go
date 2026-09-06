package application

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	installedenginetest "github.com/yyhuni/lunafox/server/internal/installedengines/testsupport"
)

const (
	directoryScanEngineID     = "engine.lunafox.directory_scan"
	screenshotEngineID        = "engine.lunafox.screenshot"
	directoryWordlistResource = "wordlists/9"
)

func TestDirectoryPlanBindsCurrentScanWebsiteURLsAndCanonicalTargetForEverySupportedType(t *testing.T) {
	directory := builtinSourcePlanPackagesForDirectoryTest(t)[directoryScanEngineID]
	packages := &planTaskPackageReaderStub{packageValue: directory}
	resources := directoryWordlistResolverForTest()
	compiler, err := NewPlanTaskCompiler(packages, resources)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}
	config := defaultEngineConfigForDirectoryTest(t, directory)
	ffuf := config["ffuf"].(map[string]any)
	ffuf["wordlist"] = directoryWordlistResource
	ffuf["match-codes"] = "200, 201"
	ffuf["delay"] = "1 - 2"

	tests := []struct {
		name      string
		target    PlanTaskTarget
		protoType agentexecutionv1.TargetType
	}{
		{name: "domain", target: PlanTaskTarget{Resource: "targets/17", Type: engineexecution.TargetTypeDomain, Value: "example.com"}, protoType: agentexecutionv1.TargetType_TARGET_TYPE_DOMAIN},
		{name: "ip", target: PlanTaskTarget{Resource: "targets/17", Type: engineexecution.TargetTypeIP, Value: "192.0.2.10"}, protoType: agentexecutionv1.TargetType_TARGET_TYPE_IP},
		{name: "cidr", target: PlanTaskTarget{Resource: "targets/17", Type: engineexecution.TargetTypeCIDR, Value: "192.0.2.0/24"}, protoType: agentexecutionv1.TargetType_TARGET_TYPE_CIDR},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := validPlanTaskRequest()
			request.StageID = "directory_scan"
			request.StepID = "directory_scan"
			request.EngineID = directoryScanEngineID
			request.Target = test.target
			request.TaskConfig = config
			request.Package = directory.Identity

			outcome, err := compiler.PlanTask(request)
			if err != nil {
				t.Fatalf("PlanTask() error = %v", err)
			}
			executable, ok := outcome.(ExecutablePlanTask)
			if !ok || executable.Plan == nil {
				t.Fatalf("PlanTask() outcome = %#v, want executable Directory plan", outcome)
			}
			plan := executable.Plan
			if plan.GetTarget().GetResource() != test.target.Resource || plan.GetTarget().GetType() != test.protoType || plan.GetTarget().GetValue() != test.target.Value {
				t.Fatalf("canonical Target changed: got %#v want %#v", plan.GetTarget(), test.target)
			}
			if plan.GetEngineRelease().GetEngine() != directoryScanEngineID || plan.GetEngineRelease().GetPackageDigest() != directory.Identity.PackageDigest || plan.GetEngineRelease().GetEngineApiMajor() != 2 {
				t.Fatalf("Directory package identity = %#v", plan.GetEngineRelease())
			}
			if !reflect.DeepEqual(plan.GetRuntimeImage().GetRefs(), directory.RuntimeImageRefs) {
				t.Fatalf("Directory Runtime Image refs = %#v, want %#v", plan.GetRuntimeImage().GetRefs(), directory.RuntimeImageRefs)
			}
			resourceBindings := plan.GetConfigResourceBindings()
			if len(resourceBindings) != 1 || resourceBindings[0].GetSectionId() != "ffuf" || resourceBindings[0].GetParamKey() != "wordlist" || resourceBindings[0].GetWordlist().GetResource() != directoryWordlistResource {
				t.Fatalf("Directory wordlist binding = %#v", resourceBindings)
			}
			if got := planStringConfigValues(plan, "ffuf", "match-codes", "delay"); got["match-codes"] != "200, 201" || got["delay"] != "1 - 2" {
				t.Fatalf("Directory opaque FFUF strings = %#v", got)
			}
		})
	}
	if len(packages.requests) != len(tests) {
		t.Fatalf("Directory exact package reads = %#v", packages.requests)
	}
	for _, identity := range packages.requests {
		if identity != directory.Identity {
			t.Fatalf("Directory package request = %#v, want %#v", identity, directory.Identity)
		}
	}
}

func TestDirectoryPlanRejectsEnabledStepWithFFUFDisabled(t *testing.T) {
	directory := builtinSourcePlanPackagesForDirectoryTest(t)[directoryScanEngineID]
	compiler, err := NewPlanTaskCompiler(&planTaskPackageReaderStub{packageValue: directory}, directoryWordlistResolverForTest())
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}
	request := validPlanTaskRequest()
	request.EngineID = directoryScanEngineID
	request.Package = directory.Identity
	request.TaskConfig = map[string]any{"ffuf": map[string]any{"enabled": false}}

	if _, err := compiler.PlanTask(request); !isPlanTaskErrorKind(err, PlanTaskConfigError) {
		t.Fatalf("PlanTask() error = %v, want all-sections-disabled config failure", err)
	}
}

func TestScanCreateSkippedScreenshotBarrierAllowsDirectoryPending(t *testing.T) {
	packages := builtinSourcePlanPackagesForDirectoryTest(t)
	directory := packages[directoryScanEngineID]
	manifest := screenshotThenDirectoryManifestForTest()
	packageReader := &selectiveScanCreatePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		directoryScanEngineID: {Package: directory},
	}}
	compilerPackages := &planTaskPackageReaderStub{packageValue: directory}
	compiler, err := NewPlanTaskCompiler(compilerPackages, directoryWordlistResolverForTest())
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}
	registrations := &scanCreateRegistrationReaderStub{exists: map[string]bool{
		screenshotEngineID: true, directoryScanEngineID: true,
	}}
	store := &scanCreateStoreCaptureStub{}
	service := NewScanCreateService(
		store,
		func(context.Context, int) (*TargetRef, error) {
			return &TargetRef{ID: 17, Name: "example.com", Type: engineexecution.TargetTypeDomain, CreatedAt: time.Now().UTC()}, nil
		},
		nil,
		scanCreateWorkflowReaderStub{manifest: manifest},
		packageReader,
	).WithEngineRegistrationReader(registrations)
	if err := service.ConfigurePlanTask(compiler, DefaultPlanTaskLimitsProvider()); err != nil {
		t.Fatalf("ConfigurePlanTask() error = %v", err)
	}
	configuration := map[string]any{"steps": map[string]any{
		"screenshot":     map[string]any{"enabled": false},
		"directory_scan": map[string]any{"enabled": true, "engineConfig": defaultEngineConfigForDirectoryTest(t, directory)},
	}}
	matchCodes := "200, 201"
	delay := "1 - 2"
	directoryConfig := configuration["steps"].(map[string]any)["directory_scan"].(map[string]any)["engineConfig"].(map[string]any)
	directoryConfig["ffuf"].(map[string]any)["match-codes"] = matchCodes
	directoryConfig["ffuf"].(map[string]any)["delay"] = delay

	result, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(17, "scanWorkflows/default", configuration))
	if err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}
	if result.CreatedCount != 1 || store.lastScan == nil || len(store.lastScan.ScanTasks) != 2 {
		t.Fatalf("Directory Scan create result = %#v scan=%#v", result, store.lastScan)
	}
	screenshot := store.lastScan.ScanTasks[0]
	if screenshot.Status != CreateTaskStatusSkipped || screenshot.SkipReason != "user_disabled" || len(screenshot.ResolvedExecutionPlan) != 0 {
		t.Fatalf("skipped Screenshot task = %#v", screenshot)
	}
	directoryTask := store.lastScan.ScanTasks[1]
	if directoryTask.Status != CreateTaskStatusPending || directoryTask.StageID != "directory_scan" || directoryTask.StepID != "directory_scan" || len(directoryTask.ResolvedExecutionPlan) == 0 {
		t.Fatalf("Directory task behind skipped barrier = %#v, want pending saved plan", directoryTask)
	}
	persistedSteps := store.lastScan.Configuration["steps"].(map[string]any)
	persistedDirectory := persistedSteps["directory_scan"].(map[string]any)["engineConfig"].(map[string]any)
	persistedFFUF := persistedDirectory["ffuf"].(map[string]any)
	if persistedFFUF["match-codes"] != matchCodes || persistedFFUF["delay"] != delay {
		t.Fatalf("persisted Scan configuration rewrote opaque FFUF strings: %#v", persistedFFUF)
	}
	if !reflect.DeepEqual(packageReader.loadCalls, []string{directoryScanEngineID}) || len(compilerPackages.requests) != 1 {
		t.Fatalf("package work = scan-create %#v compiler %#v", packageReader.loadCalls, compilerPackages.requests)
	}
	plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(directoryTask.ResolvedExecutionPlan)
	if err != nil {
		t.Fatalf("decode Directory saved plan: %v", err)
	}
	if plan.GetWorkflowStep().GetStageId() != "directory_scan" || plan.GetWorkflowStep().GetStepId() != "directory_scan" {
		t.Fatalf("Directory saved plan scope = %#v", plan.GetWorkflowStep())
	}
	if got := planStringConfigValues(plan, "ffuf", "match-codes", "delay"); got["match-codes"] != matchCodes || got["delay"] != delay {
		t.Fatalf("Directory saved plan rewrote opaque FFUF strings: %#v", got)
	}
}

func TestProductionPlanBudgetIsFourteenDaysAcrossScreenshotAndDirectory(t *testing.T) {
	packages := builtinSourcePlanPackagesForDirectoryTest(t)
	screenshot := packages[screenshotEngineID]
	directory := packages[directoryScanEngineID]
	reader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		screenshotEngineID:    {Package: screenshot},
		directoryScanEngineID: {Package: directory},
	}}
	compiler, err := NewPlanTaskCompiler(reader, directoryWordlistResolverForTest())
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}
	store := &plannedScanCreateStoreStub{}
	service := NewScanCreateService(
		store,
		func(context.Context, int) (*TargetRef, error) {
			return &TargetRef{ID: 17, Name: "example.com", Type: engineexecution.TargetTypeDomain, CreatedAt: time.Now().UTC()}, nil
		},
		nil,
		scanCreateWorkflowReaderStub{manifest: screenshotThenDirectoryManifestForTest()},
		reader,
	)
	if err := service.ConfigurePlanTask(compiler, DefaultPlanTaskLimitsProvider()); err != nil {
		t.Fatalf("ConfigurePlanTask() error = %v", err)
	}
	configuration := map[string]any{"steps": map[string]any{
		"screenshot":     map[string]any{"enabled": true, "engineConfig": defaultEngineConfigForDirectoryTest(t, screenshot)},
		"directory_scan": map[string]any{"enabled": true, "engineConfig": defaultEngineConfigForDirectoryTest(t, directory)},
	}}

	if _, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(17, "scanWorkflows/default", configuration)); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}
	if store.committed == nil || len(store.committed.ScanTasks) != 2 {
		t.Fatalf("persisted cross-Engine tasks = %#v", store.committed)
	}
	if store.committed.ScanTasks[0].Status != CreateTaskStatusPending || store.committed.ScanTasks[1].Status != CreateTaskStatusBlocked {
		t.Fatalf("cross-Stage task states = %#v", store.committed.ScanTasks)
	}
	for index, wantEngineID := range []string{screenshotEngineID, directoryScanEngineID} {
		plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(store.committed.ScanTasks[index].ResolvedExecutionPlan)
		if err != nil {
			t.Fatalf("decode %s saved plan: %v", wantEngineID, err)
		}
		if plan.GetEngineRelease().GetEngine() != wantEngineID {
			t.Fatalf("saved plan %d Engine = %q, want %q", index, plan.GetEngineRelease().GetEngine(), wantEngineID)
		}
		budget := plan.GetLimits().GetMaxExecutionDuration()
		if budget.GetSeconds() != 1_209_600 || budget.GetNanos() != 0 || budget.AsDuration() != 14*24*time.Hour {
			t.Fatalf("%s saved plan budget = %s, want exact 1,209,600s", wantEngineID, budget)
		}
	}
}

func TestScanCreateDisabledDirectoryUsesRegistrationOnlyAndPersistsNoPlan(t *testing.T) {
	packages := builtinSourcePlanPackagesForDirectoryTest(t)
	screenshot := packages[screenshotEngineID]
	manifest := screenshotThenDirectoryManifestForTest()
	packageReader := &selectiveScanCreatePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		screenshotEngineID: {Package: screenshot},
	}}
	compilerPackages := &planTaskPackageReaderStub{packageValue: screenshot}
	compiler, err := NewPlanTaskCompiler(compilerPackages, &planTaskWordlistReaderStub{err: context.Canceled})
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}
	registrations := &scanCreateRegistrationReaderStub{exists: map[string]bool{
		screenshotEngineID: true, directoryScanEngineID: true,
	}}
	store := &scanCreateStoreCaptureStub{}
	service := NewScanCreateService(
		store,
		func(context.Context, int) (*TargetRef, error) {
			return &TargetRef{ID: 17, Name: "example.com", Type: engineexecution.TargetTypeDomain, CreatedAt: time.Now().UTC()}, nil
		},
		nil,
		scanCreateWorkflowReaderStub{manifest: manifest},
		packageReader,
	).WithEngineRegistrationReader(registrations)
	if err := service.ConfigurePlanTask(compiler, FixedPlanTaskLimitsProvider(time.Hour)); err != nil {
		t.Fatalf("ConfigurePlanTask() error = %v", err)
	}
	configuration := map[string]any{"steps": map[string]any{
		"screenshot":     map[string]any{"enabled": true, "engineConfig": defaultEngineConfigForDirectoryTest(t, screenshot)},
		"directory_scan": map[string]any{"enabled": false},
	}}

	if _, err := service.CreateBatch(context.Background(), singleTargetBatchCreateInput(17, "scanWorkflows/default", configuration)); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}
	if store.lastScan == nil || len(store.lastScan.ScanTasks) != 2 {
		t.Fatalf("persisted Scan = %#v", store.lastScan)
	}
	if !reflect.DeepEqual(registrations.calls, []string{screenshotEngineID, directoryScanEngineID}) {
		t.Fatalf("registration reads = %#v", registrations.calls)
	}
	if !reflect.DeepEqual(packageReader.loadCalls, []string{screenshotEngineID}) || len(compilerPackages.requests) != 1 || compilerPackages.requests[0] != screenshot.Identity {
		t.Fatalf("disabled Directory reached package planning: loads=%#v exact=%#v", packageReader.loadCalls, compilerPackages.requests)
	}
	directoryTask := store.lastScan.ScanTasks[1]
	if directoryTask.Status != CreateTaskStatusSkipped || directoryTask.SkipReason != "user_disabled" || len(directoryTask.ResolvedExecutionPlan) != 0 || directoryTask.EngineConfig != nil || directoryTask.TaskExecutionConfig != nil {
		t.Fatalf("disabled Directory task = %#v", directoryTask)
	}
}

func builtinSourcePlanPackagesForDirectoryTest(t *testing.T) map[string]PlanTaskPackage {
	t.Helper()
	repoRoot := filepath.Clean(filepath.Join(scanApplicationDir(t), "..", "..", "..", "..", ".."))
	loaded, err := installedenginetest.LoadBuiltinSourcePackages(repoRoot)
	if err != nil {
		t.Fatalf("LoadBuiltinSourcePackages() error = %v", err)
	}
	packages := make(map[string]PlanTaskPackage, len(loaded))
	for _, pkg := range loaded {
		packages[pkg.Registration.EngineID] = PlanTaskPackage{
			Identity: PlanTaskPackageIdentity{
				EngineID:      pkg.Registration.EngineID,
				PackageDigest: pkg.Registration.PackageDigest,
			},
			PackageVersion:   pkg.Registration.PackageVersion,
			Definition:       pkg.Layout.Definition.EngineDefinition,
			RuntimeImageRefs: append([]string(nil), pkg.Layout.Definition.PackageManifest.RuntimeImage.Refs...),
		}
	}
	for _, engineID := range []string{screenshotEngineID, directoryScanEngineID} {
		if _, ok := packages[engineID]; !ok {
			t.Fatalf("builtin source package %q is unavailable: %#v", engineID, packages)
		}
	}
	return packages
}

func directoryWordlistResolverForTest() *planTaskWordlistReaderStub {
	return &planTaskWordlistReaderStub{byResource: map[string]PlanTaskWordlist{
		directoryWordlistResource: {
			Resource: directoryWordlistResource, Basename: "dir_default.txt", SizeBytes: 8, LineCount: 2, SHA256: planTaskWordlistDigest,
		},
	}}
}

func defaultEngineConfigForDirectoryTest(t *testing.T, pkg PlanTaskPackage) map[string]any {
	t.Helper()
	config, err := engineexecution.NormalizeAndValidateConfig(map[string]any{}, pkg.Definition.Execution)
	if err != nil {
		t.Fatalf("materialize %s defaults: %v", pkg.Identity.EngineID, err)
	}
	if ffuf, ok := config["ffuf"].(map[string]any); ok {
		// Engine Definition defaults are filename candidates; persisted test
		// configurations must use the canonical Catalog resource instead.
		if _, ok := ffuf["wordlist"]; ok {
			ffuf["wordlist"] = directoryWordlistResource
		}
	}
	return config
}

func screenshotThenDirectoryManifestForTest() ScanCreateWorkflowManifest {
	return ScanCreateWorkflowManifest{
		ScanWorkflowID: "default",
		Stages: []ScanCreateWorkflowStage{
			{StageID: "screenshot", Steps: []ScanCreateWorkflowStep{{StepID: "screenshot", EngineID: screenshotEngineID}}},
			{StageID: "directory_scan", Steps: []ScanCreateWorkflowStep{{StepID: "directory_scan", EngineID: directoryScanEngineID}}},
		},
	}
}

func planStringConfigValues(plan *agentexecutionv1.ResolvedEngineExecutionPlan, sectionID string, keys ...string) map[string]string {
	wanted := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		wanted[key] = struct{}{}
	}
	values := make(map[string]string, len(keys))
	for _, section := range plan.GetConfig().GetSections() {
		if section.GetSectionId() != sectionID {
			continue
		}
		for _, param := range section.GetParams() {
			if _, ok := wanted[param.GetKey()]; ok {
				values[param.GetKey()] = param.GetValue().GetStringValue()
			}
		}
	}
	return values
}
