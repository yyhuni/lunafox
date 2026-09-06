package directoryscanruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

const ffufIntegrationBinaryEnvironment = "LUNAFOX_FFUF_INTEGRATION_BINARY"

type ffufJSONRecord struct {
	URL           string `json:"url"`
	Status        int    `json:"status"`
	ContentLength int64  `json:"length"`
	ContentType   string `json:"content-type"`
	Duration      int64  `json:"duration"`
	raw           []byte
}

func TestPinnedFFUFPreservesRawPayloadShapesAndCommentLookingLines(t *testing.T) {
	binary := requirePinnedFFUFBinary(t)
	requests := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests <- request.RequestURI
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	payloads := []string{"admin", "/admin", "//admin", "#comment-looking"}
	wordlist := writeFFUFWordlist(t, payloads...)
	config := ffufIntegrationConfig(wordlist)
	records, _ := runPinnedFFUF(t, binary, server.URL+"/base/", config)
	if len(records) != len(payloads) {
		t.Fatalf("FFUF records = %#v, want %d records", records, len(payloads))
	}
	for index, payload := range payloads {
		if want := server.URL + "/base/" + payload; records[index].URL != want {
			t.Fatalf("record[%d].url = %q, want exact %q", index, records[index].URL, want)
		}
	}

	close(requests)
	var gotRequests []string
	for requestURI := range requests {
		gotRequests = append(gotRequests, requestURI)
	}
	if len(gotRequests) != len(payloads) {
		t.Fatalf("request URIs = %#v, want %d requests", gotRequests, len(payloads))
	}
	for index, wantSuffix := range []string{"/base/admin", "/base//admin", "/base///admin"} {
		if !strings.HasSuffix(gotRequests[index], wantSuffix) {
			t.Fatalf("request URI[%d] = %q, want suffix %q", index, gotRequests[index], wantSuffix)
		}
	}
}

func TestPinnedFFUFFollowRedirectsUsesOnlyDeclaredMode(t *testing.T) {
	binary := requirePinnedFFUFBinary(t)
	for _, test := range []struct {
		name            string
		followRedirects bool
		wantStatus      int
		wantFinal       int32
	}{
		{name: "disabled", wantStatus: http.StatusFound},
		{name: "enabled", followRedirects: true, wantStatus: http.StatusOK, wantFinal: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var finalRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "/final" {
					finalRequests.Add(1)
					response.WriteHeader(http.StatusOK)
					return
				}
				http.Redirect(response, request, "/final", http.StatusFound)
			}))
			defer server.Close()

			config := ffufIntegrationConfig(writeFFUFWordlist(t, "admin"))
			config.FollowRedirects = test.followRedirects
			records, _ := runPinnedFFUF(t, binary, server.URL+"/scan/", config)
			if len(records) != 1 || records[0].Status != test.wantStatus {
				t.Fatalf("FFUF records = %#v, want one status %d", records, test.wantStatus)
			}
			if got := finalRequests.Load(); got != test.wantFinal {
				t.Fatalf("final requests = %d, want %d", got, test.wantFinal)
			}
		})
	}
}

func TestPinnedFFUFStockTLSHTTP2AndHTTP1Fallback(t *testing.T) {
	binary := requirePinnedFFUFBinary(t)
	for _, test := range []struct {
		name        string
		tls         bool
		serverHTTP2 bool
		clientHTTP2 bool
		wantMajor   int32
	}{
		{name: "stock self-signed HTTPS", tls: true, wantMajor: 1},
		{name: "HTTPS HTTP2", tls: true, serverHTTP2: true, clientHTTP2: true, wantMajor: 2},
		{name: "HTTPS HTTP1 fallback", tls: true, clientHTTP2: true, wantMajor: 1},
		{name: "plain HTTP does not claim h2c", clientHTTP2: true, wantMajor: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var protocolMajor atomic.Int32
			handler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				protocolMajor.CompareAndSwap(0, int32(request.ProtoMajor))
				response.WriteHeader(http.StatusOK)
			})
			server := httptest.NewUnstartedServer(handler)
			server.EnableHTTP2 = test.serverHTTP2
			if test.tls {
				server.StartTLS()
			} else {
				server.Start()
			}
			defer server.Close()

			config := ffufIntegrationConfig(writeFFUFWordlist(t, "admin"))
			config.Http2 = test.clientHTTP2
			records, _ := runPinnedFFUF(t, binary, server.URL+"/", config)
			if len(records) != 1 || records[0].Status != http.StatusOK {
				t.Fatalf("FFUF records = %#v, want one successful record", records)
			}
			if got := protocolMajor.Load(); got != test.wantMajor {
				t.Fatalf("HTTP protocol major = %d, want %d", got, test.wantMajor)
			}
		})
	}
}

