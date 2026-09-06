package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

const (
	DiagnosticTransport = "TRANSPORT"
	DiagnosticEncoding  = "ENCODING"
	DiagnosticSyntax    = "SYNTAX"
	DiagnosticFormat    = "FORMAT"
	DiagnosticRecord    = "RECORD"
)

const (
	ReasonRequiredFieldMissing = "REQUIRED_FIELD_MISSING"
	ReasonInvalidFieldType     = "INVALID_FIELD_TYPE"
	ReasonInvalidRuleStructure = "INVALID_RULE_STRUCTURE"
	ReasonFormatMismatch       = "FORMAT_MISMATCH"
	ReasonInvalidSyntax        = "INVALID_SYNTAX"
	ReasonInvalidEncoding      = "INVALID_UTF8"
)

// ImportDiagnosticError carries location-safe validation information. It never
// includes native rule content because the HTTP boundary exposes it to users.
type ImportDiagnosticError struct {
	Kind        string
	Reason      string
	RecordIndex int
	FieldPath   string
	Line        int
	Column      int
	Err         error
}

func (err *ImportDiagnosticError) Error() string {
	if err.Err != nil {
		return err.Err.Error()
	}
	return err.Reason
}

func (err *ImportDiagnosticError) Unwrap() error { return err.Err }

func formatError(reason string, cause error) *ImportDiagnosticError {
	return &ImportDiagnosticError{Kind: DiagnosticFormat, Reason: reason, Err: cause}
}

func recordError(recordIndex int, fieldPath, reason string, cause error) *ImportDiagnosticError {
	return &ImportDiagnosticError{
		Kind:        DiagnosticRecord,
		Reason:      reason,
		RecordIndex: recordIndex,
		FieldPath:   fieldPath,
		Err:         cause,
	}
}

// ParseNativeImport accepts the FingerprintHub web_fingerprint_v4 JSON
// aggregate used by Observer Ward. The library must be validated before any
// source bytes are decoded so retired format values cannot reach storage.
func ParseNativeImport(library Library, input []byte) ([]ImportedRecord, error) {
	if library != LibraryFingerPrintHub || !library.IsSupported() {
		return nil, formatError(ReasonFormatMismatch, fmt.Errorf("unsupported library %q", library))
	}
	if len(input) == 0 {
		return nil, &ImportDiagnosticError{Kind: DiagnosticTransport, Reason: "FINGERPRINT_IMPORT_FILE_EMPTY"}
	}
	if !utf8.Valid(input) {
		line, column := invalidUTF8Position(input)
		return nil, &ImportDiagnosticError{Kind: DiagnosticEncoding, Reason: ReasonInvalidEncoding, Line: line, Column: column}
	}
	return parseFingerPrintHubJSON(input)
}

// CollapseLastWins applies the documented source-order duplicate policy only
// after every source record has passed validation.
func CollapseLastWins(records []ImportedRecord) []ImportedRecord {
	lastIndex := make(map[string]int, len(records))
	for index, record := range records {
		lastIndex[record.IdentityKey] = index
	}
	reduced := make([]ImportedRecord, 0, len(lastIndex))
	for index, record := range records {
		if lastIndex[record.IdentityKey] == index {
			reduced = append(reduced, record)
		}
	}
	return reduced
}

// EncodeNativeExport produces the source-compatible FingerprintHub JSON
// aggregate from the stored native payloads.
func EncodeNativeExport(library Library, records []PersistedRecord) ([]byte, error) {
	if library != LibraryFingerPrintHub || !library.IsSupported() {
		return nil, fmt.Errorf("unsupported library %q", library)
	}
	items, err := payloadArray(records)
	if err != nil {
		return nil, err
	}
	return json.Marshal(items)
}

