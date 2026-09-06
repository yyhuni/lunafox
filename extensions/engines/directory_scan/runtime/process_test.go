package directoryscanruntime

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestContainerFFUFProcessRunnerWritesOrdinalArtifactWithExactArgv(t *testing.T) {
	workspace := t.TempDir()
	process := &fakeFFUFProcess{stdout: io.NopCloser(strings.NewReader("record\nincomplete-tail"))}
	var gotBinary string
	var gotArgs []string
	runner := newContainerFFUFProcessRunner()
	runner.stderr = io.Discard
	runner.newProcess = func(_ context.Context, binary string, args []string, stderr io.Writer) ffufProcess {
		gotBinary = binary
		gotArgs = append([]string(nil), args...)
		if stderr != io.Discard {
			t.Fatalf("stderr writer = %#v, want io.Discard", stderr)
		}
		return process
	}
	config := defaultFFUFConfig()
	candidate := WebsiteCandidate{Ordinal: 12, Website: "https://example.com/root;literal"}
	if err := runner.Run(context.Background(), ffufInvocation{Candidate: candidate, Config: config, Workspace: workspace}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if gotBinary != FFUFBinary || valueAfter(gotArgs, "-u") != candidate.Website+"FUZZ" {
		t.Fatalf("process binary=%q args=%#v", gotBinary, gotArgs)
	}
	path, err := rawFFUFArtifactPath(workspace, candidate.Ordinal)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "record\nincomplete-tail" {
		t.Fatalf("raw artifact = %q", payload)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("raw artifact mode = %v, want 0600", info.Mode().Perm())
	}
	if process.waitCalls.Load() != 1 || process.killCalls.Load() != 0 {
		t.Fatalf("process wait=%d kill=%d", process.waitCalls.Load(), process.killCalls.Load())
	}

	if err := runner.Run(context.Background(), ffufInvocation{Candidate: candidate, Config: config, Workspace: workspace}); !errors.Is(err, os.ErrExist) {
		t.Fatalf("second Run() error = %v, want exclusive-create collision", err)
	}
}

func TestContainerFFUFProcessRunnerCopiesStdoutBeforeWaitAndClosesAfterWait(t *testing.T) {
	releaseEOF := make(chan struct{})
	reader := &stagedReader{first: []byte("complete-record\n"), releaseEOF: releaseEOF}
	var eventsMu sync.Mutex
	var events []string
	artifact := &fakeWriteCloser{onWrite: func() {
		eventsMu.Lock()
		events = append(events, "write")
		eventsMu.Unlock()
	}, onClose: func() {
		eventsMu.Lock()
		events = append(events, "close")
		eventsMu.Unlock()
	}, wrote: make(chan struct{})}
	process := &fakeFFUFProcess{stdout: io.NopCloser(reader), onWait: func() {
		eventsMu.Lock()
		events = append(events, "wait")
		eventsMu.Unlock()
	}}
	runner := testContainerFFUFRunner(process, func(string) (io.WriteCloser, error) { return artifact, nil })
	workspace := t.TempDir()
	result := make(chan error, 1)
	go func() {
		result <- runner.Run(context.Background(), ffufInvocation{
			Candidate: WebsiteCandidate{Ordinal: 0, Website: "https://example.com"},
			Config:    defaultFFUFConfig(),
			Workspace: workspace,
		})
	}()

	select {
	case <-artifact.wrote:
	case <-time.After(time.Second):
		t.Fatal("stdout was not copied while FFUF remained active")
	}
	if process.waitCalls.Load() != 0 {
		t.Fatal("process Wait ran before stdout reached the raw artifact")
	}
	close(releaseEOF)
	if err := <-result; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	eventsMu.Lock()
	defer eventsMu.Unlock()
	if !reflect.DeepEqual(events, []string{"write", "wait", "close"}) {
		t.Fatalf("process events = %#v", events)
	}
}

func TestContainerFFUFProcessRunnerPropagatesProcessAndArtifactFailures(t *testing.T) {
	wantOpenErr := errors.New("open failed")
	runner := testContainerFFUFRunner(&fakeFFUFProcess{stdout: io.NopCloser(strings.NewReader(""))}, func(string) (io.WriteCloser, error) {
		return nil, wantOpenErr
	})
	if err := runTestFFUFProcess(t, runner); !errors.Is(err, wantOpenErr) || !isFFUFSharedOperationError(err) {
		t.Fatalf("artifact open error = %v", err)
	}

	t.Run("stdout pipe", func(t *testing.T) {
		wantErr := errors.New("pipe failed")
		artifact := &fakeWriteCloser{}
		process := &fakeFFUFProcess{stdoutErr: wantErr}
		err := runTestFFUFProcess(t, testContainerFFUFRunner(process, func(string) (io.WriteCloser, error) { return artifact, nil }))
		if !errors.Is(err, wantErr) || !isFFUFSharedOperationError(err) || !artifact.closed.Load() {
			t.Fatalf("stdout pipe error = %v closed=%t", err, artifact.closed.Load())
		}
	})

	t.Run("start", func(t *testing.T) {
		wantErr := errors.New("start failed")
		artifact := &fakeWriteCloser{}
		process := &fakeFFUFProcess{stdout: io.NopCloser(strings.NewReader("")), startErr: wantErr}
		err := runTestFFUFProcess(t, testContainerFFUFRunner(process, func(string) (io.WriteCloser, error) { return artifact, nil }))
		if !errors.Is(err, wantErr) || isFFUFSharedOperationError(err) || !artifact.closed.Load() {
			t.Fatalf("start error = %v closed=%t", err, artifact.closed.Load())
		}
	})

	t.Run("stdout read", func(t *testing.T) {
		wantErr := errors.New("read failed")
		process := &fakeFFUFProcess{stdout: io.NopCloser(&readThenError{payload: []byte("partial"), err: wantErr})}
		err := runTestFFUFProcess(t, testContainerFFUFRunner(process, func(string) (io.WriteCloser, error) { return &fakeWriteCloser{}, nil }))
		if !errors.Is(err, wantErr) || !isFFUFSharedOperationError(err) || process.killCalls.Load() != 1 || process.waitCalls.Load() != 1 {
			t.Fatalf("stdout read error = %v kill=%d wait=%d", err, process.killCalls.Load(), process.waitCalls.Load())
		}
	})

	t.Run("artifact write", func(t *testing.T) {
		wantErr := errors.New("write failed")
		process := &fakeFFUFProcess{stdout: io.NopCloser(strings.NewReader("record"))}
		err := runTestFFUFProcess(t, testContainerFFUFRunner(process, func(string) (io.WriteCloser, error) {
			return &fakeWriteCloser{writeErr: wantErr}, nil
		}))
		if !errors.Is(err, wantErr) || !isFFUFSharedOperationError(err) || process.killCalls.Load() != 1 || process.waitCalls.Load() != 1 {
			t.Fatalf("artifact write error = %v kill=%d wait=%d", err, process.killCalls.Load(), process.waitCalls.Load())
		}
	})

	t.Run("non-zero wait", func(t *testing.T) {
		wantErr := errors.New("exit code 7")
		process := &fakeFFUFProcess{stdout: io.NopCloser(strings.NewReader("record\n")), waitErr: wantErr}
		err := runTestFFUFProcess(t, testContainerFFUFRunner(process, func(string) (io.WriteCloser, error) { return &fakeWriteCloser{}, nil }))
		if !errors.Is(err, wantErr) || isFFUFSharedOperationError(err) {
			t.Fatalf("wait error = %v", err)
		}
	})

	t.Run("artifact close", func(t *testing.T) {
		wantErr := errors.New("close failed")
		process := &fakeFFUFProcess{stdout: io.NopCloser(strings.NewReader("record\n"))}
		err := runTestFFUFProcess(t, testContainerFFUFRunner(process, func(string) (io.WriteCloser, error) {
			return &fakeWriteCloser{closeErr: wantErr}, nil
		}))
		if !errors.Is(err, wantErr) || !isFFUFSharedOperationError(err) {
			t.Fatalf("close error = %v", err)
		}
	})
}

func TestContainerFFUFProcessRunnerRejectsCanceledContextAndInvalidWorkspace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := atomic.Bool{}
	runner := newContainerFFUFProcessRunner()
	runner.stderr = io.Discard
	runner.newProcess = func(context.Context, string, []string, io.Writer) ffufProcess {
		called.Store(true)
		return nil
	}
	err := runner.Run(ctx, ffufInvocation{Candidate: WebsiteCandidate{Website: "https://example.com"}, Config: defaultFFUFConfig(), Workspace: t.TempDir()})
	if !errors.Is(err, context.Canceled) || called.Load() {
		t.Fatalf("canceled Run() error=%v processCalled=%t", err, called.Load())
	}

	for _, workspace := range []string{"", "relative", t.TempDir() + string(filepath.Separator) + "."} {
		if _, err := rawFFUFArtifactPath(workspace, 0); err == nil {
			t.Fatalf("rawFFUFArtifactPath(%q) accepted invalid workspace", workspace)
		}
	}
}