func TestPinnedFFUFAutoCalibrationRequestFailureRemainsBestEffortOnExitZero(t *testing.T) {
	binary := requirePinnedFFUFBinary(t)
	var calibrationRequests atomic.Int32
	var payloadRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/base/admin" {
			payloadRequests.Add(1)
			response.WriteHeader(http.StatusOK)
			return
		}
		calibrationRequests.Add(1)
		hijacker, ok := response.(http.Hijacker)
		if !ok {
			return
		}
		connection, _, err := hijacker.Hijack()
		if err == nil {
			_ = connection.Close()
		}
	}))
	defer server.Close()

	config := ffufIntegrationConfig(writeFFUFWordlist(t, "admin"))
	config.AutoCalibration = true
	config.AutoCalibrationMode = "ac"
	records, _ := runPinnedFFUF(t, binary, server.URL+"/base/", config)
	if calibrationRequests.Load() == 0 {
		t.Fatal("FFUF made no failing automatic-calibration request")
	}
	if payloadRequests.Load() != 1 {
		t.Fatalf("payload requests = %d, want 1", payloadRequests.Load())
	}
	if len(records) != 1 || records[0].Status != http.StatusOK {
		t.Fatalf("FFUF records = %#v, want successful payload after calibration failure", records)
	}
}

func TestPinnedFFUFJSONLMapsToCanonicalDirectoryObservation(t *testing.T) {
	binary := requirePinnedFFUFBinary(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/x-directory-test")
		_, _ = response.Write([]byte("hello"))
	}))
	defer server.Close()

	config := ffufIntegrationConfig(writeFFUFWordlist(t, "admin"))
	records, _ := runPinnedFFUF(t, binary, server.URL+"/", config)
	if len(records) != 1 {
		t.Fatalf("FFUF records = %#v, want one", records)
	}
	item, outcome := decodeFFUFRecord(records[0].raw)
	if outcome != ffufRecordValid || item.URL != server.URL+"/admin" || item.Status != http.StatusOK ||
		item.ContentLength != 5 || item.ContentType != "application/x-directory-test" || item.Duration < 0 {
		t.Fatalf("mapped FFUF observation = %#v outcome=%d raw=%s", item, outcome, records[0].raw)
	}
}

