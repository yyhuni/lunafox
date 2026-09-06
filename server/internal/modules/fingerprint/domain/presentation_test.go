package domain

import (
	"encoding/json"
	"testing"
)

func TestPresentFingerprintHubProjectionKeepsExtensionsSeparate(t *testing.T) {
	presentation, err := PresentFingerprint(PersistedRecord{Library: LibraryFingerPrintHub, Payload: json.RawMessage(`{"id":"example","info":{"name":"Example","severity":"low","custom":true},"http":[{"matchers":[{"type":"word"}]}],"_source_file":"web.json","custom":42}`)})
	if err != nil {
		t.Fatalf("PresentFingerprint() error = %v", err)
	}
	if string(presentation.Fields["fingerprintId"]) != `"example"` || string(presentation.Fields["displayName"]) != `"Example"` {
		t.Fatalf("fields = %#v", presentation.Fields)
	}
	if len(presentation.AdditionalFields) != 2 || presentation.AdditionalFields[0].Path != "/custom" || presentation.AdditionalFields[1].Path != "/info/custom" {
		t.Fatalf("additional fields = %#v", presentation.AdditionalFields)
	}
}

func TestPresentFingerprintRejectsRetiredLibrary(t *testing.T) {
	if _, err := PresentFingerprint(PersistedRecord{Library: "ehole", Payload: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("PresentFingerprint() accepted retired library")
	}
}
