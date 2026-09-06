package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type scheduledScanStoreCapture struct {
	created        *ScheduledScanCreate
	updated        *ScheduledScanUpdate
	batchUpdates   []ScheduledScanStatusUpdate
	batchErr       error
	loaded         *ScheduledScan
	listed         []ScheduledScan
	overviewQuery  *ScheduledScanOverviewQuery
	overviewResult *ScheduledScanOverviewProjection
}

func (store *scheduledScanStoreCapture) Create(_ context.Context, scan *ScheduledScanCreate) (*ScheduledScan, error) {
	store.created = scan
	return &ScheduledScan{
		ID:             1,
		Name:           scan.Name,
		ScanWorkflowID: scan.ScanWorkflowID,
		Configuration:  scan.Configuration,
		InputSource:    scan.InputSource,
		TargetID:       scan.TargetID,
		OrganizationID: scan.OrganizationID,
		AgentID:        scan.AgentID,
		CronExpression: scan.CronExpression,
		IsEnabled:      scan.IsEnabled,
		CreatedAt:      time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
	}, nil
}

func (store *scheduledScanStoreCapture) Update(_ context.Context, id int, scan *ScheduledScanUpdate) (*ScheduledScan, error) {
	store.updated = scan
	if store.loaded == nil {
		return nil, ErrScheduledScanNotFound
	}
	inputSource := store.loaded.InputSource
	if scan.InputSource != nil {
		inputSource = *scan.InputSource
	}
	return &ScheduledScan{
		ID:             id,
		Name:           valueOr(scan.Name, "daily"),
		ScanWorkflowID: valueOr(scan.ScanWorkflowID, "default"),
		Configuration:  scan.Configuration,
		InputSource:    inputSource,
		TargetID:       scan.TargetID,
		OrganizationID: scan.OrganizationID,
		AgentID:        scan.AgentID,
		CronExpression: valueOr(scan.CronExpression, "0 2 * * *"),
		IsEnabled:      true,
		CreatedAt:      time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
	}, nil
}

func (store *scheduledScanStoreCapture) BatchUpdateStatus(_ context.Context, updates []ScheduledScanStatusUpdate) (int, error) {
	store.batchUpdates = append([]ScheduledScanStatusUpdate(nil), updates...)
	if store.batchErr != nil {
		return 0, store.batchErr
	}
	return len(updates), nil
}

func (store *scheduledScanStoreCapture) List(context.Context, ScheduledScanListQuery) ([]ScheduledScan, int64, error) {
	return append([]ScheduledScan(nil), store.listed...), int64(len(store.listed)), nil
}

func (store *scheduledScanStoreCapture) GetOverviewSummary(_ context.Context, query ScheduledScanOverviewQuery) (*ScheduledScanOverviewProjection, error) {
	store.overviewQuery = &query
	if store.overviewResult == nil {
		return &ScheduledScanOverviewProjection{UpcomingScheduledScans: []ScheduledScanOverviewUpcoming{}}, nil
	}
	return store.overviewResult, nil
}

func (store *scheduledScanStoreCapture) GetByID(context.Context, int) (*ScheduledScan, error) {
	if store.loaded == nil {
		return nil, ErrScheduledScanNotFound
	}
	copy := *store.loaded
	return &copy, nil
}

type scheduledScanWorkflowStoreStub struct {
	workflows map[string]catalogdomain.ManagedScanWorkflow
	requests  []string
}

func (store *scheduledScanWorkflowStoreStub) GetScanWorkflowByID(id string) (*catalogdomain.ManagedScanWorkflow, error) {
	store.requests = append(store.requests, id)
	workflow, ok := store.workflows[id]
	if !ok {
		return nil, errors.New("not found")
	}
	copy := workflow
	return &copy, nil
}

type scheduledScanAgentLookupStub struct{}

func (scheduledScanAgentLookupStub) AgentExists(context.Context, int) (bool, error) { return true, nil }

type scheduledScanConfigResourceValidatorStub struct {
	err      error
	requests []scanapp.WorkflowConfigResourceValidationRequest
}

func (stub *scheduledScanConfigResourceValidatorStub) Validate(_ context.Context, request scanapp.WorkflowConfigResourceValidationRequest) error {
	stub.requests = append(stub.requests, request)
	return stub.err
}

func newScheduledScanServiceForTest(
	store ScheduledScanStore,
	workflows ScheduledScanWorkflowStore,
) *ScheduledScanService {
	return NewScheduledScanService(store, workflows).
		WithConfigResourceValidator(&scheduledScanConfigResourceValidatorStub{})
}

