package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

var nullJSON = json.RawMessage("null")

// Presentation is the format-aware, read-only view derived from persisted
// native payload. Query fields remain limited to filtering and ordering; they
// must not become a second source of truth for operator-visible rule content.
type Presentation struct {
	Fields           map[string]json.RawMessage
	AdditionalFields []AdditionalField
}

// AdditionalField preserves an unsupported-but-valid native extension for the
// detail response without allowing it to change the fixed list column matrix.
type AdditionalField struct {
	Path  string
	Value json.RawMessage
}

// PresentFingerprint constructs the fixed presentation matrix for one stored
// rule. A malformed historical payload is a data-consistency failure: callers
// must fail the read instead of silently returning incomplete records.
func PresentFingerprint(record PersistedRecord) (Presentation, error) {
	root, err := decodePresentationObject(record.Payload)
	if err != nil {
		return Presentation{}, fmt.Errorf("decode persisted %s payload: %w", record.Library, err)
	}

	if record.Library == LibraryFingerPrintHub {
		return presentFingerPrintHub(root)
	}
	return Presentation{}, fmt.Errorf("unsupported library %q", record.Library)
}

func presentFingerPrintHub(root map[string]json.RawMessage) (Presentation, error) {
	fingerprintID, err := presentationRequiredString(root, "id")
	if err != nil {
		return Presentation{}, err
	}
	httpRules, err := presentationRequiredArray(root, "http")
	if err != nil {
		return Presentation{}, err
	}
	info, err := presentationRequiredObject(root, "info")
	if err != nil {
		return Presentation{}, err
	}
	displayName, err := presentationRequiredString(info, "name")
	if err != nil {
		return Presentation{}, fmt.Errorf("info.%w", err)
	}
	fields := map[string]json.RawMessage{
		"fingerprintId": jsonString(fingerprintID),
		"displayName":   jsonString(displayName),
		"author":        presentationOptionalRaw(info, "author"),
		"tags":          presentationOptionalRaw(info, "tags"),
		"severity":      presentationOptionalRaw(info, "severity"),
		"metadata":      presentationOptionalRaw(info, "metadata"),
		"http":          httpRules,
		"sourceFile":    presentationOptionalRaw(root, "_source_file"),
	}
	additional := collectAdditional(root, setOf("id", "info", "http", "_source_file"), "")
	additional = append(additional, collectAdditional(info, setOf("name", "author", "tags", "severity", "metadata"), "/info")...)
	return Presentation{Fields: fields, AdditionalFields: sortAdditional(additional)}, nil
}

func decodePresentationObject(payload json.RawMessage) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	var root map[string]json.RawMessage
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}
	if root == nil {
		return nil, fmt.Errorf("payload must be an object")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return nil, fmt.Errorf("payload contains trailing data: %w", err)
		}
		return nil, fmt.Errorf("payload contains multiple JSON values")
	}
	return root, nil
}

func presentationRequiredString(object map[string]json.RawMessage, field string) (string, error) {
	raw, exists := object[field]
	if !exists || isNullJSON(raw) {
		return "", fmt.Errorf("%s is required", field)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s must be a non-empty string", field)
	}
	return value, nil
}

func presentationRequiredArray(object map[string]json.RawMessage, field string) (json.RawMessage, error) {
	raw, exists := object[field]
	if !exists || isNullJSON(raw) {
		return nil, fmt.Errorf("%s is required", field)
	}
	var value []json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be an array", field)
	}
	return append(json.RawMessage(nil), raw...), nil
}

func presentationRequiredObject(object map[string]json.RawMessage, field string) (map[string]json.RawMessage, error) {
	raw, exists := object[field]
	if !exists || isNullJSON(raw) {
		return nil, fmt.Errorf("%s is required", field)
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return nil, fmt.Errorf("%s must be an object", field)
	}
	return value, nil
}

func presentationOptionalRaw(object map[string]json.RawMessage, field string) json.RawMessage {
	raw, exists := object[field]
	if !exists || len(raw) == 0 {
		return append(json.RawMessage(nil), nullJSON...)
	}
	return append(json.RawMessage(nil), raw...)
}

func isNullJSON(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), nullJSON)
}

func jsonString(value string) json.RawMessage {
	raw, _ := json.Marshal(value)
	return raw
}

func collectAdditional(object map[string]json.RawMessage, known map[string]struct{}, prefix string) []AdditionalField {
	additional := make([]AdditionalField, 0)
	for key, value := range object {
		if _, exists := known[key]; exists {
			continue
		}
		additional = append(additional, AdditionalField{
			Path:  prefix + "/" + escapeJSONPointerToken(key),
			Value: append(json.RawMessage(nil), value...),
		})
	}
	return additional
}

func sortAdditional(fields []AdditionalField) []AdditionalField {
	sort.Slice(fields, func(left, right int) bool { return fields[left].Path < fields[right].Path })
	if fields == nil {
		return []AdditionalField{}
	}
	return fields
}

func escapeJSONPointerToken(value string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(value)
}

func setOf(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}
