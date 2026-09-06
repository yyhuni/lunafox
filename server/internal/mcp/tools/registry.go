package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/mcp/schema"
)

const resultBudgetBytes = 1 << 20

// maxImageBytes leaves room for the JSON/base64 representation and the fixed
// MCP metadata so a binary image cannot bypass the whole-result 1 MiB budget.
const maxImageBytes = ((resultBudgetBytes - 4096) * 3) / 4

// ResultBudgetBytes is the maximum serialized result size exposed at the MCP boundary.
const ResultBudgetBytes = resultBudgetBytes

// MaxImageBytes is the raw image ceiling compatible with ResultBudgetBytes.
const MaxImageBytes = maxImageBytes

const (
	ToolListTargets                  = "list_targets"
	ToolGetTarget                    = "get_target"
	ToolListScans                    = "list_scans"
	ToolGetScan                      = "get_scan"
	ToolListWebsites                 = "list_target_websites"
	ToolListSubdomains               = "list_target_subdomains"
	ToolListEndpoints                = "list_target_endpoints"
	ToolListDirectories              = "list_target_directories"
	ToolListHostPorts                = "list_target_host_ports"
	ToolListVulnerabilities          = "list_vulnerabilities"
	ToolGetVulnerability             = "get_vulnerability"
	ToolCreateOrganization           = "create_organization"
	ToolCreateTarget                 = "create_target"
	ToolListOrganizations            = "list_organizations"
	ToolGetOrganization              = "get_organization"
	ToolListOrganizationTargets      = "list_organization_targets"
	ToolListTargetVulnerabilities    = "list_target_vulnerabilities"
	ToolListTargetScreenshots        = "list_target_screenshots"
	ToolGetScreenshotImage           = "get_screenshot_image"
	ToolListScanWorkflows            = "list_scan_workflows"
	ToolGetScanWorkflow              = "get_scan_workflow"
	ToolGetScanWorkflowProfile       = "get_scan_workflow_profile"
	ToolListEngines                  = "list_engines"
	ToolGetEngine                    = "get_engine"
	ToolListWordlists                = "list_wordlists"
	ToolGetWordlist                  = "get_wordlist"
	ToolListServerLogEntries         = "list_server_log_entries"
	ToolListAgentLogEntries          = "list_agent_log_entries"
	ToolReviewVulnerability          = "review_vulnerability"
	ToolUnreviewVulnerability        = "unreview_vulnerability"
	ToolBatchReviewVulnerabilities   = "batch_review_vulnerabilities"
	ToolBatchUnreviewVulnerabilities = "batch_unreview_vulnerabilities"
	ToolStartScan                    = "start_scan"
	ToolGetOperation                 = "get_operation"
)

// Registry owns LunaFox's bounded investigation and constrained-creation tool
// catalog. It has no operational or unapproved mutation dependencies.
type Registry struct {
	deps Dependencies
}

func NewRegistry(deps Dependencies) *Registry { return &Registry{deps: deps} }

// ToolNames is the stable allowlisted catalog used by transport admission and
// interoperability tests.
func ToolNames() []string {
	return []string{
		ToolListTargets, ToolGetTarget, ToolListScans, ToolGetScan,
		ToolListWebsites, ToolListSubdomains, ToolListEndpoints,
		ToolListDirectories, ToolListHostPorts, ToolListVulnerabilities,
		ToolGetVulnerability, ToolCreateOrganization, ToolCreateTarget,
		ToolListOrganizations, ToolGetOrganization, ToolListOrganizationTargets,
		ToolListTargetVulnerabilities, ToolListTargetScreenshots, ToolGetScreenshotImage,
		ToolListScanWorkflows, ToolGetScanWorkflow, ToolGetScanWorkflowProfile,
		ToolListEngines, ToolGetEngine, ToolListWordlists, ToolGetWordlist,
		ToolListServerLogEntries, ToolListAgentLogEntries,
		ToolReviewVulnerability, ToolUnreviewVulnerability,
		ToolBatchReviewVulnerabilities, ToolBatchUnreviewVulnerabilities,
		ToolStartScan, ToolGetOperation,
	}
}

func (registry *Registry) HasTool(name string) bool {
	for _, candidate := range ToolNames() {
		if candidate == name {
			return true
		}
	}
	return false
}

