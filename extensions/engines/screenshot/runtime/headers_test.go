package screenshotruntime

import (
	"reflect"
	"strings"
	"testing"

	enginecontract "github.com/yyhuni/lunafox/engines/screenshot/contract"
)

func TestHeadersPreserveSeparateArgumentsAndOrder(t *testing.T) {
	config := enginecontract.CaptureConfig{PageTimeout: 15, Concurrency: 1, Retries: 0}
	config.Headers = []string{"Cookie: a=1; b=two", "Authorization: Bearer local-test", `X-Fields: a,b:c; "quoted"`, "X-Fields: second", "X-Literal: $(echo test)"}
	command, err := buildHTTPXCommand("/inputs", "/workspace", config)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for index, arg := range command.args {
		if arg == "-H" {
			got = append(got, command.args[index+1])
		}
	}
	if !reflect.DeepEqual(got, config.Headers) {
		t.Fatalf("headers = %#v, want %#v", got, config.Headers)
	}
}

func TestHeadersRejectInvalidTypedValuesWithoutDisclosure(t *testing.T) {
	for _, header := range []string{"secret-without-colon", "Bad Name: secret", "X-Secret: ", "X-Secret: secret\r\nInjected: yes", "X-Secret: secret\x00"} {
		config := enginecontract.CaptureConfig{PageTimeout: 15, Concurrency: 1, Retries: 0}
		config.Headers = []string{"X-Valid: ok", header}
		command, err := buildHTTPXCommand("/inputs", "/workspace", config)
		if err == nil {
			t.Fatal("invalid header accepted")
		}
		if !strings.Contains(err.Error(), "headers[1]") || strings.Contains(err.Error(), "secret") {
			t.Fatalf("unsafe error: %v", err)
		}
		if len(command.args) != 0 {
			t.Fatal("invalid config returned executable arguments")
		}
	}
}
