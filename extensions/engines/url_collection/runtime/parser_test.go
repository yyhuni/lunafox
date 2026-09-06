package urlcollectionruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

func TestParseHTTPXRecordPreservesInputAndOptionalMetadata(t *testing.T) {
	outcome := ParseHTTPXRecord(`{"input":"http://Example.COM:80/a","host":"example.com","status_code":999,"content_length":-1,"title":"ok\u0000","tech":[" nginx ","", "too-long-` + strings.Repeat("x", 101) + `"]}`)
	if outcome.Skip != "" || outcome.Endpoint.URL != "http://Example.COM:80/a" || outcome.Endpoint.Host != "example.com" || outcome.Endpoint.Title != "ok\x00" || outcome.Endpoint.StatusCode == nil || *outcome.Endpoint.StatusCode != 999 || outcome.Endpoint.ContentLength == nil || *outcome.Endpoint.ContentLength != -1 || len(outcome.Endpoint.Tech) != 3 || outcome.Endpoint.Tech[0] != " nginx " || outcome.Endpoint.Tech[1] != "" || !strings.HasPrefix(outcome.Endpoint.Tech[2], "too-long-") || outcome.RecoveredMetadata {
		t.Fatalf("outcome = %#v", outcome)
	}
}

func TestParseHTTPXRecordRejectsOnlyToolProtocolFailures(t *testing.T) {
	for input, want := range map[string]SkipReason{
		`not-json`: SkipMalformedJSON,
		`{"input":"https://example.com","host":3}`:                           SkipInvalidFieldType,
		`{"input":"https://example.com","host":"example.com","failed":true}`: SkipHTTPXFailed,
	} {
		if got := ParseHTTPXRecord(input).Skip; got != want {
			t.Fatalf("skip = %q, want %q", got, want)
		}
	}
	outcome := ParseHTTPXRecord(`{"input":"https://example.com","host":"other.example.com"}`)
	if outcome.Skip != "" || outcome.Endpoint.Host != "other.example.com" {
		t.Fatalf("host mismatch was rewritten or rejected: %#v", outcome)
	}
}

func TestParseHTTPXRecordRejectsInvalidUTF8BeforeJSONDecode(t *testing.T) {
	payload := append([]byte(`{"input":"https://example.com/`), 0xff)
	payload = append(payload, []byte(`","host":"example.com"}`)...)
	outcome := ParseHTTPXRecord(string(payload))
	if outcome.Skip != SkipMalformedJSON || outcome.Structural {
		t.Fatalf("invalid UTF-8 row = %#v, want non-structural malformed row", outcome)
	}
}

func TestParseHTTPXRecordRejectsUnpairedSurrogateBeforeJSONDecode(t *testing.T) {
	outcome := ParseHTTPXRecord(`{"input":"https://example.com/\uD800","host":"example.com"}`)
	if outcome.Skip != SkipMalformedJSON || outcome.Structural {
		t.Fatalf("unpaired-surrogate row = %#v, want non-structural malformed row", outcome)
	}
}

func TestParseHTTPXRecordKeepsStructuralEvidenceForRecoverableRowSkips(t *testing.T) {
	for _, input := range []string{
		`{"input":"https://example.com","host":"example.com","failed":true}`,
	} {
		outcome := ParseHTTPXRecord(input)
		if outcome.Skip == "" || !outcome.Structural {
			t.Fatalf("ParseHTTPXRecord(%s) = %#v, want structural skip", input, outcome)
		}
	}
	for _, input := range []string{`not-json`, `{"input":"https://example.com","host":3}`} {
		if ParseHTTPXRecord(input).Structural {
			t.Fatalf("ParseHTTPXRecord(%s) must not be structural", input)
		}
	}
	if outcome := ParseHTTPXRecord(`{"input":"not-a-url","host":"example.com"}`); outcome.Skip != "" || !outcome.Structural {
		t.Fatalf("candidate URL was treated as a Server validation error: %#v", outcome)
	}
}

func TestParseHTTPXRecordRejectsPresentNullAndWrongTypedFields(t *testing.T) {
	for _, input := range []string{
		`{"input":null,"host":"example.com"}`,
		`{"input":"https://example.com","host":"example.com","failed":null}`,
		`{"input":"https://example.com","host":"example.com","tech":["ok",2]}`,
		`{"input":"https://example.com","host":"example.com","status_code":200.5}`,
	} {
		want := SkipInvalidFieldType
		if input == `{"input":null,"host":"example.com"}` {
			want = SkipUnassociatedIdentity
		}
		if got := ParseHTTPXRecord(input).Skip; got != want {
			t.Fatalf("ParseHTTPXRecord(%s) skip = %q, want %q", input, got, want)
		}
	}
}

