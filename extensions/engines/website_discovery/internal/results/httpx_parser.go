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

	websitediscoverycontract "github.com/yyhuni/lunafox/engines/website_discovery/contract"
)

type httpxLine struct {
	URL             string   `json:"url"`
	Input           string   `json:"input"`
	Host            string   `json:"host"`
	Title           string   `json:"title"`
	StatusCode      *int     `json:"status_code"`
	ContentLength   *int     `json:"content_length"`
	Location        string   `json:"location"`
	Webserver       string   `json:"webserver"`
	ContentType     string   `json:"content_type"`
	Tech            []string `json:"tech"`
	ResponseBody    string   `json:"body"`
	Vhost           *bool    `json:"vhost"`
	Failed          bool     `json:"failed"`
	ResponseHeaders string   `json:"raw_header"`
}

type ParseSummary struct {
	SourceRecords    int
	ParsedItems      int
	SkippedMalformed int
	SkippedInvalid   int
	SkippedFailed    int
	SkippedDuplicate int
	SkippedOversized int
	FatalIdentity    int
}

// StreamHTTPXWebsites reads the final HTTPX artifact incrementally and emits
// raw observed website candidates. Result reporting owns global exact-URL
// deduplication.
func StreamHTTPXWebsites(ctx context.Context, filePath string, submit func(websitediscoverycontract.Website) error) (int, error) {
	summary, err := StreamHTTPXWebsitesWithSummary(ctx, filePath, submit)
	return summary.ParsedItems, err
}

// StreamHTTPXWebsitesWithSummary reads the final HTTPX artifact and returns
// parse accounting alongside admitted raw Website candidates.
func StreamHTTPXWebsitesWithSummary(ctx context.Context, filePath string, submit func(websitediscoverycontract.Website) error) (ParseSummary, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if filePath == "" {
		return ParseSummary{}, fmt.Errorf("HTTPX result artifact path is required")
	}
	if submit == nil {
		return ParseSummary{}, fmt.Errorf("website submit callback is required")
	}
	summary := ParseSummary{}
	if err := streamHTTPXFile(ctx, filePath, submit, &summary); err != nil {
		return summary, err
	}
	return summary, nil
}

func streamHTTPXFile(ctx context.Context, path string, submit func(websitediscoverycontract.Website) error, summary *ParseSummary) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open httpx output %s: %w", path, err)
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
			return fmt.Errorf("read httpx output %s: %w", path, err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		item, skip, ok := parseHTTPXLineWithOutcome(line)
		if !ok {
			switch skip {
			case httpxSkipMalformed:
				summary.SkippedMalformed++
			case httpxSkipFailed:
				summary.SkippedFailed++
			case httpxSkipUnassociatedIdentity:
				summary.FatalIdentity++
				return fmt.Errorf("httpx output row has no candidate input identity")
			default:
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

func parseHTTPXLine(line string) (websitediscoverycontract.Website, bool) {
	item, _, ok := parseHTTPXLineWithOutcome(line)
	return item, ok
}

type httpxSkipReason int

const (
	httpxSkipInvalid httpxSkipReason = iota
	httpxSkipMalformed
	httpxSkipFailed
	httpxSkipUnassociatedIdentity
)

func parseHTTPXLineWithOutcome(line string) (websitediscoverycontract.Website, httpxSkipReason, bool) {
	// encoding/json accepts invalid UTF-8 and replaces it with U+FFFD. Reject
	// the raw record first so an observed URL can never be admitted after that
	// lossy conversion.
	if err := validateJSONText([]byte(line)); err != nil {
		return websitediscoverycontract.Website{}, httpxSkipMalformed, false
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return websitediscoverycontract.Website{}, httpxSkipInvalid, false
	}
	// HTTPX's `url` is a redirect/final observation. Only `input` can bind a
	// row to the candidate that the Engine was authorized to execute. A missing,
	// null, empty, or wrongly typed input therefore makes the row unassociable,
	// even when the row otherwise looks like a valid no-finding record.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(line), &fields); err != nil {
		return websitediscoverycontract.Website{}, httpxSkipMalformed, false
	}
	rawInput, exists := fields["input"]
	if !exists || strings.TrimSpace(string(rawInput)) == "null" {
		return websitediscoverycontract.Website{}, httpxSkipUnassociatedIdentity, false
	}
	var input string
	if err := json.Unmarshal(rawInput, &input); err != nil || input == "" {
		return websitediscoverycontract.Website{}, httpxSkipUnassociatedIdentity, false
	}
	var raw httpxLine
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return websitediscoverycontract.Website{}, httpxSkipMalformed, false
	}
	raw.Input = input
	if raw.Failed {
		return websitediscoverycontract.Website{}, httpxSkipFailed, false
	}
	item := websitediscoverycontract.Website{
		URL:             preferredHTTPXURL(raw),
		Host:            raw.Host,
		Title:           raw.Title,
		StatusCode:      raw.StatusCode,
		ContentLength:   raw.ContentLength,
		Location:        raw.Location,
		Webserver:       raw.Webserver,
		ContentType:     raw.ContentType,
		Tech:            raw.Tech,
		ResponseBody:    raw.ResponseBody,
		Vhost:           raw.Vhost,
		ResponseHeaders: raw.ResponseHeaders,
	}
	return item, httpxSkipInvalid, true
}

func preferredHTTPXURL(raw httpxLine) string {
	// HTTPX url may describe a redirect or final destination. Website identity
	// is the candidate it observed, so an output row without input cannot be
	// associated safely and must be rejected by the result contract.
	return raw.Input
}
