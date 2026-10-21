package directoryscanruntime

import (
	"reflect"
	"testing"
)

func TestTargetVariantPayloadsUseFixedRuleSet(t *testing.T) {
	tests := []struct {
		name      string
		candidate string
		wantNames []string
	}{
		{
			name:      "www host",
			candidate: "https://www.example.com/path?q=1",
			wantNames: []string{"www.example.com", "example.com", "www", "example"},
		},
		{
			name:      "multi label host",
			candidate: "https://a.b.example.com/",
			wantNames: []string{"a.b.example.com", "example.com", "a", "b"},
		},
		{
			name:      "single label host",
			candidate: "http://localhost:8080/",
			wantNames: []string{"localhost"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payloads := targetVariantPayloads(test.candidate)
			wantLen := len(test.wantNames) * len(targetVariantSuffixes)
			if len(payloads) != wantLen {
				t.Fatalf("payload count = %d, want %d: %#v", len(payloads), wantLen, payloads)
			}
			for index, name := range test.wantNames {
				first := index * len(targetVariantSuffixes)
				if payloads[first] != name+targetVariantSuffixes[0] {
					t.Fatalf("payloads[%d] = %q, want prefix %q", first, payloads[first], name)
				}
			}
			again := targetVariantPayloads(test.candidate)
			if !reflect.DeepEqual(payloads, again) {
				t.Fatalf("variant generation is not deterministic: %#v vs %#v", payloads, again)
			}
		})
	}
}

func TestTargetVariantPayloadsSkipNonDomainTargets(t *testing.T) {
	for _, candidate := range []string{
		"http://127.0.0.1:8080/",
		"https://127.0.0.1/",
		"http://[::1]:8080/",
		"http:///missing-host",
		"not-a-url",
	} {
		if payloads := targetVariantPayloads(candidate); len(payloads) != 0 {
			t.Fatalf("targetVariantPayloads(%q) = %#v, want none", candidate, payloads)
		}
	}
}
