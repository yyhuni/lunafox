package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/agentexecution"
	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/ocidistribution"
	contractresults "github.com/yyhuni/lunafox/contracts/results"
)

const (
	planTaskPackageDigest         = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	planTaskImageDigest           = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	planTaskWordlistDigest        = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	planTaskEmptyDigest           = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	testCanonicalWordlistResource = "wordlists/7"
	testEmptyWordlistResource     = "wordlists/8"
	testSecondaryWordlistResource = "wordlists/10"
)

func TestPlanTaskFailureKindsDistinguishPackageLoadFromCompilation(t *testing.T) {
	want := map[PlanTaskErrorKind]string{
		PlanTaskPackageError:   "server_package_load_failed",
		PlanTaskInvalidRequest: "server_plan_compilation_request_failed",
		PlanTaskConfigError:    "server_plan_compilation_config_failed",
		PlanTaskResourceError:  "server_plan_compilation_resource_failed",
		PlanTaskProtocolError:  "server_plan_compilation_protocol_failed",
	}
	for kind, value := range want {
		if string(kind) != value {
			t.Fatalf("PlanTask failure kind = %q, want %q", kind, value)
		}
	}
}

type planTaskPackageReaderStub struct {
	packageValue PlanTaskPackage
	err          error
	requests     []PlanTaskPackageIdentity
}

func (stub *planTaskPackageReaderStub) LoadExactPackage(_ context.Context, identity PlanTaskPackageIdentity) (PlanTaskPackage, error) {
	stub.requests = append(stub.requests, identity)
	if stub.err != nil {
		return PlanTaskPackage{}, stub.err
	}
	return stub.packageValue, nil
}

type planTaskWordlistReaderStub struct {
	byResource      map[string]PlanTaskWordlist
	err             error
	requests        []string
	resolveRequests []ConfigResourceResolveRequest
}

type cancelAwarePlanTaskPackageReader struct{}

func (cancelAwarePlanTaskPackageReader) LoadExactPackage(ctx context.Context, _ PlanTaskPackageIdentity) (PlanTaskPackage, error) {
	return PlanTaskPackage{}, ctx.Err()
}

func (stub *planTaskWordlistReaderStub) ResolveConfigResource(_ context.Context, request ConfigResourceResolveRequest) (PlanTaskWordlist, error) {
	stub.requests = append(stub.requests, request.ResourceName)
	stub.resolveRequests = append(stub.resolveRequests, request)
	if stub.err != nil {
		return PlanTaskWordlist{}, stub.err
	}
	return stub.byResource[request.ResourceName], nil
}

