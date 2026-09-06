package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

func TestRunEngineCollectsAndSubmitsEndpointObservations(t *testing.T) {
	workspace := t.TempDir()
	binDir := t.TempDir()
	seedPath := filepath.Join(t.TempDir(), "website-urls.txt")
	if err := os.WriteFile(seedPath, []byte("https://example.com\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	writeURLCollectionTool(t, filepath.Join(binDir, "waymore"), `#!/bin/sh
set -eu
out=""
while [ "$#" -gt 0 ]; do case "$1" in -oU) out="$2"; shift 2;; *) shift;; esac; done
printf '%s\n' 'https://example.com/archive' > "$out"
`)
	writeURLCollectionTool(t, filepath.Join(binDir, "katana"), `#!/bin/sh
set -eu
out=""
while [ "$#" -gt 0 ]; do case "$1" in -o) out="$2"; shift 2;; *) shift;; esac; done
printf '%s\n' 'https://example.com/crawl' > "$out"
`)
	writeURLCollectionTool(t, filepath.Join(binDir, "uro"), `#!/bin/sh
set -eu
input=""; out=""
while [ "$#" -gt 0 ]; do case "$1" in -i) input="$2"; shift 2;; -o) out="$2"; shift 2;; *) shift;; esac; done
cp "$input" "$out"
`)
	writeURLCollectionTool(t, filepath.Join(binDir, "httpx"), `#!/bin/sh
set -eu
input=""; out=""
while [ "$#" -gt 0 ]; do case "$1" in -list) input="$2"; shift 2;; -o) out="$2"; shift 2;; *) shift;; esac; done
while IFS= read -r url; do printf '{"input":"%s","host":"example.com","status_code":200}\n' "$url"; done < "$input" > "$out"
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	results := &captureEndpoints{}
	execution := completeURLCollectionExecution(workspace, seedPath, results)
	if err := runEngine(context.Background(), execution); err != nil {
		t.Fatalf("runEngine() error = %v", err)
	}
	if results.calls != 1 || len(results.items) != 2 {
		t.Fatalf("Endpoint results = calls:%d items:%#v", results.calls, results.items)
	}
	for _, item := range results.items {
		if item.Host != "example.com" || !strings.HasPrefix(item.URL, "https://example.com/") {
			t.Fatalf("unexpected Endpoint %#v", item)
		}
	}
}

func TestRunEngineDoesNotTreatEmptyWebsiteFactsAsMissingInput(t *testing.T) {
	workspace := t.TempDir()
	seedPath := filepath.Join(t.TempDir(), "website-urls.txt")
	if err := os.WriteFile(seedPath, nil, 0o444); err != nil {
		t.Fatal(err)
	}
	execution := completeURLCollectionExecution(workspace, seedPath, &captureEndpoints{})
	execution.Config.Waymore.Enabled = false
	execution.Config.Katana.Enabled = false
	err := runEngine(context.Background(), execution)
	if err == nil || !strings.Contains(err.Error(), "no enabled URL Collection collector is applicable") {
		t.Fatalf("runEngine() error = %v, want collector applicability failure after baseline preparation", err)
	}
}

func TestRunEnginePropagatesTypedAcknowledgementFailure(t *testing.T) {
	workspace := t.TempDir()
	binDir := t.TempDir()
	seedPath := filepath.Join(t.TempDir(), "website-urls.txt")
	if err := os.WriteFile(seedPath, []byte("https://example.com\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"waymore", "katana", "uro", "httpx"} {
		writeURLCollectionTool(t, filepath.Join(binDir, tool), `#!/bin/sh
set -eu
out=""; input=""
while [ "$#" -gt 0 ]; do case "$1" in -o|-oU) out="$2"; shift 2;; -i|-list) input="$2"; shift 2;; *) shift;; esac; done
case "$(basename "$0")" in
  waymore|katana) printf '%s\n' 'https://example.com/path' > "$out";;
  uro) cp "$input" "$out";;
  httpx) printf '%s\n' '{"input":"https://example.com/path","host":"example.com"}' > "$out";;
esac
`)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	expected := errors.New("result acknowledgement failed")
	results := &captureEndpoints{err: expected}
	err := runEngine(context.Background(), completeURLCollectionExecution(workspace, seedPath, results))
	if !errors.Is(err, expected) {
		t.Fatalf("runEngine() error = %v, want %v", err, expected)
	}
}

func completeURLCollectionExecution(workspace, seedPath string, results *captureEndpoints) *enginecontract.Execution {
	return &enginecontract.Execution{
		Target:    enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Input:     enginecontract.Input{WebsiteURLs: testInputPath(seedPath)},
		Workspace: workspace,
		Config: enginecontract.Config{
			Waymore: enginecontract.WaymoreConfig{Enabled: true, Timeout: 60},
			Katana:  enginecontract.KatanaConfig{Enabled: true, Timeout: 60, Depth: 3, Concurrency: 10, RateLimit: 30, RequestTimeout: 10, Retries: 1, Delay: 0},
			Uro:     enginecontract.UroConfig{Enabled: true, Timeout: 60},
			HTTPX:   enginecontract.HTTPXConfig{Enabled: true, Timeout: 60, Threads: 25, RateLimit: 150, RequestTimeout: 10, Retries: 1},
		},
		Progress: captureURLCollectionProgress{},
		Results:  enginecontract.Results{Endpoints: results},
	}
}

type captureURLCollectionProgress struct{}

func (captureURLCollectionProgress) Report(context.Context, string) error { return nil }

type captureEndpoints struct {
	calls int
	items []enginecontract.Endpoint
	err   error
}

func (capture *captureEndpoints) Submit(_ context.Context, items <-chan enginecontract.Endpoint) error {
	capture.calls++
	for item := range items {
		capture.items = append(capture.items, item)
	}
	return capture.err
}

func writeURLCollectionTool(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}
