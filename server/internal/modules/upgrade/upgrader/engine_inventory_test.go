package upgrader

import (
	"strings"
	"testing"
	"time"
)

func TestParseEngineInventoryRequiresCompleteSortedPairs(t *testing.T) {
	digestA := "sha256:" + strings.Repeat("a", 64)
	valid := `{"schemaVersion":1,"components":[{"id":"engine.lunafox.port_scan.package","digest":"` + digestA + `"},{"id":"engine.lunafox.port_scan.runtime","digest":"` + digestA + `"}]}`
	inventory, err := ParseEngineInventory([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Components) != 2 {
		t.Fatalf("component count = %d, want 2", len(inventory.Components))
	}

	tests := []struct {
		name string
		raw  string
	}{
		{name: "missing package", raw: `{"schemaVersion":1,"components":[{"id":"engine.lunafox.port_scan.runtime","digest":"` + digestA + `"}]}`},
		{name: "unknown field", raw: `{"schemaVersion":1,"components":[],"extra":true}`},
		{name: "tag-like digest", raw: `{"schemaVersion":1,"components":[{"id":"engine.lunafox.port_scan.package","digest":"latest"},{"id":"engine.lunafox.port_scan.runtime","digest":"` + digestA + `"}]}`},
		{name: "trailing json", raw: valid + `{}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseEngineInventory([]byte(test.raw)); err == nil {
				t.Fatal("expected strict inventory validation error")
			}
		})
	}
}

func TestRuntimeObservationMatchesCompleteEngineInventory(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	state := ConfirmedDeploymentState{Components: []DeploymentComponent{
		{ID: runtimeServerComponent, Digest: digest},
		{ID: runtimeFrontendComponent, Digest: digest},
		{ID: runtimeNginxComponent, Digest: digest},
		{ID: runtimeAgentComponent, Digest: digest},
		{ID: runtimeBootstrapComponent, Digest: digest},
		{ID: "engine.lunafox.port_scan.package", Digest: digest},
		{ID: "engine.lunafox.port_scan.runtime", Digest: digest},
	}, NginxConfigDigest: digest}
	observation := RuntimeObservation{Images: make(map[string]string, len(state.Components))}
	for _, component := range state.Components {
		observation.Images[component.ID] = component.Digest
	}
	observation.NginxConfigDigest = digest
	observation.NginxHealthy = true
	observation.ObservedAt = time.Now().UTC()
	if !runtimeObservationMatchesState(observation, state) {
		t.Fatal("complete engine inventory should match confirmed state")
	}
	delete(observation.Images, "engine.lunafox.port_scan.runtime")
	if runtimeObservationMatchesState(observation, state) {
		t.Fatal("incomplete engine inventory must not match confirmed state")
	}
}
