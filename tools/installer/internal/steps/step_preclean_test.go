package steps

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/docker"
	"github.com/yyhuni/lunafox/tools/installer/internal/execx"
	"github.com/yyhuni/lunafox/tools/installer/internal/ui"
)

func TestStepPrecleanRunWithoutResidualContainers(t *testing.T) {
	runner := &fakePrecleanRunner{
		runFunc: func(command execx.Command) (execx.Result, error) {
			if isComposeDownCommand(command) {
				return execx.Result{Stdout: "ok"}, nil
			}
			if isListContainersCommand(command) {
				return execx.Result{Stdout: "lunafox-server-1\nlunafox-frontend-1\n"}, nil
			}
			if isRemoveContainersCommand(command) {
				t.Fatalf("unexpected remove command: %v", command.Args)
			}
			return execx.Result{}, nil
		},
	}

	installer := newStepPrecleanInstaller(t, runner)
	if err := (stepPreclean{}).Run(context.Background(), installer); err != nil {
		t.Fatalf("step preclean run failed: %v", err)
	}
}

func TestStepPrecleanRunRemovesResidualAgentContainers(t *testing.T) {
	var removed []string
	runner := &fakePrecleanRunner{
		runFunc: func(command execx.Command) (execx.Result, error) {
			if isComposeDownCommand(command) {
				return execx.Result{Stdout: "ok"}, nil
			}
			if isListContainersCommand(command) {
				return execx.Result{Stdout: "lunafox-agent\nlunafox-agent-prod\nlunafox-server-1\n"}, nil
			}
			if isRemoveContainersCommand(command) {
				removed = append([]string{}, command.Args[2:]...)
				return execx.Result{Stdout: "removed"}, nil
			}
			return execx.Result{}, nil
		},
	}

	installer := newStepPrecleanInstaller(t, runner)
	if err := (stepPreclean{}).Run(context.Background(), installer); err != nil {
		t.Fatalf("step preclean run failed: %v", err)
	}

	want := []string{"lunafox-agent", "lunafox-agent-prod"}
	if strings.Join(removed, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected removed containers: got=%v want=%v", removed, want)
	}
}

func TestStepPrecleanRunReturnsComposeError(t *testing.T) {
	runner := &fakePrecleanRunner{
		runFunc: func(command execx.Command) (execx.Result, error) {
			if isComposeDownCommand(command) {
				return execx.Result{Stderr: "compose not available", ExitCode: 1}, &execx.ExecError{
					Command: command,
					Result:  execx.Result{Stderr: "compose not available", ExitCode: 1},
					Cause:   errors.New("exit status 1"),
				}
			}
			return execx.Result{}, nil
		},
	}

	installer := newStepPrecleanInstaller(t, runner)
	err := (stepPreclean{}).Run(context.Background(), installer)
	if err == nil {
		t.Fatalf("expected compose down error")
	}
	if !strings.Contains(err.Error(), "compose down") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if !strings.Contains(err.Error(), "compose not available") {
		t.Fatalf("missing command detail in error: %v", err)
	}
}

func TestParseResidualAgentContainerNames(t *testing.T) {
	raw := "lunafox-agent\nlunafox-agent-1\nlunafox-server-1\n\nlunafox-agent\n"
	got := parseResidualAgentContainerNames(raw)
	want := []string{"lunafox-agent", "lunafox-agent-1"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected parse result: got=%v want=%v", got, want)
	}
}

type fakePrecleanRunner struct {
	runFunc func(command execx.Command) (execx.Result, error)
}

func (runner *fakePrecleanRunner) LookPath(file string) (string, error) {
	return file, nil
}

func (runner *fakePrecleanRunner) Run(_ context.Context, command execx.Command) (execx.Result, error) {
	if runner.runFunc == nil {
		return execx.Result{}, nil
	}
	return runner.runFunc(command)
}

func newStepPrecleanInstaller(t *testing.T, runner execx.Runner) *Installer {
	t.Helper()
	envPath := filepath.Join(t.TempDir(), ".env")
	return &Installer{
		options: cli.Options{
			Mode:        cli.ModeDev,
			ComposeFile: filepath.Join("docker", "docker-compose.dev.yml"),
			EnvFile:     envPath,
		},
		runner:  runner,
		printer: ui.NewPrinter(io.Discard, io.Discard),
		toolchain: docker.Toolchain{
			DockerBin:  "docker",
			ComposeCmd: []string{"docker", "compose"},
		},
	}
}

func isComposeDownCommand(command execx.Command) bool {
	if command.Name != "docker" {
		return false
	}
	if len(command.Args) < 3 {
		return false
	}
	if command.Args[0] != "compose" {
		return false
	}
	return containsArg(command.Args, "down") && containsArg(command.Args, "--remove-orphans")
}

func isListContainersCommand(command execx.Command) bool {
	if command.Name != "docker" {
		return false
	}
	return len(command.Args) >= 4 &&
		command.Args[0] == "ps" &&
		command.Args[1] == "-a" &&
		command.Args[2] == "--format"
}

func isRemoveContainersCommand(command execx.Command) bool {
	if command.Name != "docker" {
		return false
	}
	return len(command.Args) >= 3 &&
		command.Args[0] == "rm" &&
		command.Args[1] == "-f"
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}
