package domain

import (
	"bytes"
	"encoding/json"
	"testing"
)

const validFingerprintHubAggregate = `[
  {"id":"example-one","info":{"name":"Example One","severity":"info"},"http":[{"path":["/"],"matchers":[{"type":"word","words":["one"]}]}]},
  {"id":"example-two","info":{"name":"Example Two"},"http":[{"matchers":[{"type":"word","words":["two"]}]}]}
]`

func TestFingerprintHubImportExportRoundTrip(t *testing.T) {
	records, err := ParseNativeImport(LibraryFingerPrintHub, []byte(validFingerprintHubAggregate))
	if err != nil {
		t.Fatalf("ParseNativeImport() error = %v", err)
	}
	if len(records) != 2 || records[0].Fields.NativeID != "example-one" || records[0].Fields.DisplayName != "Example One" {
		t.Fatalf("records = %#v", records)
	}
	exported, err := EncodeNativeExport(LibraryFingerPrintHub, []PersistedRecord{{Library: LibraryFingerPrintHub, Payload: records[1].Payload}, {Library: LibraryFingerPrintHub, Payload: records[0].Payload}})
	if err != nil {
		t.Fatalf("EncodeNativeExport() error = %v", err)
	}
	var decoded []json.RawMessage
	if err := json.Unmarshal(exported, &decoded); err != nil || len(decoded) != 2 {
		t.Fatalf("exported aggregate = %s, unmarshal error = %v", exported, err)
	}
	var streamed bytes.Buffer
	if _, err := WriteNativeExport(&streamed, LibraryFingerPrintHub, []PersistedRecord{{Library: LibraryFingerPrintHub, Payload: records[1].Payload}, {Library: LibraryFingerPrintHub, Payload: records[0].Payload}}); err != nil {
		t.Fatalf("WriteNativeExport() error = %v", err)
	}
	if !bytes.Equal(exported, streamed.Bytes()) {
		t.Fatalf("streamed bytes = %s, want %s", streamed.Bytes(), exported)
	}
}

func TestFingerprintHubImportRejectsMalformedHTTPRules(t *testing.T) {
	_, err := ParseNativeImport(LibraryFingerPrintHub, []byte(`[{"id":"invalid","info":{"name":"Invalid"},"http":[{"matchers":[{"words":["missing-type"]}]}]}]`))
	diagnostic, ok := err.(*ImportDiagnosticError)
	if !ok || diagnostic.Kind != DiagnosticRecord || diagnostic.Reason != ReasonRequiredFieldMissing || diagnostic.FieldPath != "http[0].matchers[0].type" {
		t.Fatalf("diagnostic = %#v", diagnostic)
	}
}

func TestRetiredLibrariesFailBeforeParsingOrExport(t *testing.T) {
	for _, library := range []Library{"ehole", "goby", "wappalyzer", "fingers", "arl"} {
		t.Run(string(library), func(t *testing.T) {
			if _, err := ParseNativeImport(library, []byte(validFingerprintHubAggregate)); err == nil {
				t.Fatal("ParseNativeImport() accepted retired library")
			}
			if _, err := EncodeNativeExport(library, nil); err == nil {
				t.Fatal("EncodeNativeExport() accepted retired library")
			}
			if _, err := ParseLibrary(string(library)); err == nil {
				t.Fatal("ParseLibrary() accepted retired library")
			}
		})
	}
}