func TestPlanTaskReturnsExecutableWithExactDeterministicPlan(t *testing.T) {
	packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(testPlanTaskDefinition(
		nil,
		[]string{engineexecution.PlatformResourceSubfinderProviderConfig, engineexecution.NucleiTemplates},
		true,
	))}
	wordlists := &planTaskWordlistReaderStub{byResource: map[string]PlanTaskWordlist{
		testCanonicalWordlistResource: {Resource: testCanonicalWordlistResource, Basename: "dns.txt", SizeBytes: 8, LineCount: 2, SHA256: planTaskWordlistDigest},
	}}
	compiler, err := NewPlanTaskCompiler(packages, wordlists)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler failed: %v", err)
	}

	request := validPlanTaskRequest()
	request.TaskConfig = map[string]any{
		"scan":  map[string]any{"enabled": true, "threads": 20, "wordlist": testCanonicalWordlistResource},
		"alpha": map[string]any{"enabled": false},
	}
	outcome, err := compiler.PlanTask(request)
	if err != nil {
		t.Fatalf("PlanTask failed: %v", err)
	}
	executable, ok := outcome.(ExecutablePlanTask)
	if !ok || executable.Plan == nil {
		t.Fatalf("outcome = %#v, want ExecutablePlanTask", outcome)
	}
	plan := executable.Plan
	if plan.GetTarget().GetResource() != "targets/17" || plan.GetTarget().GetType() != agentexecutionv1.TargetType_TARGET_TYPE_DOMAIN || plan.GetTarget().GetValue() != "example.com" {
		t.Fatalf("unexpected canonical target: %#v", plan.GetTarget())
	}
	if !reflect.DeepEqual(plan.GetRuntimeImage().GetRefs(), packages.packageValue.RuntimeImageRefs) {
		t.Fatalf("runtime refs reordered or replaced: got %#v want %#v", plan.GetRuntimeImage().GetRefs(), packages.packageValue.RuntimeImageRefs)
	}
	if plan.ProtoReflect().Descriptor().Fields().ByName("inputs") != nil {
		t.Fatal("saved plan unexpectedly exposes an execution input allow-set")
	}
	if got := plan.GetConfig().GetSections(); len(got) != 2 || got[0].GetSectionId() != "alpha" || got[1].GetSectionId() != "scan" {
		t.Fatalf("config sections are not deterministic: %#v", got)
	}
	if got := plan.GetConfig().GetSections()[1].GetParams(); len(got) != 1 || got[0].GetKey() != "threads" || got[0].GetValue().GetIntegerValue() != 20 {
		t.Fatalf("resource value must be absent from scalar config: %#v", got)
	}
	if got := plan.GetConfigResourceBindings(); len(got) != 1 || got[0].GetSectionId() != "scan" || got[0].GetParamKey() != "wordlist" || got[0].GetWordlist().GetResource() != testCanonicalWordlistResource {
		t.Fatalf("unexpected config resource bindings: %#v", got)
	}
	if got := plan.GetPlatformResourceBindings(); len(got) != 1 || got[0].GetResourceId() != engineexecution.PlatformResourceSubfinderProviderConfig || got[0].GetContentType() != executionartifact.ContentTypeSubfinderProviderConfig {
		t.Fatalf("unexpected platform resource bindings: %#v", got)
	}
	if got := plan.GetRuntimeArtifactBindings(); len(got) != 1 || got[0].GetArtifactId() != engineexecution.NucleiTemplates || got[0].GetContentType() != executionartifact.ContentTypeRuntimeArtifact {
		t.Fatalf("unexpected runtime artifact bindings: %#v", got)
	}
	if !reflect.DeepEqual(wordlists.requests, []string{testCanonicalWordlistResource}) {
		t.Fatalf("unexpected wordlist resolution: %#v", wordlists.requests)
	}
	if plan.GetLimits().GetMaxExecutionDuration().AsDuration() != 2*time.Hour {
		t.Fatalf("unexpected task duration: %s", plan.GetLimits().GetMaxExecutionDuration())
	}
}

func TestPlanTaskCloudflareAccelerationMapsOnlyValidatedRuntimeImageTransport(t *testing.T) {
	packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(testPlanTaskDefinition(nil, nil, false))}
	packages.packageValue.RuntimeImageRefs = []string{
		"docker.io/yyhuni/lunafox-engine-runtime-port-scan@" + planTaskImageDigest,
		"ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@" + planTaskImageDigest,
	}
	compiler, err := NewPlanTaskCompiler(packages, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}
	compiler.WithRuntimeImageReferencesMapper(func(refs []string) ([]string, error) {
		acceleration, err := ocidistribution.BuildCloudflareAcceleration(refs)
		if err != nil {
			return nil, err
		}
		return acceleration.DownloadReferenceStrings(), nil
	})

	request := validPlanTaskRequest()
	request.TaskConfig = map[string]any{
		"scan":  map[string]any{"enabled": true, "threads": 20},
		"alpha": map[string]any{"enabled": false},
	}
	outcome, err := compiler.PlanTask(request)
	if err != nil {
		t.Fatalf("PlanTask() error = %v", err)
	}
	executable, ok := outcome.(ExecutablePlanTask)
	if !ok {
		t.Fatalf("PlanTask outcome = %#v, want executable plan", outcome)
	}
	want := []string{
		"docker.lunafox.cc.cd/yyhuni/lunafox-engine-runtime-port-scan@" + planTaskImageDigest,
		packages.packageValue.RuntimeImageRefs[0],
		packages.packageValue.RuntimeImageRefs[1],
	}
	if got := executable.Plan.GetRuntimeImage().GetRefs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("accelerated Runtime Image refs = %#v, want %#v", got, want)
	}
}

func TestPlanTaskPropagatesOperationContextThroughRequestOnlySeam(t *testing.T) {
	compiler, err := NewPlanTaskCompiler(cancelAwarePlanTaskPackageReader{}, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := validPlanTaskRequest()
	request.operationContext = ctx

	_, err = compiler.PlanTask(request)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PlanTask() error = %v, want propagated cancellation", err)
	}
}

