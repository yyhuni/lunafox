package hostport

import (
	"testing"

	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

func TestHostPortOutputUsesResourceNameAndIP(t *testing.T) {
	output := toHostPortOutput(3, service.HostPortResponse{
		IP:    "192.0.2.10",
		Hosts: []string{"api.example.com"},
		Ports: []int{443},
	})

	if output.Name != "targets/3/hostPorts/192.0.2.10" {
		t.Fatalf("expected resource name, got %q", output.Name)
	}
	if output.IP != "192.0.2.10" {
		t.Fatalf("expected IP business field, got %q", output.IP)
	}
}
