package directoryscanruntime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

type ffufInvocation struct {
	Candidate WebsiteCandidate
	Config    enginecontract.FfufConfig
	Workspace string
}

type ffufProcessRunner interface {
	Run(context.Context, ffufInvocation) error
}

type ffufProcess interface {
	StdoutPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
	Kill() error
}

type ffufProcessFactory func(context.Context, string, []string, io.Writer) ffufProcess
type rawArtifactOpener func(string) (io.WriteCloser, error)

type ffufSharedOperationError struct {
	err error
}

func (failure *ffufSharedOperationError) Error() string { return failure.err.Error() }

func (failure *ffufSharedOperationError) Unwrap() error { return failure.err }

type containerFFUFProcessRunner struct {
	binary       string
	newProcess   ffufProcessFactory
	openArtifact rawArtifactOpener
	stderr       io.Writer
}

type execFFUFProcess struct {
	command *exec.Cmd
}

func (process *execFFUFProcess) StdoutPipe() (io.ReadCloser, error) {
	return process.command.StdoutPipe()
}

func (process *execFFUFProcess) Start() error { return process.command.Start() }

func (process *execFFUFProcess) Wait() error { return process.command.Wait() }

func (process *execFFUFProcess) Kill() error {
	if process.command.Process == nil {
		return nil
	}
	return process.command.Process.Kill()
}

func newContainerFFUFProcessRunner() containerFFUFProcessRunner {
	return containerFFUFProcessRunner{
		binary:       FFUFBinary,
		newProcess:   newExecFFUFProcess,
		openArtifact: openRawFFUFArtifact,
		stderr:       os.Stderr,
	}
}

func newExecFFUFProcess(ctx context.Context, binary string, args []string, stderr io.Writer) ffufProcess {
	command := exec.CommandContext(ctx, binary, args...)
	command.Stderr = stderr
	return &execFFUFProcess{command: command}
}

// Run owns one direct FFUF child and its ordinal-named raw artifact. Stdout is
// copied while the process runs so deadline termination cannot erase complete
// records written before the child was killed.
func (runner containerFFUFProcessRunner) Run(ctx context.Context, invocation ffufInvocation) error {
	if ctx == nil {
		return errors.New("FFUF process context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if runner.binary == "" || runner.newProcess == nil || runner.openArtifact == nil || runner.stderr == nil {
		return errors.New("FFUF process runner is incomplete")
	}
	args, err := BuildFFUFArgs(invocation.Candidate.Website, invocation.Config)
	if err != nil {
		return err
	}
	artifactPath, err := rawFFUFArtifactPath(invocation.Workspace, invocation.Candidate.Ordinal)
	if err != nil {
		return err
	}
	artifact, err := runner.openArtifact(artifactPath)
	if err != nil {
		return newFFUFSharedOperationError("create FFUF raw artifact", err)
	}
	closeArtifact := func(primary error) error {
		if closeErr := artifact.Close(); closeErr != nil {
			return errors.Join(primary, newFFUFSharedOperationError("close FFUF raw artifact", closeErr))
		}
		return primary
	}

	process := runner.newProcess(ctx, runner.binary, args, runner.stderr)
	if process == nil {
		return closeArtifact(newFFUFSharedOperationError("create FFUF process", errors.New("process factory returned nil")))
	}
	stdout, err := process.StdoutPipe()
	if err != nil {
		return closeArtifact(newFFUFSharedOperationError("open FFUF stdout", err))
	}
	if err := process.Start(); err != nil {
		startErr := fmt.Errorf("start FFUF: %w", err)
		if closeErr := stdout.Close(); closeErr != nil {
			startErr = errors.Join(startErr, newFFUFSharedOperationError("close FFUF stdout", closeErr))
		}
		return closeArtifact(startErr)
	}
	if _, err := io.Copy(artifact, stdout); err != nil {
		_ = process.Kill()
		_ = process.Wait()
		return closeArtifact(newFFUFSharedOperationError("copy FFUF stdout", err))
	}
	waitErr := process.Wait()
	if waitErr != nil {
		waitErr = fmt.Errorf("wait for FFUF: %w", waitErr)
	}
	return closeArtifact(waitErr)
}

func newFFUFSharedOperationError(operation string, err error) error {
	return &ffufSharedOperationError{err: fmt.Errorf("%s: %w", operation, err)}
}

func isFFUFSharedOperationError(err error) bool {
	var failure *ffufSharedOperationError
	return errors.As(err, &failure)
}

func rawFFUFArtifactPath(workspace string, ordinal uint64) (string, error) {
	workspacePath, err := requireDirectoryWorkspace(workspace)
	if err != nil {
		return "", err
	}
	return filepath.Join(workspacePath, fmt.Sprintf("ffuf-%020d.jsonl", ordinal)), nil
}

func openRawFFUFArtifact(path string) (io.WriteCloser, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
}

func requireDirectoryWorkspace(workspace string) (string, error) {
	if workspace == "" {
		return "", errors.New("Directory workspace is required")
	}
	if workspace != strings.TrimSpace(workspace) {
		return "", errors.New("Directory workspace must not have surrounding whitespace")
	}
	if !filepath.IsAbs(workspace) {
		return "", errors.New("Directory workspace must be absolute")
	}
	if filepath.Clean(workspace) != workspace {
		return "", errors.New("Directory workspace must be clean")
	}
	info, err := os.Stat(workspace)
	if err != nil {
		return "", fmt.Errorf("inspect Directory workspace: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("Directory workspace must be an existing directory")
	}
	return workspace, nil
}
