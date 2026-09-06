package workflowmanifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadWorkflowDefinitionsUsesConfiguredRoot(t *testing.T) {
	originalRoot := WorkflowDefinitionsRoot()
	t.Cleanup(func() {
		if err := ConfigureWorkflowDefinitionsRoot(originalRoot); err != nil {
			t.Fatalf("restore workflow definition root: %v", err)
		}
	})
	root := t.TempDir()
	payload := []byte(`{"scanWorkflowId":"default","displayName":"Configured","description":"Loaded from the configured root.","stages":[{"stageId":"discovery","steps":[{"stepId":"discover","engineId":"engine.lunafox.subdomain_discovery","profileDefaultEnabled":true}]}]}`)
	if err := os.WriteFile(filepath.Join(root, "default.scan-workflow.json"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ConfigureWorkflowDefinitionsRoot(root); err != nil {
		t.Fatal(err)
	}

	items, err := loadWorkflowDefinitions()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ScanWorkflowID != "default" {
		t.Fatalf("loadWorkflowDefinitions() = %+v, want default definition only", items)
	}
}

func TestConfigureWorkflowDefinitionsRootRejectsEmptyRoot(t *testing.T) {
	if err := ConfigureWorkflowDefinitionsRoot(" \t "); err == nil {
		t.Fatal("expected empty workflow definition root to be rejected")
	}
}

func TestLoadWorkflowDefinitionsRejectsMissingConfiguredRoot(t *testing.T) {
	originalRoot := WorkflowDefinitionsRoot()
	t.Cleanup(func() {
		if err := ConfigureWorkflowDefinitionsRoot(originalRoot); err != nil {
			t.Fatalf("restore workflow definition root: %v", err)
		}
	})
	missingRoot := filepath.Join(t.TempDir(), "missing")
	if err := ConfigureWorkflowDefinitionsRoot(missingRoot); err != nil {
		t.Fatal(err)
	}

	if _, err := loadWorkflowDefinitions(); err == nil || !strings.Contains(err.Error(), missingRoot) {
		t.Fatalf("expected missing configured root error, got %v", err)
	}
}

func TestDecodeManifestRejectsLegacyStepReferenceFields(t *testing.T) {
	payload := []byte(`{"scanWorkflowId":"subdomain_discovery","displayName":"Subdomain Discovery","description":"Discover subdomains for target domains.","stages":[{"stageId":"discovery","steps":[{"stepId":"subdomain_discovery","engineId":"engine.lunafox.subdomain_discovery","workflowRef":"workflow.subdomain_discovery"}]}]}`)

	_, err := decodeManifest(payload, "test.scan-workflow.json")

	if err == nil || !strings.Contains(err.Error(), "workflowRef") {
		t.Fatalf("expected workflowRef rejection, got %v", err)
	}
}

func TestDecodeManifestRejectsLegacyUsesField(t *testing.T) {
	payload := []byte(`{"scanWorkflowId":"subdomain_discovery","displayName":"Subdomain Discovery","description":"Discover subdomains for target domains.","stages":[{"stageId":"discovery","steps":[{"stepId":"subdomain_discovery","uses":"engine.lunafox.subdomain_discovery"}]}]}`)

	_, err := decodeManifest(payload, "test.scan-workflow.json")

	if err == nil || !strings.Contains(err.Error(), "uses") {
		t.Fatalf("expected uses rejection, got %v", err)
	}
}

func TestDecodeManifestRejectsOnFailureField(t *testing.T) {
	payload := []byte(`{"scanWorkflowId":"subdomain_discovery","displayName":"Subdomain Discovery","description":"Discover subdomains for target domains.","stages":[{"stageId":"discovery","steps":[{"stepId":"subdomain_discovery","engineId":"engine.lunafox.subdomain_discovery","onFailure":"failWorkflow"}]}]}`)

	_, err := decodeManifest(payload, "test.scan-workflow.json")

	if err == nil || !strings.Contains(err.Error(), "onFailure") {
		t.Fatalf("expected onFailure rejection, got %v", err)
	}
}

func TestDecodeManifestReadsDescription(t *testing.T) {
	payload := []byte(`{"scanWorkflowId":"subdomain_discovery","displayName":"Subdomain Discovery","description":"Discover subdomains for target domains.","stages":[{"stageId":"discovery","steps":[{"stepId":"subdomain_discovery","engineId":"engine.lunafox.subdomain_discovery","profileDefaultEnabled":true}]}]}`)

	manifest, err := decodeManifest(payload, "test.scan-workflow.json")

	if err != nil {
		t.Fatalf("decodeManifest failed: %v", err)
	}
	if manifest.Description != "Discover subdomains for target domains." {
		t.Fatalf("unexpected description: %q", manifest.Description)
	}
}
