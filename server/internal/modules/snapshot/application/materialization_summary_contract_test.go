package application

import (
	"reflect"
	"testing"
)

func TestSnapshotCommandServicesExposeSummaryReturningSaveAndSync(t *testing.T) {
	services := []any{
		&DirectorySnapshotCommandService{},
		&SubdomainSnapshotCommandService{},
		&HostPortSnapshotCommandService{},
		&WebsiteSnapshotCommandService{},
		&EndpointSnapshotCommandService{},
		&ScreenshotSnapshotCommandService{},
		&VulnerabilitySnapshotCommandService{},
	}

	for _, service := range services {
		method, ok := reflect.TypeOf(service).MethodByName("SaveAndSync")
		if !ok {
			t.Fatalf("%T must expose SaveAndSync", service)
		}
		if method.Type.NumOut() != 2 {
			t.Errorf("%T SaveAndSync must return summary and error, got %d results", service, method.Type.NumOut())
		}
	}
}