func scheduledWorkflowStoreForTest() *scheduledScanWorkflowStoreStub {
	return &scheduledScanWorkflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{
		"default": {
			ScanWorkflowID: "default",
			Stages: []scanworkflow.Stage{{
				StageID: "discovery",
				Steps:   []scanworkflow.Step{{StepID: "subdomain_discovery", EngineID: "engine.lunafox.subdomain_discovery"}},
			}},
		},
	}}
}

func scheduledWorkflowStoreWithDisabledStepForTest() *scheduledScanWorkflowStoreStub {
	return &scheduledScanWorkflowStoreStub{workflows: map[string]catalogdomain.ManagedScanWorkflow{
		"default": {
			ScanWorkflowID: "default",
			Stages: []scanworkflow.Stage{{
				StageID: "discovery",
				Steps: []scanworkflow.Step{
					{StepID: "subdomain_discovery", EngineID: "engine.lunafox.subdomain_discovery"},
					{StepID: "port_scan", EngineID: "engine.lunafox.port_scan"},
				},
			}},
		},
	}}
}

func completeScheduledConfiguration() map[string]any {
	return map[string]any{"steps": map[string]any{
		"subdomain_discovery": map[string]any{"enabled": true, "engineConfig": map[string]any{"recon": map[string]any{"enabled": true, "timeout": 3600}}},
	}}
}

func (store *scheduledScanStoreCapture) Delete(context.Context, int) error {
	return nil
}

func TestScheduledScanOverviewUsesOneClockReadAndUTCMidnight(t *testing.T) {
	store := &scheduledScanStoreCapture{}
	asOfTime := time.Date(2026, 11, 1, 16, 0, 0, 0, time.UTC)
	clockReads := 0
	service := newScheduledScanServiceForTest(store, scheduledWorkflowStoreForTest()).WithClock(func() time.Time {
		clockReads++
		return asOfTime
	})

	overview, err := service.GetOverviewSummary(context.Background(), &ScheduledScanOverviewInput{})
	if err != nil {
		t.Fatalf("GetOverviewSummary() error = %v", err)
	}
	if clockReads != 1 || overview.AsOfTime != asOfTime || store.overviewQuery == nil {
		t.Fatalf("overview did not use one clock snapshot: reads=%d overview=%+v query=%+v", clockReads, overview, store.overviewQuery)
	}
	if want := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC); !store.overviewQuery.TodayStart.Equal(want) {
		t.Fatalf("today start = %s, want %s", store.overviewQuery.TodayStart, want)
	}
	if want := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC); !store.overviewQuery.TodayEnd.Equal(want) {
		t.Fatalf("today end = %s, want %s", store.overviewQuery.TodayEnd, want)
	}
	if want := asOfTime.Add(24 * time.Hour); !store.overviewQuery.Next24HoursEnd.Equal(want) {
		t.Fatalf("next 24 hour end = %s, want %s", store.overviewQuery.Next24HoursEnd, want)
	}
	if store.overviewQuery.UpcomingItemsLimit != 5 {
		t.Fatalf("upcoming item limit = %d, want 5", store.overviewQuery.UpcomingItemsLimit)
	}
}

