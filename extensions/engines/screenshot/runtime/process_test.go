package screenshotruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func installHTTPXFixture(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("HTTPX process fixture requires a POSIX executable")
	}
	directory := t.TempDir()
	path := filepath.Join(directory, "httpx")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nset -eu\n"+script+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestProcessHTTPXStreamsRowsAndRejectsNonZeroExit(t *testing.T) {
	installHTTPXFixture(t, `printf '%s\n' '{"url":"https://example.com/"}'`)
	command := httpxCommand{timeout: time.Second}
	var rows []string
	if err := (processHTTPX{}).Run(context.Background(), command, func(payload []byte) error {
		rows = append(rows, string(payload))
		return nil
	}); err != nil || len(rows) != 1 || rows[0] != `{"url":"https://example.com/"}` {
		t.Fatalf("HTTPX stream = %#v/%v", rows, err)
	}

	installHTTPXFixture(t, "exit 7")
	if err := (processHTTPX{}).Run(context.Background(), command, func([]byte) error { return nil }); err == nil {
		t.Fatal("non-zero HTTPX exit was accepted")
	}
}

func TestProcessHTTPXCleanExitDoesNotReportClosedStderr(t *testing.T) {
	installHTTPXFixture(t, `printf '%s\n' '{"url":"https://example.com/"}'`)
	command := httpxCommand{timeout: time.Second}
	for attempt := 0; attempt < 32; attempt++ {
		var rows []string
		if err := (processHTTPX{}).Run(context.Background(), command, func(payload []byte) error {
			rows = append(rows, string(payload))
			return nil
		}); err != nil {
			t.Fatalf("clean HTTPX exit attempt %d returned error: %v", attempt, err)
		}
		if len(rows) != 1 || rows[0] != `{"url":"https://example.com/"}` {
			t.Fatalf("HTTPX stream attempt %d = %#v", attempt, rows)
		}
	}
}

func TestProcessHTTPXFailsOnStderrLimit(t *testing.T) {
	installHTTPXFixture(t, "dd if=/dev/zero bs=1048577 count=1 >&2 2>/dev/null")
	err := (processHTTPX{}).Run(context.Background(), httpxCommand{timeout: time.Second}, func([]byte) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "stderr") {
		t.Fatalf("stderr overflow error = %v", err)
	}
}

func TestProcessHTTPXPropagatesCallbackAndCancellation(t *testing.T) {
	installHTTPXFixture(t, `printf '%s\n' '{"url":"https://example.com/"}'; sleep 1`)
	command := httpxCommand{timeout: 2 * time.Second}
	want := errors.New("stop row processing")
	if err := (processHTTPX{}).Run(context.Background(), command, func([]byte) error { return want }); !errors.Is(err, want) {
		t.Fatalf("callback error = %v, want %v", err, want)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := (processHTTPX{}).Run(ctx, command, func([]byte) error { return nil })
	if err == nil || !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("cancellation error = %v, context = %v", err, ctx.Err())
	}
}

func TestDecodeHTTPXRowRejectsCriticalShapeErrors(t *testing.T) {
	for _, payload := range []string{
		`[]`,
		`{"url":"https://example.com/","url":"https://other.example/"}`,
		`{"url":"https://example.com/","failed":"yes"}`,
		`{"url":"https://example.com/","screenshot_path":null}`,
		`{"url":"https://example.com/"} trailing`,
	} {
		if _, err := decodeHTTPXRow([]byte(payload)); err == nil {
			t.Fatalf("invalid HTTPX row accepted: %s", payload)
		}
	}
}

func TestDecodeHTTPXRowRejectsInvalidUTF8BeforeJSONDecode(t *testing.T) {
	payload := append([]byte(`{"url":"https://example.com/`), 0xff)
	payload = append(payload, []byte(`"}`)...)
	if _, err := decodeHTTPXRow(payload); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("invalid UTF-8 HTTPX row error = %v", err)
	}
}

func TestDecodeHTTPXRowRejectsUnpairedSurrogateBeforeJSONDecode(t *testing.T) {
	if _, err := decodeHTTPXRow([]byte(`{"url":"https://example.com/\uD800"}`)); err == nil || !strings.Contains(err.Error(), "unpaired Unicode surrogate") {
		t.Fatalf("unpaired-surrogate HTTPX row error = %v", err)
	}
}

func TestDecodeHTTPXRowOmitsInvalidOptionalStatusEvidence(t *testing.T) {
	for _, status := range []string{"null", `"200"`, "200.5", "true", "[]", "{}", "700"} {
		row, err := decodeHTTPXRow([]byte(`{"url":"https://example.com/","status_code":` + status + `}`))
		if err != nil {
			t.Fatalf("status_code %s rejected the row: %v", status, err)
		}
		if row.StatusCode != nil {
			t.Fatalf("status_code %s was retained: %v", status, *row.StatusCode)
		}
	}
	row, err := decodeHTTPXRow([]byte(`{"url":"https://example.com/","status_code":200}`))
	if err != nil || row.StatusCode == nil || *row.StatusCode != 200 {
		t.Fatalf("valid status_code was not retained: %#v/%v", row.StatusCode, err)
	}
}

func TestDecodeHTTPXRowUsesObservedURLAcrossRedirects(t *testing.T) {
	row, err := decodeHTTPXRow([]byte(`{"input":"http://example.com/","url":"https://outside.example/","status_code":302}`))
	if err != nil {
		t.Fatalf("redirected HTTPX row rejected: %v", err)
	}
	if row.Identity != "https://outside.example/" {
		t.Fatalf("redirected HTTPX row identity = %q, want observed URL", row.Identity)
	}
	if row.StatusCode == nil || *row.StatusCode != 302 {
		t.Fatalf("redirected HTTPX status = %#v, want 302", row.StatusCode)
	}
}

func TestDecodeHTTPXRowAllowsMissingInputButRequiresObservedURL(t *testing.T) {
	for _, payload := range []string{
		`{"url":"https://example.com/"}`,
		`{"input":null,"url":"https://example.com/"}`,
		`{"input":"","url":"https://example.com/"}`,
		`{"input":3,"url":"https://example.com/"}`,
	} {
		if _, err := decodeHTTPXRow([]byte(payload)); err != nil {
			t.Fatalf("payload %s rejected despite valid observed URL: %v", payload, err)
		}
	}
	for _, payload := range []string{
		`{"input":"https://example.com/"}`,
		`{"input":"https://example.com/","url":null}`,
		`{"input":"https://example.com/","url":""}`,
		`{"input":"https://example.com/","url":3}`,
	} {
		if _, err := decodeHTTPXRow([]byte(payload)); !errors.Is(err, errHTTPXRowMissingURL) {
			t.Fatalf("payload %s error = %v, want missing observed URL", payload, err)
		}
	}
}
