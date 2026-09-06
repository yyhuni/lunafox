package results

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

var (
	subdomainResultFields = map[string]struct{}{
		"dnsName": {},
	}
	hostPortResultFields = map[string]struct{}{
		"host": {},
		"ip":   {},
		"port": {},
	}
	websiteResultFields = map[string]struct{}{
		"url":             {},
		"host":            {},
		"title":           {},
		"statusCode":      {},
		"contentLength":   {},
		"location":        {},
		"webserver":       {},
		"contentType":     {},
		"tech":            {},
		"responseBody":    {},
		"vhost":           {},
		"responseHeaders": {},
	}
	endpointResultFields = map[string]struct{}{
		"url": {}, "host": {}, "title": {}, "statusCode": {}, "contentLength": {},
		"location": {}, "webserver": {}, "contentType": {}, "tech": {},
		"responseBody": {}, "responseBodyTruncated": {}, "vhost": {},
		"responseHeaders": {}, "responseHeadersTruncated": {},
	}
	screenshotResultFields = map[string]struct{}{
		"url": {}, "statusCode": {}, "image": {},
	}
	websiteTechnologyResultFields = map[string]struct{}{
		"url": {}, "tech": {},
	}
	directoryResultFields = map[string]struct{}{
		"url": {}, "status": {}, "contentLength": {}, "contentType": {}, "duration": {},
	}
	vulnerabilityResultFields = map[string]struct{}{
		"url": {}, "vulnType": {}, "severity": {}, "source": {}, "cvssScore": {}, "description": {}, "rawOutput": {},
	}
)

// decodeCanonicalResultObject checks the wire object before encoding/json
// maps it into a typed item. encoding/json otherwise accepts case variants,
// silently ignores unknown keys, and keeps the last duplicate key, all of
// which would let a result bypass the canonical item contract.
func decodeCanonicalResultObject(payload string, allowed map[string]struct{}, target any) error {
	if err := ValidateJSONText([]byte(payload)); err != nil {
		return fmt.Errorf("result item has invalid JSON text: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(payload)))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	opening, ok := token.(json.Delim)
	if !ok || opening != '{' {
		return fmt.Errorf("result item must be a JSON object")
	}
	seen := make(map[string]struct{}, len(allowed))
	for decoder.More() {
		fieldToken, err := decoder.Token()
		if err != nil {
			return err
		}
		field, ok := fieldToken.(string)
		if !ok {
			return fmt.Errorf("result item field name must be a string")
		}
		if _, exists := seen[field]; exists {
			return fmt.Errorf("duplicate result item field %q", field)
		}
		if _, allowed := allowed[field]; !allowed {
			return fmt.Errorf("unknown result item field %q", field)
		}
		seen[field] = struct{}{}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}
	}
	closing, err := decoder.Token()
	if err != nil {
		return err
	}
	if closing != json.Delim('}') {
		return fmt.Errorf("result item object is not closed")
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON content")
		}
		return err
	}
	if err := json.Unmarshal([]byte(payload), target); err != nil {
		return err
	}
	return nil
}
