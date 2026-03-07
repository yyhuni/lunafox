package workflowschema

import (
	"strings"
	"testing"
)

func TestListWorkflowsUsesSchemaFilenames(t *testing.T) {
	workflows, err := ListWorkflows()
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}
	found := false
	for _, workflowID := range workflows {
		if workflowID == "subdomain_discovery" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected subdomain_discovery in workflow list, got %v", workflows)
	}
}

func TestValidateYAMLWithoutWorkflowMetadata(t *testing.T) {
	config := []byte(`
subdomain_discovery:
  recon:
    enabled: true
    tools:
      subfinder:
        enabled: true
        timeout-runtime: 3600
        threads-cli: 10
  bruteforce:
    enabled: false
    tools:
      subdomain-bruteforce:
        enabled: false
  permutation:
    enabled: false
    tools:
      subdomain-permutation-resolve:
        enabled: false
  resolve:
    enabled: false
    tools:
      subdomain-resolve:
        enabled: false
`)
	if err := ValidateYAML("subdomain_discovery", config); err != nil {
		t.Fatalf("ValidateYAML failed: %v", err)
	}
}

func TestValidateYAMLReturnsExplicitErrorWhenWorkflowNodeIsNotMapping(t *testing.T) {
	config := []byte(`
subdomain_discovery: not-an-object
`)
	err := ValidateYAML("subdomain_discovery", config)
	if err == nil {
		t.Fatalf("expected error when workflow node is not mapping")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, `workflow "subdomain_discovery" config must be a mapping`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowIDFromSchemaFilenameRejectsInvalidName(t *testing.T) {
	_, err := workflowIDFromSchemaFilename("Subdomain-Discovery.schema.json")
	if err == nil {
		t.Fatalf("expected invalid filename rejection")
	}
}

func TestWorkflowIDFromSchemaFilenameRejectsReservedWorkflowID(t *testing.T) {
	_, err := workflowIDFromSchemaFilename("default.schema.json")
	if err == nil {
		t.Fatalf("expected reserved workflow id rejection")
	}
}
