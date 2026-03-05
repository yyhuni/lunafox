package workflowschema

import (
	"strings"
	"testing"
)

func TestListWorkflowsUsesSchemaMetadata(t *testing.T) {
	workflows, err := ListWorkflows()
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}
	found := false
	for _, workflow := range workflows {
		if workflow == "subdomain_discovery" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected subdomain_discovery in workflow list, got %v", workflows)
	}
}

func TestListWorkflowMetadataUsesSchemaFields(t *testing.T) {
	metadata, err := ListWorkflowMetadata()
	if err != nil {
		t.Fatalf("ListWorkflowMetadata failed: %v", err)
	}

	var matched *WorkflowMetadata
	for i := range metadata {
		if metadata[i].Name == "subdomain_discovery" {
			matched = &metadata[i]
			break
		}
	}
	if matched == nil {
		t.Fatalf("expected subdomain_discovery in workflow metadata list, got %v", metadata)
	}

	if matched.Title != "Subdomain Discovery" {
		t.Fatalf("unexpected title: %q", matched.Title)
	}
	if matched.Description == "" {
		t.Fatalf("expected non-empty description, got empty")
	}
}

func TestValidateYAMLWithoutVersionFields(t *testing.T) {
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

func TestParseWorkflowMetadataReturnsErrorWhenWorkflowMissing(t *testing.T) {
	payload := []byte(`{"title":"No Workflow"}`)
	_, err := parseWorkflowMetadata(payload, "missing.schema.json")
	if err == nil {
		t.Fatalf("expected error when x-workflow is missing")
	}
	if !strings.Contains(err.Error(), "missing x-workflow") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseWorkflowMetadataReturnsErrorWhenJSONInvalid(t *testing.T) {
	payload := []byte(`{invalid`)
	_, err := parseWorkflowMetadata(payload, "broken.schema.json")
	if err == nil {
		t.Fatalf("expected error for invalid metadata json")
	}
	if !strings.Contains(err.Error(), "decode metadata") {
		t.Fatalf("unexpected error: %v", err)
	}
}
