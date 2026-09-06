package workflowmanifest

import "testing"

func validManifestForValidation() Manifest {
	return Manifest{
		ScanWorkflowID: "subdomain_discovery",
		DisplayName:    "Subdomain Discovery",
		Description:    "Discover subdomains for target domains.",
		Stages: []ManifestStage{
			{
				StageID: "recon",
				Steps: []ManifestStep{
					{
						StepID:   "subdomain_discovery",
						EngineID: "engine.lunafox.subdomain_discovery",
					},
				},
			},
		},
	}
}

func TestValidateManifestRejectsDuplicateStage(t *testing.T) {
	manifest := validManifestForValidation()
	manifest.Stages = append(manifest.Stages, ManifestStage{StageID: "recon", Steps: []ManifestStep{{StepID: "dns", EngineID: "engine.lunafox.subdomain_discovery"}}})

	if err := validateManifest(manifest); err == nil {
		t.Fatal("expected duplicate stage rejection")
	}
}

func TestValidateManifestRejectsDuplicateStepIDAcrossWorkflow(t *testing.T) {
	manifest := validManifestForValidation()
	manifest.Stages = append(manifest.Stages, ManifestStage{StageID: "resolve", Steps: []ManifestStep{{StepID: "subdomain_discovery", EngineID: "engine.lunafox.subdomain_discovery"}}})

	if err := validateManifest(manifest); err == nil {
		t.Fatal("expected duplicate step rejection")
	}
}

func TestValidateManifestAllowsEmptyDescription(t *testing.T) {
	manifest := validManifestForValidation()
	manifest.Description = " "

	if err := validateManifest(manifest); err != nil {
		t.Fatalf("expected optional description to be accepted, got %v", err)
	}
}
