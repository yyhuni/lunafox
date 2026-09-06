package urlcollectionruntime

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"strings"

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

const maxRecordBytes = 4 * 1024 * 1024

type SkipReason string

const (
	SkipMalformedJSON        SkipReason = "skippedMalformedJSON"
	SkipInvalidFieldType     SkipReason = "skippedInvalidFieldType"
	SkipHTTPXFailed          SkipReason = "skippedHTTPXFailed"
	SkipInvalidURL           SkipReason = "skippedInvalidURL"
	SkipUnassociatedIdentity SkipReason = "unassociatedCandidateIdentity"
	SkipHostInconsistent     SkipReason = "skippedHostInconsistent"
	SkipOversizedRecord      SkipReason = "skippedOversizedRecord"
	SkipOversizedURL         SkipReason = "skippedOversized"
)

type HTTPXOutcome struct {
	Endpoint enginecontract.Endpoint
	Skip     SkipReason
	// Structural is true after JSON syntax and field types have been accepted.
	// Rows rejected later still prove that the tool emitted a parseable record;
	// otherwise an all-failed probe output would be misreported as corrupt.
	Structural        bool
	RecoveredMetadata bool
}

type httpxRecord struct {
	Input         string   `json:"input"`
	Host          string   `json:"host"`
	Failed        bool     `json:"failed"`
	Title         string   `json:"title"`
	StatusCode    *int     `json:"status_code"`
	ContentLength *int     `json:"content_length"`
	Location      string   `json:"location"`
	Webserver     string   `json:"webserver"`
	ContentType   string   `json:"content_type"`
	Tech          []string `json:"tech"`
	Body          string   `json:"body"`
	RawHeader     string   `json:"raw_header"`
	Vhost         *bool    `json:"vhost"`
}

// ReadBoundedLines keeps newline alignment after an oversized tool record.
// Trusted seed input uses the returned oversize marker as a terminal error;
// untrusted tool output can count and skip it.
func ReadBoundedLines(path string, visit func(string, bool) error) error {
	if path == "" || visit == nil {
		return fmt.Errorf("bounded line reader requires path and visitor")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	for {
		line, oversized, err := readBoundedLine(reader)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := visit(line, oversized); err != nil {
			return err
		}
	}
}

func readBoundedLine(reader *bufio.Reader) (string, bool, error) {
	line := make([]byte, 0, min(maxRecordBytes, 64*1024))
	over := false
	for {
		fragment, prefix, err := reader.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) && over {
				return "", true, nil
			}
			return "", false, err
		}
		if !over {
			if len(fragment) > maxRecordBytes-len(line) {
				over = true
			} else {
				line = append(line, fragment...)
			}
		}
		if !prefix {
			if over {
				return "", true, nil
			}
			return string(line), false, nil
		}
	}
}

// ParseHTTPXRecord preserves the Collector input as Endpoint identity. The
// HTTPX host is evidence only and never supplies a redirect or replacement URL.
func ParseHTTPXRecord(line string) HTTPXOutcome {
	// encoding/json replaces invalid UTF-8 in strings with U+FFFD. Treat the
	// complete raw row as malformed before decoding to preserve URL rejection
	// semantics at this tool boundary.
	if err := validateJSONText([]byte(line)); err != nil {
		return HTTPXOutcome{Skip: SkipMalformedJSON}
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return HTTPXOutcome{Skip: SkipMalformedJSON}
	}
	if !json.Valid([]byte(line)) {
		return HTTPXOutcome{Skip: SkipMalformedJSON}
	}
	input, associated, err := extractHTTPXInput([]byte(line))
	if err != nil {
		return HTTPXOutcome{Skip: SkipMalformedJSON}
	}
	if !associated {
		return HTTPXOutcome{Skip: SkipUnassociatedIdentity}
	}
	if err := validateHTTPXFieldTypes([]byte(line)); err != nil {
		return HTTPXOutcome{Skip: SkipInvalidFieldType}
	}
	var raw httpxRecord
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			return HTTPXOutcome{Skip: SkipInvalidFieldType}
		}
		return HTTPXOutcome{Skip: SkipMalformedJSON}
	}
	if raw.Failed {
		return HTTPXOutcome{Skip: SkipHTTPXFailed, Structural: true}
	}
	raw.Input = input
	item := enginecontract.Endpoint{
		URL: raw.Input, Host: raw.Host, Title: raw.Title, StatusCode: raw.StatusCode, ContentLength: raw.ContentLength,
		Location: raw.Location, Webserver: raw.Webserver, ContentType: raw.ContentType, Tech: raw.Tech,
		ResponseBody: raw.Body, ResponseHeaders: raw.RawHeader, Vhost: raw.Vhost,
	}
	return HTTPXOutcome{Endpoint: item, Structural: true}
}

