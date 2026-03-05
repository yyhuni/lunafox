package profile

import (
	"testing"
)

func TestNewLoader(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() failed: %v", err)
	}

	// Loader should initialize successfully even with no profile files
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

	// _example.yaml should be skipped, so we should have no profiles
	// unless actual profile files are added
	profiles := loader.List()

	// Check that _example.yaml is not loaded
	for _, p := range profiles {
		if p.ID == "example_profile" {
			t.Error("_example.yaml should not be loaded as a profile")
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
