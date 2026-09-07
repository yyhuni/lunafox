package application

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestBoundedWriterReportsFullWriteWithoutGrowingDiagnostic(t *testing.T) {
	writer := &boundedWriter{limit: 4}
	data := []byte("0123456789")
	written, err := writer.Write(data)
	if err != nil || written != len(data) || writer.buffer.String() != "0123" {
		t.Fatalf("written=%d err=%v buffer=%q", written, err, writer.buffer.String())
	}
}

func TestSafeGitEnvironmentPreservesProxyAndBlocksUnsafeOverrides(t *testing.T) {
	proxyVariables := []string{
		"HTTP_PROXY=http://proxy.invalid:7890",
		"HTTPS_PROXY=http://proxy.invalid:7890",
		"ALL_PROXY=socks5://proxy.invalid:7891",
		"NO_PROXY=localhost,postgres",
		"http_proxy=http://proxy.invalid:7890",
		"https_proxy=http://proxy.invalid:7890",
		"all_proxy=socks5://proxy.invalid:7891",
		"no_proxy=localhost,postgres",
	}
	for _, value := range proxyVariables {
		t.Setenv(strings.SplitN(value, "=", 2)[0], strings.SplitN(value, "=", 2)[1])
	}
	blockedVariables := []string{"GIT_SSL_NO_VERIFY=1", "GIT_SSH_COMMAND=ssh", "GIT_ASKPASS=askpass", "GIT_CONFIG_COUNT=1"}
	for _, value := range blockedVariables {
		t.Setenv(strings.SplitN(value, "=", 2)[0], strings.SplitN(value, "=", 2)[1])
	}
	environment := strings.Join(safeGitEnvironment(), "\n")
	for _, proxy := range proxyVariables {
		if !strings.Contains(environment, proxy) {
			t.Fatalf("proxy environment was dropped: %s", proxy)
		}
	}
	for _, blocked := range blockedVariables {
		if strings.Contains(environment, blocked) {
			t.Fatalf("blocked environment leaked: %s", blocked)
		}
	}
	if !strings.Contains(environment, "GIT_TERMINAL_PROMPT=0") {
		t.Fatal("terminal prompt was not disabled")
	}
}

func TestGitCloneArgsUseDomainWithoutAddressPinning(t *testing.T) {
	args := gitCloneArgs("https://example.com/org/templates.git", "/tmp/workspace")
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "curloptResolve") || strings.Contains(joined, "followRedirects") {
		t.Fatalf("clone args unexpectedly pin or disable redirects: %v", args)
	}
	if !strings.Contains(joined, "https://example.com/org/templates.git") {
		t.Fatalf("clone args dropped the domain URL: %v", args)
	}
}

func TestRunControlledGitHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	err := runControlledGit(ctx, "--version")
	if err == nil {
		t.Fatal("expected canceled command to fail")
	}
	if time.Since(started) > time.Second {
		t.Fatal("canceled command took too long")
	}
}
