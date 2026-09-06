package results

import (
	"strings"
	"testing"
)

func TestWebsiteTechnologyContractIsClosedAndCanonical(t *testing.T) {
	item := WebsiteTechnology{URL: "https://example.com/path?b=2&a=1#fragment", Tech: []string{"nginx", "Go"}}
	payload, err := EncodeWebsiteTechnology(item)
	if err != nil || payload != `{"url":"https://example.com/path?b=2\u0026a=1#fragment","tech":["nginx","Go"]}` {
		t.Fatalf("EncodeWebsiteTechnology() = %q, %v", payload, err)
	}
	decoded, err := DecodeWebsiteTechnologyItems([]string{payload})
	if err != nil || len(decoded) != 1 || decoded[0].URL != item.URL {
		t.Fatalf("DecodeWebsiteTechnologyItems() = %#v, %v", decoded, err)
	}
	if _, err := DecodeWebsiteTechnologyItems([]string{`{"url":"https://example.com","tech":[]}`}); err != nil {
		t.Fatalf("explicit empty tech must be accepted: %v", err)
	}
}

func TestWebsiteTechnologyRejectsMissingNullUnknownAndWrongFields(t *testing.T) {
	for _, payload := range []string{
		`{"url":"https://example.com"}`,
		`{"url":"https://example.com","tech":null}`,
		`{"url":"https://example.com","tech":[],"host":"example.com"}`,
		`{"url":"https://example.com","tech":"nginx"}`,
		`{"url":"https://example.com","tech":[null]}`,
		`{"url":"https://example.com","tech":[],"tech":[]}`,
		`{"url":"https://example.com","tech":[] ,"URL":"https://example.com"}`,
	} {
		if _, err := DecodeWebsiteTechnologyItems([]string{payload}); err == nil {
			t.Fatalf("expected strict rejection for %s", payload)
		}
	}
}

func TestWebsiteTechnologyPreservesValuesAndCountsUnicodeCodePoints(t *testing.T) {
	valid := strings.Repeat("界", 100)
	decoded, err := DecodeWebsiteTechnologyItems([]string{`{"url":"https://example.com","tech":["` + valid + `"]}`})
	if err != nil || decoded[0].Tech[0] != valid {
		t.Fatalf("100-code-point value was not preserved: %#v, %v", decoded, err)
	}
	for _, value := range []string{" ", "\t", "bad\x00value", strings.Repeat("界", 101)} {
		if _, err := EncodeWebsiteTechnology(WebsiteTechnology{URL: "https://example.com", Tech: []string{value}}); err == nil {
			t.Fatalf("expected invalid technology value %q", value)
		}
	}
	kept := "  nginx  "
	encoded, err := EncodeWebsiteTechnology(WebsiteTechnology{URL: "https://example.com", Tech: []string{kept}})
	if err != nil || !strings.Contains(encoded, kept) {
		t.Fatalf("valid technology string was unexpectedly trimmed: %q, %v", encoded, err)
	}
}

func TestWebsiteTechnologyUsesObservedURLAdmission(t *testing.T) {
	for _, url := range []string{" https://example.com", "ftp://example.com", "https://example.com/\n"} {
		if _, err := EncodeWebsiteTechnology(WebsiteTechnology{URL: url, Tech: []string{}}); err == nil {
			t.Fatalf("expected observed URL rejection for %q", url)
		}
	}
	for _, url := range []string{"https://example.com:443", "HTTPS://example.com/%zz"} {
		if _, err := EncodeWebsiteTechnology(WebsiteTechnology{URL: url, Tech: []string{}}); err != nil {
			t.Fatalf("expected raw URL acceptance for %q: %v", url, err)
		}
	}
}
