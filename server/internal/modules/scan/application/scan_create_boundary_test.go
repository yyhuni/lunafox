package application

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestScanApplicationDoesNotOwnEngineCatalogLoading(t *testing.T) {
	dir := scanApplicationDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read scan application dir: %v", err)
	}

	forbidden := []string{
		"internal/modules/scan/infrastructure",
		"enginepackagecatalog.LoadEnginePackagesFromDefinitionRoot",
		"DefaultBuiltinEngineDefinitionRoot",
		"server/internal/scanworkflow/manifest",
		"workflowmanifest.GetManifest",
		"repoRoot()",
		"func repoRoot",
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		source := readScanApplicationFile(t, dir, entry.Name())
		for _, snippet := range forbidden {
			if strings.Contains(source, snippet) {
				t.Fatalf("scan/application must consume catalog data through ports; %s contains %q", entry.Name(), snippet)
			}
		}
	}
}

func TestScanApplicationFileResponsibilitiesStayResourceScoped(t *testing.T) {
	dir := scanApplicationDir(t)

	assertContains := func(fileName, snippet string) {
		t.Helper()
		if !strings.Contains(readScanApplicationFile(t, dir, fileName), snippet) {
			t.Fatalf("%s must contain %q", fileName, snippet)
		}
	}
	assertNotContains := func(fileName, snippet string) {
		t.Helper()
		if strings.Contains(readScanApplicationFile(t, dir, fileName), snippet) {
			t.Fatalf("%s must not contain %q", fileName, snippet)
		}
	}

	assertContains("scan_create_catalog_ports.go", "type ScanCreateEnginePackage struct")
	assertContains("scan_create_catalog_ports.go", "type ScanCreateWorkflowReader interface")
	assertNotContains("scan_create_service.go", "type ScanCreateEnginePackage struct")
	assertNotContains("scan_create_service.go", "type ScanCreateWorkflowReader interface")

	assertContains("scan_errors.go", "ErrScanNotFound")
	assertContains("scan_task_errors.go", "ErrScanTaskNotFound")
	assertContains("facade_task.go", "type ScanTaskFacade struct")
	assertContains("facade_task.go", "func NewScanTaskFacade")
	assertNotContains("facade_task.go", "type TaskExecutionFacade struct")
	assertNotContains("facade_task.go", "func NewTaskExecutionFacade")
	assertContains("scan_task_bridge_service.go", "type ScanTaskBridgeService struct")
	assertNotContains("scan_task_bridge_service.go", "type TaskExecutionBridgeService struct")
	assertContains("facade_task_status.go", "func (service *ScanTaskFacade) ReportTerminalTaskResult")
	assertNotContains("facade_task_status.go", "func (service *ScanTaskFacade) UpdateScanTaskStatus")
	assertNotContains("facade_task_status.go", "func (service *ScanTaskFacade) UpdateTaskExecutionStatus")
	assertContains("scan_task_command_ports.go", "CommitScanTaskTerminalStatusForSession")
	assertNotContains("scan_task_command_ports.go", "UpdateScanTaskStatus")
	assertNotContains("scan_task_command_ports.go", "UpdateTaskExecutionStatus")
	assertContains("scan_stop_ports.go", "type ScanStopStore interface")
	assertNotContains("task_lifecycle_ports.go", "type TaskExecutionCanceller interface")
	assertContains("scan_task_runtime_scan_store_ports.go", "type ScanTaskRuntimeScanStore interface")
	assertNotContains("scan_task_runtime_scan_store_ports.go", "type ScanTaskScanStore interface")
	assertNotContains("scan_task_runtime_scan_store_ports.go", "type TaskExecutionScanStore interface")
	assertContains("scan_task_runtime_scan_query_ports.go", "type ScanTaskRuntimeScanQueryStore interface")
	assertNotContains("scan_task_runtime_scan_query_ports.go", "GetScanTaskScanByID")
	assertNotContains("scan_task_runtime_scan_query_ports.go", "type TaskExecutionScanQueryStore interface")
	assertNotContains("facade_scan.go", "errors.New(")
	assertNotContains("facade_task.go", "errors.New(")
}

func scanApplicationDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Dir(filename)
}

func readScanApplicationFile(t *testing.T, dir string, fileName string) string {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		t.Fatalf("read %s: %v", fileName, err)
	}
	return string(payload)
}
