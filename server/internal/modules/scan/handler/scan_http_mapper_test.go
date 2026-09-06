package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/scan/dto"
)

type scanDetailQueryStoreStub struct {
	detail *scanapp.QueryScan
}

func (stub scanDetailQueryStoreStub) List(int, int, int, string, string, string) ([]scanapp.QueryScan, int64, error) {
	return nil, 0, nil
}

func (stub scanDetailQueryStoreStub) GetDetailByID(int) (*scanapp.QueryScan, error) {
	return stub.detail, nil
}

func (stub scanDetailQueryStoreStub) GetGlobalStatsSummary() (*scanapp.QueryStatistics, error) {
	return &scanapp.QueryStatistics{}, nil
}

func TestToScanQueryInputPassesCanonicalFilterAndOrderBy(t *testing.T) {
	input := toScanQueryInput(&dto.ScanListQuery{
		Filter:  `(status=="running" || status=="failed") && targetName="acme"`,
		OrderBy: "createdAt desc",
	})

	if input.Filter != `(status=="running" || status=="failed") && targetName="acme"` {
		t.Fatalf("unexpected filter %q", input.Filter)
	}
	if input.OrderBy != "createdAt desc" {
		t.Fatalf("unexpected orderBy %q", input.OrderBy)
	}
}

func TestToScanDetailOutput_UsesConfigurationObjectWithoutYAMLField(t *testing.T) {
	scan := &scanapp.QueryScan{
		ID:             1,
		TargetID:       2,
		ScanWorkflowID: "default",
		Configuration: map[string]any{
			"steps": map[string]any{
				"subdomain_discovery": map[string]any{
					"engineConfig": map[string]any{},
				},
			},
		},
		Status:    "pending",
		CreatedAt: time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC),
	}

	output := toScanDetailOutput(scan)
	payload, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if _, ok := decoded["configuration"]; !ok {
		t.Fatalf("expected configuration field in scan detail output, got %v", decoded)
	}
	if _, ok := decoded["yamlConfiguration"]; ok {
		t.Fatalf("expected yamlConfiguration to be removed from scan detail output, got %v", decoded)
	}
}

func TestToScanOutput_ExposesFailureObject(t *testing.T) {
	scan := &scanapp.QueryScan{
		ID:             11,
		TargetID:       22,
		ScanWorkflowID: "default",
		Status:         "failed",
		ErrorMessage:   "task timed out",
		Failure:        &scanapp.FailureDetail{Kind: "task_timeout", Message: "task timed out"},
		CreatedAt:      time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC),
	}

	output := toScanOutput(scan)
	if output.Failure == nil {
		t.Fatalf("expected failure object on scan output")
	}
	if output.Failure.Kind != "task_timeout" || output.Failure.Message != "task timed out" {
		t.Fatalf("unexpected failure output: %+v", output.Failure)
	}
	if output.ScanWorkflow != "scanWorkflows/default" {
		t.Fatalf("unexpected scan workflow resource: %+v", output)
	}
}

func TestToScanOutputOmitsLegacyWorkflowFields(t *testing.T) {
	scan := &scanapp.QueryScan{
		ID:             11,
		TargetID:       22,
		ScanWorkflowID: "default",
		TriggerType:    scanapp.ScanTriggerTypeManual,
		Status:         "pending",
		CreatedAt:      time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC),
	}

	payload, err := json.Marshal(toScanOutput(scan))
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if decoded["scanWorkflow"] != "scanWorkflows/default" {
		t.Fatalf("expected canonical scanWorkflow field, got %+v", decoded)
	}
	// Legacy response aliases must stay absent after the hard-cut response rename.
	if decoded["triggerType"] != "manual" {
		t.Fatalf("expected persisted triggerType, got %+v", decoded)
	}
	for _, field := range []string{"workflow", "workflowIds", "workflowId", "workflow_ids", "scanMode", "creationMethod"} {
		if _, ok := decoded[field]; ok {
			t.Fatalf("legacy workflow field %q should be omitted from scan response: %+v", field, decoded)
		}
	}
}

