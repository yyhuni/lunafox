package domain

import "testing"

func TestParseScanTriggerTypeAcceptsOnlyClosedVocabulary(t *testing.T) {
	tests := []struct {
		value string
		want  ScanTriggerType
		ok    bool
	}{
		{value: "manual", want: ScanTriggerTypeManual, ok: true},
		{value: "scheduled", want: ScanTriggerTypeScheduled, ok: true},
		{value: "ai", want: ScanTriggerTypeAI, ok: true},
		{value: "", ok: false},
		{value: "legacy", ok: false},
		{value: "unknown", ok: false},
	}

	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			got, ok := ParseScanTriggerType(test.value)
			if ok != test.ok {
				t.Fatalf("ParseScanTriggerType(%q) validity = %t, want %t", test.value, ok, test.ok)
			}
			if test.ok && got != test.want {
				t.Fatalf("ParseScanTriggerType(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}
