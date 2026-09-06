package repository

import (
	"testing"

	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSynchronizeBuiltinScanWorkflowsUsesStableIDAndSkipsUnchangedDigest(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-workflow-sync?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ScanWorkflow{}, &model.Engine{}); err != nil {
		t.Fatal(err)
	}
	installBuiltinWorkflowTestEngine(t, db)
	repository := NewScanWorkflowRepository(db)
	first := builtinWorkflowForRepositoryTest(t, "Default Scan")
	if err := repository.SynchronizeBuiltinScanWorkflows([]catalogdomain.ManagedScanWorkflow{first}); err != nil {
		t.Fatal(err)
	}
	stored, err := repository.GetScanWorkflowByID("default")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Version != 1 || stored.DisplayName != "Default Scan" {
		t.Fatalf("unexpected first synchronization: %+v", stored)
	}
	if err := repository.SynchronizeBuiltinScanWorkflows([]catalogdomain.ManagedScanWorkflow{first}); err != nil {
		t.Fatal(err)
	}
	unchanged, err := repository.GetScanWorkflowByID("default")
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Version != 1 || unchanged.UpdateTime != stored.UpdateTime {
		t.Fatalf("unchanged synchronization rewrote workflow: before=%+v after=%+v", stored, unchanged)
	}

	changed := builtinWorkflowForRepositoryTest(t, "Updated Default Scan")
	if err := repository.SynchronizeBuiltinScanWorkflows([]catalogdomain.ManagedScanWorkflow{changed}); err != nil {
		t.Fatal(err)
	}
	updated, err := repository.GetScanWorkflowByID("default")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 || updated.DisplayName != "Updated Default Scan" || updated.ScanWorkflowID != "default" {
		t.Fatalf("changed synchronization did not update in place: %+v", updated)
	}
	if err := repository.SynchronizeBuiltinScanWorkflows([]catalogdomain.ManagedScanWorkflow{first}); err != nil {
		t.Fatal(err)
	}
	rolledBack, err := repository.GetScanWorkflowByID("default")
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack.Version != 3 || rolledBack.DisplayName != "Default Scan" || rolledBack.ScanWorkflowID != "default" {
		t.Fatalf("release rollback did not update the stable resource in place: %+v", rolledBack)
	}

	if err := db.Where("scan_workflow_id = ?", "default").Delete(&model.ScanWorkflow{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repository.SynchronizeBuiltinScanWorkflows([]catalogdomain.ManagedScanWorkflow{first}); err != nil {
		t.Fatal(err)
	}
	restored, err := repository.GetScanWorkflowByID("default")
	if err != nil || restored.ScanWorkflowID != "default" || restored.DisplayName != "Default Scan" {
		t.Fatalf("missing built-in row was not restored: %+v, %v", restored, err)
	}
}

func TestSynchronizeBuiltinScanWorkflowsRejectsUserOwnershipCollision(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-workflow-collision?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ScanWorkflow{}, &model.Engine{}); err != nil {
		t.Fatal(err)
	}
	installBuiltinWorkflowTestEngine(t, db)
	if err := db.Create(&model.ScanWorkflow{ScanWorkflowID: "default", DisplayName: "User", Description: "User-owned collision.", Stages: []byte(`[]`), IsBuiltin: false, Version: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := NewScanWorkflowRepository(db).SynchronizeBuiltinScanWorkflows([]catalogdomain.ManagedScanWorkflow{builtinWorkflowForRepositoryTest(t, "Default Scan")}); err == nil {
		t.Fatal("expected user ownership collision to reject synchronization")
	}
}

func TestSynchronizeBuiltinScanWorkflowsRejectsUnavailableEngineBeforeWriting(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-workflow-missing-engine?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ScanWorkflow{}, &model.Engine{}); err != nil {
		t.Fatal(err)
	}
	repository := NewScanWorkflowRepository(db)
	if err := repository.SynchronizeBuiltinScanWorkflows([]catalogdomain.ManagedScanWorkflow{builtinWorkflowForRepositoryTest(t, "Default Scan")}); err == nil {
		t.Fatal("expected unavailable Engine to reject synchronization")
	}
	var count int64
	if err := db.Model(&model.ScanWorkflow{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("unavailable Engine left persisted workflows: %d", count)
	}
}

func TestSynchronizeBuiltinScanWorkflowsRollsBackAllWritesWhenAnyEngineIsUnavailable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-workflow-sync-rollback?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ScanWorkflow{}, &model.Engine{}); err != nil {
		t.Fatal(err)
	}
	installBuiltinWorkflowTestEngine(t, db)
	repository := NewScanWorkflowRepository(db)
	baseline := builtinWorkflowForRepositoryTest(t, "Baseline")
	if err := repository.SynchronizeBuiltinScanWorkflows([]catalogdomain.ManagedScanWorkflow{baseline}); err != nil {
		t.Fatal(err)
	}
	changed := builtinWorkflowForRepositoryTest(t, "Should not persist")
	invalid := builtinWorkflowForRepositoryTest(t, "Unavailable engine")
	invalid.ScanWorkflowID = "secondary"
	invalid.Stages[0].Steps[0].EngineID = "engine.lunafox.unavailable"
	digest, err := scanworkflow.CanonicalWorkflowDigest(invalid.ScanWorkflowID, invalid.DisplayName, invalid.Description, invalid.Stages)
	if err != nil {
		t.Fatal(err)
	}
	invalid.DefinitionDigest = digest
	if err := repository.SynchronizeBuiltinScanWorkflows([]catalogdomain.ManagedScanWorkflow{changed, invalid}); err == nil {
		t.Fatal("expected synchronization failure")
	}
	stored, err := repository.GetScanWorkflowByID("default")
	if err != nil || stored.DisplayName != "Baseline" || stored.Version != 1 {
		t.Fatalf("failed synchronization partially updated default: %+v, %v", stored, err)
	}
}

func installBuiltinWorkflowTestEngine(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Create(&model.Engine{EngineID: "engine.lunafox.discovery", Publisher: "lunafox", PackageVersion: "1.0.0", ArtifactRef: "registry.example/lunafox/discovery:1.0.0", PackageDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111", Manifest: []byte(`{}`)}).Error; err != nil {
		t.Fatal(err)
	}
}

func builtinWorkflowForRepositoryTest(t *testing.T, displayName string) catalogdomain.ManagedScanWorkflow {
	t.Helper()
	stages := []scanworkflow.Stage{{StageID: "discovery", Steps: []scanworkflow.Step{{StepID: "discover", EngineID: "engine.lunafox.discovery", ProfileDefaultEnabled: true}}}}
	digest, err := scanworkflow.CanonicalWorkflowDigest("default", displayName, "Run discovery.", stages)
	if err != nil {
		t.Fatal(err)
	}
	return catalogdomain.ManagedScanWorkflow{
		ScanWorkflowID: "default", DisplayName: displayName, Description: "Run discovery.", Stages: stages,
		IsBuiltin: true, DefinitionDigest: digest, Version: 1,
	}
}
