package urlcollectionruntime

import (
	"reflect"
	"strings"
	"testing"

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

func TestFixedToolCommandsExposeOnlyConfirmedArguments(t *testing.T) {
	waymore, err := buildWaymoreCommand(enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, "/work/waymore.txt", enginecontract.WaymoreConfig{Timeout: 60})
	if err != nil || !reflect.DeepEqual(waymore.args, []string{"-i", "example.com", "-mode", "U", "-oU", "/work/waymore.txt"}) {
		t.Fatalf("waymore = %#v, %v", waymore, err)
	}
	katana, err := buildKatanaCommand("/run/lunafox/inputs/website-urls.txt", "/work/katana.txt", enginecontract.KatanaConfig{Timeout: 60, Depth: 3, Concurrency: 10, RateLimit: 30, RequestTimeout: 10, Retries: 1, Delay: 0})
	if err != nil {
		t.Fatal(err)
	}
	for _, argument := range []string{"-rd", "-fs", "rdn", "-dr"} {
		if !containsArgument(katana.args, argument) {
			t.Fatalf("katana args = %q, missing %q", katana.args, argument)
		}
	}
	for _, forbidden := range []string{"-H", "-proxy", "-cookie", "-ns", "-do", "-cs", "-cos"} {
		if containsArgument(katana.args, forbidden) {
			t.Fatalf("katana args must not expose %q: %q", forbidden, katana.args)
		}
	}
	httpx, err := buildHTTPXCommand("/work/candidates.txt", "/work/httpx.jsonl", enginecontract.HTTPXConfig{Timeout: 60, Threads: 25, RateLimit: 150, RequestTimeout: 10, Retries: 1})
	if err != nil || !containsArgument(httpx.args, "-random-agent") {
		t.Fatalf("httpx = %#v, %v", httpx, err)
	}
	for _, forbidden := range []string{"-fr", "-fhr", "-maxr", "-proxy", "-H", "-cookie", "-exclude", "-allow"} {
		if containsArgument(httpx.args, forbidden) {
			t.Fatalf("httpx args must not expose %q: %q", forbidden, httpx.args)
		}
	}
}

func TestUroPreservesIndependentListsAndFixedFilters(t *testing.T) {
	command, err := buildUroCommand("/work/candidates.txt", "/work/uro.txt", enginecontract.UroConfig{Timeout: 60, Whitelist: []string{"php", ".go"}, Blacklist: []string{"png"}, Filters: []string{"hasparams", "noext"}})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(command.args, " ")
	for _, expected := range []string{"-w php .go", "-b png", "-f hasparams noext"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("uro args = %q, missing %q", joined, expected)
		}
	}
}

func TestUroAllowsEmptyAndContradictoryFilterInputsWithoutFallbackPolicy(t *testing.T) {
	empty, err := buildUroCommand("/work/candidates.txt", "/work/uro.txt", enginecontract.UroConfig{Timeout: 60})
	if err != nil || !reflect.DeepEqual(empty.args, []string{"-i", "/work/candidates.txt", "-o", "/work/uro.txt"}) {
		t.Fatalf("empty Uro command = %#v, %v", empty, err)
	}
	contradictory, err := buildUroCommand("/work/candidates.txt", "/work/uro.txt", enginecontract.UroConfig{
		Timeout: 60, Whitelist: []string{"php"}, Blacklist: []string{"php"}, Filters: []string{"hasparams", "noparams"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-i", "/work/candidates.txt", "-o", "/work/uro.txt", "-w", "php", "-b", "php", "-f", "hasparams", "noparams"}
	if !reflect.DeepEqual(contradictory.args, want) {
		t.Fatalf("contradictory Uro command = %q, want %q", contradictory.args, want)
	}
}

func containsArgument(args []string, wanted string) bool {
	for _, argument := range args {
		if argument == wanted {
			return true
		}
	}
	return false
}

func TestTargetHTTPToolsPreserveAndValidateIndependentHeaders(t *testing.T) {
	builds := map[string]func([]string) (toolCommand, error){
		"katana": func(headers []string) (toolCommand, error) {
			return buildKatanaCommand("/inputs", "/output", enginecontract.KatanaConfig{Timeout: 60, Depth: 1, Concurrency: 1, RateLimit: 10, RequestTimeout: 2, Headers: headers})
		},
		"httpx": func(headers []string) (toolCommand, error) {
			return buildHTTPXCommand("/inputs", "/output", enginecontract.HTTPXConfig{Timeout: 60, Threads: 1, RateLimit: 10, RequestTimeout: 2, Headers: headers})
		},
	}
	for name, build := range builds {
		t.Run(name, func(t *testing.T) {
			headers := []string{"Cookie: a=1; b=two", "Authorization: Bearer local-test", `X-Fields: a,b:c; "quoted"`, "X-Fields: second"}
			command, err := build(headers)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for index, arg := range command.args {
				if arg == "-H" {
					got = append(got, command.args[index+1])
				}
			}
			if !reflect.DeepEqual(got, headers) {
				t.Fatalf("headers = %#v, want %#v", got, headers)
			}
			for _, header := range []string{"secret-without-colon", "Bad Name: secret", "X-Secret: ", "X-Secret: secret\r\nInjected: yes", "X-Secret: secret\x00"} {
				command, err := build([]string{"X-Valid: ok", header})
				if err == nil || !strings.Contains(err.Error(), name+".headers[1]") || strings.Contains(err.Error(), "secret") {
					t.Fatalf("invalid header error = %v", err)
				}
				if len(command.args) != 0 {
					t.Fatal("invalid config returned executable arguments")
				}
			}
		})
	}
}
