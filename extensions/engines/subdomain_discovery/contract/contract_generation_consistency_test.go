package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type scanWorkflowIdentity struct {
	ScanWorkflowID string `json:"scanWorkflowId"`
	Stages         []struct {
		StageID string `json:"stageId"`
		Steps   []struct {
			StepID   string `json:"stepId"`
			EngineID string `json:"engineId"`
		} `json:"steps"`
	} `json:"stages"`
}

type engineManifestOutputIdentity struct {
	EngineID       string           `json:"engineId"`
	DisplayNameKey *json.RawMessage `json:"displayNameKey"`
	DescriptionKey *json.RawMessage `json:"descriptionKey"`
	I18n           *json.RawMessage `json:"i18n"`
	Execution      any              `json:"execution"`
	Outputs        any              `json:"outputs"`
	Capabilities   any              `json:"capabilities"`
	Artifacts      *json.RawMessage `json:"artifacts"`
}

func TestContractDefinitionMatchesGeneratedArtifactsAndDocs(t *testing.T) {
	engineLocalID := "subdomain_discovery"

	root := repoRootForTest(t)
	assertFileDoesNotExist(t, filepath.Join(root, "extensions", "engines", engineLocalID, "artifacts", "schemas", fmt.Sprintf("%s.schema.json", engineLocalID)))

	scanWorkflow := loadScanWorkflow(t, filepath.Join(root, "extensions", "workflows", "default.scan-workflow.json"))
	require.Equal(t, "default", scanWorkflow.ScanWorkflowID)
	require.NotEmpty(t, scanWorkflow.Stages)
	require.NotEmpty(t, scanWorkflow.Stages[0].Steps)
	require.Equal(t, "engine.lunafox.subdomain_discovery", scanWorkflow.Stages[0].Steps[0].EngineID)

	docsPath := filepath.Join(root, "docs", "config-reference", engineLocalID+".md")
	docsBytes, err := os.ReadFile(docsPath)
	require.NoError(t, err)
	docs := string(docsBytes)
	require.Contains(t, docs, fmt.Sprintf("<!-- Source: extensions/engines/%s/engine.json -->", engineLocalID))
	require.Contains(t, docs, fmt.Sprintf("- Engine: `%s`", engineLocalID))
	require.NotContains(t, docs, "API Version")
	require.NotContains(t, docs, "Schema Version")
}

func TestGeneratedArtifactsDoNotCreateServerMirrors(t *testing.T) {
	engineLocalID := "subdomain_discovery"
	root := repoRootForTest(t)

	assertFileDoesNotExist(t, filepath.Join(root, "server", "internal", "workflow", "schema", fmt.Sprintf("%s.schema.json", engineLocalID)))
	assertFileDoesNotExist(t, filepath.Join(root, "server", "internal", "runtime", "manifest", fmt.Sprintf("%s.runtime.json", engineLocalID)))
	assertFileDoesNotExist(t, filepath.Join(root, "server", "internal", "workflow", "profile", "profiles", fmt.Sprintf("%s.yaml", engineLocalID)))
}

func TestEngineManifestEmbedsExecutionMetadata(t *testing.T) {
	engineLocalID := "subdomain_discovery"
	root := repoRootForTest(t)
	manifest := loadEngineManifestIdentity(t, filepath.Join(root, "extensions", "engines", engineLocalID, "engine.json"))

	require.Equal(t, "engine.lunafox.subdomain_discovery", manifest.EngineID)
	require.Nil(t, manifest.DisplayNameKey)
	require.Nil(t, manifest.DescriptionKey)
	require.Nil(t, manifest.I18n)
	require.NotNil(t, manifest.Execution)
	require.Nil(t, manifest.Outputs)
	require.Nil(t, manifest.Capabilities)
	require.Nil(t, manifest.Artifacts)
}

func loadScanWorkflow(t *testing.T, path string) scanWorkflowIdentity {
	t.Helper()
	payload, err := os.ReadFile(path)
	require.NoErrorf(t, err, "scan workflow file not found: %s", path)
	var workflow scanWorkflowIdentity
	require.NoError(t, json.Unmarshal(payload, &workflow))
	return workflow
}

func assertFileDoesNotExist(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	require.Truef(t, os.IsNotExist(err), "file should not exist: %s", path)
}

func loadEngineManifestIdentity(t *testing.T, path string) engineManifestOutputIdentity {
	t.Helper()
	payload, err := os.ReadFile(path)
	require.NoErrorf(t, err, "engine manifest not found: %s", path)
	var manifest engineManifestOutputIdentity
	require.NoError(t, json.Unmarshal(payload, &manifest))
	return manifest
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "repository root not found from %s", dir)
		dir = parent
	}
}
