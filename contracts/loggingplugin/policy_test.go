package loggingplugin

import (
	"os/exec"
	"strings"
	"testing"
)

func TestReference(t *testing.T) {
	for _, arch := range []string{"amd64", "x86_64", "arm64", "aarch64"} {
		ref, err := Reference(arch)
		if err != nil {
			t.Fatal(err)
		}
		out, err := exec.Command("bash", "-c", policy+"\nlunafox_loki_reference \"$1\"", "policy", arch).CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != ref {
			t.Fatalf("shell/Go policy diverged: %s %v", out, err)
		}
	}
	for _, arch := range []string{"", "riscv64", "linux/amd64"} {
		if _, err := Reference(arch); err == nil {
			t.Fatalf("accepted %q", arch)
		}
	}
}

// Stateful shell fake exercises real manager control flow without touching the
// developer's daemon. Read commands run in subshells just as real Docker does.
const fakeDocker = `
id=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
ref=grafana/loki-docker-driver:3.6.7-amd64
target=$ref
present=true
enabled=true
owned=true
refs=''
failure=''
on_disable=''
on_enable=''
record_id=$id
lunafox_plugin_docker() {
  printf '%s\n' "$*" >> "$TRACE"
  [ "$failure" != "$1 $2" ] || return 1
  case "$1 $2" in
    'info --format') echo amd64 ;;
    'plugin ls') if [ "$present" = true ]; then echo lunafox-loki:latest; fi ;;
    'plugin inspect') printf '%s|%s|%s|docker.logdriver/1.0\n' "$id" "$ref" "$enabled" ;;
    'volume inspect') [ "$owned" = true ] || return 1; echo "$record_id" ;;
    'volume create') owned=true ;;
    'volume rm') owned=false ;;
    'ps -aq') printf '%s' "$refs" ;;
    'inspect --type') echo "$driver" ;;
    'plugin install') present=true; ref=$target; enabled=true ;;
    'plugin enable') enabled=true; eval "$on_enable" ;;
    'plugin disable') enabled=false; eval "$on_disable" ;;
    'plugin upgrade') ref=$target ;;
    'plugin rm') present=false ;;
    *) echo "unexpected Docker call: $*" >&2; return 99 ;;
  esac
}
`

func TestManager(t *testing.T) {
	cases := []struct {
		name, setup, action, assertion string
		wantError                      bool
	}{
		{"install missing", "present=false; owned=false", "install", `[ "$present" = true ] && [ "$owned" = true ]`, false},
		{"reuse", "", "install", `! grep -Eq 'plugin (install|disable|upgrade|enable)' "$TRACE"`, false},
		{"Docker normalized reference", "ref=docker.io/grafana/loki-docker-driver:3.6.7-amd64", "start", `[ "$LUNAFOX_PLUGIN_REF" = "$target" ]`, false},
		{"enable", "enabled=false", "start", `[ "$enabled" = true ]`, false},
		{"start missing", "present=false; owned=false", "start", `[ "$owned" = true ]`, false},
		{"update", "ref=grafana/loki-docker-driver:3.6.6-amd64", "install", `[ "$ref" = "$target" ] && [ "$enabled" = true ]`, false},
		{"start mismatch", "ref=grafana/loki-docker-driver:3.6.6-amd64", "start", "", true},
		{"unknown origin", "ref=other/plugin:3.6.7-amd64", "install", "", true},
		{"missing ownership", "owned=false", "install", "", true},
		{"ID drift", "record_id=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "install", "", true},
		{"missing target", "target=''", "start", "", true},
		{"wrong architecture", "target=grafana/loki-docker-driver:3.6.7-arm64", "start", "", true},
		{"metadata failure", "present=false; owned=false; failure='volume create'", "install", "", true},
		{"enable final state failure", "enabled=false; on_enable='enabled=false'", "start", "", true},
		{"reference race", "ref=grafana/loki-docker-driver:3.6.6-amd64; on_disable='refs=external; driver=lunafox-loki'", "install", "", true},
		{"identity race", "ref=grafana/loki-docker-driver:3.6.6-amd64; on_disable='id=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'", "install", "", true},
		{"inspect failure", "failure='plugin inspect'", "start", "", true},
		{"list failure", "failure='plugin ls'", "install", "", true},
		{"install failure", "present=false; failure='plugin install'", "install", "", true},
		{"enable failure", "enabled=false; failure='plugin enable'", "start", "", true},
		{"upgrade failure", "ref=grafana/loki-docker-driver:3.6.6-amd64; failure='plugin upgrade'", "install", "", true},
		{"referenced update", "ref=grafana/loki-docker-driver:3.6.6-amd64; refs=stopped; driver=lunafox-loki", "install", "", true},
		{"enumeration failure", "ref=grafana/loki-docker-driver:3.6.6-amd64; failure='ps -aq'", "install", "", true},
		{"container inspect failure", "ref=grafana/loki-docker-driver:3.6.6-amd64; refs=other; failure='inspect --type'", "install", "", true},
		{"remove", "", "remove", `[ "$present" = false ] && [ "$owned" = false ]`, false},
		{"remove missing", "present=false", "remove", `! grep -Eq 'plugin (install|rm|disable)' "$TRACE"`, false},
		{"remove unknown", "owned=false", "remove", "", true},
		{"remove referenced", "refs=external; driver=lunafox-loki:latest", "remove", "", true},
		{"remove inspect failure", "failure='plugin inspect'", "remove", "", true},
		{"remove reference race", "on_disable='refs=external; driver=lunafox-loki'", "remove", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			action := `lunafox_plugin_prepare ` + tc.action + ` "$target"`
			if tc.action == "remove" {
				action = "lunafox_plugin_remove"
			}
			program := "set -eu\n" + Script() + fakeDocker + "\n" + tc.setup + "\n"
			if tc.wantError {
				program += "if " + action + "; then echo 'unexpected success'; exit 1; fi\n"
				program += `! grep -Eq '(^plugin rm|^volume rm|--force| -f)' "$TRACE"` + "\n"
			} else {
				program += action + "\n" + tc.assertion + "\n"
			}
			cmd := exec.Command("bash", "-c", program)
			cmd.Env = append(cmd.Environ(), "TRACE="+t.TempDir()+"/trace")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
		})
	}
}
