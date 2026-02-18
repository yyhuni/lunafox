package domain

import "testing"

func TestParseWorkflowName(t *testing.T) {
	workflow, ok := ParseWorkflowName(" subdomain_discovery ")
	if !ok || workflow != WorkflowSubdomainDiscovery {
		t.Fatalf("expected subdomain_discovery parse success, got workflow=%q ok=%v", workflow, ok)
	}

	workflow, ok = ParseWorkflowName("URL_FETCH")
	if !ok || workflow != WorkflowURLFetch {
		t.Fatalf("expected url_fetch parse success, got workflow=%q ok=%v", workflow, ok)
	}

	_, ok = ParseWorkflowName("unknown_workflow")
	if ok {
		t.Fatalf("unknown workflow should fail parse")
	}
}
