package config

import "testing"

func TestParseWorkflowCapabilities(t *testing.T) {
	items, err := ParseWorkflowCapabilities("subdomain_discovery@v1/1.0.0,asset_discovery@v2/2.1.0")
	if err != nil {
		t.Fatalf("parse workflow capabilities failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 capabilities, got %d", len(items))
	}
	if items[0].Workflow != "subdomain_discovery" || items[0].APIVersion != "v1" || items[0].SchemaVersion != "1.0.0" {
		t.Fatalf("unexpected first capability: %+v", items[0])
	}
	if items[1].Workflow != "asset_discovery" || items[1].APIVersion != "v2" || items[1].SchemaVersion != "2.1.0" {
		t.Fatalf("unexpected second capability: %+v", items[1])
	}
}

func TestParseWorkflowCapabilitiesInvalidFormat(t *testing.T) {
	_, err := ParseWorkflowCapabilities("subdomain_discovery:v1:1.0.0")
	if err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestFormatWorkflowCapabilities(t *testing.T) {
	items, err := ParseWorkflowCapabilities("subdomain_discovery@v1/1.0.0")
	if err != nil {
		t.Fatalf("parse workflow capabilities failed: %v", err)
	}

	formatted := FormatWorkflowCapabilities(items)
	if formatted != "subdomain_discovery@v1/1.0.0" {
		t.Fatalf("unexpected formatted value: %s", formatted)
	}
}
