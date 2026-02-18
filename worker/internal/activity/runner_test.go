package activity

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func withNopLogger(t *testing.T) {
	t.Helper()
	prev := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() {
		pkg.Logger = prev
	})
}

func TestRunner_Run_InvalidTimeout(t *testing.T) {
	withNopLogger(t)
	r := NewRunner(t.TempDir())

	res := r.Run(context.Background(), Command{
		Name:    "test",
		Command: "echo hi",
		Timeout: 0,
	})

	require.Error(t, res.Error)
	require.Equal(t, ExitCodeError, res.ExitCode)
}

func TestRunner_Run_ContextCanceled(t *testing.T) {
	withNopLogger(t)
	r := NewRunner(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := r.Run(ctx, Command{
		Name:    "test",
		Command: "echo hi",
		Timeout: time.Second,
	})

	require.Error(t, res.Error)
	require.Equal(t, ExitCodeError, res.ExitCode)
}

func TestRunner_Run_ExitCode(t *testing.T) {
	withNopLogger(t)
	r := NewRunner(t.TempDir())

	res := r.Run(context.Background(), Command{
		Name:    "test",
		Command: "exit 3",
		Timeout: time.Second,
	})

	require.Error(t, res.Error)
	require.Equal(t, 3, res.ExitCode)
}

func TestRunner_Run_Timeout(t *testing.T) {
	withNopLogger(t)
	r := NewRunner(t.TempDir())

	res := r.Run(context.Background(), Command{
		Name:    "test",
		Command: "sleep 0.2",
		Timeout: 50 * time.Millisecond,
	})

	require.Error(t, res.Error)
	require.Equal(t, ExitCodeTimeout, res.ExitCode)
}

func TestRunner_Run_WritesLogFile(t *testing.T) {
	withNopLogger(t)
	r := NewRunner(t.TempDir())
	logFile := filepath.Join(t.TempDir(), "run.log")

	res := r.Run(context.Background(), Command{
		Name:    "test",
		Command: "echo hi",
		LogFile: logFile,
		Timeout: time.Second,
	})

	require.NoError(t, res.Error)
	require.Equal(t, 0, res.ExitCode)

	data, err := os.ReadFile(logFile)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "# Tool: test")
	assert.Contains(t, content, "hi")
	assert.Contains(t, content, "# Status:")
}

func TestRunner_StreamOutputTruncatesLongLines(t *testing.T) {
	withNopLogger(t)
	r := NewRunner(t.TempDir())

	logFile := filepath.Join(t.TempDir(), "stream.log")
	f, err := os.Create(logFile)
	require.NoError(t, err)

	long := bytes.Repeat([]byte("a"), ScannerMaxBufSize+100)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go r.streamOutput(wg, bytes.NewReader(append(long, '\n')), f, "test", "stdout")
	wg.Wait()
	_ = f.Close()

	data, err := os.ReadFile(logFile)
	require.NoError(t, err)
	line := strings.SplitN(string(data), "\n", 2)[0]
	require.LessOrEqual(t, len(line), ScannerMaxBufSize)
}

func TestRunner_StreamOutputCleansLines(t *testing.T) {
	withNopLogger(t)
	r := NewRunner(t.TempDir())

	logFile := filepath.Join(t.TempDir(), "stream.log")
	f, err := os.Create(logFile)
	require.NoError(t, err)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go r.streamOutput(wg, strings.NewReader("hello\x1b[31m\r\nworld\x00\n"), f, "test", "stdout")
	wg.Wait()
	_ = f.Close()

	data, err := os.ReadFile(logFile)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "hello")
	assert.Contains(t, content, "world")
	assert.NotContains(t, content, "\x1b[")
}

func TestRunner_RunParallel_ContextCanceled(t *testing.T) {
	withNopLogger(t)
	r := NewRunner(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := r.RunParallel(ctx, []Command{
		{Name: "a", Command: "echo a", Timeout: time.Second},
		{Name: "b", Command: "echo b", Timeout: time.Second},
	})

	require.Len(t, results, 2)
	for _, res := range results {
		require.NotNil(t, res)
		require.Error(t, res.Error)
		require.Equal(t, ExitCodeError, res.ExitCode)
	}
}

func TestRunner_RunSequential_ContextCanceled(t *testing.T) {
	withNopLogger(t)
	r := NewRunner(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := r.RunSequential(ctx, []Command{
		{Name: "a", Command: "echo a", Timeout: time.Second},
		{Name: "b", Command: "echo b", Timeout: time.Second},
	})

	require.Len(t, results, 2)
	for _, res := range results {
		require.NotNil(t, res)
		require.Error(t, res.Error)
		require.Equal(t, ExitCodeError, res.ExitCode)
	}
}

func TestRunner_RunParallelEmpty(t *testing.T) {
	withNopLogger(t)
	r := NewRunner(t.TempDir())
	results := r.RunParallel(context.Background(), nil)
	require.Nil(t, results)
}

func TestRunner_RunSequentialEmpty(t *testing.T) {
	withNopLogger(t)
	r := NewRunner(t.TempDir())
	results := r.RunSequential(context.Background(), nil)
	require.Nil(t, results)
}

func TestCleanLineStripsANSIAndControl(t *testing.T) {
	withNopLogger(t)
	raw := "\x1b[31m hello\x1b[0m\r\x00"
	got := cleanLine(raw)
	require.Equal(t, "hello", got)
}

func TestGetMaxCmdConcurrencyInvalidEnv(t *testing.T) {
	withNopLogger(t)
	t.Setenv(EnvMaxCmdConcurrency, "0")
	require.Equal(t, DefaultMaxCmdConcurrency, getMaxCmdConcurrency())

	t.Setenv(EnvMaxCmdConcurrency, "bad")
	require.Equal(t, DefaultMaxCmdConcurrency, getMaxCmdConcurrency())
}

func TestGetMaxCmdConcurrencyValidEnv(t *testing.T) {
	withNopLogger(t)
	t.Setenv(EnvMaxCmdConcurrency, "3")
	require.Equal(t, 3, getMaxCmdConcurrency())
}

func TestRunner_RunParallel_RespectsMaxCmdConcurrency(t *testing.T) {
	require.NoError(t, pkg.InitLogger("error"))
	defer pkg.Sync()

	t.Setenv(EnvMaxCmdConcurrency, "1")

	workDir := t.TempDir()
	r := NewRunner(workDir)

	lockDir := filepath.Join(workDir, "lock")
	cmdStr := fmt.Sprintf("mkdir %q || exit 99; sleep 0.3; rmdir %q", lockDir, lockDir)

	cmds := []Command{
		{Name: "c1", Command: cmdStr, Timeout: 5 * time.Second},
		{Name: "c2", Command: cmdStr, Timeout: 5 * time.Second},
		{Name: "c3", Command: cmdStr, Timeout: 5 * time.Second},
	}

	start := time.Now()
	results := r.RunParallel(context.Background(), cmds)
	elapsed := time.Since(start)

	require.Len(t, results, len(cmds))
	for i, res := range results {
		require.NotNil(t, res, "result %d should not be nil", i)
		require.NoError(t, res.Error)
		require.Equal(t, 0, res.ExitCode)
	}

	// With concurrency=1 and 3 commands sleeping 0.3s each, total wall time should be ~0.9s.
	require.GreaterOrEqual(t, elapsed, 750*time.Millisecond)
}
