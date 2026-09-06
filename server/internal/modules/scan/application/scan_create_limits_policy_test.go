package application

import (
	"strings"
	"testing"
	"time"

	engineexecution "github.com/yyhuni/lunafox/engine-go/protocol"
)

func TestDefaultPlanTaskLimitsFitEngineProtocolContext(t *testing.T) {
	limits := DefaultPlanTaskLimitsProvider().Limits()
	if err := engineexecution.ValidateLimits(&engineexecution.ExecutionLimits{
		ProgressMessageMaxBytes: limits.ProgressMessageMaxBytes,
		ResultBatchMaxItems:     limits.ResultBatchMaxItems,
		ResultBatchMaxBytes:     limits.ResultBatchMaxBytes,
	}); err != nil {
		t.Fatalf("default PlanTask limits must fit Engine Context: %v", err)
	}
}

func TestConfigurePlanTaskRequiresCompilerAndValidServerLimitsProvider(t *testing.T) {
	validCompiler, err := NewPlanTaskCompiler(&planTaskPackageReaderStub{}, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler failed: %v", err)
	}

	tests := []struct {
		name     string
		compiler *PlanTaskCompiler
		provider PlanTaskLimitsProvider
		want     string
	}{
		{name: "missing compiler", provider: FixedPlanTaskLimitsProvider(3 * time.Second), want: "compiler is required"},
		{name: "missing provider", compiler: validCompiler, want: "limits provider is required"},
		{name: "zero execution duration", compiler: validCompiler, provider: mutatedPlanTaskLimitsProvider(func(limits *PlanTaskLimits) {
			limits.MaxExecutionDuration = 0
		}), want: "all execution limits must be positive"},
		{name: "zero progress bytes", compiler: validCompiler, provider: mutatedPlanTaskLimitsProvider(func(limits *PlanTaskLimits) {
			limits.ProgressMessageMaxBytes = 0
		}), want: "all execution limits must be positive"},
		{name: "zero result items", compiler: validCompiler, provider: mutatedPlanTaskLimitsProvider(func(limits *PlanTaskLimits) {
			limits.ResultBatchMaxItems = 0
		}), want: "all execution limits must be positive"},
		{name: "zero result bytes", compiler: validCompiler, provider: mutatedPlanTaskLimitsProvider(func(limits *PlanTaskLimits) {
			limits.ResultBatchMaxBytes = 0
		}), want: "all execution limits must be positive"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewScanCreateService(nil, nil, nil, nil, nil)
			err := service.ConfigurePlanTask(test.compiler, test.provider)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ConfigurePlanTask error = %v, want %q", err, test.want)
			}
			if service.planTaskCompiler != nil {
				t.Fatal("invalid planning dependencies partially configured the service")
			}
		})
	}
}

func TestPlanTaskLimitsProviderIsResolvedOnceAndIgnoresEnvironment(t *testing.T) {
	validCompiler, err := NewPlanTaskCompiler(&planTaskPackageReaderStub{}, nil)
	if err != nil {
		t.Fatalf("NewPlanTaskCompiler failed: %v", err)
	}
	t.Setenv("LUNAFOX_MAX_EXECUTION_DURATION", "1ns")
	if got := DefaultPlanTaskLimitsProvider().Limits().MaxExecutionDuration; got != 14*24*time.Hour || got/time.Second != 1_209_600 {
		t.Fatalf("environment changed default max execution duration to %s", got)
	}

	provided := staticPlanTaskLimitsProvider{limits: defaultPlanTaskLimits()}
	provided.limits.MaxExecutionDuration = 3 * time.Second
	service := NewScanCreateService(nil, nil, nil, nil, nil)
	if err := service.ConfigurePlanTask(validCompiler, &provided); err != nil {
		t.Fatalf("ConfigurePlanTask failed: %v", err)
	}
	provided.limits.MaxExecutionDuration = 5 * time.Second

	if service.planTaskLimits.MaxExecutionDuration != 3*time.Second {
		t.Fatalf("stored max execution duration = %s, want immutable 3s snapshot", service.planTaskLimits.MaxExecutionDuration)
	}
	if err := service.ConfigurePlanTask(validCompiler, FixedPlanTaskLimitsProvider(5*time.Second)); err == nil || !strings.Contains(err.Error(), "already configured") {
		t.Fatalf("second ConfigurePlanTask error = %v, want immutable configuration rejection", err)
	}
	if service.planTaskLimits.MaxExecutionDuration != 3*time.Second {
		t.Fatalf("reconfiguration changed max execution duration to %s", service.planTaskLimits.MaxExecutionDuration)
	}
}

func mutatedPlanTaskLimitsProvider(mutate func(*PlanTaskLimits)) PlanTaskLimitsProvider {
	limits := defaultPlanTaskLimits()
	mutate(&limits)
	return staticPlanTaskLimitsProvider{limits: limits}
}

func mustConfigurePlanTaskForTest(t *testing.T, service *ScanCreateService, compiler *PlanTaskCompiler, duration time.Duration) *ScanCreateService {
	t.Helper()
	if err := service.ConfigurePlanTask(compiler, FixedPlanTaskLimitsProvider(duration)); err != nil {
		t.Fatalf("ConfigurePlanTask failed: %v", err)
	}
	return service
}
