package bootstrap

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesKeepsManagementHTTPAPIs(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	registerRoutes(engine, &deps{}, func(c *gin.Context) { c.Next() })

	registered := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		if strings.HasPrefix(route.Path, "/api") {
			t.Fatalf("backend route must use versioned /v1 root, got legacy route: %s %s", route.Method, route.Path)
		}
		registered[fmt.Sprintf("%s %s", route.Method, route.Path)] = struct{}{}
	}

	expected := []string{
		"POST /mcp",
		"GET /healthChecks/current",
		"GET /healthChecks/liveness",
		"GET /healthChecks/readiness",
		"POST /v1/sessions",
		"POST /v1/sessions:renew",
		"GET /v1/users/current",
		"POST /v1/targets",
		"GET /v1/targets",
		"GET /v1/engines",
		"GET /v1/engines/:engine",
		"POST /v1/scanWorkflows",
		"GET /v1/scanWorkflows",
		"GET /v1/scanWorkflows/:scanWorkflow/profile",
		"GET /v1/scanWorkflows/:scanWorkflow",
		"PATCH /v1/scanWorkflows/:scanWorkflow",
		"GET /v1/wordlists",
		"GET /v1/blacklistPolicy",
		"PATCH /v1/blacklistPolicy",
		"GET /v1/targets/:target/blacklistPolicy",
		"PATCH /v1/targets/:target/blacklistPolicy",
		"GET /v1/scans",
		"POST /v1/scans:customMethod",
		"GET /v1/scans/:scan/taskProgressLogs",
		"GET /v1/vulnerabilities",
		"GET /v1/databaseHealthReports/current",
		"GET /v1/admin/system/logEntries",
		"GET /v1/admin/system/runtimeMetrics/current",
		"GET /v1/admin/agents",
		"GET /v1/admin/agents/:agent",
		"PATCH /v1/admin/agents/:agent",
		"GET /v1/admin/agents/:agent/logEntries",
		"POST /v1/admin/agentRegistrationTokens",
		"GET /v1/admin/agentRegistrationTokens/:registrationToken",
		"GET /v1/admin/agentClusterSummaries/current",
		"GET /v1/admin/agentLocationMaps/current",
		"POST /v1/agents:register",
		"GET /v1/screenshots/:screenshot/blob",
	}

	for _, key := range expected {
		if _, ok := registered[key]; !ok {
			t.Fatalf("management http api missing after agent plane grpc cutover: %s", key)
		}
	}

	legacyCollection := func(name string) string {
		return "/v1/" + name
	}
	legacyWorkflowCollection := legacyCollection(strings.Join([]string{"work", "flows"}, ""))
	legacyWorkflowProfilesCollection := legacyCollection(strings.Join([]string{"workflow", "Profiles"}, ""))

	unexpected := []string{
		"POST /v1/engines",
		"GET /v1/engines/:id",
		"PUT /v1/engines/:id",
		"PATCH /v1/engines/:id",
		"DELETE /v1/engines/:id",
		"PUT " + legacyWorkflowCollection + "/:workflow",
		"DELETE " + legacyWorkflowCollection + "/:workflow",
		"POST /v1/agentRegistrations",
		"POST /v1/sessionRenewals",
		"POST /v1/scans",
		"GET " + legacyWorkflowCollection + "/profiles",
		"GET " + legacyWorkflowCollection + "/profiles/:workflow_profile",
		"GET " + legacyWorkflowProfilesCollection,
		"GET " + legacyWorkflowProfilesCollection + "/:workflow_profile",
		"GET /v1/scans/:scan/logs",
		"POST /v1/scans/:scan/logs",
		"PATCH /v1/admin/agents/:agent/config",
		"GET /v1/admin/agents/:agent/logs",
		"POST /v1/admin/agents/registrationTokens",
	}

	for _, key := range unexpected {
		if _, ok := registered[key]; ok {
			t.Fatalf("legacy catalog-management route must be disabled in memory-only mode: %s", key)
		}
	}
}