func TestPlanTaskDoesNotPersistEngineInputMembership(t *testing.T) {
	packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(testPlanTaskDefinition(
		nil, nil, false,
	))}
	compiler, err := NewPlanTaskCompiler(packages, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}

	outcome, err := compiler.PlanTask(validPlanTaskRequest())
	if err != nil {
		t.Fatalf("PlanTask() error = %v", err)
	}
	plan, ok := outcome.(ExecutablePlanTask)
	if !ok || plan.Plan == nil {
		t.Fatalf("PlanTask() outcome = %#v, want executable plan", outcome)
	}
	if plan.Plan.ProtoReflect().Descriptor().Fields().ByName("inputs") != nil {
		t.Fatal("PlanTask persisted an Engine-specific input membership field")
	}
}

func TestPlanTaskRejectsStaleEngineAPIMajorAsPackageCompatibilityFailure(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, false)
	definition.Execution.EngineAPIMajor = 3
	packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(definition)}
	compiler, err := NewPlanTaskCompiler(packages, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}

	_, err = compiler.PlanTask(validPlanTaskRequest())
	if !isPlanTaskErrorKind(err, PlanTaskPackageError) || !strings.Contains(err.Error(), "unsupported Engine API major") {
		t.Fatalf("PlanTask() error = %v, want stale Engine API package compatibility rejection", err)
	}
	if len(packages.requests) != 1 {
		t.Fatalf("stale package was not rejected at the exact package boundary: %#v", packages.requests)
	}
}

func TestPlanTaskPreservesExplicitFingerprintResourceIntentWithoutResolvingArtifact(t *testing.T) {
	resources := make([]string, 0, 6)
	wantContentTypes := make(map[string]string)
	for _, descriptor := range executionartifact.Descriptors() {
		if !executionartifact.IsFingerprintLibraryRole(descriptor.Role) {
			continue
		}
		resources = append(resources, descriptor.PlatformResourceID)
		wantContentTypes[descriptor.PlatformResourceID] = descriptor.ContentType
	}
	// Authoring normalizes unordered declarations. Deliberately reverse the
	// input to prove task creation persists only canonical resource intent.
	for left, right := 0, len(resources)-1; left < right; left, right = left+1, right-1 {
		resources[left], resources[right] = resources[right], resources[left]
	}
	packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(testPlanTaskDefinition(nil, resources, false))}
	compiler, err := NewPlanTaskCompiler(packages, &planTaskWordlistReaderStub{err: errors.New("fingerprint planning must not read a runtime artifact")})
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}

	outcome, err := compiler.PlanTask(validPlanTaskRequest())
	if err != nil {
		t.Fatalf("PlanTask() error = %v", err)
	}
	plan, ok := outcome.(ExecutablePlanTask)
	if !ok || plan.Plan == nil {
		t.Fatalf("PlanTask() outcome = %#v, want executable plan", outcome)
	}
	bindings := plan.Plan.GetPlatformResourceBindings()
	if len(bindings) != len(resources) {
		t.Fatalf("fingerprint bindings = %#v", bindings)
	}
	previous := ""
	for _, binding := range bindings {
		if binding.GetResourceId() <= previous || binding.GetContentType() != wantContentTypes[binding.GetResourceId()] {
			t.Fatalf("fingerprint plan binding is not canonical intent: %#v", bindings)
		}
		previous = binding.GetResourceId()
	}
	encoded, err := agentexecution.MarshalResolvedEngineExecutionPlan(plan.Plan)
	if err != nil {
		t.Fatalf("MarshalResolvedEngineExecutionPlan() error = %v", err)
	}
	for _, forbidden := range []string{"source_generation", "sha256_digest", "size_bytes", "record_count", "fingerprint_artifact"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("saved plan leaked artifact identity %q: %s", forbidden, encoded)
		}
	}
}

func TestPlanTaskUnsupportedTargetReturnsSkippedWithoutResourceRead(t *testing.T) {
	packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(testPlanTaskDefinition(nil, nil, false))}
	wordlists := &planTaskWordlistReaderStub{err: errors.New("must not be called")}
	compiler, _ := NewPlanTaskCompiler(packages, wordlists)
	request := validPlanTaskRequest()
	request.Target = PlanTaskTarget{Resource: "targets/17", Type: engineexecution.TargetTypeIP, Value: "192.0.2.10"}

	outcome, err := compiler.PlanTask(request)
	if err != nil {
		t.Fatalf("PlanTask failed: %v", err)
	}
	skipped, ok := outcome.(SkippedPlanTask)
	if !ok || skipped.Reason != "target_not_applicable" {
		t.Fatalf("outcome = %#v, want stable skipped result", outcome)
	}
	if len(wordlists.requests) != 0 {
		t.Fatalf("unsupported target must not resolve runtime resources: %#v", wordlists.requests)
	}
}

