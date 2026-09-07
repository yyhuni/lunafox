package results

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	contractresults "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
	"github.com/yyhuni/lunafox/engines/subdomain_discovery/internal/validator"
)

type ParseSummary struct {
	SourceRecords    int
	ParsedItems      int
	SkippedMalformed int
	SkippedInvalid   int
	SkippedFailed    int
	SkippedDuplicate int
	SkippedOversized int
}

// StreamSubdomains reads the final result artifact incrementally and emits
// normalized candidates. Result reporting owns exact global deduplication.
func StreamSubdomains(ctx context.Context, filePath string, submit func(contractresults.Subdomain) error) (int, error) {
	summary, err := StreamSubdomainsWithSummary(ctx, filePath, submit)
	return summary.ParsedItems, err
}

// StreamSubdomainsWithSummary reads the final result artifact and returns parse
// accounting alongside normalized candidates.
func StreamSubdomainsWithSummary(ctx context.Context, filePath string, submit func(contractresults.Subdomain) error) (ParseSummary, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if filePath == "" {
		return ParseSummary{}, errors.New("subdomain result artifact path is required")
	}
	if submit == nil {
		return ParseSummary{}, errors.New("subdomain submit callback is required")
	}
	summary := ParseSummary{}
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	if err := streamSubdomainsFromFile(ctx, filePath, submit, &summary); err != nil {
		return summary, err
	}
	return summary, nil
}

func streamSubdomainsFromFile(ctx context.Context, filePath string, submit func(contractresults.Subdomain) error, summary *ParseSummary) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open subdomain result artifact %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

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
			return fmt.Errorf("read subdomain result artifact %s: %w", filePath, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		subdomain, ok := validator.NormalizeSubdomainLine(line)
		if !ok {
			summary.SkippedInvalid++
			continue
		}

		if err := ctx.Err(); err != nil {
			return err
		}
		if err := submit(contractresults.Subdomain{DNSName: subdomain}); err != nil {
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
