package application

import "testing"

func TestSnapshotFacadeConstructorsExist(t *testing.T) {
	if NewWebsiteSnapshotFacade(nil, nil) == nil {
		t.Fatalf("expected website snapshot facade")
	}
	if NewSubdomainSnapshotFacade(nil, nil) == nil {
		t.Fatalf("expected subdomain snapshot facade")
	}
	if NewEndpointSnapshotFacade(nil, nil) == nil {
		t.Fatalf("expected endpoint snapshot facade")
	}
	if NewDirectorySnapshotFacade(nil, nil) == nil {
		t.Fatalf("expected directory snapshot facade")
	}
	if NewHostPortSnapshotFacade(nil, nil) == nil {
		t.Fatalf("expected host-port snapshot facade")
	}
	if NewScreenshotSnapshotFacade(nil, nil) == nil {
		t.Fatalf("expected screenshot snapshot facade")
	}
	if NewVulnerabilitySnapshotFacade(nil, nil) == nil {
		t.Fatalf("expected vulnerability snapshot facade")
	}
}
