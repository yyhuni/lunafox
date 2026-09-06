package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type organizationListInput struct {
	pageInput
	Filter  string `json:"filter,omitempty"`
	OrderBy string `json:"order_by,omitempty"`
}

type organizationTargetsInput struct {
	pageInput
	Organization string `json:"organization"`
	Type         string `json:"type,omitempty"`
	Filter       string `json:"filter,omitempty"`
}

type targetScopedVulnerabilityInput struct {
	pageInput
	Target   string `json:"target"`
	OrderBy  string `json:"order_by,omitempty"`
	URL      string `json:"url,omitempty"`
	Severity string `json:"severity,omitempty"`
	Source   string `json:"source,omitempty"`
	VulnType string `json:"vuln_type,omitempty"`
	Reviewed *bool  `json:"reviewed,omitempty"`
}

type targetScopedScreenshotInput struct {
	pageInput
	Target  string `json:"target"`
	Filter  string `json:"filter,omitempty"`
	OrderBy string `json:"order_by,omitempty"`
}

type screenshotImageInput struct {
	Target     string `json:"target"`
	Screenshot string `json:"screenshot"`
}

type catalogListInput struct {
	pageInput
	Filter  string `json:"filter,omitempty"`
	OrderBy string `json:"order_by,omitempty"`
}

type resourceNameInput struct {
	Name string `json:"name"`
}

type workflowProfileInput struct {
	ScanWorkflow string `json:"scan_workflow"`
}

type logListInput struct {
	pageInput
	Direction string `json:"direction,omitempty"`
}

type agentLogListInput struct {
	logListInput
	AgentID   int    `json:"agent_id"`
	Container string `json:"container"`
}

type startScanToolInput struct {
	Target        string         `json:"target"`
	ScanWorkflow  string         `json:"scan_workflow"`
	Configuration map[string]any `json:"configuration"`
	Agent         string         `json:"agent,omitempty"`
	RequestID     string         `json:"request_id,omitempty"`
}

type screenshotImageMetadata struct {
	Name   string `json:"name"`
	Target string `json:"target"`
}

func (registry *Registry) listOrganizations(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input organizationListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_organizations arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	orderBy, err := normalizeOrder(input.OrderBy, map[string]string{
		"display_name": "displayName asc", "display_name desc": "displayName desc",
		"created_at": "createdAt asc", "created_at desc": "createdAt desc",
	})
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.OrganizationQueries == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.OrganizationQueries.List(ctx, OrganizationQuery{PageSize: pageSize, PageToken: input.PageToken, Filter: strings.TrimSpace(input.Filter), OrderBy: orderBy})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("organizations", len(page.Items)))
}

func (registry *Registry) getOrganization(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input resourceNameInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid get_organization arguments")
	}
	id, err := parseCanonicalResource(input.Name, "organizations")
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.OrganizationQueries == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	item, err := registry.deps.OrganizationQueries.Get(ctx, id)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, item, fmt.Sprintf("Returned organization %d.", item.ID))
}

func (registry *Registry) listOrganizationTargets(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input organizationTargetsInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_organization_targets arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	organizationID, err := parseCanonicalResource(input.Organization, "organizations")
	if err != nil {
		return toolError(ctx, err)
	}
	if input.Type != "" && input.Type != "domain" && input.Type != "ip" && input.Type != "cidr" {
		return toolError(ctx, fmt.Errorf("%w: unsupported target type", mcpErrors.ErrInvalidInput))
	}
	if registry.deps.OrganizationQueries == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.OrganizationQueries.ListTargets(ctx, organizationID, TargetQuery{PageSize: pageSize, PageToken: input.PageToken, Type: input.Type, Filter: strings.TrimSpace(input.Filter)})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("organization targets", len(page.Items)))
}

func (registry *Registry) listTargetVulnerabilities(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input targetScopedVulnerabilityInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_target_vulnerabilities arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	targetID, err := parseCanonicalTarget(input.Target)
	if err != nil {
		return toolError(ctx, err)
	}
	if input.Severity != "" && !validVulnerabilitySeverity(input.Severity) {
		return toolError(ctx, fmt.Errorf("%w: unsupported severity", mcpErrors.ErrInvalidInput))
	}
	orderBy, err := normalizeOrder(input.OrderBy, map[string]string{"created_at": "createdAt asc", "created_at desc": "createdAt desc"})
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.TargetVulns == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.TargetVulns.ListByTarget(ctx, targetID, VulnerabilityQuery{PageSize: pageSize, PageToken: input.PageToken, OrderBy: orderBy, URL: strings.TrimSpace(input.URL), Severity: input.Severity, Source: strings.TrimSpace(input.Source), VulnType: strings.TrimSpace(input.VulnType), Reviewed: input.Reviewed})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("target vulnerabilities", len(page.Items)))
}

