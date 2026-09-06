package results

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	contractresults "github.com/yyhuni/lunafox/engines/website_discovery/contract"
)

func TestStreamHTTPXWebsitesPreservesParseableRawObservations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "httpx.jsonl")
	content := "not-json\n" +
		`{"input":"https://Api.Example.COM:443/login","url":"https://Api.Example.COM:443/login","host":" Api.Example.COM. ","title":"API","status_code":200,"content_length":42,"content_type":"text/html","location":"https://example.com/next","webserver":"nginx","body":"hello","tech":[" nginx "],"vhost":true,"raw_header":"HTTP/1.1 200 OK"}` + "\n" +
		`{"input":"http://www.example.com:80/","url":"http://www.example.com:80/","host":"www.example.com","title":"WWW","status_code":301}` + "\n" +
		`{"input":"ftp://example.com","url":"ftp://example.com","status_code":200}` + "\n" +
		`{"input":"https://example.com","url":"https://example.com","status_code":999,"host":"other.example.com"}` + "\n" +
		`{"input":"https://skip.example.com","url":"https://skip.example.com","failed":true,"status_code":200}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}

	var items []contractresults.Website
	count, err := StreamHTTPXWebsites(context.Background(), path, func(item contractresults.Website) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamHTTPXWebsites failed: %v", err)
	}
	if count != 4 || len(items) != 4 {
		t.Fatalf("expected 4 parseable items, count=%d items=%#v", count, items)
	}
	if items[0].URL != "https://Api.Example.COM:443/login" || items[0].Host != " Api.Example.COM. " || items[0].Title != "API" {
		t.Fatalf("unexpected first item: %#v", items[0])
	}
	if items[0].StatusCode == nil || *items[0].StatusCode != 200 || items[0].ContentLength == nil || *items[0].ContentLength != 42 {
		t.Fatalf("unexpected first numeric fields: %#v", items[0])
	}
	if len(items[0].Tech) != 1 || items[0].Tech[0] != " nginx " || items[0].Vhost == nil || *items[0].Vhost != true {
		t.Fatalf("unexpected first metadata: %#v", items[0])
	}
	if items[1].URL != "http://www.example.com:80/" || items[1].Host != "www.example.com" || items[1].Title != "WWW" {
		t.Fatalf("unexpected input-identity item: %#v", items[1])
	}
	if items[2].URL != "ftp://example.com" || items[2].Host != "" || items[3].URL != "https://example.com" || items[3].Host != "other.example.com" {
		t.Fatalf("Server validation candidates were lost or changed: %#v", items)
	}
}

func TestStreamHTTPXWebsitesWithSummaryCountsMalformedInvalidAndFailedRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "httpx.jsonl")
	content := "not-json\n" +
		`{"input":"https://api.example.com","url":"https://api.example.com","host":"api.example.com","status_code":200}` + "\n" +
		`{"input":"ftp://example.com","url":"ftp://example.com","status_code":200}` + "\n" +
		`{"input":"https://invalid-status.example.com","url":"https://invalid-status.example.com","host":"invalid-status.example.com","status_code":999}` + "\n" +
		`{"input":"https://failed.example.com","url":"https://failed.example.com","failed":true}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}

	var items []contractresults.Website
	summary, err := StreamHTTPXWebsitesWithSummary(context.Background(), path, func(item contractresults.Website) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamHTTPXWebsitesWithSummary failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected three parseable items, got %#v", items)
	}
	if summary != (ParseSummary{SourceRecords: 5, ParsedItems: 3, SkippedMalformed: 1, SkippedFailed: 1}) {
		t.Fatalf("unexpected parse summary: %#v", summary)
	}
}

func TestStreamHTTPXWebsitesUsesInputURLBeforeRedirectURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "httpx.jsonl")
	content := `{"input":"http://Example.COM:80","url":"https://example.com","host":"example.com","status_code":200}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}

	var items []contractresults.Website
	count, err := StreamHTTPXWebsites(context.Background(), path, func(item contractresults.Website) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamHTTPXWebsites failed: %v", err)
	}
	if count != 1 || len(items) != 1 {
		t.Fatalf("expected 1 valid item, count=%d items=%#v", count, items)
	}
	if items[0].URL != "http://Example.COM:80" {
		t.Fatalf("expected input URL to be preserved before redirected url, got %#v", items[0])
	}
}

func TestStreamHTTPXWebsitesRejectsUnassociatedFinalURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "httpx.jsonl")
	content := `{"url":"https://redirected.example.com","host":"redirected.example.com","status_code":200}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}

	var items []contractresults.Website
	summary, err := StreamHTTPXWebsitesWithSummary(context.Background(), path, func(item contractresults.Website) error {
		items = append(items, item)
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "candidate input identity") {
		t.Fatalf("missing input identity did not fail the stream: %v", err)
	}
	if len(items) != 0 || summary.FatalIdentity != 1 {
		t.Fatalf("unassociated final URL was admitted: items=%#v summary=%#v", items, summary)
	}
}

