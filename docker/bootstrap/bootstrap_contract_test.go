package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapEntrypointOwnsClosedInitializationSequence(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	source, err := os.ReadFile(filepath.Join(root, "docker", "bootstrap", "bootstrap.sh"))
	if err != nil {
		t.Fatalf("read bootstrap entrypoint: %v", err)
	}
	contents := string(source)
	for _, required := range []string{
		"server engine-bootstrap",
		"server fingerprint-bootstrap",
		"server wordlist-bootstrap",
		"lunafox-engine-mount-preflight",
		"server agent-bootstrap",
		"--name \"$agent_name\"",
		"--restart unless-stopped",
		"--network \"$BOOTSTRAP_NETWORK\"",
		"--label com.docker.compose.project=lunafox",
		"-v \"$LUNAFOX_AGENT_STATE_VOLUME:/var/lib/lunafox-agent\"",
		"-v \"$LUNAFOX_ENGINE_EXECUTION_VOLUME:$LUNAFOX_ENGINE_EXECUTION_ROOT\"",
	} {
		if !strings.Contains(contents, required) {
			t.Fatalf("bootstrap entrypoint is missing %q", required)
		}
	}
	for _, forbidden := range []string{"lunafox_receipt", "echo \"$agent_authentication_token\"", "printf '%s\\n' \"$agent_authentication_token\""} {
		if strings.Contains(contents, forbidden) {
			t.Fatalf("bootstrap entrypoint must not expose %q", forbidden)
		}
	}
}