func (registry *Registry) listTargetScreenshots(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input targetScopedScreenshotInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_target_screenshots arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	targetID, err := parseCanonicalTarget(input.Target)
	if err != nil {
		return toolError(ctx, err)
	}
	orderBy, err := normalizeOrder(input.OrderBy, map[string]string{
		"created_at": "createdAt asc", "created_at desc": "createdAt desc",
		"status_code": "statusCode asc", "status_code desc": "statusCode desc",
	})
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Screenshots == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.Screenshots.ListByTarget(ctx, targetID, ScreenshotQuery{PageSize: pageSize, PageToken: input.PageToken, Filter: strings.TrimSpace(input.Filter), OrderBy: orderBy})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("target screenshots", len(page.Items)))
}

func (registry *Registry) getScreenshotImage(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input screenshotImageInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid get_screenshot_image arguments")
	}
	targetID, err := parseCanonicalTarget(input.Target)
	if err != nil {
		return toolError(ctx, err)
	}
	screenshotID, err := parseCanonicalScreenshot(input.Screenshot, targetID)
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Screenshots == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	image, err := registry.deps.Screenshots.GetImage(ctx, targetID, screenshotID)
	if err != nil {
		return toolError(ctx, err)
	}
	if len(image) == 0 {
		return toolError(ctx, mcpErrors.ErrNotFound)
	}
	if len(image) > MaxImageBytes {
		return toolError(ctx, mcpErrors.ErrResultTooLarge)
	}
	return enforceResultBudget(ctx, &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.ImageContent{Data: image, MIMEType: "image/webp"}},
		StructuredContent: screenshotImageMetadata{Name: input.Screenshot, Target: input.Target},
	})
}

func (registry *Registry) listScanWorkflows(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input catalogListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_scan_workflows arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	if input.OrderBy != "" {
		return toolError(ctx, fmt.Errorf("%w: scan workflow order is fixed", mcpErrors.ErrInvalidInput))
	}
	if registry.deps.Workflows == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.Workflows.List(ctx, CatalogListQuery{PageSize: pageSize, PageToken: input.PageToken, Filter: strings.TrimSpace(input.Filter)})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("scan workflows", len(page.Items)))
}

func (registry *Registry) getScanWorkflow(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input resourceNameInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid get_scan_workflow arguments")
	}
	if !isCanonicalScanWorkflow(input.Name) {
		return toolError(ctx, mcpErrors.ErrInvalidInput)
	}
	if registry.deps.Workflows == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	item, err := registry.deps.Workflows.Get(ctx, input.Name)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, item, "Returned scan workflow.")
}

func (registry *Registry) getScanWorkflowProfile(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input workflowProfileInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid get_scan_workflow_profile arguments")
	}
	if !isCanonicalScanWorkflow(input.ScanWorkflow) {
		return toolError(ctx, mcpErrors.ErrInvalidInput)
	}
	if registry.deps.Profiles == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	item, err := registry.deps.Profiles.GetProfile(ctx, input.ScanWorkflow)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, item, "Returned scan workflow profile.")
}

func (registry *Registry) listEngines(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input catalogListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_engines arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	if strings.TrimSpace(input.Filter) != "" || strings.TrimSpace(input.OrderBy) != "" {
		return toolError(ctx, mcpErrors.ErrInvalidInput)
	}
	if registry.deps.Engines == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.Engines.List(ctx, CatalogListQuery{PageSize: pageSize, PageToken: input.PageToken})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("engines", len(page.Items)))
}

func (registry *Registry) getEngine(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input resourceNameInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid get_engine arguments")
	}
	if !isCanonicalEngine(input.Name) {
		return toolError(ctx, mcpErrors.ErrInvalidInput)
	}
	if registry.deps.Engines == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	item, err := registry.deps.Engines.Get(ctx, input.Name)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, item, "Returned engine.")
}

func (registry *Registry) listWordlists(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input catalogListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_wordlists arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	orderBy, err := normalizeOrder(input.OrderBy, map[string]string{
		"file_name": "fileName asc", "file_name desc": "fileName desc",
		"line_count": "lineCount asc", "line_count desc": "lineCount desc",
		"file_size": "fileSize asc", "file_size desc": "fileSize desc",
		"updated_at": "updatedAt asc", "updated_at desc": "updatedAt desc",
	})
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Wordlists == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.Wordlists.List(ctx, CatalogListQuery{PageSize: pageSize, PageToken: input.PageToken, Filter: strings.TrimSpace(input.Filter), OrderBy: orderBy})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("wordlists", len(page.Items)))
}

func (registry *Registry) getWordlist(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input resourceNameInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid get_wordlist arguments")
	}
	id, err := parseCanonicalWordlist(input.Name)
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Wordlists == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	item, err := registry.deps.Wordlists.Get(ctx, id)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, item, fmt.Sprintf("Returned wordlist %d.", item.ID))
}

func (registry *Registry) listServerLogEntries(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input logListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_server_log_entries arguments")
	}
	pageSize, err := validateLogPage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	if !validLogDirection(input.Direction) {
		return toolError(ctx, mcpErrors.ErrInvalidInput)
	}
	if registry.deps.ServerLogs == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.ServerLogs.List(ctx, LogQuery{PageSize: pageSize, PageToken: input.PageToken, Direction: normalizeLogDirection(input.Direction)})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("server log entries", len(page.Items)))
}

