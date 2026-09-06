package screenshotruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	maxHTTPXRecordBytes = 4 * 1024 * 1024
	maxHTTPXStderrBytes = 1 * 1024 * 1024
)

var errHTTPXRowMissingURL = errors.New("HTTPX row has no observed URL")

type httpxRow struct {
	Identity       string
	Failed         bool
	ScreenshotPath string
	StatusCode     *int
}

func decodeHTTPXRow(payload []byte) (httpxRow, error) {
	// encoding/json otherwise replaces invalid UTF-8 in strings. Reject raw tool
	// output before decoding so the observed URL cannot be rewritten.
	if err := validateJSONText(payload); err != nil {
		return httpxRow{}, fmt.Errorf("HTTPX row has invalid JSON text: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return httpxRow{}, err
	}
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return httpxRow{}, errors.New("HTTPX row must be a JSON object")
	}
	fields := map[string]json.RawMessage{}
	for decoder.More() {
		fieldToken, err := decoder.Token()
		if err != nil {
			return httpxRow{}, err
		}
		field, ok := fieldToken.(string)
		if !ok {
			return httpxRow{}, errors.New("HTTPX row field must be a string")
		}
		if _, duplicate := fields[field]; duplicate {
			return httpxRow{}, fmt.Errorf("duplicate HTTPX row field %q", field)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return httpxRow{}, err
		}
		fields[field] = value
	}
	if _, err := decoder.Token(); err != nil {
		return httpxRow{}, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return httpxRow{}, errors.New("HTTPX row has trailing content")
		}
		return httpxRow{}, err
	}
	rawURL, ok := fields["url"]
	if !ok || bytes.Equal(bytes.TrimSpace(rawURL), []byte("null")) {
		return httpxRow{}, errHTTPXRowMissingURL
	}
	var observedURL string
	if err := json.Unmarshal(rawURL, &observedURL); err != nil || observedURL == "" {
		return httpxRow{}, fmt.Errorf("%w: url must be a non-empty string", errHTTPXRowMissingURL)
	}
	row := httpxRow{Identity: observedURL}
	if raw, ok := fields["failed"]; ok {
		if err := json.Unmarshal(raw, &row.Failed); err != nil {
			return httpxRow{}, errors.New("HTTPX row failed must be boolean")
		}
	}
	if raw, ok := fields["screenshot_path"]; ok {
		if string(raw) == "null" {
			return httpxRow{}, errors.New("HTTPX row screenshot_path must be a string")
		}
		if err := json.Unmarshal(raw, &row.ScreenshotPath); err != nil {
			return httpxRow{}, errors.New("HTTPX row screenshot_path must be a string")
		}
	}
	if raw, ok := fields["status_code"]; ok {
		// Probe status is optional evidence. A malformed, null, or out-of-range
		// value must not discard an otherwise valid screenshot; it is omitted
		// when the result item is assembled.
		var value int
		if string(raw) != "null" {
			if err := json.Unmarshal(raw, &value); err == nil && value >= 100 && value <= 599 {
				row.StatusCode = &value
			}
		}
	}
	return row, nil
}
