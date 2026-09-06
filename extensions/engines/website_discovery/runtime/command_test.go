package websitediscoveryruntime

import (
	"reflect"
	"testing"
	"time"

	websitediscoverycontract "github.com/yyhuni/lunafox/engines/website_discovery/contract"
)

func TestBuildHTTPXCommandUsesStructuredArgv(t *testing.T) {
	cmd, err := buildHTTPXCommand("/tmp/urls.txt", "/tmp/httpx.jsonl", websitediscoverycontract.HTTPXConfig{
		Enabled:        true,
		Timeout:        120,
		Threads:        50,
		RateLimit:      200,
		RequestTimeout: 15,
		Retries:        2,
	})
	if err != nil {
		t.Fatalf("buildHTTPXCommand failed: %v", err)
	}

	want := []string{"-list", "/tmp/urls.txt", "-json", "-silent", "-no-color", "-status-code", "-content-type", "-content-length", "-location", "-title", "-server", "-tech-detect", "-cdn", "-vhost", "-include-response", "-rstr", "2000", "-random-agent", "-o", "/tmp/httpx.jsonl", "-threads", "50", "-rate-limit", "200", "-timeout", "15", "-retries", "2"}
	if !reflect.DeepEqual(cmd.args, want) || cmd.timeout != 120*time.Second {
		t.Fatalf("unexpected httpx command: %+v", cmd)
	}
}
