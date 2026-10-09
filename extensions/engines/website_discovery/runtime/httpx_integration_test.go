package websitediscoveryruntime

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	websitediscoverycontract "github.com/yyhuni/lunafox/engines/website_discovery/contract"
)

func TestHTTPXHeadersReachLocalEndpoint(t *testing.T) {
	binary := os.Getenv("LUNAFOX_TEST_HTTPX_BIN")
	if binary == "" {
		t.Skip("set LUNAFOX_TEST_HTTPX_BIN to the ProjectDiscovery HTTPX v1.8.1 binary")
	}
	if !filepath.IsAbs(binary) || filepath.Base(binary) != "httpx" {
		t.Fatal("LUNAFOX_TEST_HTTPX_BIN must be an absolute path to a binary named httpx")
	}
	t.Setenv("PATH", filepath.Dir(binary)+string(os.PathListSeparator)+os.Getenv("PATH"))
	received := make(chan http.Header, 16)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/header-check" {
			select {
			case received <- request.Header.Clone():
			default:
			}
		}
		response.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(response, "<html><title>Header check</title></html>")
	}))
	defer server.Close()
	workDir := t.TempDir()
	input := filepath.Join(workDir, "urls.txt")
	if err := os.WriteFile(input, []byte(server.URL+"/header-check\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	command, err := buildHTTPXCommand(input, filepath.Join(workDir, "httpx.jsonl"), websitediscoverycontract.HTTPXConfig{
		Timeout: 30, Threads: 1, RequestTimeout: 2, Retries: 0,
		Headers: []string{"Cookie: a=1; b=two", "Authorization: Bearer local-test-token", `X-Fields: a,b:c; "quoted"`},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	if err := (containerHTTPXExecutor{}).run(ctx, command); err != nil {
		t.Fatal(err)
	}
	select {
	case headers := <-received:
		for name, want := range map[string]string{"Cookie": "a=1; b=two", "Authorization": "Bearer local-test-token", "X-Fields": `a,b:c; "quoted"`} {
			if got := headers.Get(name); got != want {
				t.Fatalf("%s = %q, want %q", name, got, want)
			}
		}
	default:
		t.Fatal("HTTPX did not reach the local endpoint")
	}
}
