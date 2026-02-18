package preset

import (
	"fmt"
	"testing"
)

// validatePreset is a test helper that validates a preset's configuration.
func validatePreset(preset *Preset) error {
	if preset == nil {
		return fmt.Errorf("preset is nil")
	}

	if err := ValidateConfiguration(preset.Configuration); err != nil {
		return fmt.Errorf("preset '%s': %w", preset.ID, err)
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
      assetfinder:
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

func TestValidateConfiguration_UnknownEngine(t *testing.T) {
	// Unknown engines should be ignored (no schema to validate against)
	config := `
unknown_engine:
  enabled: true
`
	err := ValidateConfiguration(config)
	if err != nil {
		t.Errorf("ValidateConfiguration() with unknown engine should not error, got: %v", err)
	}
}

func TestValidatePreset_Nil(t *testing.T) {
	err := validatePreset(nil)
	if err == nil {
		t.Error("validatePreset() with nil should error")
	}
}

func TestValidatePreset_Valid(t *testing.T) {
	preset := &Preset{
		ID:   "test_preset",
		Name: "Test Preset",
		Configuration: `
subdomain_discovery:
  recon:
    enabled: false
    tools:
      subfinder:
        enabled: false
      assetfinder:
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
	err := validatePreset(preset)
	if err != nil {
		t.Errorf("validatePreset() with valid preset should not error, got: %v", err)
	}
}

// TestAllPresetsValid validates all loaded presets have valid configurations.
// This test is intended to be run in CI to catch configuration errors early.
func TestAllPresetsValid(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() failed: %v", err)
	}

	presets := loader.List()
	for _, preset := range presets {
		t.Run(preset.ID, func(t *testing.T) {
			if err := validatePreset(&preset); err != nil {
				t.Errorf("Preset '%s' has invalid configuration: %v", preset.ID, err)
			}
		})
	}

	// Log summary
	t.Logf("Validated %d presets successfully", len(presets))
}
