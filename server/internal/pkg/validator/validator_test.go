package validator

import (
	"strings"
	"testing"
)

func TestTranslateError(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("init validator failed: %v", err)
	}

	type sample struct {
		Name   string `json:"name" binding:"required"`
		Count  int    `json:"count" binding:"min=2"`
		Choice string `json:"choice" binding:"oneof=a b"`
	}

	err := validate.Struct(sample{Choice: "c"})
	if err == nil {
		t.Fatalf("expected validation errors")
	}

	errs := TranslateError(err)
	if errs["name"] == "" {
		t.Fatalf("expected name error translation")
	}
	if !strings.Contains(errs["count"], "at least") {
		t.Fatalf("expected count min translation, got %q", errs["count"])
	}
	if !strings.Contains(errs["choice"], "one of") {
		t.Fatalf("expected choice oneof translation, got %q", errs["choice"])
	}
}

func TestTranslateErrorToSlice(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("init validator failed: %v", err)
	}

	type sample struct {
		Name  string `json:"name" binding:"required"`
		Count int    `json:"count" binding:"min=2"`
	}

	err := validate.Struct(sample{})
	if err == nil {
		t.Fatalf("expected validation errors")
	}

	errs := TranslateErrorToSlice(err)
	if len(errs) != 2 {
		t.Fatalf("expected 2 field errors, got %d", len(errs))
	}

	seen := map[string]bool{}
	for _, e := range errs {
		seen[e.Field] = true
		if e.Message == "" {
			t.Fatalf("expected message for field %q", e.Field)
		}
	}
	if !seen["name"] || !seen["count"] {
		t.Fatalf("expected name and count errors, got %+v", seen)
	}
}