// Register adds the complete catalog to an SDK server. Server instances are
// stateless and may be created per HTTP request by the transport.
func (registry *Registry) Register(server *mcp.Server) {
	server.AddTool(&mcp.Tool{
		Name:        ToolListTargets,
		Description: "List investigation targets with bounded pagination.",
		InputSchema: listSchema(map[string]any{
			"name":     map[string]any{"type": "string", "description": "Exact target name."},
			"type":     map[string]any{"type": "string", "enum": []string{"domain", "ip", "cidr"}},
			"order_by": map[string]any{"type": "string", "enum": []string{"name", "name desc", "created_at", "created_at desc", "last_scanned_at", "last_scanned_at desc"}},
		}),
	}, registry.listTargets)
	server.AddTool(&mcp.Tool{
		Name:        ToolGetTarget,
		Description: "Get one target and its persisted asset summary.",
		InputSchema: detailSchema(),
	}, registry.getTarget)
	server.AddTool(&mcp.Tool{
		Name:        ToolListScans,
		Description: "List scan records with bounded pagination.",
		InputSchema: listSchema(map[string]any{
			"target_id": map[string]any{"type": "integer", "minimum": 1},
			"status":    map[string]any{"type": "string"},
			"order_by":  map[string]any{"type": "string", "enum": []string{"created_at", "created_at desc"}},
		}),
	}, registry.listScans)
	server.AddTool(&mcp.Tool{
		Name:        ToolGetScan,
		Description: "Get one scan's read-only status and summary.",
		InputSchema: detailSchema(),
	}, registry.getScan)
	server.AddTool(&mcp.Tool{
		Name:        ToolListWebsites,
		Description: "List websites discovered under a target.",
		InputSchema: assetListSchema(map[string]any{
			"url":          map[string]any{"type": "string"},
			"status_code":  map[string]any{"type": "integer"},
			"webserver":    map[string]any{"type": "string"},
			"content_type": map[string]any{"type": "string"},
			"tech":         map[string]any{"type": "string"},
			"vhost":        map[string]any{"type": "boolean"},
		}, []string{"created_at", "created_at desc", "status_code", "status_code desc", "content_length", "content_length desc"}),
	}, registry.listWebsites)
	server.AddTool(&mcp.Tool{
		Name:        ToolListSubdomains,
		Description: "List subdomains discovered under a target.",
		InputSchema: assetListSchema(map[string]any{"dns_name": map[string]any{"type": "string"}}, []string{"created_at", "created_at desc", "dns_name", "dns_name desc"}),
	}, registry.listSubdomains)
	server.AddTool(&mcp.Tool{
		Name:        ToolListEndpoints,
		Description: "List endpoints discovered under a target.",
		InputSchema: assetListSchema(map[string]any{
			"url":          map[string]any{"type": "string"},
			"status_code":  map[string]any{"type": "integer"},
			"webserver":    map[string]any{"type": "string"},
			"content_type": map[string]any{"type": "string"},
			"tech":         map[string]any{"type": "string"},
			"vhost":        map[string]any{"type": "boolean"},
		}, []string{"created_at", "created_at desc", "status_code", "status_code desc", "content_length", "content_length desc"}),
	}, registry.listEndpoints)
	server.AddTool(&mcp.Tool{
		Name:        ToolListDirectories,
		Description: "List directories discovered under a target.",
		InputSchema: assetListSchema(map[string]any{
			"url":          map[string]any{"type": "string"},
			"status_code":  map[string]any{"type": "integer"},
			"content_type": map[string]any{"type": "string"},
		}, []string{"created_at", "created_at desc", "status_code", "status_code desc", "content_length", "content_length desc"}),
	}, registry.listDirectories)
	server.AddTool(&mcp.Tool{
		Name:        ToolListHostPorts,
		Description: "List aggregated host and port observations under a target.",
		InputSchema: assetListSchema(map[string]any{
			"ip":   map[string]any{"type": "string"},
			"host": map[string]any{"type": "string"},
			"port": map[string]any{"type": "integer", "minimum": 1, "maximum": 65535},
		}, []string{"created_at", "created_at desc", "ip", "ip desc"}),
	}, registry.listHostPorts)
	server.AddTool(&mcp.Tool{
		Name:        ToolListVulnerabilities,
		Description: "List persisted vulnerability findings.",
		InputSchema: listSchema(map[string]any{
			"url":       map[string]any{"type": "string"},
			"severity":  map[string]any{"type": "string", "enum": []string{"unknown", "info", "low", "medium", "high", "critical"}},
			"source":    map[string]any{"type": "string"},
			"vuln_type": map[string]any{"type": "string"},
			"reviewed":  map[string]any{"type": "boolean"},
			"order_by":  map[string]any{"type": "string", "enum": []string{"created_at", "created_at desc"}},
		}),
	}, registry.listVulnerabilities)
	server.AddTool(&mcp.Tool{
		Name:        ToolGetVulnerability,
		Description: "Get one vulnerability and its persisted structured evidence.",
		InputSchema: detailSchema(),
	}, registry.getVulnerability)
	server.AddTool(&mcp.Tool{
		Name:        ToolCreateOrganization,
		Description: "Immediately create one standalone business organization group. This mutates LunaFox data.",
		InputSchema: createOrganizationSchema(),
		Annotations: constrainedWriteAnnotations(),
	}, registry.createOrganization)
	server.AddTool(&mcp.Tool{
		Name:        ToolCreateTarget,
		Description: "Immediately create or merge a batch of 1 to 5000 targets, optionally associating the whole batch with one existing organization. This mutates LunaFox data.",
		InputSchema: createTargetSchema(),
		Annotations: constrainedWriteAnnotations(),
	}, registry.createTarget)
	registry.registerInvestigationTools(server)
}

type pageInput struct {
	PageSize  int    `json:"page_size,omitempty"`
	PageToken string `json:"page_token,omitempty"`
}

