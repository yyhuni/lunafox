package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
)

func TestRunSubdomainDiscoveryUsesCanonicalTargetInjectedResourcesAndTypedResults(t *testing.T) {
	workspace := t.TempDir()
	binDir := t.TempDir()
	argsPath := filepath.Join(t.TempDir(), "subfinder.args")
	callsPath := filepath.Join(t.TempDir(), "subfinder.calls")
	writeExecutable(t, filepath.Join(binDir, "subfinder"), `#!/bin/sh
set -eu
out=""
domain=""
: > "$SUBFINDER_ARGS_FILE"
printf '1\n' >> "$SUBFINDER_CALLS_FILE"
for arg in "$@"; do printf '%s\n' "$arg" >> "$SUBFINDER_ARGS_FILE"; done
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    -d) domain="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf 'api.%s\n' "$domain" > "$out"
`)
	writeExecutable(t, filepath.Join(binDir, "puredns"), `#!/bin/sh
set -eu
input=""
out=""
if [ "${1:-}" = resolve ]; then input="$2"; shift 2; fi
while [ "$#" -gt 0 ]; do
  case "$1" in
    --write) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
cat "$input" > "$out"
`)
	wordlist := writeResource(t, "wordlist.txt", "www\napi\n")
	resolvers := writeResource(t, "resolvers.txt", "1.1.1.1\n")
	canonicalProvider := canonicalProviderConfig(t)
	provider := writeResource(t, "provider.yaml", string(canonicalProvider))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SUBFINDER_ARGS_FILE", argsPath)
	t.Setenv("SUBFINDER_CALLS_FILE", callsPath)
	results := &captureSubdomains{}
	execution := subdomainExecution(workspace, wordlist, resolvers, provider, results)

	if err := runSubdomainDiscovery(context.Background(), execution); err != nil {
		t.Fatalf("runSubdomainDiscovery() error = %v", err)
	}
	if results.calls != 1 || len(results.items) == 0 || results.items[0].DNSName != "api.example.com" {
		t.Fatalf("typed Subdomains submission = calls:%d items:%+v", results.calls, results.items)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Fields(string(args))
	count := 0
	for index, line := range lines {
		if line == "-pc" {
			count++
			if index+1 >= len(lines) || lines[index+1] != provider {
				t.Fatalf("-pc argument = %q, want %q", lines[index+1:], provider)
			}
		}
	}
	if count != 1 {
		t.Fatalf("subfinder -pc count = %d, args = %q", count, string(args))
	}
	calls, err := os.ReadFile(callsPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(strings.Fields(string(calls))); got != 1 {
		t.Fatalf("subfinder invocation count = %d, want 1", got)
	}
	for path, want := range map[string]string{
		wordlist:  "www\napi\n",
		resolvers: "1.1.1.1\n",
		provider:  canonicalProvider,
	} {
		if got, err := os.ReadFile(path); err != nil || string(got) != want {
			t.Fatalf("read-only resource %s changed: content=%q error=%v", path, got, err)
		}
	}
}

func TestRunSubdomainDiscoveryPassesDistinctResolverPathsToMatchingPureDNSInvocations(t *testing.T) {
	workspace := t.TempDir()
	binDir := t.TempDir()
	purednsCallsPath := filepath.Join(t.TempDir(), "puredns.calls")
	writeExecutable(t, filepath.Join(binDir, "subfinder"), `#!/bin/sh
set -eu
out=""
domain=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    -d) domain="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf 'api.%s\n' "$domain" > "$out"
`)
	writeExecutable(t, filepath.Join(binDir, "puredns"), `#!/bin/sh
set -eu
mode="$1"
shift
input=""
case "$mode" in
  bruteforce) shift 2 ;;
  resolve) input="$1"; shift ;;
  *) exit 64 ;;
esac
out=""
resolvers=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -r) resolvers="$2"; shift 2 ;;
    --write) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf '%s\t%s\n' "$mode" "$resolvers" >> "$PUREDNS_CALLS_FILE"
case "$mode" in
  bruteforce) printf 'brute.example.com\n' > "$out" ;;
  resolve) cat "$input" > "$out" ;;
esac
`)
	wordlist := writeResource(t, "wordlist.txt", "www\napi\n")
	bruteforceResolvers := writeResource(t, "bruteforce-resolvers.txt", "1.1.1.1\n")
	resolveResolvers := writeResource(t, "resolve-resolvers.txt", "8.8.8.8\n")
	provider := writeResource(t, "provider.yaml", string(canonicalProviderConfig(t)))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PUREDNS_CALLS_FILE", purednsCallsPath)
	results := &captureSubdomains{}
	execution := subdomainExecution(workspace, wordlist, resolveResolvers, provider, results)
	execution.Config.Bruteforce.Enabled = true
	execution.Config.Bruteforce.Resolvers = bruteforceResolvers

	if err := runSubdomainDiscovery(context.Background(), execution); err != nil {
		t.Fatalf("runSubdomainDiscovery() error = %v", err)
	}
	calls, err := os.ReadFile(purednsCallsPath)
	if err != nil {
		t.Fatalf("read PureDNS calls: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(calls)), "\n")
	want := []string{
		"bruteforce\t" + bruteforceResolvers,
		"resolve\t" + resolveResolvers,
	}
	if len(got) != len(want) {
		t.Fatalf("PureDNS calls = %q, want %q", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("PureDNS call %d = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestEnabledConfigResourcePathsFollowSectionEnablement(t *testing.T) {
	config := enginecontract.Config{
		Bruteforce: enginecontract.BruteforceConfig{Wordlist: "/wordlist", Resolvers: "/bruteforce-resolvers"},
		Resolve:    enginecontract.ResolveConfig{Resolvers: "/resolve-resolvers"},
	}
	wordlist, bruteforceResolvers, resolveResolvers := enabledConfigResourcePaths(config)
	if wordlist != "" || bruteforceResolvers != "" || resolveResolvers != "" {
		t.Fatalf("disabled resource paths = %q, %q, %q; want zero values", wordlist, bruteforceResolvers, resolveResolvers)
	}

	config.Bruteforce.Enabled = true
	wordlist, bruteforceResolvers, resolveResolvers = enabledConfigResourcePaths(config)
	if wordlist != "/wordlist" || bruteforceResolvers != "/bruteforce-resolvers" || resolveResolvers != "" {
		t.Fatalf("bruteforce-only resource paths = %q, %q, %q", wordlist, bruteforceResolvers, resolveResolvers)
	}

	config.Resolve.Enabled = true
	wordlist, bruteforceResolvers, resolveResolvers = enabledConfigResourcePaths(config)
	if wordlist != "/wordlist" || bruteforceResolvers != "/bruteforce-resolvers" || resolveResolvers != "/resolve-resolvers" {
		t.Fatalf("enabled resource paths = %q, %q, %q", wordlist, bruteforceResolvers, resolveResolvers)
	}
}

func TestRunSubdomainDiscoveryPropagatesTypedResultAcknowledgementFailure(t *testing.T) {
	workspace := t.TempDir()
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "subfinder"), `#!/bin/sh
set -eu
out=""
domain=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    -d) domain="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf 'api.%s\n' "$domain" > "$out"
`)
	writeExecutable(t, filepath.Join(binDir, "puredns"), `#!/bin/sh
set -eu
input=""
out=""
if [ "${1:-}" = resolve ]; then input="$2"; shift 2; fi
while [ "$#" -gt 0 ]; do
  case "$1" in
    --write) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
cat "$input" > "$out"
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ackErr := errors.New("result acknowledgement unavailable")
	results := &captureSubdomains{err: ackErr}
	execution := subdomainExecution(workspace, writeResource(t, "wordlist.txt", "www\n"), writeResource(t, "resolvers.txt", "1.1.1.1\n"), writeResource(t, "provider.yaml", canonicalProviderConfig(t)), results)

	err := runSubdomainDiscovery(context.Background(), execution)
	if !errors.Is(err, ackErr) {
		t.Fatalf("runSubdomainDiscovery() error = %v, want typed result acknowledgement failure", err)
	}
	if results.calls != 1 || len(results.items) != 1 {
		t.Fatalf("typed Subdomains submission = calls:%d items:%+v", results.calls, results.items)
	}
}

func TestRunSubdomainDiscoveryRejectsMissingContext(t *testing.T) {
	execution := subdomainExecution(t.TempDir(), "wordlist", "resolvers", "provider", &captureSubdomains{})
	if err := runSubdomainDiscovery(nil, execution); err == nil || !strings.Contains(err.Error(), "execution context is required") {
		t.Fatalf("runSubdomainDiscovery() error = %v, want context validation", err)
	}
}

func TestRunSubdomainDiscoveryRejectsDisabledResolveBusinessRule(t *testing.T) {
	execution := subdomainExecution(t.TempDir(), "wordlist", "resolvers", "provider", &captureSubdomains{})
	execution.Config.Resolve.Enabled = false

	err := runSubdomainDiscovery(context.Background(), execution)
	if err == nil || !strings.Contains(err.Error(), "resolve.enabled must be true") {
		t.Fatalf("runSubdomainDiscovery() error = %v, want resolve business validation", err)
	}
}

func TestRunSubdomainDiscoveryDoesNotStartSubfinderWhenReconDisabled(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "subfinder.started")
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "subfinder"), "#!/bin/sh\ntouch \"$SUBFINDER_MARKER\"\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SUBFINDER_MARKER", marker)
	execution := subdomainExecution(t.TempDir(), "wordlist", "resolvers", "provider", &captureSubdomains{})
	execution.Config.Recon.Enabled = false
	execution.Config.Bruteforce.Enabled = false

	if err := runSubdomainDiscovery(context.Background(), execution); err != nil {
		t.Fatalf("runSubdomainDiscovery() error = %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("Subfinder started while recon was disabled: %v", err)
	}
}

func TestRunSubdomainDiscoveryPropagatesSubfinderFailure(t *testing.T) {
	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "subfinder.started")
	writeExecutable(t, filepath.Join(binDir, "subfinder"), "#!/bin/sh\n: > \"$SUBFINDER_STARTED_MARKER\"\nexit 11\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SUBFINDER_STARTED_MARKER", marker)
	execution := subdomainExecution(t.TempDir(), "wordlist", "resolvers", "provider", &captureSubdomains{})

	err := runSubdomainDiscovery(context.Background(), execution)
	if err == nil || !strings.Contains(err.Error(), "no stage command completed successfully") {
		t.Fatalf("runSubdomainDiscovery() error = %v, want Subfinder failure", err)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Fatalf("Subfinder non-zero fixture did not start: %v", statErr)
	}
}

func TestRunSubdomainDiscoveryPropagatesCancellationToSubfinder(t *testing.T) {
	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "subfinder.started")
	writeExecutable(t, filepath.Join(binDir, "subfinder"), "#!/bin/sh\nset -eu\n: > \"$SUBFINDER_STARTED_MARKER\"\nexec sleep 30\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SUBFINDER_STARTED_MARKER", marker)
	execution := subdomainExecution(t.TempDir(), "wordlist", "resolvers", "provider", &captureSubdomains{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- runSubdomainDiscovery(ctx, execution) }()

	waitForToolStart(t, marker)
	cancel()
	err := waitForRunResult(t, result)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runSubdomainDiscovery() error = %v, want cancellation", err)
	}
}

func TestSubdomainContainerHandlerDoesNotUseLegacyRuntimeCapabilities(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"contracts/engineapi/runtimekit",
		"StageExecutionCapabilities",
		"SubmitSubdomainResults",
		"GetExecutionInput",
		"Materialize",
	} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("container handler retains legacy capability %q", forbidden)
		}
	}
}

func subdomainExecution(workspace, wordlist, resolvers, provider string, results *captureSubdomains) *enginecontract.Execution {
	return &enginecontract.Execution{
		Target: enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Config: enginecontract.Config{
			Recon: enginecontract.ReconConfig{Enabled: true, Timeout: 60, Threads: 1},
			Bruteforce: enginecontract.BruteforceConfig{
				Enabled: false, Timeout: 60, Wordlist: wordlist, Resolvers: resolvers, Threads: 1, RateLimit: 1, WildcardProbeCount: 1, WildcardBatch: 1,
			},
			Resolve: enginecontract.ResolveConfig{Enabled: true, Timeout: 60, Threads: 1, RateLimit: 1, Resolvers: resolvers},
		},
		PlatformResources: enginecontract.PlatformResources{SubfinderProviderConfig: provider},
		Workspace:         workspace,
		Progress:          captureProgress{},
		Results:           enginecontract.Results{Subdomains: results},
	}
}

type captureProgress struct{}

func (captureProgress) Report(context.Context, string) error { return nil }

type captureSubdomains struct {
	calls int
	items []enginecontract.Subdomain
	err   error
}

func (capture *captureSubdomains) Submit(_ context.Context, items <-chan enginecontract.Subdomain) error {
	capture.calls++
	for item := range items {
		capture.items = append(capture.items, item)
	}
	return capture.err
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeResource(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o444); err != nil {
		t.Fatal(err)
	}
	return path
}

func canonicalProviderConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "tests", "container", "subfinder-provider-config-v2.12.0.yaml")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read canonical Subfinder provider fixture: %v", err)
	}
	if len(payload) == 0 || string(payload) == "github: []\n" {
		t.Fatalf("canonical Subfinder provider fixture is empty or reduced: %q", payload)
	}
	return string(payload)
}

func waitForToolStart(t *testing.T, marker string) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(marker); err == nil {
			return
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat tool start marker: %v", err)
		}
		select {
		case <-deadline.C:
			t.Fatalf("timed out waiting for tool start marker %s", marker)
		case <-ticker.C:
		}
	}
}

func waitForRunResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for cancelled Engine handler")
		return nil
	}
}