func TestPinnedFFUFChildDeadlineRetainsCompletePreDeadlineJSONL(t *testing.T) {
	binary := requirePinnedFFUFBinary(t)
	slowStarted := make(chan struct{})
	releaseSlow := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/admin" {
			response.Header().Set("Content-Type", "text/plain")
			_, _ = response.Write([]byte("ready"))
			return
		}
		if request.URL.Path == "/slow" {
			close(slowStarted)
			<-releaseSlow
			response.WriteHeader(http.StatusOK)
			return
		}
		response.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	defer close(releaseSlow)

	config := ffufIntegrationConfig(writeFFUFWordlist(t, "admin", "slow"))
	workspace := t.TempDir()
	runner := newContainerFFUFProcessRunner()
	runner.binary = binary
	runner.stderr = io.Discard
	ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
	defer cancel()
	err := runner.Run(ctx, ffufInvocation{
		Candidate: WebsiteCandidate{Ordinal: 4, Website: server.URL + "/"},
		Config:    config,
		Workspace: workspace,
	})
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) || err == nil {
		t.Fatalf("FFUF deadline context=%v run error=%v", ctx.Err(), err)
	}
	select {
	case <-slowStarted:
	default:
		t.Fatal("slow request did not begin before the child deadline")
	}

	path, pathErr := rawFFUFArtifactPath(workspace, 4)
	if pathErr != nil {
		t.Fatal(pathErr)
	}
	var observations []DirectoryObservation
	summary, parseErr := ParseFFUFArtifact(context.Background(), path, 4, func(observation DirectoryObservation) error {
		observations = append(observations, observation)
		return nil
	})
	if parseErr != nil {
		t.Fatalf("ParseFFUFArtifact() error = %v", parseErr)
	}
	if summary.ParsedItems != 1 || len(observations) != 1 || observations[0].Item.URL != server.URL+"/admin" {
		t.Fatalf("pre-deadline summary=%#v observations=%#v", summary, observations)
	}
}

func ffufIntegrationConfig(wordlist string) enginecontract.FfufConfig {
	config := defaultFFUFConfig()
	config.Wordlist = wordlist
	config.AutoCalibration = false
	config.MatchCodes = "all"
	config.Threads = 1
	config.Delay = "0"
	config.RequestTimeout = 5
	return config
}

func requirePinnedFFUFBinary(t *testing.T) string {
	t.Helper()
	binary := os.Getenv(ffufIntegrationBinaryEnvironment)
	explicit := binary != ""
	if binary == "" {
		resolved, err := exec.LookPath(FFUFBinary)
		if err != nil {
			t.Skipf("pinned FFUF integration unavailable; set %s", ffufIntegrationBinaryEnvironment)
		}
		binary = resolved
	}
	output, err := exec.Command(binary, "-V").CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "ffuf version: 2.2.1" {
		if explicit {
			t.Fatalf("configured FFUF must be v2.2.1: output %q, error %v", output, err)
		}
		t.Skipf("PATH FFUF is not pinned v2.2.1; set %s", ffufIntegrationBinaryEnvironment)
	}
	return binary
}

func runPinnedFFUF(
	t *testing.T,
	binary string,
	candidate string,
	config enginecontract.FfufConfig,
) ([]ffufJSONRecord, string) {
	t.Helper()
	args, err := BuildFFUFArgs(candidate, config)
	if err != nil {
		t.Fatalf("BuildFFUFArgs() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, args...)
	configHome := t.TempDir()
	if err := os.Mkdir(filepath.Join(configHome, "ffuf"), 0o700); err != nil {
		t.Fatal(err)
	}
	command.Env = append(withoutEnvironment(os.Environ(), "XDG_CONFIG_HOME"), "XDG_CONFIG_HOME="+configHome)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			t.Fatalf("FFUF context error = %v; stderr = %q", ctx.Err(), stderr.String())
		}
		t.Fatalf("FFUF run error = %v; stderr = %q; stdout = %q", err, stderr.String(), stdout.String())
	}

	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	var records []ffufJSONRecord
	for {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("decode FFUF JSONL: %v; stdout = %q", err, stdout.String())
		}
		var record ffufJSONRecord
		if err := json.Unmarshal(raw, &record); err != nil {
			t.Fatalf("decode FFUF JSONL: %v; stdout = %q", err, stdout.String())
		}
		record.raw = append([]byte(nil), raw...)
		records = append(records, record)
	}
	return records, stderr.String()
}

func writeFFUFWordlist(t *testing.T, payloads ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wordlist.txt")
	payload := ""
	if len(payloads) > 0 {
		payload = strings.Join(payloads, "\n") + "\n"
	}
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func withoutEnvironment(environment []string, key string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(environment))
	for _, value := range environment {
		if !strings.HasPrefix(value, prefix) {
			filtered = append(filtered, value)
		}
	}
	return filtered
}