// WriteNativeExport writes one complete FingerprintHub aggregate without
// building a duplicate byte buffer for artifact publication.
func WriteNativeExport(writer io.Writer, library Library, records []PersistedRecord) (int64, error) {
	if writer == nil {
		return 0, fmt.Errorf("native export writer is required")
	}
	if library != LibraryFingerPrintHub || !library.IsSupported() {
		return 0, fmt.Errorf("unsupported library %q", library)
	}
	counter := &nativeExportCounter{writer: writer}
	if err := counter.write([]byte("[")); err != nil {
		return counter.n, err
	}
	for index, record := range records {
		if index > 0 {
			if err := counter.write([]byte(",")); err != nil {
				return counter.n, err
			}
		}
		if !json.Valid(record.Payload) {
			return counter.n, fmt.Errorf("invalid persisted %s payload", record.Library)
		}
		if err := counter.write(record.Payload); err != nil {
			return counter.n, err
		}
	}
	if err := counter.write([]byte("]")); err != nil {
		return counter.n, err
	}
	return counter.n, nil
}

type nativeExportCounter struct {
	writer io.Writer
	n      int64
}

func (counter *nativeExportCounter) write(contents []byte) error {
	n, err := counter.writer.Write(contents)
	counter.n += int64(n)
	if err != nil {
		return err
	}
	if n != len(contents) {
		return io.ErrShortWrite
	}
	return nil
}

func parseFingerPrintHubJSON(input []byte) ([]ImportedRecord, error) {
	root, err := decodeJSONValue(input)
	if err != nil {
		return nil, err
	}
	items, ok := root.([]any)
	if !ok || len(items) == 0 {
		return nil, formatError(ReasonFormatMismatch, fmt.Errorf("FingerprintHub root must be a non-empty array"))
	}
	records := make([]ImportedRecord, 0, len(items))
	for index, value := range items {
		recordIndex := index + 1
		item, ok := value.(map[string]any)
		if !ok {
			return nil, recordError(recordIndex, "", ReasonInvalidFieldType, fmt.Errorf("record must be an object"))
		}
		id, err := requiredString(item, "id", recordIndex)
		if err != nil {
			return nil, err
		}
		http, ok := item["http"].([]any)
		if !ok || len(http) == 0 {
			return nil, recordError(recordIndex, "http", ReasonInvalidRuleStructure, fmt.Errorf("http must be a non-empty array"))
		}
		if err := validateFingerPrintHubHTTPItems(http, recordIndex); err != nil {
			return nil, err
		}
		info, ok := item["info"].(map[string]any)
		if !ok {
			return nil, recordError(recordIndex, "info", ReasonRequiredFieldMissing, fmt.Errorf("info is required"))
		}
		name, err := requiredString(info, "name", recordIndex)
		if err != nil {
			return nil, rebaseFieldPath(err, "info.name")
		}
		severity, err := optionalString(info, "severity", recordIndex)
		if err != nil {
			return nil, rebaseFieldPath(err, "info.severity")
		}
		payload, err := marshalPayload(item)
		if err != nil {
			return nil, err
		}
		identity, err := identityHash(id, http)
		if err != nil {
			return nil, recordError(recordIndex, "http", ReasonInvalidRuleStructure, err)
		}
		contentHash, err := ContentHash(payload)
		if err != nil {
			return nil, recordError(recordIndex, "", ReasonInvalidRuleStructure, err)
		}
		records = append(records, ImportedRecord{
			Library: LibraryFingerPrintHub, SourceIndex: recordIndex, IdentityKey: identity,
			ContentHash: contentHash, Payload: payload,
			Fields: QueryFields{DisplayName: name, NativeID: id, Severity: severity},
		})
	}
	return records, nil
}

