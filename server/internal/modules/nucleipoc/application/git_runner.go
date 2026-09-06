package application

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	defaultSyncDeadline = 2 * time.Hour
	processKillGrace    = 10 * time.Second
	maxGitDiagnostic    = 4096
)

type ProcessGitRunner struct {
	Deadline time.Duration
}

func NewProcessGitRunner() *ProcessGitRunner { return &ProcessGitRunner{Deadline: defaultSyncDeadline} }

func (runner *ProcessGitRunner) Clone(ctx context.Context, repoURL, destination string) (string, error) {
	return runner.clone(ctx, repoURL, destination)
}

func gitCloneArgs(repoURL, destination string) []string {
	return []string{"clone", "--no-recurse-submodules", "--depth", "1", "--", repoURL, destination}
}

func (runner *ProcessGitRunner) clone(ctx context.Context, repoURL, destination string) (string, error) {
	if runner == nil {
		return "", fmt.Errorf("git runner is not configured")
	}
	deadline := runner.Deadline
	if deadline <= 0 {
		deadline = defaultSyncDeadline
	}
	cloneCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	args := gitCloneArgs(repoURL, destination)
	if err := runControlledGit(cloneCtx, args...); err != nil {
		return "", fmt.Errorf("git clone failed")
	}
	var output bytes.Buffer
	lookupArgs := []string{"-C", destination, "rev-parse", "HEAD"}
	if err := runControlledGitOutput(cloneCtx, &output, lookupArgs...); err != nil {
		return "", fmt.Errorf("git commit lookup failed")
	}
	commit := strings.TrimSpace(output.String())
	if len(commit) != 40 {
		return "", fmt.Errorf("git commit identity is invalid")
	}
	return commit, nil
}

type boundedWriter struct {
	buffer bytes.Buffer
	limit  int
}

func (writer *boundedWriter) Write(value []byte) (int, error) {
	originalLength := len(value)
	if writer.limit <= 0 {
		writer.limit = maxGitDiagnostic
	}
	remaining := writer.limit - writer.buffer.Len()
	if remaining > 0 {
		if len(value) > remaining {
			value = value[:remaining]
		}
		_, _ = writer.buffer.Write(value)
	}
	// Report the full accepted length even when the diagnostic buffer is full;
	// otherwise exec.Cmd observes a short write and may turn normal Git output
	// into a misleading process failure.
	return originalLength, nil
}

func runControlledGit(ctx context.Context, args ...string) error {
	var output boundedWriter
	return runControlledGitOutput(ctx, &output, args...)
}

func runControlledGitOutput(ctx context.Context, output io.Writer, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("git arguments are required")
	}
	command := exec.Command("git", args...)
	command.Env = safeGitEnvironment()
	command.Stdout = output
	command.Stderr = output
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		terminateProcessGroup(command.Process)
		select {
		case err := <-done:
			waitForProcessGroupExit(command.Process.Pid)
			return err
		case <-time.After(processKillGrace):
			killProcessGroup(command.Process)
			waitErr := <-done
			// Wait() reaps the direct child. Confirm the process group no
			// longer exists before releasing the task's single-active slot;
			// descendants must not survive into a later sync.
			waitForProcessGroupExit(command.Process.Pid)
			return waitErr
		}
	}
}

func terminateProcessGroup(process *os.Process) {
	if process == nil {
		return
	}
	_ = syscall.Kill(-process.Pid, syscall.SIGTERM)
}
func killProcessGroup(process *os.Process) {
	if process == nil {
		return
	}
	_ = syscall.Kill(-process.Pid, syscall.SIGKILL)
}

func waitForProcessGroupExit(pid int) {
	if pid <= 0 {
		return
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(-pid, 0); err != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func safeGitEnvironment() []string {
	blocked := map[string]struct{}{
		"GIT_PROXY_COMMAND": {}, "GIT_SSH_COMMAND": {}, "GIT_ASKPASS": {}, "SSH_ASKPASS": {},
		"GIT_DIR": {}, "GIT_WORK_TREE": {}, "GIT_INDEX_FILE": {}, "GIT_OBJECT_DIRECTORY": {}, "GIT_ALTERNATE_OBJECT_DIRECTORIES": {},
		"GIT_TEMPLATE_DIR": {}, "GIT_CONFIG": {}, "GIT_CONFIG_GLOBAL": {}, "GIT_CONFIG_SYSTEM": {}, "GIT_CONFIG_NOSYSTEM": {},
		"GIT_CEILING_DIRECTORIES": {}, "GIT_DISCOVERY_ACROSS_FILESYSTEM": {}, "GIT_EXTENSIONS": {}, "GIT_ALLOW_PROTOCOL": {},
		"GIT_SSL_NO_VERIFY": {}, "GIT_HTTP_PROXY_AUTHMETHOD": {}, "GIT_PROTOCOL_FROM_USER": {},
		"GIT_HTTP_LOW_SPEED_LIMIT": {}, "GIT_HTTP_LOW_SPEED_TIME": {},
		"GIT_TRACE": {}, "GIT_TRACE_PACKET": {}, "GIT_TRACE_CURL": {}, "GIT_CURL_VERBOSE": {},
	}
	output := make([]string, 0, len(os.Environ())+3)
	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, blockedKey := blocked[key]; !blockedKey && !strings.HasPrefix(strings.ToUpper(key), "GIT_CONFIG_") {
				output = append(output, entry)
			}
		}
	}
	// Disable global/system Git configuration and every interactive helper;
	// repository URLs are untrusted input and must not select a remote helper,
	// credential prompt, SSH command, or TLS override from the server environment.
	output = append(output,
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_HTTP_LOW_SPEED_LIMIT=1024",
		"GIT_HTTP_LOW_SPEED_TIME=30",
	)
	return output
}