type fakeFFUFProcess struct {
	stdout    io.ReadCloser
	stdoutErr error
	startErr  error
	waitErr   error
	onWait    func()
	killCalls atomic.Int32
	waitCalls atomic.Int32
}

func (process *fakeFFUFProcess) StdoutPipe() (io.ReadCloser, error) {
	return process.stdout, process.stdoutErr
}

func (process *fakeFFUFProcess) Start() error { return process.startErr }

func (process *fakeFFUFProcess) Wait() error {
	process.waitCalls.Add(1)
	if process.onWait != nil {
		process.onWait()
	}
	return process.waitErr
}

func (process *fakeFFUFProcess) Kill() error {
	process.killCalls.Add(1)
	return nil
}

type fakeWriteCloser struct {
	buffer   bytes.Buffer
	writeErr error
	closeErr error
	onWrite  func()
	onClose  func()
	wrote    chan struct{}
	once     sync.Once
	closed   atomic.Bool
}

func (writer *fakeWriteCloser) Write(payload []byte) (int, error) {
	if writer.writeErr != nil {
		return 0, writer.writeErr
	}
	written, err := writer.buffer.Write(payload)
	writer.once.Do(func() {
		if writer.wrote != nil {
			close(writer.wrote)
		}
		if writer.onWrite != nil {
			writer.onWrite()
		}
	})
	return written, err
}

