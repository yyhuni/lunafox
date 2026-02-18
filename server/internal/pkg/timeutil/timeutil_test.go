package timeutil

import (
	"testing"
	"time"
)

func TestToUTC(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	value := time.Date(2026, 2, 9, 20, 30, 10, 123456789, loc)

	result := ToUTC(value)

	if result.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %s", result.Location())
	}
	if result.Format(time.RFC3339Nano) != "2026-02-09T12:30:10.123456789Z" {
		t.Fatalf("unexpected UTC value: %s", result.Format(time.RFC3339Nano))
	}
}

func TestToUTCPtr(t *testing.T) {
	if ToUTCPtr(nil) != nil {
		t.Fatal("expected nil pointer")
	}

	loc := time.FixedZone("UTC-5", -5*60*60)
	value := time.Date(2026, 2, 9, 8, 0, 0, 0, loc)

	result := ToUTCPtr(&value)
	if result == nil {
		t.Fatal("expected non-nil pointer")
	}
	if result.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %s", result.Location())
	}
	if result.Format(time.RFC3339Nano) != "2026-02-09T13:00:00Z" {
		t.Fatalf("unexpected UTC value: %s", result.Format(time.RFC3339Nano))
	}
}

func TestFormatRFC3339NanoUTC(t *testing.T) {
	loc := time.FixedZone("UTC+9", 9*60*60)
	value := time.Date(2026, 2, 9, 9, 8, 7, 654321000, loc)

	formatted := FormatRFC3339NanoUTC(value)
	if formatted != "2026-02-09T00:08:07.654321Z" {
		t.Fatalf("unexpected formatted value: %s", formatted)
	}
}
