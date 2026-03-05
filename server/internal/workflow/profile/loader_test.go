package profile

import (
	"testing"
)

func TestNewLoader(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() failed: %v", err)
	}

	// Loader should initialize successfully when generated profile files exist.
	if loader == nil {
		t.Fatal("NewLoader() returned nil loader")
	}
}

func TestLoaderHasGeneratedProfiles(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() failed: %v", err)
	}

	profiles := loader.List()
	if len(profiles) == 0 {
		t.Fatal("expected at least one generated workflow profile")
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
