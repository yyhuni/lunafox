package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/mcp/schema"
)

type testTargetReader struct {
	list func(context.Context, TargetQuery) (Page[TargetRecord], error)
	get  func(context.Context, int) (TargetRecord, error)
}

func (reader testTargetReader) List(ctx context.Context, query TargetQuery) (Page[TargetRecord], error) {
	if reader.list == nil {
		return Page[TargetRecord]{}, nil
	}
	return reader.list(ctx, query)
}

func (reader testTargetReader) Get(ctx context.Context, id int) (TargetRecord, error) {
	if reader.get == nil {
		return TargetRecord{}, nil
	}
	return reader.get(ctx, id)
}

type testVulnerabilityReader struct {
	list func(context.Context, VulnerabilityQuery) (Page[VulnerabilityRecord], error)
	get  func(context.Context, int) (VulnerabilityRecord, error)
}

type testOrganizationCreator struct {
	create func(context.Context, OrganizationCreateInput) (OrganizationCreateOutput, error)
}

func (creator testOrganizationCreator) Create(ctx context.Context, input OrganizationCreateInput) (OrganizationCreateOutput, error) {
	return creator.create(ctx, input)
}

type testTargetBatchCreator struct {
	create func(context.Context, TargetBatchCreateInput) (TargetBatchCreateOutput, error)
}

func (creator testTargetBatchCreator) Create(ctx context.Context, input TargetBatchCreateInput) (TargetBatchCreateOutput, error) {
	return creator.create(ctx, input)
}

func (reader testVulnerabilityReader) List(ctx context.Context, query VulnerabilityQuery) (Page[VulnerabilityRecord], error) {
	return reader.list(ctx, query)
}

func (reader testVulnerabilityReader) Get(ctx context.Context, id int) (VulnerabilityRecord, error) {
	return reader.get(ctx, id)
}

func toolRequest(arguments string) *mcp.CallToolRequest {
	return &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(arguments)}}
}

func TestToolNamesExposeInvestigationAndApprovedConstrainedCreationCatalog(t *testing.T) {
	want := map[string]struct{}{}
	for _, name := range []string{
		ToolListTargets, ToolGetTarget, ToolListScans, ToolGetScan,
		ToolListWebsites, ToolListSubdomains, ToolListEndpoints, ToolListDirectories,
		ToolListHostPorts, ToolListVulnerabilities, ToolGetVulnerability,
		ToolCreateOrganization, ToolCreateTarget,
		ToolListOrganizations, ToolGetOrganization, ToolListOrganizationTargets,
		ToolListTargetVulnerabilities, ToolListTargetScreenshots, ToolGetScreenshotImage,
		ToolListScanWorkflows, ToolGetScanWorkflow, ToolGetScanWorkflowProfile,
		ToolListEngines, ToolGetEngine, ToolListWordlists, ToolGetWordlist,
		ToolListServerLogEntries, ToolListAgentLogEntries,
		ToolReviewVulnerability, ToolUnreviewVulnerability,
		ToolBatchReviewVulnerabilities, ToolBatchUnreviewVulnerabilities,
		ToolStartScan, ToolGetOperation,
	} {
		want[name] = struct{}{}
	}
	got := ToolNames()
	if len(got) != len(want) {
		t.Fatalf("tool count = %d, want %d", len(got), len(want))
	}
	for _, name := range got {
		if _, ok := want[name]; !ok {
			t.Fatalf("unexpected tool %q", name)
		}
	}
	for _, excluded := range []string{"organizations", "schedule_scan", "read_task_logs", "get_screenshot_blob", "export_results", "write_target", "update_target", "delete_target"} {
		if (&Registry{}).HasTool(excluded) {
			t.Fatalf("excluded tool %q is registered", excluded)
		}
	}
}