func TestToScanOutputIncludesPlannedEngineIDsDistinctFromScanWorkflow(t *testing.T) {
	scan := &scanapp.QueryScan{
		ID:               11,
		TargetID:         22,
		ScanWorkflowID:   "full_recon",
		PlannedEngineIDs: []string{"engine.lunafox.subdomain_discovery", "engine.lunafox.port_scan", "engine.lunafox.nuclei_vulnerability"},
		Status:           "running",
		CreatedAt:        time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
	}

	payload, err := json.Marshal(toScanOutput(scan))
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if decoded["scanWorkflow"] != "scanWorkflows/full_recon" {
		t.Fatalf("expected canonical scanWorkflow field, got %+v", decoded)
	}
	engines, ok := decoded["plannedEngineIds"].([]any)
	if !ok || len(engines) != 3 {
		t.Fatalf("expected plannedEngineIds summary, got %+v", decoded["plannedEngineIds"])
	}
	if engines[0] != "engine.lunafox.subdomain_discovery" || engines[1] != "engine.lunafox.port_scan" || engines[2] != "engine.lunafox.nuclei_vulnerability" {
		t.Fatalf("unexpected plannedEngineIds order: %+v", engines)
	}
	if _, exists := decoded["engineNames"]; exists {
		t.Fatalf("legacy engineNames field must be absent: %+v", decoded)
	}
}

func TestToScanOutputOmitsAgentRuntimeFields(t *testing.T) {
	agentID := 42
	scan := &scanapp.QueryScan{
		ID:               11,
		TargetID:         22,
		ScanWorkflowID:   "default",
		Status:           "succeeded",
		AgentID:          &agentID,
		AgentName:        "test-agent-01",
		AgentStatus:      "online",
		AgentHealthState: "paused",
		AssignmentMode:   "automatic",
		CreatedAt:        time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC),
	}

	payload, err := json.Marshal(toScanOutput(scan))
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	for _, field := range []string{"agentId", "agentName", "agentStatus", "agentHealthState", "agentDeleted", "workerId", "workerName"} {
		if _, ok := decoded[field]; ok {
			t.Fatalf("runtime field %q should be omitted from scan list response: %+v", field, decoded)
		}
	}
}

func TestToScanStatisticsOutputIncludesHistoryStatusCounts(t *testing.T) {
	output := toScanStatisticsOutput(&scanapp.ScanStatistics{
		Total:     5,
		Pending:   1,
		Running:   2,
		Completed: 1,
		Failed:    1,
		Cancelled: 3,
	}, ScanHistoryRetentionPolicy{MinimumRetentionSeconds: 14 * 24 * 60 * 60, AutomaticCleanupEnabled: true})

	if output.Pending != 1 || output.Running != 2 || output.Completed != 1 || output.Failed != 1 || output.Cancelled != 3 {
		t.Fatalf("expected scan history status counts in statistics output, got %+v", output)
	}
	if output.RetentionPolicy.MinimumRetentionSeconds != 14*24*60*60 || !output.RetentionPolicy.AutomaticCleanupEnabled {
		t.Fatalf("expected effective retention policy in statistics output, got %+v", output.RetentionPolicy)
	}
}

