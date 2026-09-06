package application

import (
	"strings"
	"testing"
)

func TestParseNucleiTemplateExtractsMetadataWithoutPersistingYAML(t *testing.T) {
	raw := []byte(`id: example-template

info:
  name: Example template
  author: lunafox
  severity: HIGH
  tags: cve,headers
  description: detects an example condition
  classification:
    cve-id: CVE-2024-0001
    cwe-id: CWE-79
  reference:
    - https://example.com/advisory

http:
  - method: GET
    path:
      - "{{BaseURL}}/"
`)
	parsed, err := parseNucleiTemplate(raw)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}
	if parsed.ID != "example-template" || parsed.Severity != "high" || len(parsed.Tags) != 2 || len(parsed.CVE) != 1 || len(parsed.References) != 1 {
		t.Fatalf("parsed metadata=%+v", parsed)
	}
}

func TestParseNucleiTemplateRejectsMalformedSemanticShapes(t *testing.T) {
	cases := map[string]string{
		"empty":            "",
		"missing id":       "info:\n  name: x\nhttp: []\n",
		"missing info":     "id: x\nhttp: []\n",
		"bad severity":     "id: x\ninfo:\n  name: x\n  severity: extreme\nhttp: []\n",
		"missing protocol": "id: x\ninfo:\n  name: x\n  severity: low\n",
		"multiple docs":    "id: x\ninfo:\n  name: x\nhttp: []\n---\nid: y\n",
		"NUL in id":        "id: \"x\\0id\"\ninfo:\n  name: x\nhttp: []\n",
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseNucleiTemplate([]byte(value)); err == nil {
				t.Fatal("expected template to be rejected")
			}
		})
	}
}

func TestParseNucleiTemplateEscapesNULMetadataForPersistence(t *testing.T) {
	raw := []byte(`id: nul-metadata-template
info:
  name: "Name \0 marker"
  author: "author \0 marker"
  severity: high
  tags:
    - "tag-\0-marker"
  description: "Description \0 marker"
  remediation: "Remediation \0 marker"
  classification:
    cve-id: "CVE-2024-0001\0"
    cwe-id:
      - "CWE-79\0"
  reference:
    - "https://example.com/\0"
http:
  - method: GET
    path:
      - "{{BaseURL}}/"
`)
	parsed, err := parseNucleiTemplate(raw)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}
	for field, value := range map[string]string{
		"name":        parsed.Name,
		"author":      parsed.Author,
		"description": parsed.Description,
		"remediation": parsed.Remediation,
	} {
		if strings.ContainsRune(value, '\x00') {
			t.Fatalf("%s still contains NUL byte: %q", field, value)
		}
	}
	if parsed.Name != `Name \u0000 marker` || parsed.Author != `author \u0000 marker` || parsed.Description != `Description \u0000 marker` || parsed.Remediation != `Remediation \u0000 marker` {
		t.Fatalf("parsed scalar metadata = %+v, want escaped NUL markers", parsed)
	}
	assertEscapedNULStrings(t, "tags", parsed.Tags, []string{`tag-\u0000-marker`})
	assertEscapedNULStrings(t, "CVE", parsed.CVE, []string{`CVE-2024-0001\u0000`})
	assertEscapedNULStrings(t, "CWE", parsed.CWE, []string{`CWE-79\u0000`})
	assertEscapedNULStrings(t, "references", parsed.References, []string{`https://example.com/\u0000`})
	if parsed.ID != "nul-metadata-template" {
		t.Fatalf("template ID = %q, want unchanged identity", parsed.ID)
	}
}

func assertEscapedNULStrings(t *testing.T, field string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
	for index := range want {
		if strings.ContainsRune(got[index], '\x00') || got[index] != want[index] {
			t.Fatalf("%s[%d] = %q, want %q without a NUL byte", field, index, got[index], want[index])
		}
	}
}
