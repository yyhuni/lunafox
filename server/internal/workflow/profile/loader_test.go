package profile

import (
	"reflect"
	"testing"
)

func TestNewLoader(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() failed: %v", err)
	}

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

	p := loader.GetByID("non_existent_id")
	if p != nil {
		t.Error("GetByID() should return nil for non-existent ID")
	}
}

func TestLoaderParsesConfigurationAsObject(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() failed: %v", err)
	}

	profile := loader.GetByID("subdomain_discovery")
	if profile == nil {
		t.Fatal("expected generated profile subdomain_discovery")
	}

	if reflect.TypeOf(profile.Configuration).Kind() != reflect.Map {
		t.Fatalf("expected configuration to be map, got %T", profile.Configuration)
	}

	raw, ok := profile.Configuration["subdomain_discovery"]
	if !ok {
		t.Fatalf("expected top-level key subdomain_discovery in configuration: %#v", profile.Configuration)
	}
	if reflect.TypeOf(raw).Kind() != reflect.Map {
		t.Fatalf("expected subdomain_discovery config to be map, got %T", raw)
	}
}

