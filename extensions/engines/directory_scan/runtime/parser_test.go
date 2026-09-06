package directoryscanruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type ffufRecordFixture struct {
	URL           string `json:"url"`
	Status        int64  `json:"status"`
	ContentLength int64  `json:"length"`
	ContentType   string `json:"content-type"`
	Duration      int64  `json:"duration"`
	Position      int64  `json:"position"`
}

func TestParseFFUFArtifactSupportsLFCRLFAndIncompleteFinalRecord(t *testing.T) {
	records := [][]byte{
		marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/a", Status: 200, ContentLength: 1, ContentType: "text/plain", Duration: 10}),
		marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/b", Status: 201, ContentLength: 2, ContentType: "", Duration: 20}),
		marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/c", Status: 0, ContentLength: 0, ContentType: "application/x-test", Duration: 0}),
	}
	payload := append(append(append([]byte{}, records[0]...), '\n'), records[1]...)
	payload = append(payload, '\r', '\n')
	payload = append(payload, records[2]...)
	path := writeRawFFUFArtifact(t, payload)

	var observations []DirectoryObservation
	summary, err := ParseFFUFArtifact(context.Background(), path, 7, func(observation DirectoryObservation) error {
		observations = append(observations, observation)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseFFUFArtifact() error = %v", err)
	}
	if summary != (FFUFParseSummary{SourceRecords: 3, ParsedItems: 3}) {
		t.Fatalf("parse summary = %#v", summary)
	}
	for index, observation := range observations {
		if observation.CandidateOrdinal != 7 || observation.PhysicalRecordOrdinal != uint64(index) {
			t.Fatalf("observation[%d] ordinals = %#v", index, observation)
		}
	}
	if got := []string{observations[0].Item.URL, observations[1].Item.URL, observations[2].Item.URL}; !reflect.DeepEqual(got, []string{"https://example.com/a", "https://example.com/b", "https://example.com/c"}) {
		t.Fatalf("parsed URLs = %#v", got)
	}
}

func TestParseFFUFArtifactAcceptsExactLimitAndRealignsAfterOversizedRecord(t *testing.T) {
	exact := paddedFFUFRecord(t, maximumFFUFPhysicalRecordBytes)
	oversized := paddedFFUFRecord(t, maximumFFUFPhysicalRecordBytes+1)
	last := marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/last", Status: 999, ContentLength: math.MaxInt64, ContentType: "", Duration: math.MaxInt64})
	payload := make([]byte, 0, len(exact)+len(oversized)+len(last)+3)
	payload = append(payload, exact...)
	payload = append(payload, '\n')
	payload = append(payload, oversized...)
	payload = append(payload, '\n')
	payload = append(payload, last...)
	payload = append(payload, '\n')

	var observations []DirectoryObservation
	summary, err := ParseFFUFArtifact(context.Background(), writeRawFFUFArtifact(t, payload), 3, func(observation DirectoryObservation) error {
		observations = append(observations, observation)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseFFUFArtifact() error = %v", err)
	}
	if summary != (FFUFParseSummary{SourceRecords: 3, ParsedItems: 2, OversizedRecords: 1}) {
		t.Fatalf("parse summary = %#v", summary)
	}
	if len(observations) != 2 || observations[0].PhysicalRecordOrdinal != 0 || observations[1].PhysicalRecordOrdinal != 2 {
		t.Fatalf("observations = %#v", observations)
	}
}

func TestParseFFUFArtifactCountsMalformedInvalidOversizedAndValidIndependently(t *testing.T) {
	invalid := []byte(`{"url":"","status":200,"length":1,"content-type":"text/plain","duration":1}`)
	valid := marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/ok", Status: 200, ContentLength: 1, ContentType: "text/plain", Duration: 1})
	payload := []byte("not-json\n")
	payload = append(payload, invalid...)
	payload = append(payload, '\n')
	payload = append(payload, bytes.Repeat([]byte{'x'}, maximumFFUFPhysicalRecordBytes+1)...)
	payload = append(payload, '\n')
	payload = append(payload, valid...)
	payload = append(payload, '\n')

	var observations []DirectoryObservation
	summary, err := ParseFFUFArtifact(context.Background(), writeRawFFUFArtifact(t, payload), 9, func(observation DirectoryObservation) error {
		observations = append(observations, observation)
		return nil
	})
	if err != nil {
		t.Fatalf("ParseFFUFArtifact() error = %v", err)
	}
	want := FFUFParseSummary{SourceRecords: 4, ParsedItems: 2, MalformedRecords: 1, OversizedRecords: 1}
	if summary != want || summary.AllRecordsRejected() {
		t.Fatalf("parse summary = %#v, want %#v", summary, want)
	}
	if len(observations) != 2 || observations[0].PhysicalRecordOrdinal != 1 || observations[1].PhysicalRecordOrdinal != 3 || observations[0].Item.URL != "" {
		t.Fatalf("observations = %#v", observations)
	}
}

func TestFFUFParseSummaryDistinguishesTrueZeroFromAllRejected(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload []byte
		want    FFUFParseSummary
		allBad  bool
	}{
		{name: "true zero"},
		{name: "all malformed", payload: []byte("not-json\n"), want: FFUFParseSummary{SourceRecords: 1, MalformedRecords: 1}, allBad: true},
		{name: "all invalid", payload: []byte(`{"url":null,"status":200,"length":1,"content-type":"","duration":1}` + "\n"), want: FFUFParseSummary{SourceRecords: 1, InvalidRecords: 1}, allBad: true},
		{name: "all oversized", payload: append(bytes.Repeat([]byte{'x'}, maximumFFUFPhysicalRecordBytes+1), '\n'), want: FFUFParseSummary{SourceRecords: 1, OversizedRecords: 1}, allBad: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			summary, err := ParseFFUFArtifact(context.Background(), writeRawFFUFArtifact(t, test.payload), 0, func(DirectoryObservation) error { return nil })
			if err != nil {
				t.Fatalf("ParseFFUFArtifact() error = %v", err)
			}
			if summary != test.want || summary.AllRecordsRejected() != test.allBad {
				t.Fatalf("parse summary = %#v allRejected=%t, want %#v allRejected=%t", summary, summary.AllRecordsRejected(), test.want, test.allBad)
			}
		})
	}
}

