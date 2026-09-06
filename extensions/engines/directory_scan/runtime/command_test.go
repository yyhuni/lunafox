package directoryscanruntime

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestBuildFFUFArgsMatchesCanonicalDefaultCommand(t *testing.T) {
	args, err := BuildFFUFArgs("http://example.com", defaultFFUFConfig())
	if err != nil {
		t.Fatalf("BuildFFUFArgs() error = %v", err)
	}
	want := []string{
		"-u", "http://example.comFUZZ",
		"-w", "/run/lunafox/resources/config/ffuf/wordlist/dir_default.txt",
		"-json", "-s", "-X", "GET", "-raw", "-ac",
		"-mc", "200-299,301,302,307,401,403,405,500",
		"-t", "10", "-rate", "0", "-p", "0.1-2.0", "-timeout", "10",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("FFUF args = %#v, want %#v", args, want)
	}
}

func TestBuildFFUFArgsMapsOnlyEnabledConditionalControls(t *testing.T) {
	config := defaultFFUFConfig()
	config.Recursion = true
	config.RecursionDepth = 5
	config.RecursionStrategy = "greedy"
	config.AutoCalibrationMode = "ach"
	config.FollowRedirects = true
	config.Http2 = true
	args, err := BuildFFUFArgs("https://example.com/root/", config)
	if err != nil {
		t.Fatalf("BuildFFUFArgs() error = %v", err)
	}
	want := []string{
		"-u", "https://example.com/root/FUZZ",
		"-w", config.Wordlist,
		"-json", "-s", "-X", "GET", "-raw",
		"-recursion", "-recursion-depth", "5", "-recursion-strategy", "greedy",
		"-ach", "-mc", config.MatchCodes,
		"-t", "10", "-rate", "0", "-p", config.Delay, "-timeout", "10",
		"-r", "-http2",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("conditional FFUF args = %#v, want %#v", args, want)
	}

	config.Recursion = false
	config.AutoCalibration = false
	config.FollowRedirects = false
	config.Http2 = false
	args, err = BuildFFUFArgs("https://example.com", config)
	if err != nil {
		t.Fatalf("BuildFFUFArgs(disabled modes) error = %v", err)
	}
	for _, omitted := range []string{"-recursion", "-recursion-depth", "-recursion-strategy", "-ac", "-ach", "-r", "-http2"} {
		if slices.Contains(args, omitted) {
			t.Fatalf("disabled mode emitted %q in %#v", omitted, args)
		}
	}
}

func TestBuildFFUFArgsPreservesRawCandidateWordlistAndPayloadFile(t *testing.T) {
	wordlist := filepath.Join(t.TempDir(), "payload list.txt")
	payload := []byte("admin\n/admin\n//admin\n#comment-looking\nvalue # inline-looking\n")
	if err := os.WriteFile(wordlist, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	config := defaultFFUFConfig()
	config.Wordlist = wordlist
	config.MatchCodes = "all,200,200,200-299,250-350"
	config.Delay = "01.000-02.00"
	candidate := "https://EXAMPLE.com/root?next=/admin#fragment"
	args, err := BuildFFUFArgs(candidate, config)
	if err != nil {
		t.Fatalf("BuildFFUFArgs() error = %v", err)
	}
	if args[1] != candidate+"FUZZ" || args[3] != wordlist {
		t.Fatalf("raw candidate/wordlist changed in %#v", args)
	}
	if valueAfter(args, "-mc") != config.MatchCodes || valueAfter(args, "-p") != config.Delay {
		t.Fatalf("opaque matcher/delay changed in %#v", args)
	}
	gotPayload, err := os.ReadFile(wordlist)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotPayload, payload) {
		t.Fatalf("wordlist payload changed: %q", gotPayload)
	}
}

