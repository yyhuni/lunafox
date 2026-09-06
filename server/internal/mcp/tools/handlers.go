package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
)

func (registry *Registry) listTargets(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input targetListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_targets arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	if input.Type != "" && input.Type != "domain" && input.Type != "ip" && input.Type != "cidr" {
		return toolError(ctx, fmt.Errorf("%w: unsupported target type", mcpErrors.ErrInvalidInput))
	}
	orderBy, err := normalizeOrder(input.OrderBy, map[string]string{
		"name": "displayName asc", "name desc": "displayName desc",
		"created_at": "createdAt asc", "created_at desc": "createdAt desc",
		"last_scanned_at": "lastScannedAt asc", "last_scanned_at desc": "lastScannedAt desc",
	})
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Targets == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.Targets.List(ctx, TargetQuery{
		PageSize: pageSize, PageToken: input.PageToken, Name: strings.TrimSpace(input.Name), Type: input.Type, OrderBy: orderBy,
	})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("targets", len(page.Items)))
}

func (registry *Registry) getTarget(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input detailInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid get_target arguments")
	}
	if err := validateID(input.ID); err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Targets == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	item, err := registry.deps.Targets.Get(ctx, input.ID)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, item, fmt.Sprintf("Returned target %d.", item.ID))
}

func (registry *Registry) listScans(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input scanListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_scans arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	if input.TargetID < 0 {
		return toolError(ctx, fmt.Errorf("%w: target_id must be positive", mcpErrors.ErrInvalidInput))
	}
	orderBy, err := normalizeOrder(input.OrderBy, map[string]string{"created_at": "createdAt asc", "created_at desc": "createdAt desc"})
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Scans == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.Scans.List(ctx, ScanQuery{
		PageSize: pageSize, PageToken: input.PageToken, TargetID: input.TargetID, Status: strings.TrimSpace(input.Status), OrderBy: orderBy,
	})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("scans", len(page.Items)))
}

func (registry *Registry) getScan(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input detailInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid get_scan arguments")
	}
	if err := validateID(input.ID); err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Scans == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	item, err := registry.deps.Scans.Get(ctx, input.ID)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, item, fmt.Sprintf("Returned scan %d.", item.ID))
}

func (registry *Registry) listWebsites(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input assetListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_target_websites arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	query, err := input.assetQuery(pageSize)
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Websites == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.Websites.List(ctx, input.TargetID, query)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("websites", len(page.Items)))
}

func (registry *Registry) listSubdomains(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input assetListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_target_subdomains arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	query, err := input.assetQuery(pageSize)
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Subdomains == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.Subdomains.List(ctx, input.TargetID, query)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("subdomains", len(page.Items)))
}

func (registry *Registry) listEndpoints(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input assetListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_target_endpoints arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	query, err := input.assetQuery(pageSize)
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Endpoints == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.Endpoints.List(ctx, input.TargetID, query)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("endpoints", len(page.Items)))
}

func (registry *Registry) listDirectories(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input assetListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_target_directories arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	query, err := input.assetQuery(pageSize)
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Directories == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.Directories.List(ctx, input.TargetID, query)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("directories", len(page.Items)))
}

func (registry *Registry) listHostPorts(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input assetListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_target_host_ports arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	query, err := input.assetQuery(pageSize)
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.HostPorts == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.HostPorts.List(ctx, input.TargetID, query)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("host-port groups", len(page.Items)))
}

func (registry *Registry) listVulnerabilities(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input vulnerabilityListInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid list_vulnerabilities arguments")
	}
	pageSize, err := validatePage(input.pageInput)
	if err != nil {
		return toolError(ctx, err)
	}
	if input.Severity != "" {
		valid := map[string]bool{"unknown": true, "info": true, "low": true, "medium": true, "high": true, "critical": true}
		if !valid[input.Severity] {
			return toolError(ctx, fmt.Errorf("%w: unsupported severity", mcpErrors.ErrInvalidInput))
		}
	}
	orderBy, err := normalizeOrder(input.OrderBy, map[string]string{"created_at": "createdAt asc", "created_at desc": "createdAt desc"})
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Vulnerabilities == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	page, err := registry.deps.Vulnerabilities.List(ctx, VulnerabilityQuery{
		PageSize: pageSize, PageToken: input.PageToken, OrderBy: orderBy,
		URL: strings.TrimSpace(input.URL), Severity: input.Severity, Source: strings.TrimSpace(input.Source),
		VulnType: strings.TrimSpace(input.VulnType), Reviewed: input.Reviewed,
	})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, page, listSummary("vulnerabilities", len(page.Items)))
}

func (registry *Registry) getVulnerability(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input detailInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid get_vulnerability arguments")
	}
	if err := validateID(input.ID); err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.Vulnerabilities == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	item, err := registry.deps.Vulnerabilities.Get(ctx, input.ID)
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, item, fmt.Sprintf("Returned vulnerability %d.", item.ID))
}

func decodeRequest(req *mcp.CallToolRequest, target any) error {
	if req == nil || req.Params == nil {
		return errors.New("missing tool parameters")
	}
	return decodeArguments(req.Params.Arguments, target)
}

func (input assetListInput) assetQuery(pageSize int) (AssetQuery, error) {
	if err := validateID(input.TargetID); err != nil {
		return AssetQuery{}, err
	}
	orderBy, err := normalizeOrder(input.OrderBy, map[string]string{
		"created_at": "createdAt asc", "created_at desc": "createdAt desc",
		"status_code": "statusCode asc", "status_code desc": "statusCode desc",
		"content_length": "contentLength asc", "content_length desc": "contentLength desc",
		"url": "url asc", "url desc": "url desc", "dns_name": "dnsName asc", "dns_name desc": "dnsName desc",
		"ip": "ip asc", "ip desc": "ip desc",
	})
	if err != nil {
		return AssetQuery{}, err
	}
	if input.Port != nil && (*input.Port < 1 || *input.Port > 65535) {
		return AssetQuery{}, fmt.Errorf("%w: port out of range", mcpErrors.ErrInvalidInput)
	}
	return AssetQuery{PageSize: pageSize, PageToken: input.PageToken, OrderBy: orderBy, URL: strings.TrimSpace(input.URL), DNSName: strings.TrimSpace(input.DNSName), Host: strings.TrimSpace(input.Host), IP: strings.TrimSpace(input.IP), Port: input.Port, StatusCode: input.StatusCode, Webserver: strings.TrimSpace(input.Webserver), ContentType: strings.TrimSpace(input.ContentType), Tech: strings.TrimSpace(input.Tech), Vhost: input.Vhost}, nil
}