func TestToScanDetailOutput_IncludesAgentIDAndAgentName(t *testing.T) {
	agentID := 42
	scan := &scanapp.QueryScan{
		ID:               1,
		TargetID:         2,
		ScanWorkflowID:   "default",
		Status:           "succeeded",
		AgentID:          &agentID,
		AgentName:        "test-agent-01",
		AgentStatus:      "online",
		AgentHealthState: "paused",
		AssignmentMode:   "automatic",
		RuntimeTasks: []scanapp.QueryRuntimeTask{
			{
				ID:            7001,
				StepID:        "subdomain_discovery",
				StageID:       "discovery",
				EngineID:      "engine.lunafox.subdomain_discovery",
				Status:        "failed",
				Order:         1,
				FailureKind:   "shared_materialization_staging_failed",
				FailureDetail: "Restore Agent storage permissions.",
			},
		},
		CreatedAt: time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC),
	}

	output := toScanDetailOutput(scan)
	payload, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if decoded["agentId"] != float64(42) {
		t.Fatalf("expected agentId to be 42, got %v", decoded["agentId"])
	}
	if decoded["agentName"] != "test-agent-01" {
		t.Fatalf("expected agentName to be test-agent-01, got %v", decoded["agentName"])
	}
	if decoded["agentStatus"] != "online" || decoded["agentHealthState"] != "paused" || decoded["agentDeleted"] != false {
		t.Fatalf("expected current Agent state and explicit non-deleted projection, got %+v", decoded)
	}
	if decoded["agent"] != "agents/42" || decoded["assignmentMode"] != "automatic" {
		t.Fatalf("expected canonical Agent and automatic assignment mode, got %+v", decoded)
	}
	tasks, ok := decoded["runtimeTasks"].([]any)
	if !ok || len(tasks) != 1 {
		t.Fatalf("expected runtimeTasks to contain one task, got %+v", decoded["runtimeTasks"])
	}
	task, ok := tasks[0].(map[string]any)
	if !ok {
		t.Fatalf("expected runtime task object, got %+v", tasks[0])
	}
	if task["id"] != float64(7001) || task["stepId"] != "subdomain_discovery" || task["stageId"] != "discovery" {
		t.Fatalf("unexpected runtime task output: %+v", task)
	}
	if task["name"] != "scans/1/tasks/7001" {
		t.Fatalf("expected runtime task canonical name, got %+v", task)
	}
	if task["failureDetail"] != "Restore Agent storage permissions." {
		t.Fatalf("expected controlled failure detail, got %+v", task)
	}
	if _, ok := decoded["stageProgress"]; ok {
		t.Fatalf("stageProgress should be omitted from scan detail response: %+v", decoded)
	}
	// Legacy worker fields must be omitted
	for _, field := range []string{"workerId", "workerName"} {
		if _, ok := decoded[field]; ok {
			t.Fatalf("legacy worker field %q should be omitted from scan detail response: %+v", field, decoded)
		}
	}
}

func TestScanHandlerGetByIDSerializesCurrentAgentProjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	agentID := 1001
	store := scanDetailQueryStoreStub{detail: &scanapp.QueryScan{
		ID:               7,
		TargetID:         2,
		ScanWorkflowID:   "default",
		Status:           "running",
		AgentID:          &agentID,
		AgentName:        "Renamed Agent",
		AgentStatus:      "online",
		AgentHealthState: "healthy",
		AssignmentMode:   "pinned",
		CreatedAt:        time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC),
	}}
	queryService := scanapp.NewScanQueryService(store)
	facade := scanapp.NewScanFacade(store, nil, nil, queryService, nil, nil)
	handler := NewScanHandler(facade, ScanHistoryRetentionPolicy{})
	router := gin.New()
	router.GET("/v1/scans/:scan", handler.GetByID)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/scans/7", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET Scan detail status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode Scan detail response: %v", err)
	}
	if response["agent"] != "agents/1001" || response["agentName"] != "Renamed Agent" || response["agentStatus"] != "online" || response["agentHealthState"] != "healthy" || response["agentDeleted"] != false || response["assignmentMode"] != "pinned" {
		t.Fatalf("Scan detail Agent projection = %+v", response)
	}
}

func TestToRuntimeTaskOutputIncludesStableSkipReason(t *testing.T) {
	output := toRuntimeTaskOutputs(1, []scanapp.QueryRuntimeTask{{
		ID:         7002,
		StepID:     "port_scan",
		StageID:    "ports",
		EngineID:   "engine.lunafox.port_scan",
		Status:     "skipped",
		SkipReason: "target_not_applicable",
	}})
	if len(output) != 1 || output[0].SkipReason != "target_not_applicable" {
		t.Fatalf("runtime task skip reason = %+v, want target_not_applicable", output)
	}
}

