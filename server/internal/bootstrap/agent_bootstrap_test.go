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
		ID:                  7,
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
	if credentials != (agentBootstrapCredentials{AgentID: 7, InstanceID: "agent-1", DisplayName: "LunaFox Agent", AuthenticationToken: "a1b2c3d4"}) {
		t.Fatalf("credentials = %+v", credentials)
	}
	if err := writeAgentBootstrapCredentials(path, agent); err == nil {
		t.Fatal("expected existing credential file to fail closed")
	}
}

func TestEnsureAgentBootstrapCredentialsReusesDatabaseIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.json")
	agent := &agentdomain.Agent{ID: 7, InstanceID: "agent-1", DisplayName: "Initial name", AuthenticationToken: "deadbeef"}
	created := 0
	create := func() (*agentdomain.Agent, error) { created++; return agent, nil }
	find := func(token string) (*agentdomain.Agent, error) {
		if token != agent.AuthenticationToken {
			t.Fatal("wrong token")
		}
		return agent, nil
	}
	if err := ensureAgentBootstrapCredentials(path, "", find, create); err != nil {
		t.Fatal(err)
	}
	agent.DisplayName = "Renamed on server"
	if err := ensureAgentBootstrapCredentials(path, agent.InstanceID, find, create); err != nil {
		t.Fatal(err)
	}
	if created != 1 {
		t.Fatalf("created %d agents", created)
	}
}

func TestEnsureAgentBootstrapCredentialsRejectsPartialState(t *testing.T) {
	for _, scenario := range []string{"missing-file", "missing-binding", "wrong-binding", "wrong-agent", "corrupt", "symlink", "public-file", "executable-file", "trailing-json"} {
		t.Run(scenario, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "agent.json")
			agent := &agentdomain.Agent{ID: 7, InstanceID: "agent-1", DisplayName: "Agent", AuthenticationToken: "deadbeef"}
			if err := writeAgentBootstrapCredentials(path, agent); err != nil {
				t.Fatal(err)
			}
			binding := agent.InstanceID
			switch scenario {
			case "missing-file":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "missing-binding":
				binding = ""
			case "wrong-binding":
				binding = "other"
			case "wrong-agent":
				agent.InstanceID = "other"
			case "corrupt":
				if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Rename(path, path+".real"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+".real", path); err != nil {
					t.Fatal(err)
				}
			case "public-file":
				if err := os.Chmod(path, 0o644); err != nil {
					t.Fatal(err)
				}
			case "executable-file":
				if err := os.Chmod(path, 0o700); err != nil {
					t.Fatal(err)
				}
			case "trailing-json":
				f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
				if err != nil {
					t.Fatal(err)
				}
				_, err = f.WriteString("{}")
				if err != nil {
					t.Fatal(err)
				}
				if err = f.Close(); err != nil {
					t.Fatal(err)
				}
			}
			err := ensureAgentBootstrapCredentials(path, binding, func(string) (*agentdomain.Agent, error) { return agent, nil }, func() (*agentdomain.Agent, error) { t.Fatal("must not register from partial state"); return nil, nil })
			if err == nil {
				t.Fatal("expected explicit failure")
			}
		})
	}
}
