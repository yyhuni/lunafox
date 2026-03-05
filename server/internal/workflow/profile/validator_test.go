package profile

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// validateProfile is a test helper that validates a profile's configuration.
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
	err := ValidateConfiguration("")
	if err != nil {
		t.Errorf("ValidateConfiguration() with empty string should not error, got: %v", err)
	}
}

func TestValidateConfiguration_InvalidYAML(t *testing.T) {
	err := ValidateConfiguration("invalid: yaml: content:")
	if err == nil {
		t.Error("ValidateConfiguration() with invalid YAML should error")
	}
}

func TestValidateConfiguration_ValidSubdomainDiscovery(t *testing.T) {
	config := `
subdomain_discovery:
  recon:
    enabled: true
    tools:
      subfinder:
        enabled: true
        timeout-runtime: 3600
        threads-cli: 10
  bruteforce:
    enabled: false
    tools:
      subdomain-bruteforce:
        enabled: false
  permutation:
    enabled: false
    tools:
      subdomain-permutation-resolve:
        enabled: false
  resolve:
    enabled: true
    tools:
      subdomain-resolve:
        enabled: true
        timeout-runtime: 3600
        threads-cli: 50
        rate-limit-cli: 1000
`
	err := ValidateConfiguration(config)
	if err != nil {
		t.Errorf("ValidateConfiguration() with valid config should not error, got: %v", err)
	}
}

func TestValidateConfiguration_InvalidSubdomainDiscovery(t *testing.T) {
	// Missing required field 'enabled' in subfinder when it should be required
	config := `
subdomain_discovery:
  recon:
    enabled: true
    tools:
      subfinder:
        timeout-runtime: 3600
`
	err := ValidateConfiguration(config)
	if err == nil {
		t.Error("ValidateConfiguration() with invalid subdomain_discovery config should error")
	}
}

func TestValidateConfiguration_UnknownWorkflow(t *testing.T) {
	config := `
unknown_workflow:
  enabled: true
`
	err := ValidateConfiguration(config)
	if err == nil {
		t.Fatal("ValidateConfiguration() should reject unknown workflow keys")
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
		ID:   "test_profile",
		Name: "Test Profile",
		Configuration: `
subdomain_discovery:
  recon:
    enabled: false
    tools:
      subfinder:
        enabled: false
  bruteforce:
    enabled: false
    tools:
      subdomain-bruteforce:
        enabled: false
  permutation:
    enabled: false
    tools:
      subdomain-permutation-resolve:
        enabled: false
  resolve:
    enabled: false
    tools:
      subdomain-resolve:
        enabled: false
`,
	}
	err := validateProfile(profile)
	if err != nil {
		t.Errorf("validateProfile() with valid profile should not error, got: %v", err)
	}
}

// TestAllProfilesValid validates all loaded profiles have valid configurations.
// This test is intended to be run in CI to catch configuration errors early.
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

	// Log summary
	t.Logf("Validated %d profiles successfully", len(profiles))
}

func TestExampleProfileTemplateHasVersionedSubdomainConfig(t *testing.T) {
	data, err := profilesFS.ReadFile("presets/_example.yaml")
	if err != nil {
		t.Fatalf("read _example.yaml failed: %v", err)
	}

	var profile Profile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		t.Fatalf("parse _example.yaml failed: %v", err)
	}

	if err := ValidateConfiguration(profile.Configuration); err != nil {
		t.Fatalf("_example.yaml configuration should pass schema validation, got: %v", err)
	}
}