func TestPlanTaskRejectsNonCanonicalTargetBeforePackageRead(t *testing.T) {
	tests := []struct {
		name   string
		target PlanTaskTarget
	}{
		{name: "uppercase domain", target: PlanTaskTarget{Resource: "targets/17", Type: engineexecution.TargetTypeDomain, Value: "Example.com"}},
		{name: "invalid domain labels", target: PlanTaskTarget{Resource: "targets/17", Type: engineexecution.TargetTypeDomain, Value: "bad..example"}},
		{name: "non-canonical IPv4", target: PlanTaskTarget{Resource: "targets/17", Type: engineexecution.TargetTypeIP, Value: "192.000.2.1"}},
		{name: "unmasked IPv4 CIDR", target: PlanTaskTarget{Resource: "targets/17", Type: engineexecution.TargetTypeCIDR, Value: "192.0.2.3/24"}},
		{name: "IPv6", target: PlanTaskTarget{Resource: "targets/17", Type: engineexecution.TargetTypeIP, Value: "2001:db8::1"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(testPlanTaskDefinition(nil, nil, false))}
			compiler, _ := NewPlanTaskCompiler(packages, nil)
			request := validPlanTaskRequest()
			request.Target = test.target

			if _, err := compiler.PlanTask(request); !isPlanTaskErrorKind(err, PlanTaskInvalidRequest) {
				t.Fatalf("error = %v, want classified invalid request", err)
			}
			if len(packages.requests) != 0 {
				t.Fatalf("invalid Target reached exact package reader: %#v", packages.requests)
			}
		})
	}
}

func TestPlanTaskDoesNotHideInvalidConfigBehindSkipped(t *testing.T) {
	packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(testPlanTaskDefinition(nil, nil, false))}
	compiler, _ := NewPlanTaskCompiler(packages, nil)
	request := validPlanTaskRequest()
	request.Target = PlanTaskTarget{Resource: "targets/17", Type: engineexecution.TargetTypeIP, Value: "192.0.2.10"}
	request.TaskConfig = map[string]any{"scan": map[string]any{"enabled": true, "threads": "fast"}, "alpha": map[string]any{"enabled": false}}

	if _, err := compiler.PlanTask(request); !isPlanTaskErrorKind(err, PlanTaskConfigError) {
		t.Fatalf("error = %v, want classified config error", err)
	}
}

