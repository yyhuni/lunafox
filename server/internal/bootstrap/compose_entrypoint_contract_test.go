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

func TestComposeBootstrapRequiresKnownRegistryAndIgnoresLegacyAccelerationKey(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	script, err := filepath.Abs(filepath.Join(root, "docker", "bootstrap", "bootstrap.sh"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name         string
		registry     string
		acceleration *string
		wantSuccess  bool
		wantWarning  bool
	}{
		{name: "Docker Hub without the legacy key", registry: "docker.io", wantSuccess: true},
		{name: "GHCR without the legacy key", registry: "ghcr.io", wantSuccess: true},
		{name: "legacy enabled key is ignored", registry: "ghcr.io", acceleration: stringPointer("true"), wantSuccess: true, wantWarning: true},
		{name: "legacy malformed key is ignored", registry: "docker.io", acceleration: stringPointer("sometimes"), wantSuccess: true, wantWarning: true},
		{name: "unknown registry fails", registry: "registry.example"},
		{name: "unknown registry fails with the legacy key", registry: "registry.example", acceleration: stringPointer("false")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			calls := filepath.Join(directory, "calls")
			fakeServer := "#!/bin/sh\nprintf '%s\\n' \"$1\" >> \"$CALLS\"\n"
			if err := os.WriteFile(filepath.Join(directory, "server"), []byte(fakeServer), 0o755); err != nil {
				t.Fatal(err)
			}
			environment := composeBootstrapEnvironment(map[string]string{
				"PATH":                          directory + ":" + os.Getenv("PATH"),
				"CALLS":                         calls,
				"ENGINE_INSTALL_INVENTORY_PATH": "/inventory",
				"ENGINE_INSTALL_REGISTRY":       test.registry,
				"FINGERPRINT_BOOTSTRAP_PATH":    "/fingerprints",
				"WORDLISTS_SOURCE_PATH":         "/wordlists",
				"AGENT_VERSION":                 "1.0.0",
			})
			if test.acceleration != nil {
				environment = append(environment, "ENGINE_INSTALL_CF_ACCELERATION="+*test.acceleration)
			}
			command := exec.Command("bash", script)
			command.Env = environment
			output, err := command.CombinedOutput()
			if (err == nil) != test.wantSuccess {
				t.Fatalf("exit = %v, output = %s", err, output)
			}
			if !test.wantSuccess {
				return
			}
			data, err := os.ReadFile(calls)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(string(data), "engine-bootstrap\n") {
				t.Fatalf("bootstrap call order = %s", data)
			}
			if test.wantWarning && !strings.Contains(string(output), "ENGINE_INSTALL_CF_ACCELERATION is deprecated") {
				t.Fatalf("legacy key warning missing from output: %s", output)
			}
		})
	}
}

func composeBootstrapEnvironment(overrides map[string]string) []string {
	environment := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, found := strings.Cut(entry, "=")
		if !found || key == "ENGINE_INSTALL_CF_ACCELERATION" || key == "ENGINE_INSTALL_REGISTRY" {
			continue
		}
		if _, overridden := overrides[key]; !overridden {
			environment = append(environment, entry)
		}
	}
	for key, value := range overrides {
		environment = append(environment, key+"="+value)
	}
	return environment
}

func stringPointer(value string) *string {
	return &value
}
