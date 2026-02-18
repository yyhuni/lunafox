package activity

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
)

// ansiRegex matches ANSI escape sequences (colors, cursor movement, etc.)
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// controlCharReplacer removes control characters in a single pass
var controlCharReplacer = strings.NewReplacer(
	"\x00", "", // NUL
	"\r", "", // CR
	"\b", "", // Backspace
	"\f", "", // Form feed
	"\v", "", // Vertical tab
)

type outputLineBuffer struct {
	logFile      *os.File
	activityName string
	streamName   string
	lineBuf      []byte
	truncated    bool
	discarding   bool
}

func newOutputLineBuffer(logFile *os.File, activityName, streamName string) *outputLineBuffer {
	return &outputLineBuffer{
		logFile:      logFile,
		activityName: activityName,
		streamName:   streamName,
		lineBuf:      make([]byte, 0, ScannerInitBufSize),
	}
}

func (buffer *outputLineBuffer) flushLine() {
	if len(buffer.lineBuf) == 0 && !buffer.truncated {
		return
	}

	line := cleanLine(string(buffer.lineBuf))
	if line != "" {
		if buffer.logFile != nil {
			_, _ = fmt.Fprintln(buffer.logFile, line)
		}
		pkg.Logger.Debug("Activity output",
			zap.String("activity", buffer.activityName),
			zap.String("stream", buffer.streamName),
			zap.String("line", line))
	}

	if buffer.truncated {
		pkg.Logger.Warn("Activity output line truncated",
			zap.String("activity", buffer.activityName),
			zap.String("stream", buffer.streamName),
			zap.Int("maxBytes", ScannerMaxBufSize))
	}

	buffer.lineBuf = buffer.lineBuf[:0]
	buffer.truncated = false
	buffer.discarding = false
}

func (buffer *outputLineBuffer) appendChunk(chunk []byte) {
	if len(chunk) == 0 || buffer.discarding {
		return
	}

	if len(buffer.lineBuf)+len(chunk) <= ScannerMaxBufSize {
		buffer.lineBuf = append(buffer.lineBuf, chunk...)
		return
	}

	remain := ScannerMaxBufSize - len(buffer.lineBuf)
	if remain > 0 {
		buffer.lineBuf = append(buffer.lineBuf, chunk[:remain]...)
	}
	buffer.truncated = true
	buffer.discarding = true
}

func (buffer *outputLineBuffer) consumeData(data []byte) {
	for len(data) > 0 {
		i := bytes.IndexAny(data, "\r\n")
		if i == -1 {
			buffer.appendChunk(data)
			return
		}

		buffer.appendChunk(data[:i])
		delim := data[i]
		data = data[i+1:]
		if delim == '\r' && len(data) > 0 && data[0] == '\n' {
			data = data[1:]
		}
		buffer.flushLine()
	}
}

func (r *Runner) streamOutput(wg *sync.WaitGroup, reader io.Reader, logFile *os.File, activityName, streamName string) {
	defer wg.Done()

	const readChunkSize = 8 * 1024
	readBuffer := make([]byte, readChunkSize)
	lineBuffer := newOutputLineBuffer(logFile, activityName, streamName)

	for {
		n, err := reader.Read(readBuffer)
		if n > 0 {
			lineBuffer.consumeData(readBuffer[:n])
		}

		if err != nil {
			if err != io.EOF {
				pkg.Logger.Warn("Error reading output stream",
					zap.String("activity", activityName),
					zap.String("stream", streamName),
					zap.Error(err))
			}
			lineBuffer.flushLine()
			return
		}
	}
}

// cleanLine removes ANSI escape sequences and control characters from output
func cleanLine(line string) string {
	line = ansiRegex.ReplaceAllString(line, "")
	line = controlCharReplacer.Replace(line)
	return strings.TrimSpace(line)
}
