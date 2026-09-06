package workflowmanifest

import "testing"

func TestListManifests(t *testing.T) {
	items, err := ListManifests()
	if err != nil {
		t.Fatalf("ListManifests failed: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected workflow definitions")
	}
	var found *Manifest
	for i := range items {
		if items[i].ScanWorkflowID == "default" {
			found = &items[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected default workflow, got %+v", items)
	}
	if found.DisplayName != "Default Scan" {
		t.Fatalf("unexpected displayName: %q", found.DisplayName)
	}
	if found.Description == "" {
		t.Fatal("expected workflow description")
	}
	if len(found.Stages) != 7 {
		t.Fatalf("expected default workflow to have seven stages, got %+v", found.Stages)
	}
	if len(found.Stages[0].Steps) != 1 {
		t.Fatalf("expected single step, got %+v", found.Stages[0].Steps)
	}
	if found.Stages[0].Steps[0].EngineID != "engine.lunafox.subdomain_discovery" {
		t.Fatalf("unexpected step engine ref: %+v", found.Stages[0].Steps[0])
	}
}

func TestGetManifest(t *testing.T) {
	manifest, err := GetManifest("default")
	if err != nil {
		t.Fatalf("GetManifest failed: %v", err)
	}
	if manifest.ScanWorkflowID != "default" {
		t.Fatalf("unexpected scanWorkflowId: %q", manifest.ScanWorkflowID)
	}
}

func TestDefaultWorkflowStagesSubdomainDiscoveryBeforePortScan(t *testing.T) {
	manifest, err := GetManifest("default")
	if err != nil {
		t.Fatalf("GetManifest(default) failed: %v", err)
	}
	if manifest.ScanWorkflowID != "default" {
		t.Fatalf("unexpected scanWorkflowId: %q", manifest.ScanWorkflowID)
	}
	if len(manifest.Stages) != 7 {
		t.Fatalf("expected seven staged workflow stages, got %+v", manifest.Stages)
	}
	if manifest.Stages[0].StageID != "discovery" || len(manifest.Stages[0].Steps) != 1 {
		t.Fatalf("expected discovery stage first, got %+v", manifest.Stages[0])
	}
	if manifest.Stages[0].Steps[0].EngineID != "engine.lunafox.subdomain_discovery" {
		t.Fatalf("expected subdomain discovery first, got %+v", manifest.Stages[0].Steps[0])
	}
	if manifest.Stages[1].StageID != "ports" || len(manifest.Stages[1].Steps) != 1 {
		t.Fatalf("expected ports stage second, got %+v", manifest.Stages[1])
	}
	if manifest.Stages[1].Steps[0].EngineID != "engine.lunafox.port_scan" {
		t.Fatalf("expected port_scan second, got %+v", manifest.Stages[1].Steps[0])
	}
	if manifest.Stages[2].StageID != "websites" || len(manifest.Stages[2].Steps) != 1 {
		t.Fatalf("expected websites stage third, got %+v", manifest.Stages[2])
	}
	if manifest.Stages[2].Steps[0].EngineID != "engine.lunafox.website_discovery" {
		t.Fatalf("expected website discovery third, got %+v", manifest.Stages[2].Steps[0])
	}
	if manifest.Stages[3].StageID != "url_collection" || len(manifest.Stages[3].Steps) != 2 {
		t.Fatalf("expected Fingerprint Detection and URL Collection after Website Discovery, got %+v", manifest.Stages[3])
	}
	if manifest.Stages[3].Steps[0].StepID != "fingerprint_detection" || manifest.Stages[3].Steps[0].EngineID != "engine.lunafox.fingerprint_detection" {
		t.Fatalf("expected Fingerprint Detection first in URL Collection stage, got %+v", manifest.Stages[3].Steps[0])
	}
	if manifest.Stages[3].Steps[1].StepID != "url_collection" || manifest.Stages[3].Steps[1].EngineID != "engine.lunafox.url_collection" {
		t.Fatalf("expected URL Collection beside Fingerprint Detection, got %+v", manifest.Stages[3].Steps[1])
	}
	if manifest.Stages[4].StageID != "screenshot" || len(manifest.Stages[4].Steps) != 1 || manifest.Stages[4].Steps[0].EngineID != "engine.lunafox.screenshot" {
		t.Fatalf("expected Screenshot after URL Collection, got %+v", manifest.Stages[4])
	}
	if manifest.Stages[5].StageID != "directory_scan" || len(manifest.Stages[5].Steps) != 1 {
		t.Fatalf("expected Directory Scan in a dedicated final stage, got %+v", manifest.Stages[5])
	}
	if manifest.Stages[5].Steps[0].StepID != "directory_scan" || manifest.Stages[5].Steps[0].EngineID != "engine.lunafox.directory_scan" {
		t.Fatalf("unexpected Directory Scan step identity: %+v", manifest.Stages[5].Steps[0])
	}
	if manifest.Stages[6].StageID != "nuclei_vulnerability" || len(manifest.Stages[6].Steps) != 1 || manifest.Stages[6].Steps[0].StepID != "nuclei_vulnerability" || manifest.Stages[6].Steps[0].EngineID != "engine.lunafox.nuclei_vulnerability" {
		t.Fatalf("unexpected Nuclei vulnerability stage: %+v", manifest.Stages[6])
	}
}

func TestRemovedEngineNamedWorkflowsAreNotRegistered(t *testing.T) {
	for _, workflowID := range []string{"subdomain_discovery", "port_scan", "nuclei_vulnerability"} {
		if manifest, err := GetManifest(workflowID); err == nil {
			t.Fatalf("expected workflow %q to be removed, got %+v", workflowID, manifest)
		}
	}
}

func TestManifestFilenamesMatchWorkflowIDs(t *testing.T) {
	items, err := loadWorkflowDefinitions()
	if err != nil {
		t.Fatalf("loadWorkflowDefinitions failed: %v", err)
	}
	for _, manifest := range items {
		expectedFilename := manifestFilename(manifest.ScanWorkflowID)
		if expectedFilename != manifestFilename(manifest.ScanWorkflowID) {
			t.Fatalf("unexpected manifest filename for scanWorkflowId %q", manifest.ScanWorkflowID)
		}
	}
}

func TestDecodeManifestRejectsUnknownField(t *testing.T) {
	_, err := decodeManifest([]byte(`{"scanWorkflowId":"subdomain_discovery","displayName":"Subdomain Discovery","description":"Discover subdomains for target domains.","stages":[{"stageId":"recon","steps":[{"stepId":"scan","engineId":"engine.lunafox.subdomain_discovery","engineConfig":{},"unknown":true}]}]}`), "test")
	if err == nil {
		t.Fatal("expected unknown field rejection")
	}
}
