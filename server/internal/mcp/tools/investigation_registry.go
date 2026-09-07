package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yyhuni/lunafox/server/internal/mcp/schema"
)

func (registry *Registry) registerInvestigationTools(server *mcp.Server) {
	server.AddTool(&mcp.Tool{
		Name:        ToolListOrganizations,
		Description: "List deployment-wide organization business groups with bounded pagination.",
		InputSchema: listSchema(map[string]any{
			"filter":   map[string]any{"type": "string"},
			"order_by": map[string]any{"type": "string", "enum": []string{"display_name", "display_name desc", "created_at", "created_at desc"}},
		}),
	}, registry.listOrganizations)
	server.AddTool(&mcp.Tool{
		Name:        ToolGetOrganization,
		Description: "Get one organization business group by canonical resource name.",
		InputSchema: canonicalNameSchema("organizations"),
	}, registry.getOrganization)
	server.AddTool(&mcp.Tool{
		Name:        ToolListOrganizationTargets,
		Description: "List targets associated with one organization business group.",
		InputSchema: organizationTargetsSchema(),
	}, registry.listOrganizationTargets)
	server.AddTool(&mcp.Tool{
		Name:        ToolListTargetVulnerabilities,
		Description: "List persisted vulnerability findings under one target without raw evidence.",
		InputSchema: targetVulnerabilitiesSchema(),
	}, registry.listTargetVulnerabilities)
	server.AddTool(&mcp.Tool{
		Name:        ToolListTargetScreenshots,
		Description: "List screenshot metadata under one target without image bytes.",
		InputSchema: targetScreenshotsSchema(),
	}, registry.listTargetScreenshots)
	server.AddTool(&mcp.Tool{
		Name:        ToolGetScreenshotImage,
		Description: "Return exactly one target-owned screenshot as image/webp content.",
		InputSchema: screenshotImageSchema(),
	}, registry.getScreenshotImage)
	server.AddTool(&mcp.Tool{
		Name:        ToolListScanWorkflows,
		Description: "List current scan workflow catalog entries.",
		InputSchema: listSchema(map[string]any{"filter": map[string]any{"type": "string"}}),
	}, registry.listScanWorkflows)
	server.AddTool(&mcp.Tool{
		Name:        ToolGetScanWorkflow,
		Description: "Get one current scan workflow by canonical resource name.",
		InputSchema: canonicalNameSchema("scanWorkflows"),
	}, registry.getScanWorkflow)
	server.AddTool(&mcp.Tool{
		Name:        ToolGetScanWorkflowProfile,
		Description: "Materialize a complete editable configuration profile for one scan workflow.",
		InputSchema: objectSchema(map[string]any{"scan_workflow": canonicalReferenceProperty("scanWorkflows")}, "scan_workflow"),
	}, registry.getScanWorkflowProfile)
	server.AddTool(&mcp.Tool{
		Name:        ToolListEngines,
		Description: "List installed Engine catalog projections.",
		InputSchema: listSchema(map[string]any{}),
	}, registry.listEngines)
	server.AddTool(&mcp.Tool{
		Name:        ToolGetEngine,
		Description: "Get one installed Engine catalog projection.",
		InputSchema: canonicalNameSchema("engines"),
	}, registry.getEngine)
	server.AddTool(&mcp.Tool{
		Name:        ToolListWordlists,
		Description: "List safe Wordlist metadata without file content or paths.",
		InputSchema: listSchema(map[string]any{
			"filter":   map[string]any{"type": "string"},
			"order_by": map[string]any{"type": "string", "enum": []string{"file_name", "file_name desc", "line_count", "line_count desc", "file_size", "file_size desc", "updated_at", "updated_at desc"}},
		}),
	}, registry.listWordlists)
	server.AddTool(&mcp.Tool{
		Name:        ToolGetWordlist,
		Description: "Get safe metadata for one Wordlist.",
		InputSchema: canonicalNameSchema("wordlists"),
	}, registry.getWordlist)
	server.AddTool(&mcp.Tool{
		Name:        ToolListServerLogEntries,
		Description: "List fixed-source LunaFox Server log entries with bounded follow pagination.",
		InputSchema: logListSchema(),
	}, registry.listServerLogEntries)
	server.AddTool(&mcp.Tool{
		Name:        ToolListAgentLogEntries,
		Description: "List fixed Agent runtime-container log entries with bounded follow pagination.",
		InputSchema: agentLogListSchema(),
	}, registry.listAgentLogEntries)
	server.AddTool(&mcp.Tool{
		Name:        ToolReviewVulnerability,
		Description: "Immediately mark one vulnerability reviewed. This mutates LunaFox data.",
		InputSchema: vulnerabilityDispositionSchema(),
		Annotations: vulnerabilityDispositionAnnotations(false),
	}, registry.reviewVulnerability)
	server.AddTool(&mcp.Tool{
		Name:        ToolUnreviewVulnerability,
		Description: "Immediately mark one vulnerability unreviewed. This mutates LunaFox data.",
		InputSchema: vulnerabilityDispositionSchema(),
		Annotations: vulnerabilityDispositionAnnotations(true),
	}, registry.unreviewVulnerability)
	server.AddTool(&mcp.Tool{
		Name:        ToolBatchReviewVulnerabilities,
		Description: "Atomically mark up to 5000 vulnerabilities reviewed. This mutates LunaFox data.",
		InputSchema: vulnerabilityBatchDispositionSchema(),
		Annotations: vulnerabilityDispositionAnnotations(false),
	}, registry.batchReviewVulnerabilities)
	server.AddTool(&mcp.Tool{
		Name:        ToolBatchUnreviewVulnerabilities,
		Description: "Atomically mark up to 5000 vulnerabilities unreviewed. This mutates LunaFox data.",
		InputSchema: vulnerabilityBatchDispositionSchema(),
		Annotations: vulnerabilityDispositionAnnotations(true),
	}, registry.batchUnreviewVulnerabilities)
	server.AddTool(&mcp.Tool{
		Name:        ToolStartScan,
		Description: "Immediately start one target scan from a complete editable workflow configuration and return a polling operation.",
		InputSchema: startScanSchema(),
		Annotations: startScanAnnotations(),
	}, registry.startScan)
	server.AddTool(&mcp.Tool{
		Name:        ToolGetOperation,
		Description: "Get one retained Scan-backed polling operation by canonical resource name.",
		InputSchema: operationNameSchema(),
	}, registry.getOperation)
}

