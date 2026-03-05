package workflowschema

import "testing"

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

func TestReadWorkflowNameFromSchema(t *testing.T) {
	workflow, err := readWorkflowNameFromSchema("subdomain_discovery.schema.json")
	if err != nil {
		t.Fatalf("readWorkflowNameFromSchema failed: %v", err)
	}
	if workflow != "subdomain_discovery" {
		t.Fatalf("unexpected workflow name: %q", workflow)
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
