package results

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	contractresults "github.com/yyhuni/lunafox/engines/port_scan/contract"
)

type naabuLine struct {
	Host string `json:"host"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

type ParseSummary struct {
	SourceRecords    int
	ParsedItems      int
	SkippedMalformed int
	SkippedInvalid   int
	SkippedFailed    int
	SkippedDuplicate int
	SkippedOversized int
}

func StreamNaabuHostPorts(ctx context.Context, filePaths []string, submit func(contractresults.HostPort) error) (int, error) {
	summary, err := StreamNaabuHostPortsWithSummary(ctx, filePaths, submit)
	return summary.ParsedItems, err
}

func StreamNaabuHostPortsWithSummary(ctx context.Context, filePaths []string, submit func(contractresults.HostPort) error) (ParseSummary, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if submit == nil {
		return ParseSummary{}, fmt.Errorf("host-port submit callback is required")
	}
	summary := ParseSummary{}
	for _, filePath := range filePaths {
		path := strings.TrimSpace(filePath)
		if path == "" {
			continue
		}
		err := streamNaabuFile(ctx, path, submit, &summary)
		if err != nil {
			return summary, err
		}
	}
	return summary, nil
}

func streamNaabuFile(ctx context.Context, path string, submit func(contractresults.HostPort) error, summary *ParseSummary) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open naabu output %s: %w", path, err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, oversized, err := readResultArtifactLine(reader)
		if errors.Is(err, io.EOF) {
			break
		}
		summary.SourceRecords++
		if oversized {
			summary.SkippedOversized++
			continue
		}
		if err != nil {
			return fmt.Errorf("read naabu output %s: %w", path, err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		item, malformed, ok := parseNaabuLine(line)
		if !ok {
			if malformed {
				summary.SkippedMalformed++
			} else {
				summary.SkippedInvalid++
			}
			continue
		}
		if err := submit(item); err != nil {
			return err
		}
		summary.ParsedItems++
	}
	return nil
}

const maxResultArtifactRecordBytes = 4 * 1024 * 1024

// readResultArtifactLine remains local because scanner artifact parsing belongs
// to this Engine rather than to a shared engine-author layer.
func readResultArtifactLine(reader *bufio.Reader) (string, bool, error) {
	line := make([]byte, 0, min(maxResultArtifactRecordBytes, 64*1024))
	overLimit := false
	for {
		fragment, isPrefix, err := reader.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) && overLimit {
				return "", true, nil
			}
			return "", false, err
		}
		if !overLimit {
			if len(fragment) > maxResultArtifactRecordBytes-len(line) {
				overLimit = true
			} else {
				line = append(line, fragment...)
			}
		}
		if !isPrefix {
			if overLimit {
				return "", true, nil
			}
			return string(line), false, nil
		}
	}
}

func parseNaabuLine(line string) (contractresults.HostPort, bool, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return contractresults.HostPort{}, false, false
	}
	var raw naabuLine
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return contractresults.HostPort{}, true, false
	}
	item, ok := normalizeNaabuHostPort(raw)
	if !ok {
		return contractresults.HostPort{}, false, false
	}
	return item, false, true
}