func TestConstrainedCreationToolsValidateAndReturnBoundedResults(t *testing.T) {
	contextKey := struct{}{}
	ctx := context.WithValue(context.Background(), contextKey, "preserved")
	registry := NewRegistry(Dependencies{
		Organizations: testOrganizationCreator{create: func(got context.Context, input OrganizationCreateInput) (OrganizationCreateOutput, error) {
			if got.Value(contextKey) != "preserved" {
				t.Fatal("organization creator lost tool context")
			}
			if input.Name != "  Platform  " || input.Description != "group" {
				t.Fatalf("unexpected organization input: %+v", input)
			}
			return OrganizationCreateOutput{ResourceName: "organizations/8", DisplayName: "Platform"}, nil
		}},
		TargetCreator: testTargetBatchCreator{create: func(got context.Context, input TargetBatchCreateInput) (TargetBatchCreateOutput, error) {
			if got.Value(contextKey) != "preserved" {
				t.Fatal("target creator lost tool context")
			}
			if len(input.Names) != 2 || input.OrganizationID == nil || *input.OrganizationID != 8 {
				t.Fatalf("unexpected target input: %+v", input)
			}
			return TargetBatchCreateOutput{
				CreatedCount:         1,
				FailedCount:          1,
				FailedTargets:        []FailedTarget{{Name: "***", Reason: "unrecognized target format"}},
				Organization:         "organizations/8",
				AssociationCompleted: true,
			}, nil
		}},
	})

	organizationResult, err := registry.createOrganization(ctx, toolRequest(`{"name":"  Platform  ","description":"group"}`))
	if err != nil || organizationResult == nil || organizationResult.IsError {
		t.Fatalf("create organization = %#v, %v", organizationResult, err)
	}
	organizationOutput := organizationResult.StructuredContent.(OrganizationCreateOutput)
	if organizationOutput.ResourceName != "organizations/8" || organizationOutput.DisplayName != "Platform" {
		t.Fatalf("unexpected organization output: %+v", organizationOutput)
	}

	targetResult, err := registry.createTarget(ctx, toolRequest(`{"targets":[{"name":"example.com"},{"name":"***"}],"organization":"organizations/8"}`))
	if err != nil || targetResult == nil || targetResult.IsError {
		t.Fatalf("create target = %#v, %v", targetResult, err)
	}
	targetOutput := targetResult.StructuredContent.(TargetBatchCreateOutput)
	if targetOutput.CreatedCount != 1 || targetOutput.FailedCount != 1 || !targetOutput.AssociationCompleted {
		t.Fatalf("unexpected target output: %+v", targetOutput)
	}
	if len(targetOutput.FailedTargets) != 1 || targetOutput.FailedTargets[0].Name != "***" {
		t.Fatalf("partial acceptance details = %+v", targetOutput.FailedTargets)
	}

	invalidResult, err := registry.createTarget(ctx, toolRequest(`{"targets":[{"name":"example.com"}],"organization":"organizations/08"}`))
	if err != nil || invalidResult == nil || !invalidResult.IsError {
		t.Fatalf("invalid organization reference = %#v, %v", invalidResult, err)
	}
	if invalidResult.StructuredContent.(map[string]any)["error"].(map[string]string)["category"] != string(mcpErrors.CategoryInvalidInput) {
		t.Fatalf("invalid reference category = %#v", invalidResult.StructuredContent)
	}

	if _, err := registry.createOrganization(ctx, toolRequest(`{"name":"Platform","unexpected":true}`)); err == nil {
		t.Fatal("unknown organization argument unexpectedly accepted")
	}
	emptyResult, err := registry.createTarget(ctx, toolRequest(`{"targets":[]}`))
	if err != nil || emptyResult == nil || !emptyResult.IsError {
		t.Fatalf("empty target batch = %#v, %v", emptyResult, err)
	}
}

func TestConstrainedCreationMapsExpectedErrorsAndDeclaresMutation(t *testing.T) {
	registry := NewRegistry(Dependencies{
		Organizations: testOrganizationCreator{create: func(context.Context, OrganizationCreateInput) (OrganizationCreateOutput, error) {
			return OrganizationCreateOutput{}, mcpErrors.ErrAlreadyExists
		}},
	})
	result, err := registry.createOrganization(context.Background(), toolRequest(`{"name":"Taken"}`))
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("duplicate organization = %#v, %v", result, err)
	}
	if result.StructuredContent.(map[string]any)["error"].(map[string]string)["category"] != string(mcpErrors.CategoryAlreadyExists) {
		t.Fatalf("duplicate category = %#v", result.StructuredContent)
	}
	annotation := constrainedWriteAnnotations()
	if annotation.ReadOnlyHint || annotation.DestructiveHint == nil || *annotation.DestructiveHint {
		t.Fatalf("unexpected write annotation: %+v", annotation)
	}
	for _, schema := range []map[string]any{createOrganizationSchema(), createTargetSchema()} {
		if schema["additionalProperties"] != false {
			t.Fatalf("write schema is not closed: %#v", schema)
		}
	}
	targetSchema := createTargetSchema()
	properties := targetSchema["properties"].(map[string]any)
	organization := properties["organization"].(map[string]any)
	if organization["pattern"] != "^organizations/[1-9][0-9]*$" {
		t.Fatalf("organization schema pattern = %#v", organization["pattern"])
	}
	items := properties["targets"].(map[string]any)["items"].(map[string]any)
	if items["additionalProperties"] != false {
		t.Fatalf("target item schema is not closed: %#v", items)
	}
}

