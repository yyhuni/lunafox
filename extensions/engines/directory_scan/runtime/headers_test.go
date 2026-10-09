package directoryscanruntime

import (
	"reflect"
	"strings"
	"testing"
)

func TestHeadersPreserveSeparateArgumentsAndOrder(t *testing.T) {
	config := defaultFFUFConfig()
	config.Headers = []string{"Cookie: a=1; b=two", "Authorization: Bearer local-test", `X-Fields: a,b:c; "quoted"`, "X-Fields: second", "X-Literal: $(echo test)"}
	args, err := BuildFFUFArgs("http://example.com/", config)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for index, arg := range args {
		if arg == "-H" {
			got = append(got, args[index+1])
		}
	}
	if !reflect.DeepEqual(got, config.Headers) {
		t.Fatalf("headers = %#v, want %#v", got, config.Headers)
	}
}

func TestHeadersRejectInvalidTypedValuesWithoutDisclosure(t *testing.T) {
	for _, header := range []string{"secret-without-colon", "Bad Name: secret", "X-Secret: ", "X-Secret: secret\r\nInjected: yes", "X-Secret: secret\x00"} {
		config := defaultFFUFConfig()
		config.Headers = []string{"X-Valid: ok", header}
		args, err := BuildFFUFArgs("http://example.com/", config)
		if err == nil {
			t.Fatal("invalid header accepted")
		}
		if !strings.Contains(err.Error(), "headers[1]") || strings.Contains(err.Error(), "secret") {
			t.Fatalf("unsafe error: %v", err)
		}
		if len(args) != 0 {
			t.Fatal("invalid config returned executable arguments")
		}
	}
}
