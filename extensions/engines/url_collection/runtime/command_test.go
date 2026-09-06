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
