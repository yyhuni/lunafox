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
		if metadata[i].WorkflowID == "subdomain_discovery" {
			matched = &metadata[i]
			break
		}
	}
	if matched == nil {
		t.Fatalf("expected subdomain_discovery in workflow metadata list, got %v", metadata)
	}

	if matched.DisplayName != "Subdomain Discovery" {
		t.Fatalf("unexpected displayName: %q", matched.DisplayName)
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

func TestParseWorkflowMetadataRejectsInvalidWorkflowID(t *testing.T) {
	payload := []byte(`{"x-workflow":"Subdomain-Discovery","title":"Broken"}`)
	_, err := parseWorkflowMetadata(payload, "invalid.schema.json")
	if err == nil {
		t.Fatalf("expected invalid workflow id error")
	}
	if !strings.Contains(err.Error(), "invalid x-workflow") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseWorkflowMetadataRejectsReservedWorkflowID(t *testing.T) {
	payload := []byte(`{"x-workflow":"default","title":"Broken"}`)
	_, err := parseWorkflowMetadata(payload, "reserved.schema.json")
	if err == nil {
		t.Fatalf("expected reserved workflow id error")
	}
	if !strings.Contains(err.Error(), "reserved x-workflow") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseWorkflowMetadataRequiresMetadataDisplayName(t *testing.T) {
	withMetadataName := []byte(`{"x-workflow":"subdomain_discovery","title":"Title","x-metadata":{"name":"Display"}}`)
	meta, err := parseWorkflowMetadata(withMetadataName, "meta.schema.json")
	if err != nil {
		t.Fatalf("parseWorkflowMetadata failed: %v", err)
	}
	if meta.DisplayName != "Display" {
		t.Fatalf("expected x-metadata.name to be used, got %q", meta.DisplayName)
	}

	withTitle := []byte(`{"x-workflow":"subdomain_discovery","title":"Title Only"}`)
	_, err = parseWorkflowMetadata(withTitle, "title.schema.json")
	if err == nil {
		t.Fatalf("expected error when x-metadata.name is missing")
	}
	if !strings.Contains(err.Error(), "missing x-metadata.name") {
		t.Fatalf("unexpected error: %v", err)
	}

	withFallback := []byte(`{"x-workflow":"subdomain_discovery"}`)
	_, err = parseWorkflowMetadata(withFallback, "fallback.schema.json")
	if err == nil {
		t.Fatalf("expected error when x-metadata.name is missing")
	}
	if !strings.Contains(err.Error(), "missing x-metadata.name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseWorkflowMetadataIgnoresTitleField(t *testing.T) {
	payload := []byte(`{"x-workflow":"subdomain_discovery","title":123,"x-metadata":{"name":"Display"}}`)
	meta, err := parseWorkflowMetadata(payload, "title-type.schema.json")
	if err != nil {
		t.Fatalf("expected title to be ignored, got error: %v", err)
	}
	if meta.DisplayName != "Display" {
		t.Fatalf("unexpected displayName: %q", meta.DisplayName)
	}
}

func TestListWorkflowMetadataDisplayNameContract(t *testing.T) {
	metadata, err := ListWorkflowMetadata()
	if err != nil {
		t.Fatalf("ListWorkflowMetadata failed: %v", err)
	}
	if len(metadata) == 0 {
		t.Fatalf("expected at least one workflow metadata entry")
	}
	for _, item := range metadata {
		if strings.TrimSpace(item.DisplayName) == "" {
			t.Fatalf("workflow %q must define non-empty x-metadata.name", item.WorkflowID)
		}
	}
}

func TestValidateWorkflowMetadataListRejectsDuplicateWorkflowID(t *testing.T) {
	items := []WorkflowMetadata{
		{WorkflowID: "subdomain_discovery"},
		{WorkflowID: "subdomain_discovery"},
	}
	err := validateWorkflowMetadataList(items)
	if err == nil {
		t.Fatalf("expected duplicate workflow id error")
	}
	if !strings.Contains(err.Error(), "duplicate schema for workflow") {
		t.Fatalf("unexpected error: %v", err)
	}
}
