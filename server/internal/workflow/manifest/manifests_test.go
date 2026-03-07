package workflowmanifest

import "testing"

func TestListWorkflowMetadata(t *testing.T) {
	items, err := ListWorkflowMetadata()
	if err != nil {
		t.Fatalf("ListWorkflowMetadata failed: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected embedded manifests")
	}
	var found *WorkflowMetadata
	for i := range items {
		if items[i].WorkflowID == "subdomain_discovery" {
			found = &items[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected subdomain_discovery manifest metadata, got %+v", items)
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
	if manifest.DefaultProfileID != "subdomain_discovery" {
		t.Fatalf("unexpected defaultProfileId: %q", manifest.DefaultProfileID)
	}
	if manifest.ConfigSchemaID != "lunafox://schemas/workflows/subdomain_discovery" {
		t.Fatalf("unexpected configSchemaId: %q", manifest.ConfigSchemaID)
	}
	if len(manifest.Stages) == 0 {
		t.Fatalf("expected stages")
	}
}

func TestDecodeManifestRejectsUnknownField(t *testing.T) {
	_, err := decodeManifest([]byte(`{"manifestVersion":"v1","workflowId":"subdomain_discovery","displayName":"Subdomain Discovery","description":"demo","configSchemaId":"lunafox://schemas/workflows/subdomain_discovery","supportedTargetTypeIds":["domain"],"defaultProfileId":"subdomain_discovery","stages":[],"unknown":true}`), "test")
	if err == nil {
		t.Fatalf("expected unknown field rejection")
	}
}

func TestValidateManifestRejectsDuplicateStage(t *testing.T) {
	knownProfiles := map[string]struct{}{"subdomain_discovery": {}}
	manifest := Manifest{
		ManifestVersion:        "v1",
		WorkflowID:             "subdomain_discovery",
		DisplayName:            "Subdomain Discovery",
		ConfigSchemaID:         "lunafox://schemas/workflows/subdomain_discovery",
		SupportedTargetTypeIDs: []string{"domain"},
		DefaultProfileID:       "subdomain_discovery",
		Stages: []ManifestStage{
			{StageID: "recon", DisplayName: "Recon", Tools: []ManifestTool{{ToolID: "subfinder"}}},
			{StageID: "recon", DisplayName: "Recon 2", Tools: []ManifestTool{{ToolID: "subfinder-2"}}},
		},
	}
	if err := validateManifest(manifest, knownProfiles); err == nil {
		t.Fatalf("expected duplicate stage rejection")
	}
}

func TestValidateManifestRejectsMissingProfile(t *testing.T) {
	manifest := Manifest{
		ManifestVersion:        "v1",
		WorkflowID:             "subdomain_discovery",
		DisplayName:            "Subdomain Discovery",
		ConfigSchemaID:         "lunafox://schemas/workflows/subdomain_discovery",
		SupportedTargetTypeIDs: []string{"domain"},
		DefaultProfileID:       "missing_profile",
		Stages: []ManifestStage{
			{StageID: "recon", DisplayName: "Recon", Tools: []ManifestTool{{ToolID: "subfinder"}}},
		},
	}
	if err := validateManifest(manifest, map[string]struct{}{}); err == nil {
		t.Fatalf("expected missing profile rejection")
	}
}
