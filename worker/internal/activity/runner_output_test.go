package activity

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type chunkedReader struct {
	chunks [][]byte
	index  int
}

func (reader *chunkedReader) Read(p []byte) (int, error) {
	if reader.index >= len(reader.chunks) {
		return 0, io.EOF
	}
	chunk := reader.chunks[reader.index]
	reader.index++
	n := copy(p, chunk)
	return n, nil
}

func streamToLogFile(t *testing.T, runner *Runner, reader io.Reader) string {
	t.Helper()

	logFile := filepath.Join(t.TempDir(), "stream.log")
	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("create log file failed: %v", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go runner.streamOutput(wg, reader, file, nil, "test", "stdout")
	wg.Wait()
	_ = file.Close()

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file failed: %v", err)
	}
	return string(data)
}

func linesFromLog(content string) []string {
	if content == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(content, "\n"), "\n")
}

func TestRunner_StreamOutput_CRLFAcrossReadBoundaries(t *testing.T) {
	withNopLogger(t)
	runner := NewRunner(t.TempDir())

	reader := &chunkedReader{
		chunks: [][]byte{
			[]byte("line-1\r"),
			[]byte("\nline-2\r"),
			[]byte("\nline-3"),
			[]byte("\r"),
			[]byte("\n"),
		},
	}

	content := streamToLogFile(t, runner, reader)
	lines := linesFromLog(content)

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), lines)
	}
	if lines[0] != "line-1" || lines[1] != "line-2" || lines[2] != "line-3" {
		t.Fatalf("unexpected lines: %q", lines)
	}
}

func TestRunner_StreamOutput_EOFFlushesLastLineWithoutNewline(t *testing.T) {
	withNopLogger(t)
	runner := NewRunner(t.TempDir())

	content := streamToLogFile(t, runner, strings.NewReader("tail-without-newline"))
	lines := linesFromLog(content)

	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %q", len(lines), lines)
	}
	if lines[0] != "tail-without-newline" {
		t.Fatalf("unexpected tail line: %q", lines[0])
	}
}

func TestRunner_StreamOutput_TruncationDoesNotAffectFollowingLine(t *testing.T) {
	withNopLogger(t)
	runner := NewRunner(t.TempDir())

	longLine := bytes.Repeat([]byte("a"), ScannerMaxBufSize+128)
	payload := append(longLine, '\n')
	payload = append(payload, []byte("ok-line\n")...)

	content := streamToLogFile(t, runner, bytes.NewReader(payload))
	lines := linesFromLog(content)

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if len(lines[0]) != ScannerMaxBufSize {
		t.Fatalf("expected truncated first line length %d, got %d", ScannerMaxBufSize, len(lines[0]))
	}
	if lines[1] != "ok-line" {
		t.Fatalf("expected second line ok-line, got %q", lines[1])
	}
}