func TestStreamHTTPXWebsitesFailsMissingInputEvenWhenOtherRowsAreMalformed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "httpx.jsonl")
	content := "not-json\n" + `{"url":"https://redirected.example.com","host":"redirected.example.com"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}
	_, err := StreamHTTPXWebsitesWithSummary(context.Background(), path, func(contractresults.Website) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "candidate input identity") {
		t.Fatalf("expected unassociated identity failure, got %v", err)
	}
}

func TestStreamHTTPXWebsitesPreservesResponseEvidenceWithLineBreaks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "httpx.jsonl")
	content := `{"input":"https://example.com","host":"example.com","status_code":403,"body":"challenge\r\ncontinue","raw_header":"HTTP/1.1 403 Forbidden\r\nServer: cloudflare\r\n"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}

	var items []contractresults.Website
	count, err := StreamHTTPXWebsites(context.Background(), path, func(item contractresults.Website) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamHTTPXWebsites failed: %v", err)
	}
	if count != 1 || len(items) != 1 {
		t.Fatalf("expected one website with response evidence, count=%d items=%#v", count, items)
	}
	if items[0].ResponseBody != "challenge\r\ncontinue" {
		t.Fatalf("unexpected response body: %q", items[0].ResponseBody)
	}
	if items[0].ResponseHeaders != "HTTP/1.1 403 Forbidden\r\nServer: cloudflare\r\n" {
		t.Fatalf("unexpected response headers: %q", items[0].ResponseHeaders)
	}
}

func TestStreamHTTPXWebsitesLeavesHostAndNumericValidationToServer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "httpx.jsonl")
	content := `{"input":"https://api.example.com","host":"bad host","status_code":999,"content_length":-1}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}

	var items []contractresults.Website
	summary, err := StreamHTTPXWebsitesWithSummary(context.Background(), path, func(item contractresults.Website) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamHTTPXWebsitesWithSummary failed: %v", err)
	}
	if len(items) != 1 || items[0].Host != "bad host" || items[0].StatusCode == nil || *items[0].StatusCode != 999 || items[0].ContentLength == nil || *items[0].ContentLength != -1 {
		t.Fatalf("Server validation candidate was changed or dropped: %#v", items)
	}
	if summary != (ParseSummary{SourceRecords: 1, ParsedItems: 1}) {
		t.Fatalf("unexpected parse summary: %#v", summary)
	}
}

func TestStreamHTTPXWebsitesRejectsInvalidUTF8BeforeJSONDecode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "httpx.jsonl")
	payload := append([]byte(`{"input":"https://example.com/`), 0xff)
	payload = append(payload, []byte(`","host":"example.com"}`)...)
	payload = append(payload, '\n')
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}

	var items []contractresults.Website
	summary, err := StreamHTTPXWebsitesWithSummary(context.Background(), path, func(item contractresults.Website) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamHTTPXWebsitesWithSummary failed: %v", err)
	}
	if len(items) != 0 || summary.SourceRecords != 1 || summary.SkippedMalformed != 1 {
		t.Fatalf("invalid UTF-8 record was admitted or misclassified: items=%#v summary=%#v", items, summary)
	}
}

func TestStreamHTTPXWebsitesRejectsUnpairedSurrogateBeforeJSONDecode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "httpx.jsonl")
	if err := os.WriteFile(path, []byte(`{"input":"https://example.com/\uD800","host":"example.com"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}

	summary, err := StreamHTTPXWebsitesWithSummary(context.Background(), path, func(contractresults.Website) error {
		t.Fatal("unpaired-surrogate record must not be submitted")
		return nil
	})
	if err != nil {
		t.Fatalf("StreamHTTPXWebsitesWithSummary failed: %v", err)
	}
	if summary != (ParseSummary{SourceRecords: 1, SkippedMalformed: 1}) {
		t.Fatalf("unpaired-surrogate record summary = %#v", summary)
	}
}

func TestStreamHTTPXWebsitesSkipsOversizedRecordAndContinues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "httpx.jsonl")
	content := strings.Repeat("x", maxResultArtifactRecordBytes+1) + "\n" +
		`{"input":"https://api.example.com","host":"api.example.com","status_code":200}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}

	var items []contractresults.Website
	summary, err := StreamHTTPXWebsitesWithSummary(context.Background(), path, func(item contractresults.Website) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamHTTPXWebsitesWithSummary failed: %v", err)
	}
	if len(items) != 1 || items[0].URL != "https://api.example.com" {
		t.Fatalf("unexpected submitted items: %#v", items)
	}
	if summary != (ParseSummary{SourceRecords: 2, ParsedItems: 1, SkippedOversized: 1}) {
		t.Fatalf("unexpected parse summary: %#v", summary)
	}
}

func TestStreamHTTPXWebsitesRejectsMissingFinalArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.jsonl")
	_, err := StreamHTTPXWebsites(context.Background(), path, func(contractresults.Website) error { return nil })
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected missing artifact error, got %v", err)
	}
}
