package websitediscoveryruntime

import (
	"reflect"
	"strings"
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

func TestBuildHTTPXCommandPreservesHeaderArguments(t *testing.T) {
	headers := []string{"Authorization: Bearer test-token", "Cookie: a=1; b=two", `X-Fields: a,b:c; "quoted"`, "X-Fields: other"}
	command, err := buildHTTPXCommand("/tmp/urls.txt", "/tmp/httpx.jsonl", websitediscoverycontract.HTTPXConfig{Timeout: 60, Headers: headers})
	if err != nil {
		t.Fatal(err)
	}
	want := make([]string, 0, len(headers)*2)
	for _, header := range headers {
		want = append(want, "-H", header)
	}
	if got := command.args[len(command.args)-len(want):]; !reflect.DeepEqual(got, want) {
		t.Fatalf("header argv = %#v, want %#v", got, want)
	}
	withoutHeaders, err := buildHTTPXCommand("/tmp/urls.txt", "/tmp/httpx.jsonl", websitediscoverycontract.HTTPXConfig{Timeout: 60, Headers: []string{}})
	if err != nil || !reflect.DeepEqual(command.args[:len(command.args)-len(want)], withoutHeaders.args) {
		t.Fatalf("empty headers changed argv: %#v, %v", withoutHeaders.args, err)
	}
}

func TestBuildHTTPXCommandRejectsInvalidHeadersWithoutExposingValues(t *testing.T) {
	for _, header := range []string{"secret-token", "Bad Name: secret-token", "Authorization: ", "Authorization: secret-token\r\nX-Injected: true", "Authorization: secret-token\x00"} {
		_, err := buildHTTPXCommand("urls", "output", websitediscoverycontract.HTTPXConfig{Timeout: 60, Headers: []string{header}})
		if err == nil || !strings.Contains(err.Error(), "httpx.headers[0]") || strings.Contains(err.Error(), "secret-token") {
			t.Fatalf("header validation error = %v", err)
		}
	}
}