func TestConstrainedCreationBatchBoundariesAndOptionalGrouping(t *testing.T) {
	var calls []TargetBatchCreateInput
	registry := NewRegistry(Dependencies{
		TargetCreator: testTargetBatchCreator{create: func(_ context.Context, input TargetBatchCreateInput) (TargetBatchCreateOutput, error) {
			calls = append(calls, input)
			return TargetBatchCreateOutput{
				CreatedCount:         len(input.Names),
				AssociationCompleted: input.OrganizationID != nil,
			}, nil
		}},
	})

	one, err := registry.createTarget(context.Background(), toolRequest(`{"targets":[{"name":"one.example"}]}`))
	if err != nil || one == nil || one.IsError || len(calls) != 1 || calls[0].OrganizationID != nil {
		t.Fatalf("ungrouped singleton = %#v, %v, calls=%+v", one, err, calls)
	}

	bulkNames := make([]map[string]string, 5000)
	for index := range bulkNames {
		bulkNames[index] = map[string]string{"name": fmt.Sprintf("target-%d.example", index)}
	}
	bulkArguments, err := json.Marshal(map[string]any{"targets": bulkNames})
	if err != nil {
		t.Fatalf("marshal bulk arguments: %v", err)
	}
	bulk, err := registry.createTarget(context.Background(), toolRequest(string(bulkArguments)))
	if err != nil || bulk == nil || bulk.IsError || len(calls) != 2 || len(calls[1].Names) != 5000 || calls[1].OrganizationID != nil {
		t.Fatalf("ungrouped maximum batch = %#v, %v, calls=%+v", bulk, err, calls)
	}

	zero, err := registry.createTarget(context.Background(), toolRequest(`{"targets":[]}`))
	if err != nil || zero == nil || !zero.IsError || toolFailureCategory(t, zero) != string(mcpErrors.CategoryInvalidInput) {
		t.Fatalf("zero-item batch = %#v, %v", zero, err)
	}
	tooMany := make([]map[string]string, 5001)
	for index := range tooMany {
		tooMany[index] = map[string]string{"name": fmt.Sprintf("overflow-%d.example", index)}
	}
	tooManyArguments, err := json.Marshal(map[string]any{"targets": tooMany})
	if err != nil {
		t.Fatalf("marshal overflow arguments: %v", err)
	}
	overflow, err := registry.createTarget(context.Background(), toolRequest(string(tooManyArguments)))
	if err != nil || overflow == nil || !overflow.IsError || toolFailureCategory(t, overflow) != string(mcpErrors.CategoryInvalidInput) {
		t.Fatalf("overflow batch = %#v, %v", overflow, err)
	}
	perTargetOrganization, err := registry.createTarget(context.Background(), toolRequest(`{"targets":[{"name":"one.example","organization":"organizations/8"}]}`))
	if err == nil || perTargetOrganization != nil {
		t.Fatalf("per-target organization unexpectedly accepted: %#v, %v", perTargetOrganization, err)
	}
	if len(calls) != 2 {
		t.Fatalf("invalid batches reached creator: %d calls", len(calls))
	}
}

