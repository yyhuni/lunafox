package repository

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScanWorkflowRepositoryListsFilteredWorkflowsInStableOrder(t *testing.T) {
	repository := newScanWorkflowRepositoryForTest(t)
	for _, workflow := range []catalogdomain.ManagedScanWorkflow{
		builtinWorkflowForRepositoryTest(t, "Zulu built-in"),
		userWorkflowForRepositoryTest(t, "wf-123e4567-e89b-12d3-a456-426614174001", "Alpha workflow", "matches alpha"),
		userWorkflowForRepositoryTest(t, "wf-123e4567-e89b-12d3-a456-426614174002", "Bravo workflow", "matches bravo"),
	} {
		workflow := workflow
		if err := repository.CreateScanWorkflow(&workflow); err != nil {
			t.Fatalf("create %q: %v", workflow.ScanWorkflowID, err)
		}
	}

	first, total, err := repository.ListScanWorkflows(catalogdomain.ScanWorkflowListFilter{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(first) != 2 || first[0].ScanWorkflowID != "default" || first[1].DisplayName != "Alpha workflow" {
		t.Fatalf("first page = %+v, total = %d", first, total)
	}
	second, total, err := repository.ListScanWorkflows(catalogdomain.ScanWorkflowListFilter{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(second) != 1 || second[0].DisplayName != "Bravo workflow" {
		t.Fatalf("second page = %+v, total = %d", second, total)
	}
	filtered, total, err := repository.ListScanWorkflows(catalogdomain.ScanWorkflowListFilter{Page: 1, PageSize: 10, Filter: " BRAVO "})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(filtered) != 1 || filtered[0].ScanWorkflowID != "wf-123e4567-e89b-12d3-a456-426614174002" {
		t.Fatalf("filtered page = %+v, total = %d", filtered, total)
	}
}

func TestScanWorkflowRepositoryPersistsRequestIDAndUsesCompareAndSwap(t *testing.T) {
	repository := newScanWorkflowRepositoryForTest(t)
	requestID := uuid.NewString()
	workflow := userWorkflowForRepositoryTest(t, "wf-123e4567-e89b-12d3-a456-426614174003", "Original", "")
	workflow.RequestID = requestID
	if err := repository.CreateScanWorkflow(&workflow); err != nil {
		t.Fatal(err)
	}
	replayed, err := repository.FindScanWorkflowByRequestID(requestID)
	if err != nil || replayed == nil || replayed.ScanWorkflowID != workflow.ScanWorkflowID {
		t.Fatalf("requestId lookup = %+v, %v", replayed, err)
	}

	workflow.DisplayName = "Updated"
	updated, err := repository.UpdateUserScanWorkflow(&workflow, 1)
	if err != nil || !updated || workflow.Version != 2 {
		t.Fatalf("first CAS = updated %t version %d error %v", updated, workflow.Version, err)
	}
	workflow.Description = "stale update"
	updated, err = repository.UpdateUserScanWorkflow(&workflow, 1)
	if err != nil || updated {
		t.Fatalf("stale CAS = updated %t error %v", updated, err)
	}
	stored, err := repository.GetScanWorkflowByID(workflow.ScanWorkflowID)
	if err != nil || stored.DisplayName != "Updated" || stored.Description != "" || stored.Version != 2 {
		t.Fatalf("stored workflow = %+v, %v", stored, err)
	}
}

func newScanWorkflowRepositoryForTest(t *testing.T) *ScanWorkflowRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "-")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ScanWorkflow{}); err != nil {
		t.Fatal(err)
	}
	return NewScanWorkflowRepository(db)
}

func userWorkflowForRepositoryTest(t *testing.T, id, displayName, description string) catalogdomain.ManagedScanWorkflow {
	t.Helper()
	workflow := catalogdomain.ManagedScanWorkflow{
		ScanWorkflowID: id,
		DisplayName:    displayName,
		Description:    description,
		Stages: []scanworkflow.Stage{{StageID: "discovery", Steps: []scanworkflow.Step{{
			StepID: "discover", EngineID: "engine.lunafox.discovery", ProfileDefaultEnabled: true,
		}}}},
		Version: 1,
	}
	if err := workflow.Validate(); err != nil {
		t.Fatal(err)
	}
	return workflow
}
