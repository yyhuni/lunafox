package bootstrap

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestPlaneNamingGuardRejectsRetiredBoundaryTerms(t *testing.T) {
	repoRoot := repoRootFromBootstrapPackage(t)

	files := []string{
		"agent",
		"worker",
		"server",
		"contracts",
		"proto",
		"docker",
		"extensions/engines",
		"scripts",
		"tools",
		"docs/harness",
		"openspec/specs",
		"openspec/changes",
	}
	bannedContent := []string{
		"AgentRuntimeService",
		"AgentRuntimeDataPlaneService",
		"WorkerRuntimeDataPlaneService",
		"AgentControlPlaneService",
		"AgentDataPlaneService",
		"WorkerExecutionDataPlaneService",
		"AgentRuntimeGateway",
		"RunAgentWorkerRuntimeDataPlaneServer",
		"LocalAgentRuntimeDataPlaneClient",
		"LocalAgentExecutionClient",
		"NewLocalAgentExecutionClient",
		"ServeWorkerExecutionDataPlane",
		"AgentRuntimeLifecycleService",
		"TaskRuntimeService",
		"GetProviderConfig",
		"ProviderConfigGet",
		"provider_config.get",
		"CommandProviderConfigGet",
		"WorkerProviderConfig",
		"ProviderConfigDataPlane",
		"RuntimeConfigReader",
		"RuntimeExecutionClient",
		"agentplane",
		"workerplane",
		"RUNTIME_GRPC_URL",
		"RuntimeGRPCURL",
		"--runtime-grpc-url",
		"LUNAFOX_RUNTIME_SOCKET",
		"LUNAFOX_RUNTIME_VOLUME",
		"lunafox-runtime-agent-api-key",
		"lunafox-runtime-worker-task-session-token",
		"runtime-smoke-agent",
		"runtime-smoke-server",
		"lunafox.runtime.v1",
		"runtimehost",
		"Post" + "ResultBatch",
		"ResultBatch" + "Post",
		"CommandResultBatch" + "Post",
		"result_batch." + "post",
		"data" + "Type",
		"Data" + "Type",
		"data_" + "type",
		"Batch" + "Upsert" + "Assets",
		"AssetBatch" + "Upsert" + "DataPlanes",
	}
	bannedPathFragments := []string{
		"proto/lunafox/runtime/v1",
		"contracts/gen/lunafox/runtime/v1",
		"server/internal/grpc/runtime/",
		"agent/internal/grpc/runtime/",
		"worker/internal/grpc/runtime/",
		"agent/internal/runtime/",
		"worker/internal/server/runtime_client.go",
		"agent/internal/workerplane",
		"worker/internal/agentplane",
	}

	var hits []string
	for _, rel := range files {
		root := filepath.Join(repoRoot, rel)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			relPath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			relPath = filepath.ToSlash(relPath)
			if relPath == "server/internal/bootstrap/plane_naming_guard_test.go" {
				return nil
			}
			if strings.HasPrefix(relPath, "openspec/changes/archive/") ||
				strings.HasPrefix(relPath, "openspec/archive/") ||
				strings.HasPrefix(relPath, "openspec/changes/refactor-worker-execution-data-plane-naming/") {
				return nil
			}
			if !isNamingGuardFile(relPath) {
				return nil
			}
			for _, fragment := range bannedPathFragments {
				if strings.Contains(relPath, fragment) {
					hits = append(hits, relPath+" contains retired path fragment "+fragment)
				}
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			content := string(data)
			for _, term := range bannedContent {
				if strings.HasPrefix(relPath, "openspec/changes/") && !isWorkerExecutionDataPlaneNamingTerm(term) {
					continue
				}
				if containsRetiredTerm(content, term) {
					if isAllowedLegacyDenyReference(relPath, content, term) {
						continue
					}
					hits = append(hits, relPath+" contains retired term "+term)
				}
			}
			return nil
		})
		if err != nil {
			if os.IsNotExist(err) && isPublicProjectionRoot(repoRoot) && isOmittedFromPublicProjection(rel) {
				continue
			}
			if rel == "worker" && os.IsNotExist(err) {
				continue
			}
			t.Fatalf("walk %s: %v", rel, err)
		}
	}

	if len(hits) > 0 {
		t.Fatalf("retired plane naming found:\n%s", strings.Join(hits, "\n"))
	}
}

func isPublicProjectionRoot(repoRoot string) bool {
	_, err := os.Stat(filepath.Join(repoRoot, "PUBLIC_PROVENANCE.json"))
	return err == nil
}

func isOmittedFromPublicProjection(relativeRoot string) bool {
	switch relativeRoot {
	case "agent", "tools", "docs/harness", "openspec/specs", "openspec/changes":
		return true
	default:
		return false
	}
}

func repoRootFromBootstrapPackage(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(cwd, "..", "..", ".."))
}

func isWorkerExecutionDataPlaneNamingTerm(term string) bool {
	switch term {
	case "AgentControlPlaneService",
		"AgentDataPlaneService",
		"WorkerExecutionDataPlaneService",
		"LocalAgentExecutionClient",
		"NewLocalAgentExecutionClient",
		"ServeWorkerExecutionDataPlane",
		"GetProviderConfig",
		"ProviderConfigGet",
		"provider_config.get",
		"CommandProviderConfigGet",
		"WorkerProviderConfig",
		"ProviderConfigDataPlane",
		"RuntimeConfigReader",
		"RuntimeExecutionClient",
		"agentplane",
		"workerplane":
		return true
	default:
		return false
	}
}

func containsRetiredTerm(content string, term string) bool {
	switch term {
	case "GetProviderConfig", "ProviderConfigGet", "ProviderConfigDataPlane":
		return regexp.MustCompile(`(^|[^A-Za-z])`+regexp.QuoteMeta(term)).FindStringIndex(content) != nil
	case "provider_config.get":
		return regexp.MustCompile(`(^|[^a-z_])`+regexp.QuoteMeta(term)).FindStringIndex(content) != nil
	default:
		return strings.Contains(content, term)
	}
}

func isNamingGuardFile(path string) bool {
	switch filepath.Ext(path) {
	case ".go", ".proto", ".json", ".md", ".sh", ".yml", ".yaml", ".conf":
		return true
	default:
		return false
	}
}

func isAllowedLegacyDenyReference(path string, content string, term string) bool {
	if term != "provider_config.get" {
		return false
	}
	switch path {
	case "contracts/runtimeengine/hostcommand/hostcommand_naming_test.go",
		"server/internal/scanworkflow/manifest/validate.go",
		"server/internal/scanworkflow/manifest/validate_test.go",
		"tools/engine-manifest-artifact-gen/manifest.go",
		"tools/engine-manifest-artifact-gen/runtime_manifest_test.go":
		return strings.Contains(content, "legacy") ||
			strings.Contains(content, "unsupported") ||
			strings.Contains(content, "removed") ||
			strings.Contains(content, "rejection")
	default:
		return false
	}
}