func TestDecodeFFUFRecordEnforcesCompleteDirectoryBoundaries(t *testing.T) {
	maximum := ffufRecordFixture{
		URL:           directoryURLAtByteLimit(2000),
		Status:        999,
		ContentLength: math.MaxInt64,
		ContentType:   strings.Repeat("t", 1024),
		Duration:      math.MaxInt64,
		Position:      123,
	}
	item, outcome := decodeFFUFRecord(marshalFFUFRecord(t, maximum))
	if outcome != ffufRecordValid || item.URL != maximum.URL || item.Status != 999 ||
		item.ContentLength != math.MaxInt64 || item.ContentType != maximum.ContentType || item.Duration != math.MaxInt64 {
		t.Fatalf("maximum record = %#v outcome=%d", item, outcome)
	}

	zero, outcome := decodeFFUFRecord([]byte(`{"url":"https://example.com/%00","status":0,"length":0,"content-type":"","duration":0,"extra":true}`))
	if outcome != ffufRecordValid || zero.URL != "https://example.com/%00" || zero.Status != 0 || zero.ContentLength != 0 || zero.Duration != 0 {
		t.Fatalf("zero record = %#v outcome=%d", zero, outcome)
	}

	replacement, outcome := decodeFFUFRecord([]byte(`{"url":"https://example.com/\uFFFD","status":200,"length":1,"content-type":"","duration":1}`))
	if outcome != ffufRecordValid || replacement.URL != "https://example.com/\uFFFD" {
		t.Fatalf("literal replacement-rune record = %#v outcome=%d", replacement, outcome)
	}
}

