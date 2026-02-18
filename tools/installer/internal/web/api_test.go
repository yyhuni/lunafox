package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/installapp"
)

type mockInstallService struct {
	startFn     func(options cli.Options) (string, error)
	snapshotFn  func(jobID string) (installapp.InstallStateSnapshot, error)
	subscribeFn func(jobID string, afterID int64) (<-chan installapp.InstallEvent, func(), error)
}

func (service mockInstallService) Start(options cli.Options) (string, error) {
	return service.startFn(options)
}

func (service mockInstallService) Snapshot(jobID string) (installapp.InstallStateSnapshot, error) {
	return service.snapshotFn(jobID)
}

func (service mockInstallService) Subscribe(jobID string, afterID int64) (<-chan installapp.InstallEvent, func(), error) {
	return service.subscribeFn(jobID, afterID)
}

type mockRunningErr struct {
	jobID string
}

func (err mockRunningErr) Error() string        { return "running" }
func (err mockRunningErr) RunningJobID() string { return err.jobID }

type mockNotFoundErr struct{}

func (mockNotFoundErr) Error() string       { return "missing" }
func (mockNotFoundErr) IsJobNotFound() bool { return true }

func newRouterForAPI(t *testing.T, service InstallService) *Router {
	t.Helper()
	return newRouterForAPIWithMode(t, cli.ModeProd, service)
}

func newRouterForAPIWithMode(t *testing.T, mode string, service InstallService) *Router {
	t.Helper()
	root := t.TempDir()
	dockerDir := filepath.Join(root, "docker")
	return NewRouter(cli.Options{
		Mode:           mode,
		PublicURL:      "https://example.com:8083",
		PublicPort:     "8083",
		AgentServerURL: "http://server:8080",
		AgentNetwork:   "lunafox_network",
		RootDir:        root,
		DockerDir:      dockerDir,
		ComposeDev:     filepath.Join(dockerDir, "docker-compose.dev.yml"),
		ComposeProd:    filepath.Join(dockerDir, "docker-compose.yml"),
		ComposeFile:    filepath.Join(dockerDir, "docker-compose.yml"),
		GoProxy:        "https://proxy.golang.org,direct",
	}, service)
}