func TestBuildFFUFArgsAcceptsVerifiedZeroByteWordlistPath(t *testing.T) {
	wordlist := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(wordlist, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	config := defaultFFUFConfig()
	config.Wordlist = wordlist
	args, err := BuildFFUFArgs("http://example.com", config)
	if err != nil {
		t.Fatalf("BuildFFUFArgs() error = %v", err)
	}
	if valueAfter(args, "-w") != wordlist {
		t.Fatalf("zero-byte wordlist path = %q, want %q", valueAfter(args, "-w"), wordlist)
	}
	info, err := os.Stat(wordlist)
	if err != nil || info.Size() != 0 {
		t.Fatalf("zero-byte wordlist = %#v, %v", info, err)
	}
}

func TestBuildFFUFArgsRejectsPreflightAndConfigViolations(t *testing.T) {
	if args, err := BuildFFUFArgs("https://example.com/FUZZ", defaultFFUFConfig()); err != nil || valueAfter(args, "-u") != "https://example.com/FUZZFUZZ" {
		t.Fatalf("BuildFFUFArgs() = %#v, %v; literal candidate bytes must be preserved", args, err)
	}
	config := defaultFFUFConfig()
	config.MatchCodes = "200, 201"
	if _, err := BuildFFUFArgs("https://example.com", config); err == nil {
		t.Fatal("BuildFFUFArgs() accepted semantically invalid FFUF config")
	}
}

func TestBuildFFUFArgsHasClosedNegativeFlagSurface(t *testing.T) {
	config := defaultFFUFConfig()
	config.Recursion = true
	config.AutoCalibrationMode = "ach"
	config.FollowRedirects = true
	config.Http2 = true
	args, err := BuildFFUFArgs("https://example.com", config)
	if err != nil {
		t.Fatalf("BuildFFUFArgs() error = %v", err)
	}
	forbidden := []string{
		"-H", "-b", "-x", "-cc", "-ck", "-d", "-replay-proxy", "-request", "-request-proto",
		"-acc", "-acs", "-ack",
		"-c", "-V", "-v", "-noninteractive", "-search",
		"-ms", "-mw", "-ml", "-mt", "-mr", "-mmode",
		"-fc", "-fs", "-fw", "-fl", "-ft", "-fr", "-fmode",
		"-e", "-D", "-ic", "-input-cmd", "-input-num", "-input-shell", "-mode",
		"-se", "-sa", "-sf",
		"-scraperfile", "-scrapers", "-enc", "-sni", "-ignore-body",
		"-maxtime", "-maxtime-job", "-config",
		"-audit-log", "-debug-log", "-o", "-od", "-of", "-or",
	}
	for _, flag := range forbidden {
		if slices.Contains(args, flag) {
			t.Fatalf("closed FFUF argv contains forbidden flag %q: %#v", flag, args)
		}
	}
	if valueAfter(args, "-X") != "GET" {
		t.Fatalf("FFUF method = %q, want fixed GET", valueAfter(args, "-X"))
	}
	if slices.Contains(args, "-ac") || !slices.Contains(args, "-ach") {
		t.Fatalf("ach mode must emit only -ach: %#v", args)
	}
}

func TestBuildFFUFArgsAppendsFUZZAfterExactQueryAndFragmentPrefixes(t *testing.T) {
	for _, candidate := range []string{
		"http://example.com?path=/admin",
		"http://example.com#fragment",
		"http://example.com?path=/admin#fragment",
		"http://example.com//",
	} {
		t.Run(strings.ReplaceAll(candidate, "/", "_"), func(t *testing.T) {
			args, err := BuildFFUFArgs(candidate, defaultFFUFConfig())
			if err != nil {
				t.Fatalf("BuildFFUFArgs() error = %v", err)
			}
			if args[1] != candidate+"FUZZ" {
				t.Fatalf("FFUF target = %q, want %q", args[1], candidate+"FUZZ")
			}
		})
	}
}

func valueAfter(args []string, flag string) string {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == flag {
			return args[index+1]
		}
	}
	return ""
}