func TestCreateScheduledScanPersistsInternalWorkflowIDAndStepConfig(t *testing.T) {
	store := &scheduledScanStoreCapture{}
	service := newScheduledScanServiceForTest(store, scheduledWorkflowStoreForTest())

	result, err := service.Create(context.Background(), &CreateScheduledScanInput{
		Name:           "daily",
		ScanWorkflow:   "scanWorkflows/default",
		Configuration:  completeScheduledConfiguration(),
		InputSource:    scanapp.InputSourceScanSnapshot,
		Target:         "targets/7",
		CronExpression: "0 2 * * *",
		IsEnabled:      boolPtr(true),
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if store.created == nil || store.created.ScanWorkflowID != "default" {
		t.Fatalf("expected internal scan workflow id persisted, got %+v", store.created)
	}
	if _, ok := store.created.Configuration["steps"]; !ok {
		t.Fatalf("expected step-scoped configuration persisted, got %+v", store.created.Configuration)
	}
	if result.ScanWorkflowID != "default" {
		t.Fatalf("unexpected result scan workflow: %+v", result)
	}
}

func TestCreateScheduledScanPersistsCanonicalStepBranches(t *testing.T) {
	store := &scheduledScanStoreCapture{}
	service := newScheduledScanServiceForTest(store, scheduledWorkflowStoreWithDisabledStepForTest())

	_, err := service.Create(context.Background(), &CreateScheduledScanInput{
		Name:         "daily",
		ScanWorkflow: "scanWorkflows/default",
		Configuration: map[string]any{"steps": map[string]any{
			"subdomain_discovery": map[string]any{
				"enabled":      true,
				"engineConfig": map[string]any{"recon": map[string]any{"enabled": true}},
			},
			"port_scan": map[string]any{
				"enabled":      false,
				"engineConfig": map[string]any{"must": "be dropped"},
			},
		}},
		InputSource:    scanapp.InputSourceScanSnapshot,
		Target:         "targets/7",
		CronExpression: "0 2 * * *",
	})
	if err == nil {
		t.Fatal("expected disabled engineConfig to be rejected before persistence")
	}

	_, err = service.Create(context.Background(), &CreateScheduledScanInput{
		Name:         "daily",
		ScanWorkflow: "scanWorkflows/default",
		Configuration: map[string]any{"steps": map[string]any{
			"subdomain_discovery": map[string]any{
				"enabled":      true,
				"engineConfig": map[string]any{"recon": map[string]any{"enabled": true}},
			},
			"port_scan": map[string]any{"enabled": false},
		}},
		InputSource:    scanapp.InputSourceScanSnapshot,
		Target:         "targets/7",
		CronExpression: "0 2 * * *",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	steps := store.created.Configuration["steps"].(map[string]any)
	if disabled := steps["port_scan"].(map[string]any); len(disabled) != 1 || disabled["enabled"] != false {
		t.Fatalf("disabled Step was not canonicalized: %#v", disabled)
	}
	if enabled := steps["subdomain_discovery"].(map[string]any); enabled["enabled"] != true || enabled["engineConfig"] == nil {
		t.Fatalf("enabled Step lost engineConfig: %#v", enabled)
	}
}

func TestCreateScheduledScanRejectsAllUserDisabledWorkflow(t *testing.T) {
	service := newScheduledScanServiceForTest(&scheduledScanStoreCapture{}, scheduledWorkflowStoreForTest())

	_, err := service.Create(context.Background(), &CreateScheduledScanInput{
		Name:           "disabled",
		ScanWorkflow:   "scanWorkflows/default",
		Configuration:  map[string]any{"steps": map[string]any{"subdomain_discovery": map[string]any{"enabled": false}}},
		InputSource:    scanapp.InputSourceScanSnapshot,
		Target:         "targets/7",
		CronExpression: "0 2 * * *",
	})
	if err == nil || !strings.Contains(err.Error(), "at least one workflow Step must be enabled") {
		t.Fatalf("expected all-disabled scheduled scan rejection, got %v", err)
	}
}

func TestCreateScheduledScanRejectsManifestTypedRefWorkflow(t *testing.T) {
	service := newScheduledScanServiceForTest(&scheduledScanStoreCapture{}, scheduledWorkflowStoreForTest())

	_, err := service.Create(context.Background(), &CreateScheduledScanInput{
		Name:           "daily",
		ScanWorkflow:   "engine.lunafox.subdomain_discovery",
		Configuration:  completeScheduledConfiguration(),
		InputSource:    scanapp.InputSourceScanSnapshot,
		CronExpression: "0 2 * * *",
	})
	if err == nil {
		t.Fatal("expected manifest typed ref scanWorkflow to fail")
	}
}

func TestUpdateScheduledScanRejectsConfigurationWithoutWorkflow(t *testing.T) {
	service := newScheduledScanServiceForTest(&scheduledScanStoreCapture{}, scheduledWorkflowStoreForTest())

	_, err := service.Update(context.Background(), 1, &UpdateScheduledScanInput{
		Configuration: map[string]any{"steps": map[string]any{}},
	})
	if err == nil {
		t.Fatal("expected configuration update without scanWorkflow to fail")
	}
}

func TestScheduledScanAgentUpdateDoesNotMutatePriorScans(t *testing.T) {
	previousAgentID := 42
	nextAgentID := 43
	store := &scheduledScanStoreCapture{loaded: &ScheduledScan{ID: 12, Name: "daily", ScanWorkflowID: "default", InputSource: scanapp.InputSourceScanSnapshot, AgentID: &previousAgentID}}
	service := newScheduledScanServiceForTest(store, scheduledWorkflowStoreForTest()).WithAgentLookup(scheduledScanAgentLookupStub{})
	nextAgent := "agents/43"
	if _, err := service.Update(context.Background(), 12, &UpdateScheduledScanInput{Agent: &nextAgent}); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if store.updated == nil || store.updated.AgentID == nil || *store.updated.AgentID != nextAgentID || !store.updated.AgentSet {
		t.Fatalf("scheduled Agent update was not persisted for future triggers: %#v", store.updated)
	}
	if store.loaded.AgentID == nil || *store.loaded.AgentID != previousAgentID {
		t.Fatalf("scheduled update must not mutate already-created Scan affinity: %#v", store.loaded)
	}
}

func TestScheduledScanServiceListPreservesHandoffOutcomeTotals(t *testing.T) {
	store := &scheduledScanStoreCapture{listed: []ScheduledScan{{
		ID: 23, RunCount: 10, SuccessfulHandoffCount: 8, FailedHandoffCount: 1,
	}}}
	service := newScheduledScanServiceForTest(store, scheduledWorkflowStoreForTest())

	items, total, err := service.List(context.Background(), ScheduledScanListQuery{Page: 1, PageSize: 20})
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("List() = %+v, %d, %v", items, total, err)
	}
	if items[0].RunCount != 10 || items[0].SuccessfulHandoffCount != 8 || items[0].FailedHandoffCount != 1 {
		t.Fatalf("List() changed handoff totals: %+v", items[0])
	}
}

func TestScheduledScanCreateValidatesResourcesBeforeStore(t *testing.T) {
	tests := []struct {
		name  string
		cause scanapp.ConfigResourceValidationCause
		err   error
	}{
		{name: "unavailable", cause: scanapp.ConfigResourceUnavailable, err: scanapp.NewConfigResourceUnavailableError("field", "wordlist", "dns.txt", errors.New("missing"))},
		{name: "validation unavailable", cause: scanapp.ConfigResourceValidationUnavailable, err: scanapp.NewConfigResourceValidationUnavailableError("field", "wordlist", "dns.txt", errors.New("database down"))},
		{name: "internal", cause: scanapp.ConfigResourceInternal, err: scanapp.NewConfigResourceInternalError("field", "wordlist", "dns.txt", errors.New("missing dependency"))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &scheduledScanStoreCapture{}
			validator := &scheduledScanConfigResourceValidatorStub{err: test.err}
			service := NewScheduledScanService(store, scheduledWorkflowStoreForTest()).WithConfigResourceValidator(validator)
			_, err := service.Create(context.Background(), &CreateScheduledScanInput{
				Name: "daily", ScanWorkflow: "scanWorkflows/default", Configuration: completeScheduledConfiguration(), InputSource: scanapp.InputSourceScanSnapshot,
				Target: "targets/7", CronExpression: "0 2 * * *",
			})
			var typed *scanapp.ConfigResourceValidationError
			if !errors.As(err, &typed) || typed.ConfigResourceValidationCause() != string(test.cause) {
				t.Fatalf("Create() error = %v, want cause %q", err, test.cause)
			}
			if len(validator.requests) != 1 || store.created != nil {
				t.Fatalf("validation/store calls = %d/%#v", len(validator.requests), store.created)
			}
		})
	}
}

func TestScheduledScanConfigurationUpdateValidatesBeforeStore(t *testing.T) {
	store := &scheduledScanStoreCapture{}
	typed := scanapp.NewConfigResourceUnavailableError("field", "wordlist", "dns.txt", errors.New("missing"))
	validator := &scheduledScanConfigResourceValidatorStub{err: typed}
	service := NewScheduledScanService(store, scheduledWorkflowStoreForTest()).WithConfigResourceValidator(validator)
	workflow := "scanWorkflows/default"
	_, err := service.Update(context.Background(), 12, &UpdateScheduledScanInput{
		ScanWorkflow: &workflow, Configuration: completeScheduledConfiguration(),
	})
	if !errors.Is(err, typed) {
		t.Fatalf("Update() error = %v, want typed resource failure", err)
	}
	if len(validator.requests) != 1 || store.updated != nil {
		t.Fatalf("validation/store calls = %d/%#v", len(validator.requests), store.updated)
	}
}

func TestScheduledScanIsEnabledOnlyUpdateSkipsWorkflowAndResourceValidation(t *testing.T) {
	store := &scheduledScanStoreCapture{loaded: &ScheduledScan{ID: 12, Name: "daily", ScanWorkflowID: "default", InputSource: scanapp.InputSourceScanSnapshot}}
	workflows := scheduledWorkflowStoreForTest()
	validator := &scheduledScanConfigResourceValidatorStub{err: errors.New("must not be called")}
	service := NewScheduledScanService(store, workflows).WithConfigResourceValidator(validator)
	enabled := true
	if _, err := service.Update(context.Background(), 12, &UpdateScheduledScanInput{IsEnabled: &enabled}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if len(workflows.requests) != 0 || len(validator.requests) != 0 {
		t.Fatalf("isEnabled-only update performed dynamic validation: workflows=%#v resources=%#v", workflows.requests, validator.requests)
	}
	if store.updated == nil || store.updated.IsEnabled == nil || !*store.updated.IsEnabled {
		t.Fatalf("isEnabled-only update was not persisted: %#v", store.updated)
	}
}

func TestScheduledScanBatchStatusUpdateSortsExplicitStatesWithoutDynamicValidation(t *testing.T) {
	store := &scheduledScanStoreCapture{}
	workflows := scheduledWorkflowStoreForTest()
	validator := &scheduledScanConfigResourceValidatorStub{err: errors.New("must not be called")}
	service := NewScheduledScanService(store, workflows).WithConfigResourceValidator(validator)

	updatedCount, err := service.BatchUpdateStatus(context.Background(), []ScheduledScanStatusUpdate{
		{ID: 9, IsEnabled: false},
		{ID: 3, IsEnabled: true},
	})
	if err != nil {
		t.Fatalf("BatchUpdateStatus() error = %v", err)
	}
	if updatedCount != 2 || len(store.batchUpdates) != 2 {
		t.Fatalf("BatchUpdateStatus() = %d, updates=%+v", updatedCount, store.batchUpdates)
	}
	if store.batchUpdates[0].ID != 3 || !store.batchUpdates[0].IsEnabled || store.batchUpdates[1].ID != 9 || store.batchUpdates[1].IsEnabled {
		t.Fatalf("service did not preserve sorted explicit states: %+v", store.batchUpdates)
	}
	if len(workflows.requests) != 0 || len(validator.requests) != 0 {
		t.Fatalf("batch status update performed dynamic validation: workflows=%#v resources=%#v", workflows.requests, validator.requests)
	}
}

func TestScheduledScanBatchStatusUpdateRejectsInvalidIDsBeforeStore(t *testing.T) {
	tests := []struct {
		name    string
		updates []ScheduledScanStatusUpdate
	}{
		{name: "empty", updates: nil},
		{name: "non-positive", updates: []ScheduledScanStatusUpdate{{ID: 0, IsEnabled: true}}},
		{name: "duplicate", updates: []ScheduledScanStatusUpdate{{ID: 1, IsEnabled: true}, {ID: 1, IsEnabled: false}}},
		{name: "too many", updates: make([]ScheduledScanStatusUpdate, MaxScheduledScanBatchStatusUpdates+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &scheduledScanStoreCapture{}
			service := newScheduledScanServiceForTest(store, scheduledWorkflowStoreForTest())
			_, err := service.BatchUpdateStatus(context.Background(), test.updates)
			if !errors.Is(err, ErrScheduledScanInvalidArgument) {
				t.Fatalf("BatchUpdateStatus() error = %v, want invalid argument", err)
			}
			if len(store.batchUpdates) != 0 {
				t.Fatalf("invalid batch reached store: %+v", store.batchUpdates)
			}
		})
	}
}

func TestScheduledScanSaveFailsClosedWhenValidatorIsNotAssembled(t *testing.T) {
	store := &scheduledScanStoreCapture{}
	service := NewScheduledScanService(store, scheduledWorkflowStoreForTest())
	_, err := service.Create(context.Background(), &CreateScheduledScanInput{
		Name: "daily", ScanWorkflow: "scanWorkflows/default", Configuration: completeScheduledConfiguration(), InputSource: scanapp.InputSourceScanSnapshot,
		Target: "targets/7", CronExpression: "0 2 * * *",
	})
	var typed *scanapp.ConfigResourceValidationError
	if !errors.As(err, &typed) || typed.ConfigResourceValidationCause() != string(scanapp.ConfigResourceInternal) || store.created != nil {
		t.Fatalf("Create() = error %v store %#v, want typed internal and no persistence", err, store.created)
	}
}

func valueOr(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}

func boolPtr(value bool) *bool {
	return &value
}