func (registry *Registry) listAgentLogEntries(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input agentLogListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_agent_log_entries arguments")
	}
	pageSize, err := validateLogPage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	if input.AgentID <= 0 || strings.TrimSpace(input.Container) == "" || !validLogDirection(input.Direction) {
		return toolError(ctx, mcpErrors.ErrInvalidInput)
	}
	if registry.deps.AgentLogs == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.AgentLogs.List(ctx, AgentLogQuery{AgentID: input.AgentID, Container: strings.TrimSpace(input.Container), PageSize: pageSize, PageToken: input.PageToken, Direction: normalizeLogDirection(input.Direction)})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("agent log entries", len(page.Items)))
}

func (registry *Registry) startScan(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input startScanToolInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid start_scan arguments")
	}
	if _, err := parseCanonicalTarget(input.Target); err != nil || !isCanonicalScanWorkflow(input.ScanWorkflow) || input.Configuration == nil {
		return toolError(ctx, mcpErrors.ErrInvalidInput)
	}
	if strings.TrimSpace(input.Agent) != "" && !isCanonicalAgent(input.Agent) {
		return toolError(ctx, mcpErrors.ErrInvalidInput)
	}
	if registry.deps.ScanStarter == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	result, err := registry.deps.ScanStarter.Start(ctx, StartScanInput{
		Target: input.Target, ScanWorkflow: input.ScanWorkflow, Configuration: input.Configuration, Agent: input.Agent, RequestID: input.RequestID,
	})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, result, "Started one scan and created a polling operation.")
}

func (registry *Registry) getOperation(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input resourceNameInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid get_operation arguments")
	}
	if !isCanonicalOperation(input.Name) {
		return toolError(ctx, mcpErrors.ErrInvalidInput)
	}
	if registry.deps.Operations == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	result, err := registry.deps.Operations.GetOperation(ctx, input.Name)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, result, "Returned scan operation.")
}

func parseCanonicalResource(name, collection string) (int, error) {
	id, err := httpdto.ParseResourceNameID(name, collection)
	if err != nil || fmt.Sprintf("%s/%d", collection, id) != strings.TrimSpace(name) {
		return 0, fmt.Errorf("%w: canonical %s resource name is required", mcpErrors.ErrInvalidInput, collection)
	}
	return id, nil
}

func parseCanonicalTarget(name string) (int, error) {
	id, err := resourcenames.ParseTarget(name)
	if err != nil || resourcenames.Target(id) != strings.TrimSpace(name) {
		return 0, fmt.Errorf("%w: canonical targets/{id} resource name is required", mcpErrors.ErrInvalidInput)
	}
	return id, nil
}

func parseCanonicalWordlist(name string) (int, error) {
	id, err := resourcenames.ParseWordlist(name)
	if err != nil || resourcenames.Wordlist(id) != strings.TrimSpace(name) {
		return 0, fmt.Errorf("%w: canonical wordlists/{id} resource name is required", mcpErrors.ErrInvalidInput)
	}
	return id, nil
}

func parseCanonicalScreenshot(name string, targetID int) (int, error) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 4 || parts[0] != "targets" || parts[2] != "screenshots" {
		return 0, fmt.Errorf("%w: canonical target screenshot resource name is required", mcpErrors.ErrInvalidInput)
	}
	parent, err := parseCanonicalTarget(parts[0] + "/" + parts[1])
	if err != nil || parent != targetID {
		return 0, fmt.Errorf("%w: screenshot target does not match target", mcpErrors.ErrInvalidInput)
	}
	return parseCanonicalResource("screenshots/"+parts[3], "screenshots")
}

func isCanonicalScanWorkflow(name string) bool {
	id, err := resourcenames.ParseScanWorkflow(name)
	return err == nil && resourcenames.ScanWorkflow(id) == strings.TrimSpace(name)
}

func isCanonicalEngine(name string) bool {
	id, err := resourcenames.ParseEngine(name)
	return err == nil && resourcenames.Engine(id) == strings.TrimSpace(name)
}

func isCanonicalAgent(name string) bool {
	id, err := httpdto.ParseResourceNameID(name, "agents")
	return err == nil && httpdto.AgentName(id) == strings.TrimSpace(name)
}

func isCanonicalOperation(name string) bool {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 2 || parts[0] != "operations" {
		return false
	}
	if len(parts[1]) != 36 {
		return false
	}
	for index, character := range parts[1] {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return false
			}
			continue
		}
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func validVulnerabilitySeverity(value string) bool {
	switch value {
	case "unknown", "info", "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func validateLogPage(input pageInput) (int, error) {
	if input.PageSize == 0 {
		return 200, nil
	}
	if input.PageSize < 1 || input.PageSize > 500 {
		return 0, fmt.Errorf("%w: page_size must be between 1 and 500", mcpErrors.ErrInvalidInput)
	}
	return input.PageSize, nil
}

func validLogDirection(value string) bool {
	switch normalizeLogDirection(value) {
	case "", "newer", "older":
		return true
	default:
		return false
	}
}

func normalizeLogDirection(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