func TestParseHTTPXRecordMarksMissingInputAsUnassociatedIdentity(t *testing.T) {
	for _, input := range []string{
		`{"host":"example.com","failed":true}`,
		`{"input":"","host":"example.com"}`,
		`{"input":3,"host":"example.com"}`,
	} {
		if got := ParseHTTPXRecord(input).Skip; got != SkipUnassociatedIdentity {
			t.Fatalf("ParseHTTPXRecord(%s) skip = %q, want %q", input, got, SkipUnassociatedIdentity)
		}
	}
}

func TestParseHTTPXRecordPreservesTechNULWithoutMetadataRecovery(t *testing.T) {
	outcome := ParseHTTPXRecord(`{"input":"https://example.com","host":"example.com","tech":["ng\u0000inx"]}`)
	if outcome.Skip != "" || outcome.RecoveredMetadata || len(outcome.Endpoint.Tech) != 1 || outcome.Endpoint.Tech[0] != "ng\x00inx" {
		t.Fatalf("outcome = %#v", outcome)
	}
}

func TestParseHTTPXRecordPreservesResponseEvidenceAndOriginalRedirectInput(t *testing.T) {
	body := strings.Repeat("界", 1000)
	headers := strings.Repeat("界", 70_000)
	outcome := ParseHTTPXRecord(`{"input":"https://example.com/original","host":"example.com","status_code":302,"location":"https://outside.example/next","body":"` + body + `","raw_header":"` + headers + `"}`)
	if outcome.Skip != "" || outcome.Endpoint.URL != "https://example.com/original" || outcome.Endpoint.Location != "https://outside.example/next" {
		t.Fatalf("redirect evidence changed Endpoint identity: %#v", outcome)
	}
	if outcome.Endpoint.ResponseBody != body || outcome.Endpoint.ResponseHeaders != headers || outcome.Endpoint.ResponseBodyTruncated || outcome.Endpoint.ResponseHeadersTruncated {
		t.Fatalf("response evidence was changed or truncated: %#v", outcome.Endpoint)
	}
}

func TestParseHTTPXRecordClassifiesAllPresentWrongTypesAsUnrecoverable(t *testing.T) {
	for _, input := range []string{
		`{"input":"https://example.com","host":"example.com","title":null}`,
		`{"input":"https://example.com","host":"example.com","location":1}`,
		`{"input":"https://example.com","host":"example.com","webserver":false}`,
		`{"input":"https://example.com","host":"example.com","content_type":{}}`,
		`{"input":"https://example.com","host":"example.com","body":[]}`,
		`{"input":"https://example.com","host":"example.com","raw_header":1}`,
		`{"input":"https://example.com","host":"example.com","vhost":"false"}`,
	} {
		if outcome := ParseHTTPXRecord(input); outcome.Skip != SkipInvalidFieldType || outcome.Structural {
			t.Fatalf("ParseHTTPXRecord(%s) = %#v, want non-structural invalid-field-type", input, outcome)
		}
	}
}

func TestReadBoundedLinesDrainsOversizedRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", maxRecordBytes+1)+"\nnext\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := ReadBoundedLines(path, func(line string, oversized bool) error {
		if oversized {
			got = append(got, "oversized")
			return nil
		}
		got = append(got, line)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "oversized,next" {
		t.Fatalf("lines = %#v", got)
	}
}

func TestReadBoundedLinesAcceptsExactLimitAndMarksLimitPlusOne(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", maxRecordBytes)+"\n"+strings.Repeat("y", maxRecordBytes+1)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var oversized []bool
	var lengths []int
	if err := ReadBoundedLines(path, func(line string, over bool) error {
		oversized = append(oversized, over)
		lengths = append(lengths, len(line))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(oversized) != 2 || oversized[0] || !oversized[1] || lengths[0] != maxRecordBytes || lengths[1] != 0 {
		t.Fatalf("bounded records = oversize:%v lengths:%v", oversized, lengths)
	}
}

func TestInTargetScopeUsesURLIdentity(t *testing.T) {
	if !InTargetScope(enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, "https://api.example.com") || !InTargetScope(enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, "https://api.example.com/%zz") || InTargetScope(enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, "https://example.net") || !InTargetScope(enginecontract.Target{Type: enginecontract.TargetTypeCIDR, Value: "192.0.2.0/24"}, "http://192.0.2.4") || InTargetScope(enginecontract.Target{Type: enginecontract.TargetTypeCIDR, Value: "192.0.2.0/24"}, "http://192.0.3.4") {
		t.Fatal("scope identity check failed")
	}
}