func TestScanCreateMappersParseCanonicalAgentAndRejectMalformedReferences(t *testing.T) {
	input, err := toScanCreateQuickInput(&dto.CreateQuickScanRequest{
		Targets:       []string{"example.com"},
		ScanWorkflow:  "scanWorkflows/default",
		Configuration: map[string]any{},
		InputSource:   "scanSnapshot",
		Agent:         "agents/42",
	})
	if err != nil || input.AgentID == nil || *input.AgentID != 42 {
		t.Fatalf("canonical Agent input = %#v, %v", input, err)
	}
	if input.TriggerType != scanapp.ScanTriggerTypeManual {
		t.Fatalf("quick create trigger type = %q, want manual", input.TriggerType)
	}
	if input.InputSource != scanapp.InputSourceScanSnapshot {
		t.Fatalf("quick create input source = %q, want scanSnapshot", input.InputSource)
	}

	batch, err := toScanBatchCreateInput(&dto.BatchCreateScanRequest{
		Requests:     []dto.BatchCreateScanItem{{Target: "targets/7"}},
		ScanWorkflow: "scanWorkflows/default",
		InputSource:  "targetInventory",
	})
	if err != nil || batch.TriggerType != scanapp.ScanTriggerTypeManual {
		t.Fatalf("batch create trigger type = %#v, %v; want manual", batch, err)
	}
	if batch.InputSource != scanapp.InputSourceTargetInventory {
		t.Fatalf("batch create input source = %q, want targetInventory", batch.InputSource)
	}

	if _, err := toScanCreateQuickInput(&dto.CreateQuickScanRequest{Agent: "agents/not-a-number"}); err == nil {
		t.Fatal("expected malformed Agent resource name to fail")
	}

	var request dto.CreateQuickScanRequest
	if err := json.Unmarshal([]byte(`{"targets":["example.com"],"scanWorkflow":"scanWorkflows/default","configuration":{},"agentId":42}`), &request); err == nil {
		t.Fatal("expected legacy agentId request field to fail")
	}
}

func TestScanResponseProjectionsIncludePersistedTriggerType(t *testing.T) {
	scan := scanapp.QueryScan{
		ID:             91,
		TargetID:       7,
		ScanWorkflowID: "default",
		TriggerType:    scanapp.ScanTriggerTypeScheduled,
		Status:         "pending",
		CreatedAt:      time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC),
	}

	if output := toScanOutput(&scan); output.TriggerType != "scheduled" {
		t.Fatalf("list Scan response triggerType = %q, want scheduled", output.TriggerType)
	}
	if output := toScanDetailOutput(&scan); output.TriggerType != "scheduled" {
		t.Fatalf("detail Scan response triggerType = %q, want scheduled", output.TriggerType)
	}
	if output := toQuickScanOutput(&scanapp.QuickScanResult{Scans: []scanapp.QueryScan{scan}}); len(output.Scans) != 1 || output.Scans[0].TriggerType != "scheduled" {
		t.Fatalf("quick-create Scan response = %+v, want scheduled triggerType", output.Scans)
	}
	if output := toBatchScanOutput(&scanapp.BatchScanResult{Scans: []scanapp.QueryScan{scan}, CreatedCount: 1}); len(output.Scans) != 1 || output.Scans[0].TriggerType != "scheduled" {
		t.Fatalf("batch-create Scan response = %+v, want scheduled triggerType", output.Scans)
	}
}

func TestToScanDetailOutputMarksDeletedAgent(t *testing.T) {
	agentID := 42
	output := toScanDetailOutput(&scanapp.QueryScan{ID: 1, TargetID: 2, ScanWorkflowID: "default", AgentID: &agentID, AgentDeleted: true, AssignmentMode: "pinned"})
	if output.AgentDeleted == nil || !*output.AgentDeleted || output.Agent != "agents/42" || output.AssignmentMode != "pinned" {
		t.Fatalf("unexpected deleted Agent detail output: %+v", output)
	}
	payload, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal deleted Agent detail: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal deleted Agent detail: %v", err)
	}
	for _, field := range []string{"agentName", "agentStatus", "agentHealthState"} {
		if _, exists := decoded[field]; exists {
			t.Fatalf("deleted Agent field %q must be omitted: %+v", field, decoded)
		}
	}
}

func TestToScanDetailOutputOmitsAgentProjectionWhenUnassigned(t *testing.T) {
	output := toScanDetailOutput(&scanapp.QueryScan{ID: 1, TargetID: 2, ScanWorkflowID: "default", AssignmentMode: "automatic"})
	payload, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal unassigned Scan detail: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal unassigned Scan detail: %v", err)
	}
	for _, field := range []string{"agent", "agentId", "agentName", "agentStatus", "agentHealthState", "agentDeleted"} {
		if _, exists := decoded[field]; exists {
			t.Fatalf("unassigned Agent field %q must be omitted: %+v", field, decoded)
		}
	}
}
