package workflowmanifest

import "testing"

func TestListManifests(t *testing.T) {
	items, err := ListManifests()
	if err != nil {
		t.Fatalf("ListManifests failed: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected embedded manifests")
	}
	var found *Manifest
	for i := range items {
		if items[i].WorkflowID == "subdomain_discovery" {
			found = &items[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected subdomain_discovery manifest, got %+v", items)
	}
	if found.DisplayName != "Subdomain Discovery" {
		t.Fatalf("unexpected displayName: %q", found.DisplayName)
	}
	if found.Description == "" {
		t.Fatalf("expected non-empty description")
	}
}

func TestGetManifest(t *testing.T) {
	manifest, err := GetManifest("subdomain_discovery")
	if err != nil {
		t.Fatalf("GetManifest failed: %v", err)
	}
	if manifest.WorkflowID != "subdomain_discovery" {
		t.Fatalf("unexpected workflowId: %q", manifest.WorkflowID)
	}
	if manifest.Executor.Type != "builtin" || manifest.Executor.Ref != "subdomain_discovery" {
		t.Fatalf("unexpected executor binding: %+v", manifest.Executor)
	}
	if len(manifest.Stages) == 0 {
		t.Fatalf("expected stages")
	}
}

func TestDecodeManifestRejectsUnknownField(t *testing.T) {
	_, err := decodeManifest([]byte(`{"manifestVersion":"v1","workflowId":"subdomain_discovery","displayName":"Subdomain Discovery","description":"demo","executor":{"type":"builtin","ref":"subdomain_discovery"},"stages":[],"unknown":true}`), "test")
	if err == nil {
		t.Fatalf("expected unknown field rejection")
	}
}

func TestValidateManifestRejectsDuplicateStage(t *testing.T) {
	manifest := Manifest{
		ManifestVersion: "v1",
		WorkflowID:      "subdomain_discovery",
		DisplayName:     "Subdomain Discovery",
		Executor:        ManifestExecutor{Type: "builtin", Ref: "subdomain_discovery"},
		Stages: []ManifestStage{
			{StageID: "recon", DisplayName: "Recon", Tools: []ManifestTool{{ToolID: "subfinder"}}},
			{StageID: "recon", DisplayName: "Recon 2", Tools: []ManifestTool{{ToolID: "subfinder-2"}}},
		},
	}
	if err := validateManifest(manifest); err == nil {
		t.Fatalf("expected duplicate stage rejection")
	}
}

func TestValidateManifestRejectsMissingExecutorBinding(t *testing.T) {
	manifest := Manifest{
		ManifestVersion: "v1",
		WorkflowID:      "subdomain_discovery",
		DisplayName:     "Subdomain Discovery",
		Stages: []ManifestStage{{
			StageID: "recon", DisplayName: "Recon", Tools: []ManifestTool{{ToolID: "subfinder"}},
		}},
	}
	if err := validateManifest(manifest); err == nil {
		t.Fatalf("expected missing executor rejection")
	}
}
