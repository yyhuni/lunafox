package results

import (
	"bytes"
	"strings"
	"testing"
)

func observedURLAtByteLimit(limit int) string {
	prefix := "https://example.com/"
	return prefix + strings.Repeat("x", limit-len(prefix))
}

func TestValidateObservedAssetURLPreservesAcceptedBytes(t *testing.T) {
	valid := []string{
		"HTTPS://EXAMPLE.com:443/a%2Fb?b=2&a=1#fragment",
		"https://example.com/?x=%00",
		"https://example.com/?x=%0d%0aInjected",
		"https://example.com/%zz",
		observedURLAtByteLimit(ObservedAssetURLMaxBytes),
	}
	for _, raw := range valid {
		t.Run(raw, func(t *testing.T) {
			got, err := ValidateObservedAssetURL(raw)
			if err != nil {
				t.Fatalf("ValidateObservedAssetURL(%q): %v", raw, err)
			}
			if got != raw {
				t.Fatalf("ValidateObservedAssetURL() = %q, want unchanged %q", got, raw)
			}
		})
	}
}

func TestValidateObservedAssetURLRejectsOnlyTransportUnsafeValues(t *testing.T) {
	invalidUTF8 := string(append([]byte("https://example.com/"), 0xff))
	for name, raw := range map[string]string{
		"empty":           "",
		"trimmed":         " https://example.com",
		"wrong scheme":    "ftp://example.com",
		"actual NUL":      "https://example.com/\x00payload",
		"actual CR":       "https://example.com/\rpayload",
		"actual LF":       "https://example.com/\npayload",
		"invalid UTF-8":   invalidUTF8,
		"over byte limit": observedURLAtByteLimit(ObservedAssetURLMaxBytes + 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ValidateObservedAssetURL(raw); err == nil {
				t.Fatalf("ValidateObservedAssetURL(%q) unexpectedly succeeded", raw)
			}
		})
	}
}

func TestDeriveObservedAssetURLHostDoesNotParsePayload(t *testing.T) {
	raw := "HTTPS://Api.Example.COM:443/%zz?x=%0d%0a#fragment"
	host, err := DeriveObservedAssetURLHost(raw)
	if err != nil {
		t.Fatalf("DeriveObservedAssetURLHost(%q): %v", raw, err)
	}
	if host != "api.example.com" {
		t.Fatalf("derived Host = %q, want api.example.com", host)
	}
}

func TestDeriveObservedAssetURLHostRejectsAmbiguousAuthority(t *testing.T) {
	for name, raw := range map[string]string{
		"missing host":        "https:///payload",
		"multiple separators": "https://example.com:443:444/payload",
		"unsupported IPv6":    "https://[2001:db8::1]/payload",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DeriveObservedAssetURLHost(raw); err == nil {
				t.Fatalf("DeriveObservedAssetURLHost(%q) unexpectedly succeeded", raw)
			}
		})
	}
}

func TestReadObservedAssetURLLinesPreservesAcceptedRecords(t *testing.T) {
	values := []string{
		"HTTPS://EXAMPLE.com:443/a%2Fb?b=2&a=1#fragment",
		"https://example.com/?x=%0d%0aInjected",
		"https://example.com/%zz",
	}
	var got []string
	err := ReadObservedAssetURLLines(bytes.NewBufferString(strings.Join(values, "\n")+"\n"), func(raw string) error {
		got = append(got, raw)
		return nil
	})
	if err != nil {
		t.Fatalf("ReadObservedAssetURLLines() error = %v", err)
	}
	if strings.Join(got, "\n") != strings.Join(values, "\n") {
		t.Fatalf("records = %#v, want %#v", got, values)
	}
}

func TestReadObservedAssetURLLinesRejectsUnrecoverableLineFraming(t *testing.T) {
	for name, payload := range map[string]string{
		"CRLF":            "https://example.com/\r\n",
		"unterminated":    "https://example.com/",
		"empty record":    "\n",
		"overlong record": observedURLAtByteLimit(ObservedAssetURLMaxBytes+1) + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			if err := ReadObservedAssetURLLines(bytes.NewBufferString(payload), func(string) error { return nil }); err == nil {
				t.Fatalf("ReadObservedAssetURLLines(%q) unexpectedly succeeded", payload)
			}
		})
	}
}
