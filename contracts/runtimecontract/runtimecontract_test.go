package runtimecontract

import "testing"

func TestNormalizeVersion(t *testing.T) {
	cases := map[string]string{
		"v1.2.3":  "v1.2.3",
		"V2.0.0":  "V2.0.0",
		" 1.0.0 ": "1.0.0",
		"":        "",
	}
	for input, expected := range cases {
		if got := NormalizeVersion(input); got != expected {
			t.Fatalf("NormalizeVersion(%q)=%q, want %q", input, got, expected)
		}
	}
}

func TestVersionValidators(t *testing.T) {
	if !IsValidAPIVersion("v1") {
		t.Fatalf("expected apiVersion valid")
	}
	if IsValidAPIVersion("1") {
		t.Fatalf("expected apiVersion invalid")
	}

	if !IsValidSchemaVersion("1.0.0") {
		t.Fatalf("expected schemaVersion valid")
	}
	if IsValidSchemaVersion("v1.0.0-beta+1") {
		t.Fatalf("expected schemaVersion with leading v invalid")
	}
	if IsValidSchemaVersion("1.0") {
		t.Fatalf("expected schemaVersion invalid")
	}
}

func TestVersionFieldMessageHelpers(t *testing.T) {
	if got := APIVersionFieldMessage("apiVersion"); got != APIVersionFormatMessage {
		t.Fatalf("unexpected apiVersion message: %s", got)
	}
	if got := APIVersionFieldMessage("workflow_api_version"); got != "workflow_api_version must match v<major>" {
		t.Fatalf("unexpected workflow api message: %s", got)
	}
	if got := SchemaVersionFieldMessage("schemaVersion"); got != SchemaVersionFormatMessage {
		t.Fatalf("unexpected schemaVersion message: %s", got)
	}
	if got := SchemaVersionFieldMessage("workflow_schema_version"); got != "workflow_schema_version must match MAJOR.MINOR.PATCH(+suffix)" {
		t.Fatalf("unexpected workflow schema message: %s", got)
	}
}

func TestBuildTaskWorkspaceDirAndConfigPath(t *testing.T) {
	workspace := BuildTaskWorkspaceDir(7, 11)
	if workspace != "/opt/lunafox/results/scan_7/task_11" {
		t.Fatalf("unexpected workspace path: %s", workspace)
	}
	configPath := BuildTaskConfigPath(workspace)
	if configPath != "/opt/lunafox/results/scan_7/task_11/task_config.yaml" {
		t.Fatalf("unexpected config path: %s", configPath)
	}
	if DefaultRuntimeSocketPath() != "/run/lunafox/worker-runtime.sock" {
		t.Fatalf("unexpected runtime socket path: %s", DefaultRuntimeSocketPath())
	}
}