func TestPlanTaskClassifiesPackageResourceAndProtocolFailures(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*planTaskPackageReaderStub, *planTaskWordlistReaderStub, *PlanTaskRequest)
		want   PlanTaskErrorKind
	}{
		{
			name: "package reader",
			mutate: func(packages *planTaskPackageReaderStub, _ *planTaskWordlistReaderStub, _ *PlanTaskRequest) {
				packages.err = errors.New("cache corrupt")
			},
			want: PlanTaskPackageError,
		},
		{
			name: "package identity",
			mutate: func(packages *planTaskPackageReaderStub, _ *planTaskWordlistReaderStub, _ *PlanTaskRequest) {
				packages.packageValue.Identity.PackageDigest = planTaskWordlistDigest
			},
			want: PlanTaskPackageError,
		},
		{
			name: "wordlist",
			mutate: func(_ *planTaskPackageReaderStub, wordlists *planTaskWordlistReaderStub, _ *PlanTaskRequest) {
				wordlists.err = errors.New("not found")
			},
			want: PlanTaskResourceError,
		},
		{
			name: "wordlist basename control character",
			mutate: func(_ *planTaskPackageReaderStub, wordlists *planTaskWordlistReaderStub, _ *PlanTaskRequest) {
				wordlist := wordlists.byResource[testCanonicalWordlistResource]
				wordlist.Basename = "dns\x00.txt"
				wordlists.byResource[testCanonicalWordlistResource] = wordlist
			},
			want: PlanTaskResourceError,
		},
		{
			name: "invalid package runtime ref",
			mutate: func(packages *planTaskPackageReaderStub, _ *planTaskWordlistReaderStub, _ *PlanTaskRequest) {
				packages.packageValue.RuntimeImageRefs = []string{"docker.io/lunafox/runtime:latest"}
			},
			want: PlanTaskPackageError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(testPlanTaskDefinition(nil, nil, true))}
			wordlists := &planTaskWordlistReaderStub{byResource: map[string]PlanTaskWordlist{
				testCanonicalWordlistResource: {Resource: testCanonicalWordlistResource, Basename: "dns.txt", SizeBytes: 8, LineCount: 2, SHA256: planTaskWordlistDigest},
			}}
			request := validPlanTaskRequest()
			request.TaskConfig = map[string]any{"scan": map[string]any{"enabled": true, "threads": 10, "wordlist": testCanonicalWordlistResource}, "alpha": map[string]any{"enabled": false}}
			test.mutate(packages, wordlists, &request)
			compiler, _ := NewPlanTaskCompiler(packages, wordlists)
			_, err := compiler.PlanTask(request)
			if !isPlanTaskErrorKind(err, test.want) {
				t.Fatalf("error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestPlanTaskPreservesTypedConfigResourceFailures(t *testing.T) {
	diagnostic := errors.New("private dependency diagnostic")
	tests := []struct {
		name      string
		resolver  error
		wantCause ConfigResourceValidationCause
		wantKind  PlanTaskErrorKind
	}{
		{
			name: "unavailable",
			resolver: NewConfigResourceUnavailableError(
				`configuration.steps["step"].engineConfig.scan.wordlist`, "wordlist", testCanonicalWordlistResource, diagnostic,
			),
			wantCause: ConfigResourceUnavailable, wantKind: PlanTaskResourceError,
		},
		{
			name: "validation unavailable",
			resolver: NewConfigResourceValidationUnavailableError(
				`configuration.steps["step"].engineConfig.scan.wordlist`, "wordlist", testCanonicalWordlistResource, diagnostic,
			),
			wantCause: ConfigResourceValidationUnavailable, wantKind: PlanTaskResourceError,
		},
		{
			name: "internal",
			resolver: NewConfigResourceInternalError(
				`configuration.steps["step"].engineConfig.scan.wordlist`, "wordlist", testCanonicalWordlistResource, diagnostic,
			),
			wantCause: ConfigResourceInternal, wantKind: PlanTaskResourceError,
		},
		{name: "untyped resolver failure becomes internal", resolver: diagnostic, wantCause: ConfigResourceInternal, wantKind: PlanTaskResourceError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := testPlanTaskDefinition(nil, nil, true)
			packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(definition)}
			resources := &planTaskWordlistReaderStub{err: test.resolver}
			compiler, _ := NewPlanTaskCompiler(packages, resources)
			request := validPlanTaskRequest()
			request.TaskConfig = map[string]any{
				"scan":  map[string]any{"enabled": true, "threads": 10, "wordlist": testCanonicalWordlistResource},
				"alpha": map[string]any{"enabled": false},
			}
			_, err := compiler.PlanTask(request)
			if !isPlanTaskErrorKind(err, test.wantKind) {
				t.Fatalf("PlanTask() error = %v, want kind %q", err, test.wantKind)
			}
			var typed *ConfigResourceValidationError
			if !errors.As(err, &typed) || typed.ConfigResourceValidationCause() != string(test.wantCause) {
				t.Fatalf("PlanTask() error = %v, want resource cause %q", err, test.wantCause)
			}
			if !errors.Is(err, diagnostic) {
				t.Fatalf("PlanTask() lost private wrapped cause: %v", err)
			}
		})
	}
}

func TestPlanTaskDoesNotResolveResourcesForDisabledConfigSection(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, true)
	packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(definition)}
	resources := &planTaskWordlistReaderStub{err: errors.New("disabled section must not resolve")}
	compiler, _ := NewPlanTaskCompiler(packages, resources)
	request := validPlanTaskRequest()
	request.TaskConfig = map[string]any{
		"scan":  map[string]any{"enabled": false},
		"alpha": map[string]any{"enabled": true, "label": "ok"},
	}
	outcome, err := compiler.PlanTask(request)
	if err != nil {
		t.Fatalf("PlanTask() error = %v", err)
	}
	executable := outcome.(ExecutablePlanTask)
	if len(executable.Plan.GetConfigResourceBindings()) != 0 || len(resources.resolveRequests) != 0 {
		t.Fatalf("disabled section produced resource work: bindings=%#v requests=%#v", executable.Plan.GetConfigResourceBindings(), resources.resolveRequests)
	}
}

