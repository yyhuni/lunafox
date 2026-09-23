package bootstrap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestComposeBootstrapOrdersInitializationAndPropagatesFailure(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	script, err := filepath.Abs(filepath.Join(root, "docker", "bootstrap", "bootstrap.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, failure := range []string{"", "engine-bootstrap", "fingerprint-bootstrap", "wordlist-bootstrap", "agent-bootstrap"} {
		t.Run("failure="+failure, func(t *testing.T) {
			directory := t.TempDir()
			fakeServer := "#!/bin/sh\nprintf '%s\\n' \"$1\" >> \"$CALLS\"\n[ \"$1\" != \"$FAIL_AT\" ]\n"
			if err := os.WriteFile(filepath.Join(directory, "server"), []byte(fakeServer), 0o755); err != nil {
				t.Fatal(err)
			}
			calls := filepath.Join(directory, "calls")
			command := exec.Command("bash", script)
			command.Env = append(os.Environ(),
				"PATH="+directory+":"+os.Getenv("PATH"),
				"CALLS="+calls,
				"FAIL_AT="+failure,
				"ENGINE_INSTALL_INVENTORY_PATH=/inventory",
				"ENGINE_INSTALL_REGISTRY=docker.io",
				"ENGINE_INSTALL_CF_ACCELERATION=false",
				"FINGERPRINT_BOOTSTRAP_PATH=/fingerprints",
				"WORDLISTS_SOURCE_PATH=/wordlists",
				"AGENT_VERSION=1.0.0",
			)
			output, err := command.CombinedOutput()
			if (err != nil) != (failure != "") {
				t.Fatalf("exit: %v, %s", err, output)
			}
			data, err := os.ReadFile(calls)
			if err != nil {
				t.Fatal(err)
			}
			expected := []string{"engine-bootstrap", "fingerprint-bootstrap", "wordlist-bootstrap", "agent-bootstrap"}
			if failure != "" {
				for index, name := range expected {
					if name == failure {
						expected = expected[:index+1]
						break
					}
				}
			}
			if string(data) != strings.Join(expected, "\n")+"\n" {
				t.Fatalf("initialization sequence = %s", data)
			}
		})
	}
}

func TestComposeBootstrapDoesNotCreateResidentContainers(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	source, err := os.ReadFile(filepath.Join(root, "docker", "bootstrap", "bootstrap.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"docker run", "lunafox-loki", "lunafox_receipt", "AGENT_AUTHENTICATION_TOKEN"} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("unexpected host lifecycle behavior: %s", forbidden)
		}
	}
}
