package dto

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestUpdateBlacklistPolicyRequestTracksPatternsPresence(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		present bool
		null    bool
		want    []string
	}{
		{name: "omitted", body: `{"name":"blacklistPolicy","etag":"etag"}`},
		{name: "null", body: `{"name":"blacklistPolicy","patterns":null,"etag":"etag"}`, present: true, null: true},
		{name: "empty", body: `{"name":"blacklistPolicy","patterns":[],"etag":"etag"}`, present: true, want: []string{}},
		{name: "values", body: `{"name":"blacklistPolicy","patterns":["example.com"],"etag":"etag"}`, present: true, want: []string{"example.com"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var request UpdateBlacklistPolicyRequest
			if err := json.Unmarshal([]byte(test.body), &request); err != nil {
				t.Fatal(err)
			}
			if request.PatternsPresent() != test.present || request.PatternsNull() != test.null || !reflect.DeepEqual(request.Patterns, test.want) {
				t.Fatalf("request = %#v present=%t null=%t, want present=%t null=%t patterns=%v", request, request.PatternsPresent(), request.PatternsNull(), test.present, test.null, test.want)
			}
		})
	}
}

func TestUpdateBlacklistPolicyRequestRejectsUnknownFieldsAndIgnoresOutputTime(t *testing.T) {
	var request UpdateBlacklistPolicyRequest
	if err := json.Unmarshal([]byte(`{"name":"blacklistPolicy","patterns":[],"etag":"etag","updateTime":{"ignored":true}}`), &request); err != nil {
		t.Fatalf("output-only updateTime should be ignored: %v", err)
	}
	if err := json.Unmarshal([]byte(`{"name":"blacklistPolicy","patterns":[],"etag":"etag","legacyPatterns":[]}`), &request); err == nil {
		t.Fatal("unknown legacy field must be rejected")
	}
}

func TestBlacklistPolicyResponseAlwaysUsesFourFields(t *testing.T) {
	encoded, err := json.Marshal(BlacklistPolicyResponse{Name: "blacklistPolicy", Patterns: []string{}, ETag: "etag"})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 4 {
		t.Fatalf("response fields = %s, want exactly four", encoded)
	}
	for _, field := range []string{"name", "patterns", "etag", "updateTime"} {
		if _, ok := fields[field]; !ok {
			t.Fatalf("missing %q in %s", field, encoded)
		}
	}
}
