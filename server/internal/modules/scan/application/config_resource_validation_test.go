package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
)

type configResourceResolverStub struct {
	requests []ConfigResourceResolveRequest
	err      error
}

func (stub *configResourceResolverStub) ResolveConfigResource(_ context.Context, request ConfigResourceResolveRequest) (PlanTaskWordlist, error) {
	stub.requests = append(stub.requests, request)
	if stub.err != nil {
		return PlanTaskWordlist{}, stub.err
	}
	return PlanTaskWordlist{
		Resource: request.ResourceName, Basename: "dns.txt",
		SizeBytes: 8, LineCount: 2, SHA256: planTaskWordlistDigest,
	}, nil
}

type configResourcePackageReaderStub struct {
	packages map[string]ScanCreateEnginePackage
	requests []string
	err      error
}

func (stub *configResourcePackageReaderStub) LoadEnginePackage(_ context.Context, engineID string) (ScanCreateEnginePackage, error) {
	stub.requests = append(stub.requests, engineID)
	if stub.err != nil {
		return ScanCreateEnginePackage{}, stub.err
	}
	return stub.packages[engineID], nil
}

func TestConfigResourceValidationErrorExposesOnlySafeStableView(t *testing.T) {
	diagnostic := errors.New("open /srv/private/wordlists/secret.txt: permission denied")
	tests := []struct {
		name  string
		cause ConfigResourceValidationCause
		new   func(error) *ConfigResourceValidationError
	}{
		{name: "unavailable", cause: ConfigResourceUnavailable, new: func(err error) *ConfigResourceValidationError {
			return NewConfigResourceUnavailableError(`configuration.steps["step"].engineConfig.scan.wordlist`, "wordlist", testCanonicalWordlistResource, err)
		}},
		{name: "validation unavailable", cause: ConfigResourceValidationUnavailable, new: func(err error) *ConfigResourceValidationError {
			return NewConfigResourceValidationUnavailableError(`configuration.steps["step"].engineConfig.scan.wordlist`, "wordlist", testCanonicalWordlistResource, err)
		}},
		{name: "internal", cause: ConfigResourceInternal, new: func(err error) *ConfigResourceValidationError {
			return NewConfigResourceInternalError(`configuration.steps["step"].engineConfig.scan.wordlist`, "wordlist", testCanonicalWordlistResource, err)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			typed := test.new(diagnostic)
			wrapped := fmt.Errorf("outer boundary: %w", typed)
			var found *ConfigResourceValidationError
			if !errors.As(wrapped, &found) || found != typed {
				t.Fatalf("errors.As() did not preserve typed error: %v", wrapped)
			}
			if !errors.Is(wrapped, diagnostic) {
				t.Fatalf("wrapped diagnostic cause was not retained: %v", wrapped)
			}
			if found.ConfigResourceValidationCause() != string(test.cause) ||
				found.ConfigResourceValidationField() != `configuration.steps["step"].engineConfig.scan.wordlist` ||
				found.ConfigResourceValidationKind() != "wordlist" ||
				found.ConfigResourceValidationName() != testCanonicalWordlistResource {
				t.Fatalf("unexpected safe view: %#v", found)
			}
			if strings.Contains(found.Error(), "/srv/private") || strings.Contains(found.Error(), "permission denied") {
				t.Fatalf("public error leaked private diagnostic: %q", found.Error())
			}
		})
	}
}

func TestWorkflowConfigResourceValidatorVisitsOnlyEnabledScopes(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, true)
	definition.EngineID = "engine.lunafox.first"
	definition.Execution.ConfigSections[0].Params = append(definition.Execution.ConfigSections[0].Params,
		engineexecution.ParamDefinition{
			Key: "exclude", Type: engineexecution.ParamTypeString, Default: "exclude.txt", MinLength: intPointer(1),
			Resource: &engineexecution.ParamResourceBinding{Kind: engineexecution.ConfigResourceKindWordlist},
		},
	)
	definition.Execution.ConfigSections[1].Params = append(definition.Execution.ConfigSections[1].Params,
		engineexecution.ParamDefinition{
			Key: "disabled-wordlist", Type: engineexecution.ParamTypeString, Default: "disabled.txt", MinLength: intPointer(1),
			Resource: &engineexecution.ParamResourceBinding{Kind: engineexecution.ConfigResourceKindWordlist},
		},
	)
	disabledDefinition := testPlanTaskDefinition(nil, nil, true)
	disabledDefinition.EngineID = "engine.lunafox.disabled"

	packages := &configResourcePackageReaderStub{packages: map[string]ScanCreateEnginePackage{
		definition.EngineID:         {Package: PlanTaskPackage{Definition: definition}},
		disabledDefinition.EngineID: {Package: PlanTaskPackage{Definition: disabledDefinition}},
	}}
	resources := &configResourceResolverStub{}
	validator, err := NewWorkflowConfigResourceValidator(packages, resources)
	if err != nil {
		t.Fatalf("NewWorkflowConfigResourceValidator() error = %v", err)
	}
	request := WorkflowConfigResourceValidationRequest{
		Workflow: ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{
			StageID: "stage", Steps: []ScanCreateWorkflowStep{
				{StepID: "first", EngineID: definition.EngineID},
				{StepID: "disabled", EngineID: disabledDefinition.EngineID},
			},
		}}},
		Configuration: map[string]any{"steps": map[string]any{
			"first": map[string]any{"enabled": true, "engineConfig": map[string]any{
				"scan":  map[string]any{"enabled": true, "threads": 10, "wordlist": testCanonicalWordlistResource, "exclude": testSecondaryWordlistResource},
				"alpha": map[string]any{"enabled": false},
			}},
			"disabled": map[string]any{"enabled": false},
		}},
	}

	if err := validator.Validate(context.Background(), request); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !reflect.DeepEqual(packages.requests, []string{definition.EngineID}) {
		t.Fatalf("package reads = %#v, want only enabled Step", packages.requests)
	}
	want := map[string]ConfigResourceResolveRequest{
		testCanonicalWordlistResource: {
			Field:        `configuration.steps["first"].engineConfig.scan.wordlist`,
			ResourceKind: "wordlist", ResourceName: testCanonicalWordlistResource,
		},
		testSecondaryWordlistResource: {
			Field:        `configuration.steps["first"].engineConfig.scan.exclude`,
			ResourceKind: "wordlist", ResourceName: testSecondaryWordlistResource,
		},
	}
	if len(resources.requests) != len(want) {
		t.Fatalf("resource reads = %#v", resources.requests)
	}
	for _, got := range resources.requests {
		if expected, ok := want[got.ResourceName]; !ok || got != expected {
			t.Fatalf("unexpected resource request: %#v", got)
		}
	}
}

