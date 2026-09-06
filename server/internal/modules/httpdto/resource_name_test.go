package httpdto

import (
	"os"
	"strings"
	"testing"
)

func TestResourceNameHelpersDoNotHandWriteSharedCanonicalNames(t *testing.T) {
	source, err := os.ReadFile("resource_name.go")
	if err != nil {
		t.Fatalf("read resource_name.go: %v", err)
	}

	for _, forbidden := range []string{
		`fmt.Sprintf("targets/%d", id)`,
		`fmt.Sprintf("scans/%d/tasks/%d", scanID, taskID)`,
	} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("resource_name.go still hand-writes shared resource helper %s", forbidden)
		}
	}
}

func TestParseResourceIDSegmentRejectsInvalidSegments(t *testing.T) {
	tests := []string{"", "0", "-1", "abc", "scans/1", "1/2"}
	for _, value := range tests {
		if _, err := ParseResourceIDSegment(value); err == nil {
			t.Fatalf("ParseResourceIDSegment(%q) succeeded, want error", value)
		}
	}
}

func TestParseResourceIDSegmentReturnsPositiveInteger(t *testing.T) {
	id, err := ParseResourceIDSegment("42")
	if err != nil {
		t.Fatalf("ParseResourceIDSegment failed: %v", err)
	}
	if id != 42 {
		t.Fatalf("expected id 42, got %d", id)
	}
}

func TestParseResourceNameIDRejectsWrongShape(t *testing.T) {
	tests := []string{"", "1", "targets", "targets/0", "scans/1", "targets/1/websites/2"}
	for _, value := range tests {
		if _, err := ParseResourceNameID(value, "targets"); err == nil {
			t.Fatalf("ParseResourceNameID(%q) succeeded, want error", value)
		}
	}
}

func TestParseResourceNameIDReturnsID(t *testing.T) {
	id, err := ParseResourceNameID("targets/42", "targets")
	if err != nil {
		t.Fatalf("ParseResourceNameID failed: %v", err)
	}
	if id != 42 {
		t.Fatalf("expected id 42, got %d", id)
	}
}

func TestParseResourceNameIDsReturnsIDs(t *testing.T) {
	ids, err := ParseResourceNameIDs([]string{"websites/7", "websites/8"}, "websites")
	if err != nil {
		t.Fatalf("ParseResourceNameIDs failed: %v", err)
	}
	if len(ids) != 2 || ids[0] != 7 || ids[1] != 8 {
		t.Fatalf("unexpected ids: %v", ids)
	}
}

func TestParseNestedResourceNameIDReturnsChildID(t *testing.T) {
	id, err := ParseNestedResourceNameID("targets/4/websites/9", "targets", "websites")
	if err != nil {
		t.Fatalf("ParseNestedResourceNameID failed: %v", err)
	}
	if id != 9 {
		t.Fatalf("expected child id 9, got %d", id)
	}
}

func TestParseNestedResourceNameIDsReturnsChildIDs(t *testing.T) {
	ids, err := ParseNestedResourceNameIDs([]string{"targets/4/websites/9", "targets/4/websites/10"}, "targets", "websites")
	if err != nil {
		t.Fatalf("ParseNestedResourceNameIDs failed: %v", err)
	}
	if len(ids) != 2 || ids[0] != 9 || ids[1] != 10 {
		t.Fatalf("unexpected ids: %v", ids)
	}
}

func TestParseScanWorkflowNameRejectsManifestTypedRefs(t *testing.T) {
	tests := []string{
		"scanWorkflows/engine.lunafox.subdomain_discovery",
		"scanWorkflows/runtime.subdomain_discovery",
		"scanWorkflows/schema.engine.subdomain_discovery",
		"scanWorkflows/system://subdomain_discovery",
	}
	for _, value := range tests {
		if _, err := ParseScanWorkflowName(value); err == nil {
			t.Fatalf("ParseScanWorkflowName(%q) succeeded, want error", value)
		}
	}
}

func TestParseScanWorkflowNameUsesCanonicalResourceNameParser(t *testing.T) {
	tests := []string{
		"scanWorkflows/Subdomain_Discovery",
		"scanWorkflows/subdomain-discovery",
		"scanWorkflows/_subdomain_discovery",
	}
	for _, value := range tests {
		if _, err := ParseScanWorkflowName(value); err == nil {
			t.Fatalf("ParseScanWorkflowName(%q) succeeded, want error", value)
		}
	}
}

func TestParseScanWorkflowNameAcceptsScanWorkflowID(t *testing.T) {
	workflowID, err := ParseScanWorkflowName("scanWorkflows/subdomain_discovery")
	if err != nil {
		t.Fatalf("ParseScanWorkflowName failed: %v", err)
	}
	if workflowID != "subdomain_discovery" {
		t.Fatalf("expected subdomain_discovery, got %q", workflowID)
	}
}
