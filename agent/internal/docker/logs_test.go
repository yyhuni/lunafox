package docker

import (
	"strings"
	"testing"
	"time"
)

func TestTruncateErrorMessage(t *testing.T) {
	short := "short message"
	if got := TruncateErrorMessage(short); got != short {
		t.Fatalf("expected message to stay unchanged")
	}

	long := strings.Repeat("x", maxErrorBytes+10)
	got := TruncateErrorMessage(long)
	if len(got) != maxErrorBytes {
		t.Fatalf("expected length %d, got %d", maxErrorBytes, len(got))
	}
	if got != long[:maxErrorBytes] {
		t.Fatalf("unexpected truncation result")
	}
}

func TestLogLineWriterTruncatesLongLine(t *testing.T) {
	var chunks []StreamLogChunk
	writer := newLogLineWriter("stdout", false, func(chunk StreamLogChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})

	line := strings.Repeat("a", maxLogLineBytes+128) + "\n"
	if _, err := writer.Write([]byte(line)); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected one chunk, got %d", len(chunks))
	}
	if !chunks[0].Truncated {
		t.Fatalf("expected truncated chunk")
	}
	if len(chunks[0].Line) != maxLogLineBytes {
		t.Fatalf("expected line length %d, got %d", maxLogLineBytes, len(chunks[0].Line))
	}
}

func TestParseDockerTimestamp(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	raw := now.Format(time.RFC3339Nano) + " hello world"
	parsed, line := parseDockerTimestamp(raw)
	if parsed.IsZero() {
		t.Fatalf("expected parsed timestamp")
	}
	if parsed.Location() != time.UTC {
		t.Fatalf("expected UTC timestamp")
	}
	if line != "hello world" {
		t.Fatalf("unexpected line content: %q", line)
	}

	parsed, line = parseDockerTimestamp("not-a-ts hello")
	if !parsed.IsZero() {
		t.Fatalf("expected zero timestamp for invalid input")
	}
	if line != "not-a-ts hello" {
		t.Fatalf("expected original line on parse failure")
	}
}
