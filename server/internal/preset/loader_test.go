package preset

import (
	"testing"
)

func TestNewLoader(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() failed: %v", err)
	}

	// Loader should initialize successfully even with no preset files
	// (only _example.yaml which is skipped)
	if loader == nil {
		t.Fatal("NewLoader() returned nil loader")
	}
}

func TestLoaderSkipsUnderscoreFiles(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() failed: %v", err)
	}

	// _example.yaml should be skipped, so we should have no presets
	// unless actual preset files are added
	presets := loader.List()

	// Check that _example.yaml is not loaded
	for _, p := range presets {
		if p.ID == "example_preset" {
			t.Error("_example.yaml should not be loaded as a preset")
		}
	}
}

func TestLoaderGetByID(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() failed: %v", err)
	}

	// Non-existent ID should return nil
	p := loader.GetByID("non_existent_id")
	if p != nil {
		t.Error("GetByID() should return nil for non-existent ID")
	}
}