func (writer *fakeWriteCloser) Close() error {
	writer.closed.Store(true)
	if writer.onClose != nil {
		writer.onClose()
	}
	return writer.closeErr
}

type stagedReader struct {
	first      []byte
	releaseEOF <-chan struct{}
	sent       bool
}

func (reader *stagedReader) Read(buffer []byte) (int, error) {
	if !reader.sent {
		reader.sent = true
		return copy(buffer, reader.first), nil
	}
	<-reader.releaseEOF
	return 0, io.EOF
}

type readThenError struct {
	payload []byte
	err     error
	sent    bool
}

func (reader *readThenError) Read(buffer []byte) (int, error) {
	if !reader.sent {
		reader.sent = true
		return copy(buffer, reader.payload), nil
	}
	return 0, reader.err
}

func testContainerFFUFRunner(process ffufProcess, opener rawArtifactOpener) containerFFUFProcessRunner {
	return containerFFUFProcessRunner{
		binary: FFUFBinary,
		newProcess: func(context.Context, string, []string, io.Writer) ffufProcess {
			return process
		},
		openArtifact: opener,
		stderr:       io.Discard,
	}
}

func runTestFFUFProcess(t *testing.T, runner containerFFUFProcessRunner) error {
	t.Helper()
	return runner.Run(context.Background(), ffufInvocation{
		Candidate: WebsiteCandidate{Ordinal: 0, Website: "https://example.com"},
		Config:    defaultFFUFConfig(),
		Workspace: t.TempDir(),
	})
}