func TestPlanTaskRejectsInvalidResourceValueBeforeResolver(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, true)
	packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(definition)}
	resources := &planTaskWordlistReaderStub{err: errors.New("must not be reached")}
	compiler, _ := NewPlanTaskCompiler(packages, resources)
	request := validPlanTaskRequest()
	request.TaskConfig = map[string]any{
		"scan":  map[string]any{"enabled": true, "threads": 10, "wordlist": ""},
		"alpha": map[string]any{"enabled": false},
	}
	if _, err := compiler.PlanTask(request); !isPlanTaskErrorKind(err, PlanTaskConfigError) {
		t.Fatalf("PlanTask() error = %v, want config failure", err)
	}
	if len(resources.resolveRequests) != 0 {
		t.Fatalf("invalid input reached resolver: %#v", resources.resolveRequests)
	}
}

func TestPlanTaskPreservesVerifiedZeroByteWordlistDescriptor(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, true)
	packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(definition)}
	resources := &planTaskWordlistReaderStub{byResource: map[string]PlanTaskWordlist{
		testEmptyWordlistResource: {
			Resource: testEmptyWordlistResource, Basename: "empty.txt", SizeBytes: 0, LineCount: 0, SHA256: planTaskEmptyDigest,
		},
	}}
	compiler, err := NewPlanTaskCompiler(packages, resources)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}
	request := validPlanTaskRequest()
	request.TaskConfig = map[string]any{
		"scan":  map[string]any{"enabled": true, "threads": 10, "wordlist": testEmptyWordlistResource},
		"alpha": map[string]any{"enabled": false},
	}
	outcome, err := compiler.PlanTask(request)
	if err != nil {
		t.Fatalf("PlanTask() error = %v", err)
	}
	plan := outcome.(ExecutablePlanTask).Plan
	bindings := plan.GetConfigResourceBindings()
	if len(bindings) != 1 || bindings[0].GetWordlist().GetSizeBytes() != 0 ||
		bindings[0].GetWordlist().GetLineCount() != 0 || bindings[0].GetWordlist().GetSha256Digest() != planTaskEmptyDigest {
		t.Fatalf("zero-byte wordlist binding = %#v", bindings)
	}
}

func TestPlanTaskPreservesOpaqueFFUFStringsInSavedPlan(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, false)
	definition.Execution.ConfigSections[0].Params = append(definition.Execution.ConfigSections[0].Params,
		engineexecution.ParamDefinition{Key: "match-codes", Type: engineexecution.ParamTypeString, Default: "200"},
		engineexecution.ParamDefinition{Key: "delay", Type: engineexecution.ParamTypeString, Default: "0"},
	)
	packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(definition)}
	compiler, err := NewPlanTaskCompiler(packages, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler() error = %v", err)
	}
	request := validPlanTaskRequest()
	matchCodes := "200, 201"
	delay := "1 - 2"
	request.TaskConfig = map[string]any{
		"scan": map[string]any{
			"enabled": true, "threads": 10, "match-codes": matchCodes, "delay": delay,
		},
		"alpha": map[string]any{"enabled": false},
	}
	outcome, err := compiler.PlanTask(request)
	if err != nil {
		t.Fatalf("PlanTask() semantically interpreted opaque FFUF strings: %v", err)
	}
	plan := outcome.(ExecutablePlanTask).Plan
	got := map[string]string{}
	for _, section := range plan.GetConfig().GetSections() {
		if section.GetSectionId() != "scan" {
			continue
		}
		for _, param := range section.GetParams() {
			if param.GetKey() == "match-codes" || param.GetKey() == "delay" {
				got[param.GetKey()] = param.GetValue().GetStringValue()
			}
		}
	}
	if got["match-codes"] != matchCodes || got["delay"] != delay {
		t.Fatalf("saved plan opaque strings = %#v", got)
	}
}

func TestPlanTaskTargetApplicabilityDoesNotDependOnRuntimeInputRoles(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, false)
	definition.Execution.SupportedTargetTypes = []string{engineexecution.TargetTypeDomain}
	packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(definition)}
	compiler, _ := NewPlanTaskCompiler(packages, nil)
	request := validPlanTaskRequest()
	request.Target = PlanTaskTarget{Resource: "targets/17", Type: engineexecution.TargetTypeCIDR, Value: "192.0.2.0/24"}

	outcome, err := compiler.PlanTask(request)
	if err != nil {
		t.Fatalf("PlanTask failed: %v", err)
	}
	if _, ok := outcome.(SkippedPlanTask); !ok {
		t.Fatalf("runtime input Registry must not make CIDR applicable: %#v", outcome)
	}
}

