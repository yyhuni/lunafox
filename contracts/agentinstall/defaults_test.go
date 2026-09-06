package agentinstall

import "testing"

func TestDefaultAgentDockerNetwork(t *testing.T) {
	if DefaultAgentDockerNetwork != "lunafox_network" {
		t.Fatalf("unexpected default agent docker network: %s", DefaultAgentDockerNetwork)
	}
}
