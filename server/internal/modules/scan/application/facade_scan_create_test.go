package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
)

func TestWrapScanInvalidWorkflow_PreservesSentinelAndDetail(t *testing.T) {
	source := invalidScanWorkflowf("workflow must use scanWorkflows/{workflow}")
	err := wrapScanInvalidScanWorkflow(source)
	if !errors.Is(err, ErrScanInvalidScanWorkflow) {
		t.Fatalf("expected ErrScanInvalidScanWorkflow sentinel, got: %v", err)
	}
	if !strings.Contains(err.Error(), "scanWorkflows/{workflow}") {
		t.Fatalf("expected scan workflow detail to be preserved, got: %v", err)
	}
}

func TestWrapScanInvalidWorkflowFallbackToSentinel(t *testing.T) {
	err := wrapScanInvalidScanWorkflow(ErrCreateInvalidScanWorkflow)
	if !errors.Is(err, ErrScanInvalidScanWorkflow) {
		t.Fatalf("expected ErrScanInvalidScanWorkflow sentinel, got: %v", err)
	}
	if err.Error() != ErrScanInvalidScanWorkflow.Error() {
		t.Fatalf("expected sentinel-only error, got %v", err)
	}
}

func TestScanFacadePreservesConfigResourceFailuresForQuickAndBatch(t *testing.T) {
	definition := testPlanTaskDefinition(nil, nil, true)
	exact := planTaskExactPackage(definition)
	diagnostic := errors.New("catalog unavailable")
	typed := NewConfigResourceValidationUnavailableError(
		`configuration.steps["step"].engineConfig.scan.wordlist`, "wordlist", "dns.txt", diagnostic,
	)
	resources := &planTaskWordlistReaderStub{err: typed}
	packages := &planTaskPackageReaderStub{packageValue: exact}
	compiler, _ := NewPlanTaskCompiler(packages, resources)
	store := &plannedScanCreateStoreStub{}
	manifest := ScanCreateWorkflowManifest{ScanWorkflowID: "default", Stages: []ScanCreateWorkflowStage{{
		StageID: "stage", Steps: []ScanCreateWorkflowStep{{StepID: "step", EngineID: exact.Identity.EngineID}},
	}}}
	packageReader := scanCreateEnginePackageReaderStub{enginePackages: map[string]ScanCreateEnginePackage{
		exact.Identity.EngineID: scanCreatePackageFromPlanPackage(exact),
	}}
	createService := NewScanCreateService(
		store,
		func(context.Context, int) (*TargetRef, error) {
			return &TargetRef{ID: 17, Name: "example.com", Type: "domain", CreatedAt: time.Now().UTC()}, nil
		},
		func(context.Context, []string) (*QuickTargetResolution, error) {
			return &QuickTargetResolution{Targets: []TargetRef{{ID: 17, Name: "example.com", Type: "domain", CreatedAt: time.Now().UTC()}}, TargetStats: QuickTargetStats{Created: 1}}, nil
		},
		scanCreateWorkflowReaderStub{manifest: manifest},
		packageReader,
	)
	mustConfigurePlanTaskForTest(t, createService, compiler, time.Minute)
	facade := NewScanFacade(nil, nil, nil, nil, nil, createService)
	configuration := completeScanCreateConfigurationForTest(t, map[string]engineexecution.ExecutionDefinition{"step": definition.Execution})

	tests := []struct {
		name string
		call func() error
	}{
		{name: "quick", call: func() error {
			_, err := facade.CreateQuick(context.Background(), &CreateQuickRequest{Targets: []string{"example.com"}, ScanWorkflow: "scanWorkflows/default", Configuration: configuration, InputSource: InputSourceScanSnapshot, TriggerType: ScanTriggerTypeManual})
			return err
		}},
		{name: "batch", call: func() error {
			_, err := facade.CreateBatch(context.Background(), &CreateBatchRequest{Requests: []CreateBatchItem{{TargetID: 17}}, ScanWorkflow: "scanWorkflows/default", Configuration: configuration, InputSource: InputSourceScanSnapshot, TriggerType: ScanTriggerTypeManual})
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.call()
			var got *ConfigResourceValidationError
			if !errors.As(err, &got) || got != typed || !errors.Is(err, diagnostic) {
				t.Fatalf("facade error = %v, want exact typed cause", err)
			}
			if errors.Is(err, ErrScanEngineConfigInvalid) {
				t.Fatalf("typed resource failure collapsed into Engine config error: %v", err)
			}
			if store.committed != nil {
				t.Fatalf("facade resource failure committed Scan: %#v", store.committed)
			}
		})
	}
}