func TestWorkflowConfigResourceValidatorPreservesTypedFailure(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, true)
	packages := &configResourcePackageReaderStub{packages: map[string]ScanCreateEnginePackage{
		definition.EngineID: {Package: planTaskExactPackage(definition)},
	}}
	diagnostic := errors.New("catalog connection refused")
	typed := NewConfigResourceValidationUnavailableError(
		`configuration.steps["step"].engineConfig.scan.wordlist`, "wordlist", testCanonicalWordlistResource, diagnostic,
	)
	resources := &configResourceResolverStub{err: typed}
	validator, _ := NewWorkflowConfigResourceValidator(packages, resources)
	err := validator.Validate(context.Background(), WorkflowConfigResourceValidationRequest{
		Workflow: ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{
			StageID: "stage", Steps: []ScanCreateWorkflowStep{{StepID: "step", EngineID: definition.EngineID}},
		}}},
		Configuration: map[string]any{"steps": map[string]any{"step": map[string]any{
			"enabled": true, "engineConfig": map[string]any{
				"scan":  map[string]any{"enabled": true, "threads": 10, "wordlist": testCanonicalWordlistResource},
				"alpha": map[string]any{"enabled": false},
			},
		}}},
	})
	var got *ConfigResourceValidationError
	if !errors.As(err, &got) || got != typed || !errors.Is(err, diagnostic) {
		t.Fatalf("Validate() error = %v, want exact typed failure", err)
	}
}

func TestWorkflowConfigResourceValidatorClassifiesUntypedResolverFailureAsInternal(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, true)
	packages := &configResourcePackageReaderStub{packages: map[string]ScanCreateEnginePackage{
		definition.EngineID: {Package: planTaskExactPackage(definition)},
	}}
	diagnostic := errors.New("unexpected resolver implementation failure")
	resources := &configResourceResolverStub{err: diagnostic}
	validator, _ := NewWorkflowConfigResourceValidator(packages, resources)
	err := validator.Validate(context.Background(), WorkflowConfigResourceValidationRequest{
		Workflow: ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{
			StageID: "stage", Steps: []ScanCreateWorkflowStep{{StepID: "step", EngineID: definition.EngineID}},
		}}},
		Configuration: map[string]any{"steps": map[string]any{"step": map[string]any{
			"enabled": true, "engineConfig": map[string]any{
				"scan":  map[string]any{"enabled": true, "threads": 10, "wordlist": testCanonicalWordlistResource},
				"alpha": map[string]any{"enabled": false},
			},
		}}},
	})
	var typed *ConfigResourceValidationError
	if !errors.As(err, &typed) || typed.ConfigResourceValidationCause() != string(ConfigResourceInternal) || !errors.Is(err, diagnostic) {
		t.Fatalf("Validate() error = %v, want typed internal cause", err)
	}
}

func TestWorkflowConfigResourceValidatorRequiresProductionDependencies(t *testing.T) {
	if _, err := NewWorkflowConfigResourceValidator(nil, &configResourceResolverStub{}); err == nil {
		t.Fatal("expected missing package reader to fail construction")
	}
	if _, err := NewWorkflowConfigResourceValidator(&configResourcePackageReaderStub{}, nil); err == nil {
		t.Fatal("expected missing resource resolver to fail construction")
	}
	var validator *WorkflowConfigResourceValidator
	var typed *ConfigResourceValidationError
	if err := validator.Validate(context.Background(), WorkflowConfigResourceValidationRequest{}); !errors.As(err, &typed) || typed.ConfigResourceValidationCause() != string(ConfigResourceInternal) {
		t.Fatalf("nil validator error = %v, want typed internal failure", err)
	}
}
