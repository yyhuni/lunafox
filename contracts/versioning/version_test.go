package versioning

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"v1.2.3":  "v1.2.3",
		".0.0":  ".0.0",
		" 1.0.0 ": "1.0.0",
		"":        "",
	}
	for input, expected := range cases {
		if got := Normalize(input); got != expected {
			t.Fatalf("Normalize(%q)=%q, want %q", input, got, expected)
		}
	}
}

func TestVersionValidators(t *testing.T) {
	if !IsValidSemVer("1.0.0") {
		t.Fatalf("expected SemVer valid")
	}
	if !IsValidSemVer("1.0.0-beta+1") {
		t.Fatalf("expected SemVer suffix valid")
	}
	if IsValidSemVer("v1.0.0-beta+1") {
		t.Fatalf("expected SemVer with leading v invalid")
	}
	if IsValidSemVer("1.0") {
		t.Fatalf("expected SemVer invalid")
	}
}

func TestVersionFieldMessageHelpers(t *testing.T) {
	if got := SemVerFieldMessage("version"); got != SemVerFormatMessage {
		t.Fatalf("unexpected version message: %s", got)
	}
	if got := SemVerFieldMessage("runtime_schema_version"); got != "runtime_schema_version must match MAJOR.MINOR.PATCH(-suffix or +suffix)" {
		t.Fatalf("unexpected runtime schema message: %s", got)
	}
}