func TestConstrainedCreationMapsNotFoundAndWriteResultBudget(t *testing.T) {
	registry := NewRegistry(Dependencies{
		TargetCreator: testTargetBatchCreator{create: func(context.Context, TargetBatchCreateInput) (TargetBatchCreateOutput, error) {
			return TargetBatchCreateOutput{}, mcpErrors.ErrNotFound
		}},
	})
	missing, err := registry.createTarget(context.Background(), toolRequest(`{"targets":[{"name":"example.test"}],"organization":"organizations/404"}`))
	if err != nil || missing == nil || !missing.IsError || toolFailureCategory(t, missing) != string(mcpErrors.CategoryNotFound) {
		t.Fatalf("missing organization = %#v, %v", missing, err)
	}

	registry = NewRegistry(Dependencies{
		TargetCreator: testTargetBatchCreator{create: func(context.Context, TargetBatchCreateInput) (TargetBatchCreateOutput, error) {
			return TargetBatchCreateOutput{FailedTargets: []FailedTarget{{Name: strings.Repeat("x", resultBudgetBytes)}}, FailedCount: 1}, nil
		}},
	})
	tooLarge, err := registry.createTarget(context.Background(), toolRequest(`{"targets":[{"name":"example.test"}]}`))
	if err != nil || tooLarge == nil || !tooLarge.IsError || toolFailureCategory(t, tooLarge) != string(mcpErrors.CategoryResultTooLarge) {
		t.Fatalf("oversized write result = %#v, %v", tooLarge, err)
	}
	encoded, marshalErr := json.Marshal(tooLarge)
	if marshalErr != nil {
		t.Fatalf("marshal oversized write result: %v", marshalErr)
	}
	if strings.Contains(string(encoded), strings.Repeat("x", 1024)) {
		t.Fatal("oversized write result contains a partial payload")
	}
}

func toolFailureCategory(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil || !result.IsError {
		t.Fatalf("expected tool failure, got %#v", result)
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("tool failure structured content = %#v", result.StructuredContent)
	}
	failure, ok := structured["error"].(map[string]string)
	if !ok {
		t.Fatalf("tool failure payload = %#v", structured["error"])
	}
	return failure["category"]
}

func TestToolHandlersUseClosedObjectArgumentsAndBoundedPagination(t *testing.T) {
	reader := testTargetReader{list: func(_ context.Context, query TargetQuery) (Page[TargetRecord], error) {
		if query.PageSize != 20 {
			t.Fatalf("default page size = %d, want 20", query.PageSize)
		}
		return Page[TargetRecord]{Items: []TargetRecord{{ID: 1, Name: "example.test"}}, TotalSize: 1}, nil
	}}
	registry := NewRegistry(Dependencies{Targets: reader})

	result, err := registry.listTargets(context.Background(), toolRequest(`{}`))
	if err != nil || result == nil || result.IsError {
		t.Fatalf("default list result = %#v, %v", result, err)
	}

	result, err = registry.listTargets(context.Background(), toolRequest(`{"page_size":101}`))
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("oversized page result = %#v, %v", result, err)
	}
	if !strings.Contains(string(result.StructuredContent.(map[string]any)["error"].(map[string]string)["category"]), "INVALID_ARGUMENT") {
		t.Fatalf("oversized page category = %#v", result.StructuredContent)
	}

	if _, err := registry.listTargets(context.Background(), toolRequest(`{"unknown":true}`)); err == nil {
		t.Fatal("unknown argument unexpectedly accepted")
	} else {
		var rpcErr *jsonrpc.Error
		if !errors.As(err, &rpcErr) || rpcErr.Code != jsonrpc.CodeInvalidParams {
			t.Fatalf("unknown argument error = %v", err)
		}
	}

	pageSchema := listSchema(map[string]any{})
	if pageSchema["type"] != "object" || pageSchema["additionalProperties"] != false {
		t.Fatalf("list schema is not a closed object: %#v", pageSchema)
	}
	properties := pageSchema["properties"].(map[string]any)
	pageSize := properties["page_size"].(map[string]any)
	if pageSize["default"] != schema.DefaultPageSize || pageSize["maximum"] != schema.MaxPageSize {
		t.Fatalf("page_size schema = %#v", pageSize)
	}
	detail := detailSchema()
	if detail["additionalProperties"] != false || len(detail["required"].([]string)) != 1 {
		t.Fatalf("detail schema = %#v", detail)
	}
}

