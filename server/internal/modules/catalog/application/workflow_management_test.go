package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type workflowStoreStub struct {
	workflows map[string]catalogdomain.ManagedScanWorkflow
	listCalls []catalogdomain.ScanWorkflowListFilter
}

func (stub *workflowStoreStub) GetScanWorkflowByID(id string) (*catalogdomain.ManagedScanWorkflow, error) {
	workflow, ok := stub.workflows[id]
	if !ok {
		return nil, catalogdomain.ErrScanWorkflowNotFound
	}
	return &workflow, nil
}
func (stub *workflowStoreStub) ListScanWorkflows(filter catalogdomain.ScanWorkflowListFilter) ([]catalogdomain.ManagedScanWorkflow, int64, error) {
	stub.listCalls = append(stub.listCalls, filter)
	out := make([]catalogdomain.ManagedScanWorkflow, 0, len(stub.workflows))
	for _, workflow := range stub.workflows {
		out = append(out, workflow)
	}
	return out, int64(len(out) + 1), nil
}
func (stub *workflowStoreStub) CreateScanWorkflow(workflow *catalogdomain.ManagedScanWorkflow) error {
	if _, exists := stub.workflows[workflow.ScanWorkflowID]; exists {
		return errors.New("duplicate")
	}
	stub.workflows[workflow.ScanWorkflowID] = *workflow
	return nil
}
func (stub *workflowStoreStub) FindScanWorkflowByRequestID(requestID string) (*catalogdomain.ManagedScanWorkflow, error) {
	for _, workflow := range stub.workflows {
		if workflow.RequestID == requestID {
			copy := workflow
			return &copy, nil
		}
	}
	return nil, nil
}
func (stub *workflowStoreStub) UpdateUserScanWorkflow(workflow *catalogdomain.ManagedScanWorkflow, expectedVersion int64) (bool, error) {
	current, ok := stub.workflows[workflow.ScanWorkflowID]
	if !ok || current.Version != expectedVersion || current.IsBuiltin {
		return false, nil
	}
	workflow.Version = expectedVersion + 1
	stub.workflows[workflow.ScanWorkflowID] = *workflow
	return true, nil
}

type workflowEngineResolverStub map[string]bool

func (stub workflowEngineResolverStub) HasEngine(_ context.Context, engineID string) (bool, error) {
	return stub[engineID], nil
}