// extractHTTPXInput is intentionally separate from the permissive evidence
// decoder. `input` is the only field that binds a tool row to an attempted
// candidate; a missing/null/wrong-typed value is therefore an unassociable
// protocol result, not a recoverable URL-content error.
func extractHTTPXInput(payload []byte) (string, bool, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return "", false, err
	}
	raw, ok := fields["input"]
	if !ok || string(bytes.TrimSpace(raw)) == "null" {
		return "", false, nil
	}
	var input string
	if err := json.Unmarshal(raw, &input); err != nil || input == "" {
		return "", false, nil
	}
	return input, true, nil
}

func validateHTTPXFieldTypes(payload []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return err
	}
	for _, name := range []string{"input", "host", "title", "location", "webserver", "content_type", "body", "raw_header"} {
		if value, exists := fields[name]; exists && !validJSONType[string](value) {
			return fmt.Errorf("httpx %s must be string", name)
		}
	}
	for _, name := range []string{"failed", "vhost"} {
		if value, exists := fields[name]; exists && !validJSONType[bool](value) {
			return fmt.Errorf("httpx %s must be boolean", name)
		}
	}
	for _, name := range []string{"status_code", "content_length"} {
		if value, exists := fields[name]; exists && !validJSONType[int](value) {
			return fmt.Errorf("httpx %s must be integer", name)
		}
	}
	if value, exists := fields["tech"]; exists {
		var technology []string
		if string(value) == "null" || json.Unmarshal(value, &technology) != nil {
			return errors.New("httpx tech must be a string array")
		}
	}
	return nil
}

func validJSONType[T any](value json.RawMessage) bool {
	if string(value) == "null" {
		return false
	}
	var decoded T
	return json.Unmarshal(value, &decoded) == nil
}

func InTargetScope(target enginecontract.Target, rawURL string) bool {
	host, err := deriveCandidateURLHost(rawURL)
	if err != nil {
		return false
	}
	switch target.Type {
	case enginecontract.TargetTypeDomain:
		return strings.EqualFold(host, target.Value) || strings.HasSuffix(strings.ToLower(host), "."+strings.ToLower(target.Value))
	case enginecontract.TargetTypeIP:
		return host == target.Value
	case enginecontract.TargetTypeCIDR:
		prefix, err := netip.ParsePrefix(target.Value)
		address, addressErr := netip.ParseAddr(host)
		return err == nil && addressErr == nil && address.Is4() && prefix.Contains(address)
	default:
		return false
	}
}

func observedEndpointFromURL(raw string) (enginecontract.Endpoint, error) {
	host, err := deriveCandidateURLHost(raw)
	if err != nil {
		return enginecontract.Endpoint{}, err
	}
	return enginecontract.Endpoint{URL: raw, Host: host}, nil
}

// deriveCandidateURLHost extracts only the authority used to route a raw URL
// through the local tool pipeline. It never parses or rebuilds path, query, or
// fragment bytes, which are still part of the observation sent to Server.
func deriveCandidateURLHost(raw string) (string, error) {
	if strings.ContainsAny(raw, "\x00\r\n") {
		return "", errors.New("candidate URL contains unsafe control characters")
	}
	offset := 0
	switch {
	case len(raw) >= len("http://") && strings.EqualFold(raw[:len("http://")], "http://"):
		offset = len("http://")
	case len(raw) >= len("https://") && strings.EqualFold(raw[:len("https://")], "https://"):
		offset = len("https://")
	default:
		return "", errors.New("candidate must be an HTTP URL with an authority")
	}
	authorityEnd := len(raw)
	for index := offset; index < len(raw); index++ {
		switch raw[index] {
		case '/', '?', '#':
			authorityEnd = index
			index = len(raw)
		}
	}
	authority := raw[offset:authorityEnd]
	if authority == "" {
		return "", errors.New("candidate URL authority is required")
	}
	hostPort := authority
	if at := strings.LastIndexByte(hostPort, '@'); at >= 0 {
		hostPort = hostPort[at+1:]
	}
	if hostPort == "" || strings.TrimSpace(hostPort) != hostPort || strings.HasPrefix(hostPort, "[") || strings.Count(hostPort, ":") > 1 {
		return "", errors.New("candidate URL authority host is invalid")
	}
	host := hostPort
	if colon := strings.IndexByte(hostPort, ':'); colon >= 0 {
		host = hostPort[:colon]
	}
	if host == "" || strings.TrimSpace(host) != host {
		return "", errors.New("candidate URL authority host is invalid")
	}
	return host, nil
}