func validateFingerPrintHubHTTPItems(httpItems []any, recordIndex int) error {
	for httpIndex, candidate := range httpItems {
		item, ok := candidate.(map[string]any)
		if !ok || len(item) == 0 {
			return recordError(recordIndex, fmt.Sprintf("http[%d]", httpIndex), ReasonInvalidRuleStructure, fmt.Errorf("http item must be a non-empty object"))
		}
		matchers, ok := item["matchers"].([]any)
		if !ok || len(matchers) == 0 {
			return recordError(recordIndex, fmt.Sprintf("http[%d].matchers", httpIndex), ReasonInvalidRuleStructure, fmt.Errorf("http item must include a non-empty matchers array"))
		}
		for matcherIndex, matcher := range matchers {
			matcherObject, ok := matcher.(map[string]any)
			if !ok || len(matcherObject) == 0 {
				return recordError(recordIndex, fmt.Sprintf("http[%d].matchers[%d]", httpIndex, matcherIndex), ReasonInvalidRuleStructure, fmt.Errorf("matcher must be a non-empty object"))
			}
			if _, err := requiredString(matcherObject, "type", recordIndex); err != nil {
				return rebaseFieldPath(err, fmt.Sprintf("http[%d].matchers[%d].type", httpIndex, matcherIndex))
			}
		}
	}
	return nil
}

func decodeJSONValue(input []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, jsonDiagnostic(input, err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, jsonDiagnostic(input, err)
	}
	return value, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return fmt.Errorf("multiple JSON values")
	}
	return err
}

func jsonDiagnostic(input []byte, cause error) *ImportDiagnosticError {
	diagnostic := &ImportDiagnosticError{Kind: DiagnosticSyntax, Reason: ReasonInvalidSyntax, Err: cause}
	var syntaxErr *json.SyntaxError
	if errors.As(cause, &syntaxErr) {
		diagnostic.Line, diagnostic.Column = byteOffsetPosition(input, syntaxErr.Offset)
	}
	return diagnostic
}

func payloadArray(records []PersistedRecord) ([]json.RawMessage, error) {
	items := make([]json.RawMessage, 0, len(records))
	for _, record := range records {
		if !json.Valid(record.Payload) {
			return nil, fmt.Errorf("invalid persisted %s payload", record.Library)
		}
		items = append(items, append(json.RawMessage(nil), record.Payload...))
	}
	return items, nil
}

func marshalPayload(value any) (json.RawMessage, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return json.RawMessage(payload), nil
}

func requiredString(item map[string]any, field string, recordIndex int) (string, error) {
	value, exists := item[field]
	if !exists || value == nil {
		return "", recordError(recordIndex, field, ReasonRequiredFieldMissing, fmt.Errorf("%s is required", field))
	}
	text, ok := nonEmptyString(value)
	if !ok {
		return "", recordError(recordIndex, field, ReasonInvalidFieldType, fmt.Errorf("%s must be a non-empty string", field))
	}
	return text, nil
}

func optionalString(item map[string]any, field string, recordIndex int) (*string, error) {
	value, exists := item[field]
	if !exists || value == nil {
		return nil, nil
	}
	text, ok := nonEmptyString(value)
	if !ok {
		return nil, recordError(recordIndex, field, ReasonInvalidFieldType, fmt.Errorf("%s must be a non-empty string", field))
	}
	return &text, nil
}

func nonEmptyString(value any) (string, bool) {
	text, ok := value.(string)
	return text, ok && strings.TrimSpace(text) != ""
}

func rebaseFieldPath(err error, fieldPath string) error {
	var diagnostic *ImportDiagnosticError
	if errors.As(err, &diagnostic) {
		clone := *diagnostic
		clone.FieldPath = fieldPath
		return &clone
	}
	return err
}

func byteOffsetPosition(input []byte, offset int64) (int, int) {
	if offset <= 0 {
		return 0, 0
	}
	limit := int(offset - 1)
	if limit > len(input) {
		limit = len(input)
	}
	line, column := 1, 1
	for _, character := range input[:limit] {
		if character == '\n' {
			line, column = line+1, 1
		} else {
			column++
		}
	}
	return line, column
}

func invalidUTF8Position(input []byte) (int, int) {
	line, column := 1, 1
	for len(input) > 0 {
		character, size := utf8.DecodeRune(input)
		if character == utf8.RuneError && size == 1 {
			return line, column
		}
		if character == '\n' {
			line, column = line+1, 1
		} else {
			column++
		}
		input = input[size:]
	}
	return 0, 0
}