func TestStartAPIAccepted(t *testing.T) {
	router := newRouterForAPI(t, mockInstallService{
		startFn: func(options cli.Options) (string, error) {
			if options.PublicURL != "https://example.com:8083" {
				t.Fatalf("unexpected url: %s", options.PublicURL)
			}
			return "job-1", nil
		},
		snapshotFn: func(string) (installapp.InstallStateSnapshot, error) {
			return installapp.InstallStateSnapshot{}, nil
		},
		subscribeFn: func(string, int64) (<-chan installapp.InstallEvent, func(), error) {
			return make(chan installapp.InstallEvent), func() {}, nil
		},
	})

	body := bytes.NewBufferString(`{"publicHost":"example.com","publicPort":"8083","useGoProxyCN":false}`)
	request := httptest.NewRequest(http.MethodPost, "/api/start", body)
	recorder := httptest.NewRecorder()
	router.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	var resp startResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.JobID != "job-1" || resp.State != "running" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestStartAPIConflict(t *testing.T) {
	router := newRouterForAPI(t, mockInstallService{
		startFn: func(cli.Options) (string, error) {
			return "", mockRunningErr{jobID: "job-running"}
		},
		snapshotFn: func(string) (installapp.InstallStateSnapshot, error) {
			return installapp.InstallStateSnapshot{}, nil
		},
		subscribeFn: func(string, int64) (<-chan installapp.InstallEvent, func(), error) {
			return make(chan installapp.InstallEvent), func() {}, nil
		},
	})

	body := bytes.NewBufferString(`{"publicHost":"example.com","publicPort":"8083","useGoProxyCN":false}`)
	request := httptest.NewRequest(http.MethodPost, "/api/start", body)
	recorder := httptest.NewRecorder()
	router.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "JOB_ALREADY_RUNNING") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestStartAPIDevRejectsEmptyPublicHost(t *testing.T) {
	router := newRouterForAPIWithMode(t, cli.ModeDev, mockInstallService{
		startFn: func(cli.Options) (string, error) {
			t.Fatalf("service start should not be called when options are invalid")
			return "", nil
		},
		snapshotFn: func(string) (installapp.InstallStateSnapshot, error) {
			return installapp.InstallStateSnapshot{}, nil
		},
		subscribeFn: func(string, int64) (<-chan installapp.InstallEvent, func(), error) {
			return make(chan installapp.InstallEvent), func() {}, nil
		},
	})

	body := bytes.NewBufferString(`{"publicHost":"","publicPort":"8083","useGoProxyCN":true}`)
	request := httptest.NewRequest(http.MethodPost, "/api/start", body)
	recorder := httptest.NewRecorder()
	router.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "必须填写公网主机") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestStateAPINotFound(t *testing.T) {
	router := newRouterForAPI(t, mockInstallService{
		startFn: func(cli.Options) (string, error) { return "job-1", nil },
		snapshotFn: func(string) (installapp.InstallStateSnapshot, error) {
			return installapp.InstallStateSnapshot{}, mockNotFoundErr{}
		},
		subscribeFn: func(string, int64) (<-chan installapp.InstallEvent, func(), error) {
			return make(chan installapp.InstallEvent), func() {}, nil
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/state?jobId=missing", nil)
	recorder := httptest.NewRecorder()
	router.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "JOB_NOT_FOUND") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestEventsAPIStream(t *testing.T) {
	events := make(chan installapp.InstallEvent, 2)
	events <- installapp.InstallEvent{
		ID:    1,
		JobID: "job-1",
		Type:  installapp.EventLog,
		Data:  installapp.LogEvent{Message: "hello\n"},
	}
	events <- installapp.InstallEvent{
		ID:    2,
		JobID: "job-1",
		Type:  installapp.EventDone,
		Data:  installapp.InstallStateSnapshot{JobID: "job-1", State: "succeeded"},
	}
	close(events)

	router := newRouterForAPI(t, mockInstallService{
		startFn: func(cli.Options) (string, error) { return "job-1", nil },
		snapshotFn: func(string) (installapp.InstallStateSnapshot, error) {
			return installapp.InstallStateSnapshot{}, nil
		},
		subscribeFn: func(string, int64) (<-chan installapp.InstallEvent, func(), error) {
			return events, func() {}, nil
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/events?jobId=job-1", nil)
	request = request.WithContext(context.Background())
	recorder := httptest.NewRecorder()
	router.Handler().ServeHTTP(recorder, request)

	result := recorder.Result()
	defer result.Body.Close()
	body, _ := io.ReadAll(result.Body)
	text := string(body)
	if result.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", result.StatusCode, text)
	}
	if !strings.Contains(text, "event: log") || !strings.Contains(text, "event: done") {
		t.Fatalf("unexpected sse body: %s", text)
	}
}

func TestNetworkReachabilityRejectInvalidHost(t *testing.T) {
	router := newRouterForAPI(t, mockInstallService{
		startFn: func(cli.Options) (string, error) { return "job-1", nil },
		snapshotFn: func(string) (installapp.InstallStateSnapshot, error) {
			return installapp.InstallStateSnapshot{}, nil
		},
		subscribeFn: func(string, int64) (<-chan installapp.InstallEvent, func(), error) {
			return make(chan installapp.InstallEvent), func() {}, nil
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/network/reachability?host=https://bad&port=8083", nil)
	recorder := httptest.NewRecorder()
	router.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "INVALID_HOST") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestNetworkReachabilityRejectEmptyHost(t *testing.T) {
	router := newRouterForAPI(t, mockInstallService{
		startFn: func(cli.Options) (string, error) { return "job-1", nil },
		snapshotFn: func(string) (installapp.InstallStateSnapshot, error) {
			return installapp.InstallStateSnapshot{}, nil
		},
		subscribeFn: func(string, int64) (<-chan installapp.InstallEvent, func(), error) {
			return make(chan installapp.InstallEvent), func() {}, nil
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/network/reachability?host=&port=8083", nil)
	recorder := httptest.NewRecorder()
	router.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "INVALID_HOST") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "必须填写公网主机") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestNetworkReachabilityProdRejectsLoopback(t *testing.T) {
	router := newRouterForAPI(t, mockInstallService{
		startFn: func(cli.Options) (string, error) { return "job-1", nil },
		snapshotFn: func(string) (installapp.InstallStateSnapshot, error) {
			return installapp.InstallStateSnapshot{}, nil
		},
		subscribeFn: func(string, int64) (<-chan installapp.InstallEvent, func(), error) {
			return make(chan installapp.InstallEvent), func() {}, nil
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/network/reachability?host=localhost&port=8083", nil)
	recorder := httptest.NewRecorder()
	router.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"ok":false`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}
