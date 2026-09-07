package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

func TestWriteAgentBootstrapCredentialsWritesSecretOnlyToRestrictedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bootstrap", "agent.json")
	agent := &agentdomain.Agent{
		InstanceID:          "agent-1",
		DisplayName:         "LunaFox Agent",
		AuthenticationToken: "a1b2c3d4",
	}
	if err := writeAgentBootstrapCredentials(path, agent); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat credentials: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("credentials mode = %o, want 600", info.Mode().Perm())
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read credentials: %v", err)
	}
	var credentials agentBootstrapCredentials
	if err := json.Unmarshal(contents, &credentials); err != nil {
		t.Fatalf("decode credentials: %v", err)
	}
	if credentials != (agentBootstrapCredentials{InstanceID: "agent-1", DisplayName: "LunaFox Agent", AuthenticationToken: "a1b2c3d4"}) {
		t.Fatalf("credentials = %+v", credentials)
	}
	if err := writeAgentBootstrapCredentials(path, agent); err == nil {
		t.Fatal("expected existing credential file to fail closed")
	}
}