func validPlanTaskRequest() PlanTaskRequest {
	return PlanTaskRequest{
		Execution:  "executions/scan-23-task-31",
		Task:       "scans/23/tasks/31",
		Scan:       "scans/23",
		Workflow:   "scanWorkflows/default",
		StageID:    "stage",
		StepID:     "step",
		EngineID:   "engine.lunafox.port_scan",
		Target:     PlanTaskTarget{Resource: "targets/17", Type: engineexecution.TargetTypeDomain, Value: "example.com"},
		TaskConfig: map[string]any{"scan": map[string]any{"enabled": true, "threads": 10}, "alpha": map[string]any{"enabled": false}},
		Package:    PlanTaskPackageIdentity{EngineID: "engine.lunafox.port_scan", PackageDigest: planTaskPackageDigest},
		Limits: PlanTaskLimits{
			MaxExecutionDuration:    2 * time.Hour,
			ProgressMessageMaxBytes: 4096,
			ResultBatchMaxItems:     uint32(contractresults.DefaultResultBatchMaxItems),
			ResultBatchMaxBytes:     uint32(contractresults.DefaultResultBatchMaxBytes),
		},
	}
}

func TestPlanTaskRejectsEnabledStepWithAllConfigSectionsDisabled(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, false)
	packages := &planTaskPackageReaderStub{packageValue: planTaskExactPackage(definition)}
	compiler, _ := NewPlanTaskCompiler(packages, nil)
	request := validPlanTaskRequest()
	request.TaskConfig = map[string]any{
		"scan":  map[string]any{"enabled": false},
		"alpha": map[string]any{"enabled": false},
	}

	_, err := compiler.PlanTask(request)
	if !isPlanTaskErrorKind(err, PlanTaskConfigError) {
		t.Fatalf("expected all-disabled config sections to fail validation, got %v", err)
	}
}

func planTaskExactPackage(definition enginecontract.EngineDefinition) PlanTaskPackage {
	return PlanTaskPackage{
		Identity:       PlanTaskPackageIdentity{EngineID: "engine.lunafox.port_scan", PackageDigest: planTaskPackageDigest},
		PackageVersion: "1.0.0",
		Definition:     definition,
		RuntimeImageRefs: []string{
			"docker.io/lunafox/lunafox-engine-runtime-port-scan@" + planTaskImageDigest,
			"ghcr.io/lunafox/lunafox-engine-runtime-port-scan@" + planTaskImageDigest,
		},
	}
}

func testPlanTaskDefinition(_ []string, executionResources []string, withWordlist bool) enginecontract.EngineDefinition {
	params := []engineexecution.ParamDefinition{{Key: "threads", Type: engineexecution.ParamTypeInteger, Default: 10, Minimum: intPointer(1)}}
	if withWordlist {
		params = append(params, engineexecution.ParamDefinition{
			Key: "wordlist", Type: engineexecution.ParamTypeString, Default: "dns.txt", MinLength: intPointer(1),
			Resource: &engineexecution.ParamResourceBinding{Kind: engineexecution.ConfigResourceKindWordlist},
		})
	}
	return enginecontract.EngineDefinition{
		ManifestVersion: enginecontract.SupportedRootManifestVersion,
		EngineID:        "engine.lunafox.port_scan",
		Publisher:       "lunafox",
		Execution: engineexecution.ExecutionDefinition{
			EngineAPIMajor:       2,
			SupportedTargetTypes: []string{engineexecution.TargetTypeDomain},
			ExecutionResources:   executionResources,
			ConfigSections: []engineexecution.ConfigSectionDefinition{
				{ID: "scan", DefaultEnabled: true, Params: params},
				{ID: "alpha", Params: []engineexecution.ParamDefinition{{Key: "label", Type: engineexecution.ParamTypeString, Default: "ok"}}},
			},
		},
	}
}

func intPointer(value int) *int { return &value }

func isPlanTaskErrorKind(err error, kind PlanTaskErrorKind) bool {
	var classified *PlanTaskError
	return errors.As(err, &classified) && classified.Kind == kind
}