func TestScanWorkflowManagementCreateIsIdempotentAndGeneratesUserID(t *testing.T) {
	store := &workflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{}}
	service, err := NewScanWorkflowManagementService(store, workflowEngineResolverStub{"engine.lunafox.discovery": true})
	if err != nil {
		t.Fatal(err)
	}
	requestID := uuid.NewString()
	input := CreateManagedScanWorkflowInput{RequestID: requestID, DisplayName: "Discovery", Description: "Run discovery.", Stages: workflowTestStages()}
	first, err := service.CreateScanWorkflow(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if err := scanworkflow.ValidateUserScanWorkflowID(first.ScanWorkflowID); err != nil {
		t.Fatalf("generated id %q: %v", first.ScanWorkflowID, err)
	}
	second, err := service.CreateScanWorkflow(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if second.ScanWorkflowID != first.ScanWorkflowID || len(store.workflows) != 1 {
		t.Fatalf("idempotent create failed: first=%+v second=%+v", first, second)
	}
}

func TestScanWorkflowManagementPreservesExplicitProfileDefaultsAndETagDigest(t *testing.T) {
	store := &workflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{}}
	service, err := NewScanWorkflowManagementService(store, workflowEngineResolverStub{
		"engine.lunafox.discovery": true,
		"engine.lunafox.port_scan": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	requestID := uuid.NewString()
	stages := []scanworkflow.Stage{{StageID: "discovery", Steps: []scanworkflow.Step{
		{StepID: "discover", EngineID: "engine.lunafox.discovery", ProfileDefaultEnabled: true},
		{StepID: "ports", EngineID: "engine.lunafox.port_scan", ProfileDefaultEnabled: false},
	}}}
	created, err := service.CreateScanWorkflow(context.Background(), CreateManagedScanWorkflowInput{
		RequestID: requestID, DisplayName: "Mixed defaults", Description: "", Stages: stages,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Stages[0].Steps[0].ProfileDefaultEnabled != true || created.Stages[0].Steps[1].ProfileDefaultEnabled != false {
		t.Fatalf("Create lost explicit Profile defaults: %+v", created.Stages)
	}
	oldETag := created.ETag
	updatedStages := []scanworkflow.Stage{{StageID: "discovery", Steps: []scanworkflow.Step{
		{StepID: "discover", EngineID: "engine.lunafox.discovery", ProfileDefaultEnabled: false},
		{StepID: "ports", EngineID: "engine.lunafox.port_scan", ProfileDefaultEnabled: true},
	}}}
	updated, err := service.UpdateScanWorkflow(context.Background(), UpdateManagedScanWorkflowInput{
		ScanWorkflowID: created.ScanWorkflowID,
		ETag:           oldETag,
		UpdateMask:     []string{"stages"},
		Stages:         &updatedStages,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ETag == oldETag || updated.Stages[0].Steps[0].ProfileDefaultEnabled != false || updated.Stages[0].Steps[1].ProfileDefaultEnabled != true {
		t.Fatalf("Update did not round-trip Profile defaults or advance ETag: old=%s updated=%+v", oldETag, updated)
	}
}

func TestScanWorkflowManagementListBindsTokenToQueryShape(t *testing.T) {
	store := &workflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{"default": builtinWorkflowForApplicationTest(t)}}
	service, err := NewScanWorkflowManagementService(store, workflowEngineResolverStub{"engine.lunafox.discovery": true})
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.ListScanWorkflows(context.Background(), ListManagedScanWorkflowsInput{PageSize: 1, Filter: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if first.NextPageToken == "" {
		t.Fatal("expected next token")
	}
	if _, err := service.ListScanWorkflows(context.Background(), ListManagedScanWorkflowsInput{PageSize: 1, Filter: "changed", PageToken: first.NextPageToken}); !errors.Is(err, ErrInvalidScanWorkflowPageToken) {
		t.Fatalf("query mismatch error = %v", err)
	}
}

func TestScanWorkflowManagementRejectsBuiltinUpdate(t *testing.T) {
	workflow := builtinWorkflowForApplicationTest(t)
	store := &workflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{"default": workflow}}
	service, err := NewScanWorkflowManagementService(store, workflowEngineResolverStub{"engine.lunafox.discovery": true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateScanWorkflow(context.Background(), UpdateManagedScanWorkflowInput{ScanWorkflowID: "default"}); !errors.Is(err, ErrScanWorkflowImmutable) {
		t.Fatalf("built-in update error = %v", err)
	}
}

func TestScanWorkflowManagementKeepsUnavailableWorkflowReadable(t *testing.T) {
	workflow := builtinWorkflowForApplicationTest(t)
	store := &workflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{"default": workflow}}
	service, err := NewScanWorkflowManagementService(store, workflowEngineResolverStub{"engine.lunafox.discovery": false})
	if err != nil {
		t.Fatal(err)
	}
	item, err := service.GetScanWorkflow(context.Background(), "default")
	if err != nil {
		t.Fatalf("GetScanWorkflow must remain readable: %v", err)
	}
	if item.IsExecutable {
		t.Fatalf("unavailable Engine must make the read projection non-executable: %+v", item)
	}
	list, err := service.ListScanWorkflows(context.Background(), ListManagedScanWorkflowsInput{PageSize: 10})
	if err != nil {
		t.Fatalf("ListScanWorkflows must remain readable: %v", err)
	}
	if len(list.Results) != 1 || list.Results[0].IsExecutable {
		t.Fatalf("unavailable Workflow list projection is wrong: %+v", list.Results)
	}
}

func TestScanWorkflowManagementExecutableProjectionDependsOnlyOnEngineAvailability(t *testing.T) {
	workflow := builtinWorkflowForApplicationTest(t)
	store := &workflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{"default": workflow}}
	service, err := NewScanWorkflowManagementService(store, workflowEngineResolverStub{"engine.lunafox.discovery": true})
	if err != nil {
		t.Fatal(err)
	}
	item, err := service.GetScanWorkflow(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	if !item.IsExecutable {
		t.Fatalf("installed Engines must keep Workflow executable independently of deployment resource state: %+v", item)
	}
}

func workflowTestStages() []scanworkflow.Stage {
	return []scanworkflow.Stage{{StageID: "discovery", Steps: []scanworkflow.Step{{StepID: "discover", EngineID: "engine.lunafox.discovery", ProfileDefaultEnabled: true}}}}
}
func builtinWorkflowForApplicationTest(t *testing.T) catalogdomain.ManagedScanWorkflow {
	t.Helper()
	stages := workflowTestStages()
	digest, err := scanworkflow.CanonicalWorkflowDigest("default", "Default", "Run discovery.", stages)
	if err != nil {
		t.Fatal(err)
	}
	return catalogdomain.ManagedScanWorkflow{ScanWorkflowID: "default", DisplayName: "Default", Description: "Run discovery.", Stages: stages, IsBuiltin: true, DefinitionDigest: digest, Version: 1}
}
