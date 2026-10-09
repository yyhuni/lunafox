package urlcollectionruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

func TestPinnedHTTPToolsDeliverHeaders(t *testing.T) {
	for _, tool := range []struct {
		name, env, version string
		build              func(string, string, []string) (toolCommand, error)
	}{
		{"katana", "LUNAFOX_TEST_KATANA_BIN", "v1.6.1", func(input, output string, headers []string) (toolCommand, error) {
			return buildKatanaCommand(input, output, enginecontract.KatanaConfig{Timeout: 30, Depth: 1, Concurrency: 1, RateLimit: 10, RequestTimeout: 5, Headers: headers})
		}},
		{"httpx", "LUNAFOX_TEST_HTTPX_BIN", "v1.10.0", func(input, output string, headers []string) (toolCommand, error) {
			return buildHTTPXCommand(input, output, enginecontract.HTTPXConfig{Timeout: 30, Threads: 1, RateLimit: 10, RequestTimeout: 5, Headers: headers})
		}},
	} {
		t.Run(tool.name, func(t *testing.T) {
			binary := os.Getenv(tool.env)
			if binary == "" {
				t.Skipf("set %s to pinned %s %s", tool.env, tool.name, tool.version)
			}
			version, err := exec.Command(binary, "-version").CombinedOutput()
			if err != nil || !strings.Contains(string(version), tool.version) {
				t.Fatalf("expected %s %s: %s, %v", tool.name, tool.version, version, err)
			}
			received := make(chan http.Header, 32)
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "/header-check" {
					select {
					case received <- request.Header.Clone():
					default:
					}
				}
				response.Header().Set("Content-Type", "text/html")
				_, _ = response.Write([]byte("<html><title>Header check</title></html>"))
			}))
			defer server.Close()
			workspace := t.TempDir()
			if tool.name == "httpx" {
				// Mirror the Runtime Image's offline optional DIT classifier setup.
				t.Setenv("HOME", workspace)
				if err := os.Mkdir(filepath.Join(workspace, ".dit"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(workspace, ".dit", "model.json"), []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			input := filepath.Join(workspace, "urls.txt")
			output := filepath.Join(workspace, "output.txt")
			if err := os.WriteFile(input, []byte(server.URL+"/header-check\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			command, err := tool.build(input, output, []string{"Cookie: a=1; b=two", "Authorization: Bearer local-test-token", `X-Fields: a,b:c; "quoted"`})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			if log, err := exec.CommandContext(ctx, binary, command.args...).CombinedOutput(); err != nil {
				t.Fatalf("%s: %s, %v", tool.name, log, err)
			}
			data, err := os.ReadFile(output)
			if err != nil || !strings.Contains(string(data), server.URL+"/header-check") {
				t.Fatalf("missing local endpoint result: %s, %v", data, err)
			}
			select {
			case headers := <-received:
				for name, want := range map[string]string{"Cookie": "a=1; b=two", "Authorization": "Bearer local-test-token", "X-Fields": `a,b:c; "quoted"`} {
					if got := headers.Get(name); got != want {
						t.Fatalf("%s = %q, want %q", name, got, want)
					}
				}
			default:
				t.Fatalf("%s did not reach the local endpoint", tool.name)
			}
		})
	}
}