type targetListInput struct {
	pageInput
	Name    string `json:"name,omitempty"`
	Type    string `json:"type,omitempty"`
	OrderBy string `json:"order_by,omitempty"`
}

type scanListInput struct {
	pageInput
	TargetID int    `json:"target_id,omitempty"`
	Status   string `json:"status,omitempty"`
	OrderBy  string `json:"order_by,omitempty"`
}

type detailInput struct {
	ID int `json:"id"`
}

type assetListInput struct {
	pageInput
	TargetID    int    `json:"target_id"`
	OrderBy     string `json:"order_by,omitempty"`
	URL         string `json:"url,omitempty"`
	DNSName     string `json:"dns_name,omitempty"`
	Host        string `json:"host,omitempty"`
	IP          string `json:"ip,omitempty"`
	Port        *int   `json:"port,omitempty"`
	StatusCode  *int   `json:"status_code,omitempty"`
	Webserver   string `json:"webserver,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Tech        string `json:"tech,omitempty"`
	Vhost       *bool  `json:"vhost,omitempty"`
}

type vulnerabilityListInput struct {
	pageInput
	OrderBy  string `json:"order_by,omitempty"`
	URL      string `json:"url,omitempty"`
	Severity string `json:"severity,omitempty"`
	Source   string `json:"source,omitempty"`
	VulnType string `json:"vuln_type,omitempty"`
	Reviewed *bool  `json:"reviewed,omitempty"`
}

func listSchema(properties map[string]any) map[string]any {
	return objectSchema(withPageProperties(properties))
}

func assetListSchema(properties map[string]any, orderBy []string) map[string]any {
	properties["target_id"] = map[string]any{"type": "integer", "minimum": 1}
	properties["order_by"] = map[string]any{"type": "string", "enum": orderBy}
	return objectSchema(withPageProperties(properties), "target_id")
}

func detailSchema() map[string]any {
	return objectSchema(map[string]any{
		"id": map[string]any{"type": "integer", "minimum": 1},
	}, "id")
}

func withPageProperties(properties map[string]any) map[string]any {
	result := map[string]any{
		"page_size":  map[string]any{"type": "integer", "minimum": 1, "maximum": schema.MaxPageSize, "default": schema.DefaultPageSize},
		"page_token": map[string]any{"type": "string"},
	}
	for key, value := range properties {
		result[key] = value
	}
	return result
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func decodeArguments(raw json.RawMessage, target any) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	if raw[0] != '{' {
		return errors.New("tool arguments must be a JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("multiple JSON values")
	} else if !errors.Is(err, io.EOF) {
		return errors.New("invalid trailing JSON")
	}
	return nil
}

func invalidParams(message string) error {
	return &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: message}
}

func validatePage(input pageInput) (int, error) {
	pageSize, err := schema.NormalizePageSize(input.PageSize)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", mcpErrors.ErrInvalidInput, err)
	}
	return pageSize, nil
}

func validateID(id int) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", mcpErrors.ErrInvalidInput)
	}
	return nil
}

func normalizeOrder(value string, allowed map[string]string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if normalized, ok := allowed[trimmed]; ok {
		return normalized, nil
	}
	return "", fmt.Errorf("%w: unsupported order_by", mcpErrors.ErrInvalidInput)
}

func quoteFilter(value string) string { return strconv.Quote(strings.TrimSpace(value)) }

func toolError(ctx context.Context, err error) (*mcp.CallToolResult, error) {
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return nil, context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = mcpErrors.ErrDeadlineExceeded
	}
	if mcpErrors.IsExpected(err) {
		failure := mcpErrors.From(err)
		if state := CallStateFromContext(ctx); state != nil {
			state.Mark(string(failure.Category), "")
		}
		text := string(failure.Category) + ": " + failure.Message + " Recovery: " + failure.Recovery
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
			StructuredContent: map[string]any{
				"error": map[string]string{
					"category": string(failure.Category),
					"message":  failure.Message,
					"recovery": failure.Recovery,
				},
			},
		}, nil
	}
	if state := CallStateFromContext(ctx); state != nil {
		state.Mark(string(mcpErrors.CategoryInternal), "-32603")
	}
	return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: "Internal MCP tool error"}
}

func successResult(ctx context.Context, output any, summary string) (*mcp.CallToolResult, error) {
	return enforceResultBudget(ctx, &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: summary}},
		StructuredContent: output,
	})
}

// enforceResultBudget measures the complete MCP wire result, including every
// Content block (for example the base64 representation of ImageContent), the
// structured payload, and fixed protocol metadata. Measuring only the
// structured value would let a large unstructured block bypass the boundary.
func enforceResultBudget(ctx context.Context, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
	if result == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	if err := ctx.Err(); err != nil {
		return toolError(ctx, err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	if len(encoded) > resultBudgetBytes {
		return toolError(ctx, mcpErrors.ErrResultTooLarge)
	}
	if !result.IsError {
		if state := CallStateFromContext(ctx); state != nil {
			state.Mark("success", "")
		}
	}
	return result, nil
}

func listSummary(kind string, count int) string {
	return fmt.Sprintf("Returned %d %s.", count, kind)
}