func TestToolResultBudgetReturnsSafeErrorWithoutTruncation(t *testing.T) {
	registry := NewRegistry(Dependencies{Targets: testTargetReader{list: func(context.Context, TargetQuery) (Page[TargetRecord], error) {
		return Page[TargetRecord]{Items: []TargetRecord{{ID: 1, Name: strings.Repeat("x", resultBudgetBytes)}}, TotalSize: 1}, nil
	}}})

	result, err := registry.listTargets(context.Background(), toolRequest(`{}`))
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("over-budget result = %#v, %v", result, err)
	}
	encoded, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		t.Fatalf("marshal over-budget result: %v", marshalErr)
	}
	if len(encoded) > 4096 || strings.Contains(string(encoded), strings.Repeat("x", 1024)) {
		t.Fatalf("over-budget response contains a partial payload: %d bytes", len(encoded))
	}
	structured := result.StructuredContent.(map[string]any)
	if structured["error"].(map[string]string)["category"] != string(mcpErrors.CategoryResultTooLarge) {
		t.Fatalf("over-budget category = %#v", structured)
	}
}

func TestVulnerabilityListOmitsRawOutputButDetailIncludesIt(t *testing.T) {
	raw := json.RawMessage(`{"evidence":{"request":"GET /admin"}}`)
	reader := testVulnerabilityReader{
		list: func(context.Context, VulnerabilityQuery) (Page[VulnerabilityRecord], error) {
			return Page[VulnerabilityRecord]{Items: []VulnerabilityRecord{{ID: 3, RawOutput: nil}}, TotalSize: 1}, nil
		},
		get: func(context.Context, int) (VulnerabilityRecord, error) {
			return VulnerabilityRecord{ID: 3, RawOutput: raw}, nil
		},
	}
	registry := NewRegistry(Dependencies{Vulnerabilities: reader})

	listResult, err := registry.listVulnerabilities(context.Background(), toolRequest(`{}`))
	if err != nil || listResult == nil || listResult.IsError {
		t.Fatalf("vulnerability list = %#v, %v", listResult, err)
	}
	listPage := listResult.StructuredContent.(Page[VulnerabilityRecord])
	if len(listPage.Items) != 1 || len(listPage.Items[0].RawOutput) != 0 {
		t.Fatalf("vulnerability list exposed raw output: %#v", listPage)
	}

	detailResult, err := registry.getVulnerability(context.Background(), toolRequest(`{"id":3}`))
	if err != nil || detailResult == nil || detailResult.IsError {
		t.Fatalf("vulnerability detail = %#v, %v", detailResult, err)
	}
	detail := detailResult.StructuredContent.(VulnerabilityRecord)
	if string(detail.RawOutput) != string(raw) {
		t.Fatalf("vulnerability detail raw output = %s, want %s", detail.RawOutput, raw)
	}
}

func TestToolErrorsAreSanitizedAndDeadlineAware(t *testing.T) {
	registry := NewRegistry(Dependencies{Targets: testTargetReader{list: func(ctx context.Context, _ TargetQuery) (Page[TargetRecord], error) {
		if ctx.Err() != nil {
			return Page[TargetRecord]{}, ctx.Err()
		}
		return Page[TargetRecord]{}, errors.New("SQL SELECT secret from /srv/private stack trace")
	}}})
	result, err := registry.listTargets(context.Background(), toolRequest(`{}`))
	if err == nil || result != nil || strings.Contains(err.Error(), "SQL") || strings.Contains(err.Error(), "/srv/private") {
		t.Fatalf("internal error leaked or was converted to a tool result: result=%#v err=%v", result, err)
	}
	if !strings.Contains(err.Error(), "Internal MCP tool error") {
		t.Fatalf("internal error message = %v", err)
	}

	deadlineCtx, cancel := context.WithCancel(context.Background())
	cancel()
	deadlineResult, deadlineErr := registry.listTargets(deadlineCtx, toolRequest(`{}`))
	if !errors.Is(deadlineErr, context.Canceled) || deadlineResult != nil {
		t.Fatalf("cancelled tool = %#v, %v", deadlineResult, deadlineErr)
	}

	result, err = successResult(deadlineCtx, map[string]string{"ok": "no"}, "ignored")
	if !errors.Is(err, context.Canceled) || result != nil {
		t.Fatalf("cancelled success result = %#v, %v", result, err)
	}

}