func canonicalReferenceProperty(collection string) map[string]any {
	return map[string]any{"type": "string", "pattern": "^" + collection + "/[^/]+$"}
}

func canonicalNameSchema(collection string) map[string]any {
	return objectSchema(map[string]any{"name": canonicalReferenceProperty(collection)}, "name")
}

func organizationTargetsSchema() map[string]any {
	properties := withPageProperties(map[string]any{
		"organization": canonicalReferenceProperty("organizations"),
		"type":         map[string]any{"type": "string", "enum": []string{"domain", "ip", "cidr"}},
		"filter":       map[string]any{"type": "string"},
	})
	return objectSchema(properties, "organization")
}

func targetVulnerabilitiesSchema() map[string]any {
	properties := withPageProperties(map[string]any{
		"target":    canonicalReferenceProperty("targets"),
		"url":       map[string]any{"type": "string"},
		"severity":  map[string]any{"type": "string", "enum": []string{"unknown", "info", "low", "medium", "high", "critical"}},
		"source":    map[string]any{"type": "string"},
		"vuln_type": map[string]any{"type": "string"},
		"reviewed":  map[string]any{"type": "boolean"},
		"order_by":  map[string]any{"type": "string", "enum": []string{"created_at", "created_at desc"}},
	})
	return objectSchema(properties, "target")
}

func targetScreenshotsSchema() map[string]any {
	properties := withPageProperties(map[string]any{
		"target":   canonicalReferenceProperty("targets"),
		"filter":   map[string]any{"type": "string"},
		"order_by": map[string]any{"type": "string", "enum": []string{"created_at", "created_at desc", "status_code", "status_code desc"}},
	})
	return objectSchema(properties, "target")
}

func screenshotImageSchema() map[string]any {
	return objectSchema(map[string]any{
		"target":     canonicalReferenceProperty("targets"),
		"screenshot": map[string]any{"type": "string", "pattern": "^targets/[1-9][0-9]*/screenshots/[1-9][0-9]*$"},
	}, "target", "screenshot")
}

func logListSchema() map[string]any {
	return objectSchema(map[string]any{
		"page_size":  map[string]any{"type": "integer", "minimum": 1, "maximum": 500, "default": 200},
		"page_token": map[string]any{"type": "string"},
		"direction":  map[string]any{"type": "string", "enum": []string{"newer", "older"}},
	})
}

func agentLogListSchema() map[string]any {
	properties := logListSchema()["properties"].(map[string]any)
	properties["agent_id"] = map[string]any{"type": "integer", "minimum": 1}
	properties["container"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 128, "pattern": "^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$"}
	return objectSchema(properties, "agent_id", "container")
}

func startScanSchema() map[string]any {
	stepSchema := objectSchema(map[string]any{
		"enabled": map[string]any{"type": "boolean"},
		"engineConfig": map[string]any{
			"type": "object",
			// Workflow-defined section IDs are dynamic. Their contents remain
			// strictly decoded by the canonical Scan create validator at submit.
			"additionalProperties": true,
		},
	}, "enabled")
	configurationSchema := objectSchema(map[string]any{
		"steps": map[string]any{
			"type": "object", "minProperties": 1,
			"additionalProperties": stepSchema,
		},
	}, "steps")
	return objectSchema(map[string]any{
		"target":        canonicalReferenceProperty("targets"),
		"scan_workflow": canonicalReferenceProperty("scanWorkflows"),
		"configuration": configurationSchema,
		"agent":         map[string]any{"type": "string", "pattern": "^agents/[1-9][0-9]*$"},
		"request_id":    map[string]any{"type": "string", "minLength": 1, "maxLength": 128},
	}, "target", "scan_workflow", "configuration")
}

func operationNameSchema() map[string]any {
	return objectSchema(map[string]any{
		"name": map[string]any{"type": "string", "pattern": "^operations/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"},
	}, "name")
}

func vulnerabilityDispositionSchema() map[string]any {
	return objectSchema(map[string]any{
		"name":       map[string]any{"type": "string", "pattern": "^vulnerabilities/[1-9][0-9]*$"},
		"request_id": map[string]any{"type": "string", "minLength": 1, "maxLength": 128},
	}, "name")
}

func vulnerabilityBatchDispositionSchema() map[string]any {
	return objectSchema(map[string]any{
		"names": map[string]any{
			"type": "array", "minItems": 1, "maxItems": maxMCPVulnerabilityDispositionItems,
			"items": map[string]any{"type": "string", "pattern": "^vulnerabilities/[1-9][0-9]*$"},
		},
		"request_id": map[string]any{"type": "string", "minLength": 1, "maxLength": 128},
	}, "names")
}

func vulnerabilityDispositionAnnotations(destructive bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &destructive, IdempotentHint: true}
}

func startScanAnnotations() *mcp.ToolAnnotations {
	destructive := false
	return &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &destructive, IdempotentHint: false}
}

// Keep shared schema constants imported by this registration module close to
// the log-specific override, rather than weakening the general page schema.
var _ = schema.MaxPageSize
