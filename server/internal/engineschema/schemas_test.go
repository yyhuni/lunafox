package engineschema

import (
	"strings"
	"testing"
)

func TestListEnginesUsesSchemaMetadata(t *testing.T) {
	engines, err := ListEngines()
	if err != nil {
		t.Fatalf("ListEngines failed: %v", err)
	}
	found := false
	for _, engine := range engines {
		if engine == "subdomain_discovery" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected subdomain_discovery in engine list, got %v", engines)
	}
}

func TestReadEngineNameFromSchema(t *testing.T) {
	engine, err := readEngineNameFromSchema("subdomain_discovery-v1-1.0.0.schema.json")
	if err != nil {
		t.Fatalf("readEngineNameFromSchema failed: %v", err)
	}
	if engine != "subdomain_discovery" {
		t.Fatalf("unexpected engine name: %q", engine)
	}
}

func TestExtractSchemaVersionRejectsInvalidFormatsWithStableMessage(t *testing.T) {
	_, _, err := extractSchemaVersion(map[string]any{
		"apiVersion":    "version1",
		"schemaVersion": "1.0.0",
	})
	if err == nil || !strings.Contains(err.Error(), "apiVersion must match v<major>") {
		t.Fatalf("unexpected apiVersion error: %v", err)
	}

	_, _, err = extractSchemaVersion(map[string]any{
		"apiVersion":    "v1",
		"schemaVersion": "1.0",
	})
	if err == nil || !strings.Contains(err.Error(), "schemaVersion must match MAJOR.MINOR.PATCH(+suffix)") {
		t.Fatalf("unexpected schemaVersion error: %v", err)
	}
}