func TestDecodeFFUFRecordRejectsMalformedAndInvalidForms(t *testing.T) {
	valid := `{"url":"https://example.com","status":200,"length":1,"content-type":"text/plain","duration":1}`
	for _, raw := range []string{"", "not-json", valid + valid, `{"url":`} {
		if _, outcome := decodeFFUFRecord([]byte(raw)); outcome != ffufRecordMalformed {
			t.Fatalf("decodeFFUFRecord(%q) outcome = %d, want malformed", raw, outcome)
		}
	}

	invalidUTF8 := append([]byte(`{"url":"https://example.com/`), 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`","status":200,"length":1,"content-type":"","duration":1}`)...)
	invalid := []string{
		`[]`, `null`,
		`{"url":"https://example.com","status":200,"length":1,"content-type":"text/plain"}`,
		`{"url":null,"status":200,"length":1,"content-type":"text/plain","duration":1}`,
		`{"url":"https://example.com","status":"200","length":1,"content-type":"text/plain","duration":1}`,
		`{"url":"https://example.com","status":200,"length":null,"content-type":"text/plain","duration":1}`,
		`{"url":"https://example.com","status":200,"length":1,"content-type":null,"duration":1}`,
		`{"url":"https://example.com","status":200,"length":1,"content-type":"text/plain","duration":null}`,
		`{"url":"https://example.com/\uD800","status":200,"length":1,"content-type":"text/plain","duration":1}`,
		`{"url":"u","status":200,"length":9223372036854775808,"content-type":"","duration":1}`,
		`{"url":"u","status":200,"length":1.0,"content-type":"","duration":1}`,
		`{"url":"u","status":200,"length":1,"content-type":"","duration":9223372036854775808}`,
		`{"url":"u","url":"v","status":200,"length":1,"content-type":"","duration":1}`,
	}
	for _, raw := range invalid {
		if _, outcome := decodeFFUFRecord([]byte(raw)); outcome != ffufRecordInvalid {
			t.Fatalf("decodeFFUFRecord(%q) outcome = %d, want invalid", raw, outcome)
		}
	}
	if _, outcome := decodeFFUFRecord(invalidUTF8); outcome != ffufRecordInvalid {
		t.Fatalf("invalid UTF-8 outcome = %d, want invalid", outcome)
	}
}

func TestDecodeFFUFRecordLeavesSemanticAcceptanceToResultEncoder(t *testing.T) {
	record := fmt.Sprintf(`{"url":"","status":1000,"length":-1,"content-type":"%s","duration":-2}`, strings.Repeat("t", 1025))
	item, outcome := decodeFFUFRecord([]byte(record))
	if outcome != ffufRecordValid || item.URL != "" || item.Status != 1000 || item.ContentLength != -1 || item.ContentType != strings.Repeat("t", 1025) || item.Duration != -2 {
		t.Fatalf("semantic values were rejected or changed: %#v outcome=%d", item, outcome)
	}
}

func TestParseFFUFArtifactPropagatesContextCallbackAndIOFailures(t *testing.T) {
	valid := marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com", Status: 200, ContentLength: 1, ContentType: "", Duration: 1})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ParseFFUFArtifact(ctx, writeRawFFUFArtifact(t, append(valid, '\n')), 0, func(DirectoryObservation) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled parse error = %v", err)
	}

	wantCallbackErr := errors.New("callback failed")
	summary, err := ParseFFUFArtifact(context.Background(), writeRawFFUFArtifact(t, append(valid, '\n')), 0, func(DirectoryObservation) error {
		return wantCallbackErr
	})
	if !errors.Is(err, wantCallbackErr) || summary.SourceRecords != 1 || summary.ParsedItems != 0 {
		t.Fatalf("callback parse summary=%#v error=%v", summary, err)
	}

	wantReadErr := errors.New("read failed")
	if _, err := parseFFUFStream(context.Background(), errorReader{err: wantReadErr}, 0, func(DirectoryObservation) error { return nil }); !errors.Is(err, wantReadErr) {
		t.Fatalf("stream read error = %v", err)
	}

	missing := filepath.Join(t.TempDir(), "missing.jsonl")
	if _, err := ParseFFUFArtifact(context.Background(), missing, 0, func(DirectoryObservation) error { return nil }); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing artifact error = %v", err)
	}
	if _, err := ParseFFUFArtifact(context.Background(), t.TempDir(), 0, func(DirectoryObservation) error { return nil }); err == nil {
		t.Fatal("directory artifact was accepted")
	}
}

type errorReader struct {
	err error
}

func (reader errorReader) Read([]byte) (int, error) { return 0, reader.err }

func marshalFFUFRecord(t *testing.T, record ffufRecordFixture) []byte {
	t.Helper()
	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func paddedFFUFRecord(t *testing.T, size int) []byte {
	t.Helper()
	prefix := []byte(`{"url":"https://example.com/","status":200,"length":1,"content-type":"","duration":1,"padding":"`)
	suffix := []byte(`"}`)
	if size < len(prefix)+len(suffix) {
		t.Fatalf("padded FFUF record size %d is too small", size)
	}
	payload := make([]byte, 0, size)
	payload = append(payload, prefix...)
	payload = append(payload, bytes.Repeat([]byte{'x'}, size-len(prefix)-len(suffix))...)
	payload = append(payload, suffix...)
	return payload
}

func directoryURLAtByteLimit(limit int) string {
	const prefix = "https://example.com/"
	return prefix + strings.Repeat("u", limit-len(prefix))
}

func writeRawFFUFArtifact(t *testing.T, payload []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffuf.jsonl")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
