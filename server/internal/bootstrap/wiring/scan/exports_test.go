package scanwiring

import (
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/installedengines"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type installedPackageQueryStub struct{}

func (*installedPackageQueryStub) ListInstalledEnginePackages() ([]installedengines.ResolvedInstalledEnginePackage, error) {
	return nil, nil
}

func TestProductionPlanTaskLimitsProviderUsesCodeOwnedDefaults(t *testing.T) {
	provider := productionPlanTaskLimitsProvider()
	if provider == nil {
		t.Fatal("production PlanTask limits provider is missing")
	}
	limits := provider.Limits()
	if limits.MaxExecutionDuration != 14*24*time.Hour || limits.MaxExecutionDuration/time.Second != 1_209_600 {
		t.Fatalf("production max execution duration = %s, want exact 14d code-owned default", limits.MaxExecutionDuration)
	}
	if limits.ProgressMessageMaxBytes == 0 || limits.ResultBatchMaxItems == 0 || limits.ResultBatchMaxBytes == 0 {
		t.Fatalf("production PlanTask limits are not positive: %#v", limits)
	}
}

func (*installedPackageQueryStub) GetInstalledEnginePackage(string) (installedengines.ResolvedInstalledEnginePackage, error) {
	return installedengines.ResolvedInstalledEnginePackage{}, nil
}

func TestNewPlanTaskCompilerFailsFastWhenProductionDependencyIsMissing(t *testing.T) {
	if _, err := newPlanTaskCompiler(nil, nil); err == nil || !strings.Contains(err.Error(), "exact Engine Package reader") {
		t.Fatalf("missing installed package query error = %v", err)
	}

	query := &installedPackageQueryStub{}
	if _, err := newPlanTaskCompiler(query, nil); err == nil || !strings.Contains(err.Error(), "config resource resolver") {
		t.Fatalf("missing config resource resolver error = %v", err)
	}
}

func TestNewPlanTaskCompilerRejectsAmbiguousCloudflareAccelerationConfiguration(t *testing.T) {
	query := &installedPackageQueryStub{}
	if _, err := newPlanTaskCompiler(query, nil, true, false); err == nil || !strings.Contains(err.Error(), "at most one Cloudflare acceleration") {
		t.Fatalf("ambiguous Cloudflare acceleration error = %v", err)
	}
}

func TestNewScanTaskStoreAdapterRetainsDiagnosticTerminalCapability(t *testing.T) {
	store := NewScanTaskStoreAdapter(scanrepo.NewScanTaskRepository(nil))
	if _, ok := store.(scanapp.EngineDiagnosticTerminalTaskStore); !ok {
		t.Fatal("scan task store adapter dropped the required diagnostic terminal capability")
	}
}

func TestNewConfigResourceValidationComponentsRejectsIncompleteAssembly(t *testing.T) {
	if _, _, err := NewConfigResourceValidationComponents(nil, nil); err == nil || !strings.Contains(err.Error(), "installed Engine Package") {
		t.Fatalf("missing installed package query error = %v", err)
	}
	if _, _, err := NewConfigResourceValidationComponents(&installedPackageQueryStub{}, nil); err == nil || !strings.Contains(err.Error(), "wordlist catalog") {
		t.Fatalf("missing wordlist Catalog error = %v", err)
	}
}
