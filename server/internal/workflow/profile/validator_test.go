package profile

import (
	"fmt"
	"strings"
	"testing"
)

func validSubdomainDiscoveryConfig() map[string]any {
	return map[string]any{
		"subdomain_discovery": map[string]any{
			"recon": map[string]any{
				"enabled": true,
				"tools": map[string]any{
					"subfinder": map[string]any{
						"enabled":         true,
						"timeout-runtime": 3600,
						"threads-cli":     10,
					},
				},
			},
			"bruteforce": map[string]any{
				"enabled": false,
				"tools": map[string]any{
					"subdomain-bruteforce": map[string]any{
						"enabled": false,
					},
				},
			},
			"permutation": map[string]any{
				"enabled": false,
				"tools": map[string]any{
					"subdomain-permutation-resolve": map[string]any{
						"enabled": false,
					},
				},
			},
			"resolve": map[string]any{
				"enabled": true,
				"tools": map[string]any{
					"subdomain-resolve": map[string]any{
						"enabled":         true,
						"timeout-runtime": 3600,
						"threads-cli":     50,
						"rate-limit-cli":  1000,
					},
				},
			},
		},
	}
}

func validateProfile(profile *Profile) error {
	if profile == nil {
		return fmt.Errorf("profile is nil")
	}

	if err := ValidateConfiguration(profile.Configuration); err != nil {
		return fmt.Errorf("profile '%s': %w", profile.ID, err)
	}

	return nil
}

func TestValidateConfiguration_Empty(t *testing.T) {
	err := ValidateConfiguration(nil)
	if err != nil {
		t.Errorf("ValidateConfiguration() with nil should not error, got: %v", err)
	}
}

func TestValidateConfiguration_InvalidRootType(t *testing.T) {
	err := ValidateConfiguration([]any{"invalid"})
	if err == nil {
		t.Error("ValidateConfiguration() with invalid root type should error")
	}
}

func TestValidateConfiguration_ValidSubdomainDiscovery(t *testing.T) {
	err := ValidateConfiguration(validSubdomainDiscoveryConfig())
	if err != nil {
		t.Errorf("ValidateConfiguration() with valid config should not error, got: %v", err)
	}
}

func TestValidateConfiguration_InvalidSubdomainDiscovery(t *testing.T) {
	config := map[string]any{
		"subdomain_discovery": map[string]any{
			"recon": map[string]any{
				"enabled": true,
				"tools": map[string]any{
					"subfinder": map[string]any{
						"timeout-runtime": 3600,
					},
				},
			},
		},
	}
	if err := ValidateConfiguration(config); err == nil {
		t.Error("ValidateConfiguration() with invalid subdomain_discovery config should error")
	}
}

func TestValidateConfiguration_UnknownWorkflow(t *testing.T) {
	config := map[string]any{
		"unknown_workflow": map[string]any{"enabled": true},
	}
	err := ValidateConfiguration(config)
	if err == nil {
		t.Fatal("ValidateConfiguration() should reject unknown workflow keys")
	}
	if !strings.Contains(err.Error(), "unknown workflow key") {
		t.Fatalf("expected unknown workflow key error, got: %v", err)
	}
}

func TestExtractWorkflowIDsFromObject_OnlyExtracts(t *testing.T) {
	root, err := normalizeConfiguration(validSubdomainDiscoveryConfig())
	if err != nil {
		t.Fatalf("normalizeConfiguration() returned error: %v", err)
	}
	ids := extractWorkflowIDsFromConfig(root)
	if len(ids) != 1 || ids[0] != "subdomain_discovery" {
		t.Fatalf("expected [subdomain_discovery], got %v", ids)
	}
}

func TestExtractWorkflowIDsFromObject_DoesNotValidateKnownWorkflowSet(t *testing.T) {
	root, err := normalizeConfiguration(map[string]any{
		"unknown_workflow": map[string]any{"enabled": true},
	})
	if err != nil {
		t.Fatalf("normalizeConfiguration() should succeed, got: %v", err)
	}
	ids := extractWorkflowIDsFromConfig(root)
	if len(ids) != 1 || ids[0] != "unknown_workflow" {
		t.Fatalf("expected [unknown_workflow], got %v", ids)
	}
}

func TestValidateAndExtractWorkflowIDs_RejectsUnknownWorkflow(t *testing.T) {
	_, err := ValidateAndExtractWorkflowIDs(map[string]any{
		"unknown_workflow": map[string]any{"enabled": true},
	})
	if err == nil {
		t.Fatal("ValidateAndextractWorkflowIDs() should reject unknown workflow keys")
	}
	if !strings.Contains(err.Error(), "unknown workflow key") {
		t.Fatalf("expected unknown workflow key error, got: %v", err)
	}
}

func TestValidateProfile_Nil(t *testing.T) {
	err := validateProfile(nil)
	if err == nil {
		t.Error("validateProfile() with nil should error")
	}
}

func TestValidateProfile_Valid(t *testing.T) {
	profile := &Profile{
		ID:            "test_profile",
		Name:          "Test Profile",
		Configuration: validSubdomainDiscoveryConfig(),
	}
	err := validateProfile(profile)
	if err != nil {
		t.Errorf("validateProfile() with valid profile should not error, got: %v", err)
	}
}

func TestAllProfilesValid(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() failed: %v", err)
	}

	profiles := loader.List()
	for _, profile := range profiles {
		t.Run(profile.ID, func(t *testing.T) {
			if err := validateProfile(&profile); err != nil {
				t.Errorf("Profile '%s' has invalid configuration: %v", profile.ID, err)
			}
		})
	}

	t.Logf("Validated %d profiles successfully", len(profiles))
}

func TestGeneratedSubdomainProfilePassesSchemaValidation(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() failed: %v", err)
	}
	profile := loader.GetByID("subdomain_discovery")
	if profile == nil {
		t.Fatal("expected generated profile subdomain_discovery")
	}
	if err := ValidateConfiguration(profile.Configuration); err != nil {
		t.Fatalf("generated profile configuration should pass schema validation, got: %v", err)
	}
}
