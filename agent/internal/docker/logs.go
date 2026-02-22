package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
)

const (
	maxErrorBytes   = 4096
	maxLogLineBytes = 16 * 1024
)

// StreamLogChunk is one parsed log line from Docker logs API.
type StreamLogChunk struct {
	TS        time.Time
	Stream    string
	Line      string
	Truncated bool
}

// StreamLogsOptions configures Docker logs streaming behavior.
type StreamLogsOptions struct {
	Container  string
	Tail       int
	Follow     bool
	Since      *time.Time
	Timestamps bool
}

// TailLogs returns the last N lines of container logs, truncated to 4KB.
func (c *Client) TailLogs(ctx context.Context, containerID string, lines int) (string, error) {
	reader, err := c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
		Tail:       strconv.Itoa(lines),
	})
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return "", err
	}

	out := buf.String()
	out = strings.TrimSpace(out)
	if len(out) > maxErrorBytes {
		out = out[len(out)-maxErrorBytes:]
	}
	return out, nil
}

// StreamLogs streams container logs line-by-line and demultiplexes stdout/stderr.
func (c *Client) StreamLogs(ctx context.Context, opts StreamLogsOptions, emit func(StreamLogChunk) error) error {
	if c == nil || c.cli == nil {
		return errors.New("docker client is unavailable")
	}
	if emit == nil {
		return errors.New("emit callback is required")
	}
	containerName := strings.TrimSpace(opts.Container)
	if containerName == "" {
		return errors.New("container is required")
	}

	tail := opts.Tail
	if tail < 0 {
		tail = 0
	}

	since := ""
	if opts.Since != nil {
		since = opts.Since.UTC().Format(time.RFC3339Nano)
	}

	reader, err := c.cli.ContainerLogs(ctx, containerName, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     opts.Follow,
		Tail:       strconv.Itoa(tail),
		Timestamps: opts.Timestamps,
		Since:      since,
	})
	if err != nil {
		return err
	}
	defer reader.Close()

	return consumeDockerLogs(reader, opts.Timestamps, emit)
}

func consumeDockerLogs(reader io.Reader, timestamps bool, emit func(StreamLogChunk) error) error {
	buffered := bufio.NewReader(reader)
	peek, err := buffered.Peek(8)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return err
	}

	if len(peek) == 8 && looksLikeDockerHeader(peek) {
		return consumeMultiplexed(buffered, timestamps, emit)
	}
	return consumePlain(buffered, timestamps, emit)
}

func looksLikeDockerHeader(header []byte) bool {
	if len(header) < 8 {
		return false
	}
	if header[0] != 1 && header[0] != 2 {
		return false
	}
	if header[1] != 0 || header[2] != 0 || header[3] != 0 {
		return false
	}
	size := binary.BigEndian.Uint32(header[4:8])
	return size <= 16*1024*1024
}

func consumeMultiplexed(reader *bufio.Reader, timestamps bool, emit func(StreamLogChunk) error) error {
	stdoutWriter := newLogLineWriter("stdout", timestamps, emit)
	stderrWriter := newLogLineWriter("stderr", timestamps, emit)

	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(reader, header); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return err
		}

		size := binary.BigEndian.Uint32(header[4:8])
		if size == 0 {
			continue
		}
		payload := make([]byte, size)
		if _, err := io.ReadFull(reader, payload); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return err
		}

		switch header[0] {
		case 1:
			if _, err := stdoutWriter.Write(payload); err != nil {
				return err
			}
		case 2:
			if _, err := stderrWriter.Write(payload); err != nil {
				return err
			}
		default:
			if _, err := stdoutWriter.Write(payload); err != nil {
				return err
			}
		}
	}

	if err := stdoutWriter.Flush(); err != nil {
		return err
	}
	if err := stderrWriter.Flush(); err != nil {
		return err
	}
	return nil
}

func consumePlain(reader *bufio.Reader, timestamps bool, emit func(StreamLogChunk) error) error {
	writer := newLogLineWriter("stdout", timestamps, emit)
	if _, err := io.Copy(writer, reader); err != nil {
		return err
	}
	return writer.Flush()
}

type logLineWriter struct {
	stream     string
	timestamps bool
	emit       func(StreamLogChunk) error
	buf        []byte
	truncated  bool
}

func newLogLineWriter(stream string, timestamps bool, emit func(StreamLogChunk) error) *logLineWriter {
	return &logLineWriter{stream: stream, timestamps: timestamps, emit: emit}
}

func (writer *logLineWriter) Write(p []byte) (int, error) {
	for _, b := range p {
		if b == '\n' {
			if err := writer.emitLine(); err != nil {
				return 0, err
			}
			writer.buf = writer.buf[:0]
			writer.truncated = false
			continue
		}
		if len(writer.buf) < maxLogLineBytes {
			writer.buf = append(writer.buf, b)
			continue
		}
		writer.truncated = true
	}
	return len(p), nil
}

func (writer *logLineWriter) Flush() error {
	if len(writer.buf) == 0 && !writer.truncated {
		return nil
	}
	if err := writer.emitLine(); err != nil {
		return err
	}
	writer.buf = writer.buf[:0]
	writer.truncated = false
	return nil
}

func (writer *logLineWriter) emitLine() error {
	ts := time.Now().UTC()
	line := string(bytes.TrimRight(writer.buf, "\r"))
	if writer.timestamps {
		parsedTS, trimmed := parseDockerTimestamp(line)
		if !parsedTS.IsZero() {
			ts = parsedTS
			line = trimmed
		}
	}

	chunk := StreamLogChunk{
		TS:        ts,
		Stream:    writer.stream,
		Line:      line,
		Truncated: writer.truncated,
	}
	if err := writer.emit(chunk); err != nil {
		return err
	}
	return nil
}

func parseDockerTimestamp(line string) (time.Time, string) {
	idx := strings.IndexByte(line, ' ')
	if idx <= 0 {
		return time.Time{}, line
	}
	rawTS := strings.TrimSpace(line[:idx])
	if rawTS == "" {
		return time.Time{}, line
	}
	parsed, err := time.Parse(time.RFC3339Nano, rawTS)
	if err != nil {
		return time.Time{}, line
	}
	return parsed.UTC(), strings.TrimLeft(line[idx+1:], " ")
}

// TruncateErrorMessage clamps message length to 4KB.
func TruncateErrorMessage(message string) string {
	if len(message) <= maxErrorBytes {
		return message
	}
	return message[:maxErrorBytes]
}

func IsContainerNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	return strings.Contains(msg, "no such container") || strings.Contains(msg, "container not found")
}
