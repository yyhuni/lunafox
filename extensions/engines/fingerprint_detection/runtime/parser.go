package fingerprintdetectionruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var errObserverWardMissingTarget = errors.New("Observer Ward result has no observed target")

type observerWardResult struct {
	InputTarget string
	Target      string
	Success     bool
	Matched     []observerWardMatched
}

type observerWardMatched struct {
	BaseURL string
	Result  observerWardMatchedResult
}

type observerWardMatchedResult struct {
	Status *int
	Names  []string
}

func parseObserverWardResult(payload []byte) (observerWardResult, error) {
	fields, err := decodeJSONObject(payload, map[string]struct{}{"input_target": {}, "target": {}, "success": {}, "matched": {}, "task_id": {}, "record": {}, "error": {}})
	if err != nil {
		return observerWardResult{}, fmt.Errorf("Observer Ward result: %w", err)
	}
	var result observerWardResult
	// input_target is retained only as optional diagnostic data. Observer Ward
	// may omit, normalize, repeat, or otherwise change it; none of those forms
	// can replace the observed target below or gate result admission.
	if rawInputTarget, ok := fields["input_target"]; ok && !bytes.Equal(bytes.TrimSpace(rawInputTarget), []byte("null")) {
		_ = json.Unmarshal(rawInputTarget, &result.InputTarget)
	}
	rawTarget, ok := fields["target"]
	if !ok || bytes.Equal(bytes.TrimSpace(rawTarget), []byte("null")) {
		return observerWardResult{}, fmt.Errorf("%w: target is required", errObserverWardMissingTarget)
	}
	if err := json.Unmarshal(rawTarget, &result.Target); err != nil || result.Target == "" {
		return observerWardResult{}, fmt.Errorf("%w: target must be a non-empty string", errObserverWardMissingTarget)
	}
	if err := requiredJSONField(fields, "success", &result.Success); err != nil {
		return observerWardResult{}, err
	}
	rawMatched, ok := fields["matched"]
	if !ok || bytes.Equal(bytes.TrimSpace(rawMatched), []byte("null")) {
		return observerWardResult{}, errors.New("matched is required and must be an array")
	}
	var matched []json.RawMessage
	if err := json.Unmarshal(rawMatched, &matched); err != nil {
		return observerWardResult{}, errors.New("matched must be an array")
	}
	result.Matched = make([]observerWardMatched, 0, len(matched))
	for index, raw := range matched {
		item, err := parseObserverWardMatched(raw)
		if err != nil {
			return observerWardResult{}, fmt.Errorf("matched[%d]: %w", index, err)
		}
		result.Matched = append(result.Matched, item)
	}
	return result, nil
}

func parseObserverWardMatched(payload []byte) (observerWardMatched, error) {
	fields, err := decodeJSONObject(payload, map[string]struct{}{"base_url": {}, "result": {}})
	if err != nil {
		return observerWardMatched{}, err
	}
	var item observerWardMatched
	if err := requiredJSONField(fields, "base_url", &item.BaseURL); err != nil || item.BaseURL == "" {
		if err == nil {
			err = errors.New("base_url must be a non-empty string")
		}
		return observerWardMatched{}, err
	}
	rawResult, ok := fields["result"]
	if !ok {
		return observerWardMatched{}, errors.New("result is required")
	}
	resultFields, err := decodeJSONObject(rawResult, map[string]struct{}{
		"title": {}, "length": {}, "status": {}, "favicon": {}, "certificate": {}, "name": {}, "fingerprints": {},
	})
	if err != nil {
		return observerWardMatched{}, err
	}
	if rawStatus, exists := resultFields["status"]; exists && !bytes.Equal(bytes.TrimSpace(rawStatus), []byte("null")) {
		var status int
		if err := json.Unmarshal(rawStatus, &status); err != nil {
			return observerWardMatched{}, errors.New("result.status must be an integer or null")
		}
		item.Result.Status = &status
	}
	if rawNames, exists := resultFields["name"]; exists {
		if bytes.Equal(bytes.TrimSpace(rawNames), []byte("null")) {
			return observerWardMatched{}, errors.New("result.name must be a string array")
		}
		if err := json.Unmarshal(rawNames, &item.Result.Names); err != nil {
			return observerWardMatched{}, errors.New("result.name must be a string array")
		}
	}
	return item, nil
}

// trustedObserverWardTechnology extracts only the current row's trusted
// technology names. The top-level success field is deliberately not used as
// an attribution signal; a concrete HTTP status is required.
func trustedObserverWardTechnology(result observerWardResult) (bool, []string) {
	trusted := false
	tech := make([]string, 0)
	for _, matched := range result.Matched {
		if matched.Result.Status == nil || *matched.Result.Status < 100 || *matched.Result.Status > 599 {
			continue
		}
		trusted = true
		tech = append(tech, matched.Result.Names...)
	}
	return trusted, tech
}

func decodeJSONObject(payload []byte, allowed map[string]struct{}) (map[string]json.RawMessage, error) {
	// encoding/json replaces invalid UTF-8 in strings with U+FFFD. Reject raw
	// tool output before decoding so candidate identities cannot be rewritten.
	if err := validateJSONText(payload); err != nil {
		return nil, fmt.Errorf("value has invalid JSON text: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return nil, errors.New("value must be a JSON object")
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		fieldToken, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		field, ok := fieldToken.(string)
		if !ok {
			return nil, errors.New("object field must be a string")
		}
		if _, duplicate := fields[field]; duplicate {
			return nil, fmt.Errorf("duplicate field %q", field)
		}
		if _, known := allowed[field]; !known {
			return nil, fmt.Errorf("unknown field %q", field)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		fields[field] = value
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("trailing JSON content")
		}
		return nil, err
	}
	return fields, nil
}

func requiredJSONField(fields map[string]json.RawMessage, name string, target any) error {
	raw, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("%s is required", name)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("%s has an invalid type", name)
	}
	return nil
}
