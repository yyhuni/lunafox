package directoryscanruntime

import (
	"context"
	"testing"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

func TestDenoiseAuthRedirectMatching(t *testing.T) {
	enabled := directoryDenoiseOptions{authRedirect: true}
	tests := []struct {
		name     string
		status   int
		url      string
		location string
		want     bool
	}{
		{name: "absolute login path", status: 302, url: "https://example.com/admin", location: "/login", want: true},
		{name: "child path", status: 301, url: "https://example.com/a", location: "https://example.com/auth/login/step", want: true},
		{name: "cross host sso", status: 302, url: "https://example.com/a", location: "https://sso.example.net/cas/login", want: true},
		{name: "relative login from root", status: 302, url: "https://example.com/", location: "login", want: true},
		{name: "relative login with query from root", status: 302, url: "https://example.com/", location: "signin?next=/a", want: true},
		{name: "relative login below directory is conservative", status: 302, url: "https://example.com/deep/page", location: "login", want: false},
		{name: "lookalike path", status: 302, url: "https://example.com/a", location: "/docs/login-guide/", want: false},
		{name: "ordinary redirect", status: 302, url: "https://example.com/a", location: "/admin/", want: false},
		{name: "non redirect status", status: 200, url: "https://example.com/a", location: "/login", want: false},
		{name: "missing location", status: 302, url: "https://example.com/a", location: "", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := enginecontract.Directory{URL: test.url, Status: test.status}
			if got := enabled.filteredByAuthRedirect(item, test.location); got != test.want {
				t.Fatalf("filteredByAuthRedirect() = %t, want %t", got, test.want)
			}
		})
	}
	if (directoryDenoiseOptions{}).filteredByAuthRedirect(enginecontract.Directory{URL: "https://example.com/a", Status: 302}, "/login") {
		t.Fatal("disabled auth denoise must keep the record")
	}
}

func TestDenoiseBackupHTMLMatching(t *testing.T) {
	enabled := directoryDenoiseOptions{backupHTML: true}
	tests := []struct {
		name        string
		url         string
		contentType string
		want        bool
	}{
		{name: "html zip", url: "https://example.com/backup.zip", contentType: "text/html; charset=utf-8", want: true},
		{name: "html sql", url: "https://example.com/db.sql", contentType: "Text/HTML", want: true},
		{name: "real zip", url: "https://example.com/backup.zip", contentType: "application/zip", want: false},
		{name: "plain sql", url: "https://example.com/db.sql", contentType: "text/plain", want: false},
		{name: "empty content type", url: "https://example.com/backup.zip", contentType: "", want: false},
		{name: "non backup suffix", url: "https://example.com/index.html", contentType: "text/html", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := enginecontract.Directory{URL: test.url, ContentType: test.contentType}
			if got := enabled.filteredByBackupHTML(item); got != test.want {
				t.Fatalf("filteredByBackupHTML() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestParseFFUFArtifactWithOptionsAppliesDenoise(t *testing.T) {
	records := [][]byte{
		marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/admin", Status: 302, ContentLength: 0, ContentType: "text/html", Duration: 1, RedirectLocation: "/login"}),
		marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/real", Status: 302, ContentLength: 0, ContentType: "text/html", Duration: 1, RedirectLocation: "/admin/"}),
		marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/x", Status: 302, ContentLength: 0, ContentType: "text/html", Duration: 1, RedirectLocation: "https://sso.example.net/auth/login"}),
		marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/", Status: 302, ContentLength: 0, ContentType: "text/html", Duration: 1, RedirectLocation: "login"}),
		marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/", Status: 302, ContentLength: 0, ContentType: "text/html", Duration: 1, RedirectLocation: "signin?next=/a"}),
		marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/backup.zip", Status: 200, ContentLength: 10, ContentType: "text/html", Duration: 1}),
		marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/backup.zip", Status: 200, ContentLength: 10, ContentType: "application/zip", Duration: 1}),
		marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/db.sql", Status: 200, ContentLength: 10, ContentType: "text/html", Duration: 1}),
		marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/db.sql", Status: 200, ContentLength: 10, ContentType: "text/plain", Duration: 1}),
		marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/docs/login-guide/", Status: 200, ContentLength: 10, ContentType: "text/html", Duration: 1}),
	}
	payload := joinFFUFRecords(records)

	t.Run("disabled keeps every valid record", func(t *testing.T) {
		visited := 0
		summary, err := ParseFFUFArtifactWithOptions(context.Background(), writeRawFFUFArtifact(t, payload), 0, directoryDenoiseOptions{}, func(DirectoryObservation) error {
			visited++
			return nil
		})
		if err != nil {
			t.Fatalf("ParseFFUFArtifactWithOptions() error = %v", err)
		}
		if visited != len(records) || summary.ParsedItems != uint64(len(records)) {
			t.Fatalf("visited=%d parsed=%d, want %d", visited, summary.ParsedItems, len(records))
		}
		if summary.FilteredAuthRedirect != 0 || summary.FilteredBackupHTML != 0 {
			t.Fatalf("unexpected filters: %#v", summary)
		}
	})

	t.Run("enabled drops only strict matches", func(t *testing.T) {
		visited := 0
		summary, err := ParseFFUFArtifactWithOptions(
			context.Background(),
			writeRawFFUFArtifact(t, payload),
			0,
			directoryDenoiseOptions{authRedirect: true, backupHTML: true},
			func(DirectoryObservation) error {
				visited++
				return nil
			},
		)
		if err != nil {
			t.Fatalf("ParseFFUFArtifactWithOptions() error = %v", err)
		}
		if summary.FilteredAuthRedirect != 4 || summary.FilteredBackupHTML != 2 {
			t.Fatalf("filtered counts = %#v", summary)
		}
		if visited != len(records)-6 || summary.ParsedItems != uint64(len(records)-6) {
			t.Fatalf("visited=%d parsed=%d, want %d", visited, summary.ParsedItems, len(records)-6)
		}
		if summary.AllRecordsRejected() {
			t.Fatal("partial filtering must not be a rejection")
		}
	})

	t.Run("all filtered is not a rejection", func(t *testing.T) {
		onlyFiltered := joinFFUFRecords([][]byte{
			marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/admin", Status: 302, ContentLength: 0, ContentType: "text/html", Duration: 1, RedirectLocation: "/login"}),
		})
		summary, err := ParseFFUFArtifactWithOptions(
			context.Background(),
			writeRawFFUFArtifact(t, onlyFiltered),
			0,
			directoryDenoiseOptions{authRedirect: true},
			func(DirectoryObservation) error { t.Fatal("filtered record must not be visited"); return nil },
		)
		if err != nil {
			t.Fatalf("ParseFFUFArtifactWithOptions() error = %v", err)
		}
		if summary.AllRecordsRejected() {
			t.Fatal("filtered-only run must not be reported as rejected")
		}
		if summary.FilteredAuthRedirect != 1 || summary.ParsedItems != 0 {
			t.Fatalf("summary = %#v", summary)
		}
	})
}

func joinFFUFRecords(records [][]byte) []byte {
	var payload []byte
	for _, record := range records {
		payload = append(payload, record...)
		payload = append(payload, '\n')
	}
	return payload
}
