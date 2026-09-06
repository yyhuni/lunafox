package agentdata

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type executionInputBlacklistMetricSample struct {
	ctx         context.Context
	value       int64
	attributes  map[string]string
	optionCount int
}

type executionInputBlacklistCounterStub struct {
	samples []executionInputBlacklistMetricSample
}

func (stub *executionInputBlacklistCounterStub) Add(ctx context.Context, value int64, options ...metric.AddOption) {
	attributes := map[string]string{}
	config := metric.NewAddConfig(options)
	attributeSet := config.Attributes()
	for _, value := range attributeSet.ToSlice() {
		attributes[string(value.Key)] = value.Value.AsString()
	}
	stub.samples = append(stub.samples, executionInputBlacklistMetricSample{
		ctx: ctx, value: value, attributes: attributes, optionCount: len(options),
	})
}

func installExecutionInputBlacklistObservationTestTelemetry(t *testing.T) (*observer.ObservedLogs, *executionInputBlacklistCounterStub) {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)
	previousLogger := pkg.Logger
	previousSugar := pkg.Sugar
	previousCounter := executionInputBlacklistRecords
	counter := &executionInputBlacklistCounterStub{}
	pkg.Logger = logger
	pkg.Sugar = logger.Sugar()
	executionInputBlacklistRecords = counter
	t.Cleanup(func() {
		pkg.Logger = previousLogger
		pkg.Sugar = previousSugar
		executionInputBlacklistRecords = previousCounter
	})
	return logs, counter
}

func TestExecutionInputBlacklistObservationLogsExactlyOnceWithPartialCounts(t *testing.T) {
	logs, counter := installExecutionInputBlacklistObservationTestTelemetry(t)
	observation := newExecutionInputBlacklistObservation(artifactRoleSubdomains, 23, 31)
	if observation == nil {
		t.Fatal("expected subdomains observation")
	}
	observation.setStats(func() executionInputBlacklistStats {
		return executionInputBlacklistStats{examined: 4, excluded: 2, emitted: 2}
	})
	// The raw error intentionally contains values that must not leave this
	// terminal observation as a structured log field or metric attribute.
	observation.observe(status.Error(codes.DataLoss, "matched https://secret.example/192.0.2.11"))
	observation.observe(status.Error(codes.DataLoss, "second observation is forbidden"))

	entries := logs.FilterMessage("execution input blacklist materialized").All()
	if len(entries) != 1 {
		t.Fatalf("blacklist terminal logs = %d, want exactly one", len(entries))
	}
	entry := entries[0]
	if entry.Level != zapcore.ErrorLevel {
		t.Fatalf("terminal log level = %s, want error", entry.Level)
	}
	want := map[string]any{
		"scan.id":                   int64(23),
		"task.id":                   int64(31),
		"execution.input.role":      "subdomains",
		"blacklist.filter.outcome":  "failed",
		"blacklist.filter.examined": uint64(4),
		"blacklist.filter.excluded": uint64(2),
		"blacklist.filter.emitted":  uint64(2),
		"failure.kind":              "immutable_input_invalid",
	}
	if got := entry.ContextMap(); !reflect.DeepEqual(got, want) {
		t.Fatalf("terminal log fields = %#v, want %#v", got, want)
	}
	serialized := entry.Message + " " + strings.Join(mapStringValues(entry.ContextMap()), " ")
	for _, forbidden := range []string{"secret.example", "192.0.2.11", "matched"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("terminal log leaked %q: %q", forbidden, serialized)
		}
	}
	assertExecutionInputBlacklistMetricSamples(t, counter.samples, "subdomains", "failed", executionInputBlacklistStats{examined: 4, excluded: 2, emitted: 2})
}

func TestExecutionInputBlacklistObservationClassifiesTerminalLevelsAndFailureKinds(t *testing.T) {
	tests := []struct {
		name                 string
		err                  error
		transportInterrupted bool
		wantOutcome          string
		wantFailureKind      string
		wantLevel            zapcore.Level
	}{
		{name: "completed", wantOutcome: "completed", wantLevel: zapcore.InfoLevel},
		{name: "cancelled", err: context.Canceled, wantOutcome: "cancelled", wantFailureKind: "context_cancelled", wantLevel: zapcore.WarnLevel},
		{name: "deadline", err: context.DeadlineExceeded, wantOutcome: "cancelled", wantFailureKind: "deadline_exceeded", wantLevel: zapcore.WarnLevel},
		{name: "retryable transfer interruption", err: status.Error(codes.Unavailable, "connection reset"), transportInterrupted: true, wantOutcome: "failed", wantFailureKind: "transfer_interrupted", wantLevel: zapcore.WarnLevel},
		{name: "snapshot dependency unavailable", err: status.Error(codes.Unavailable, "snapshot store unavailable"), wantOutcome: "failed", wantFailureKind: "input_unavailable", wantLevel: zapcore.ErrorLevel},
		{name: "immutable input corrupt", err: status.Error(codes.DataLoss, "corrupt snapshot"), wantOutcome: "failed", wantFailureKind: "immutable_input_invalid", wantLevel: zapcore.ErrorLevel},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logs, _ := installExecutionInputBlacklistObservationTestTelemetry(t)
			observation := newExecutionInputBlacklistObservation(artifactRoleWebsiteURLs, 23, 31)
			observation.setStats(func() executionInputBlacklistStats { return executionInputBlacklistStats{examined: 1, emitted: 1} })
			if test.transportInterrupted {
				observation.markTransportInterrupted()
			}
			observation.observe(test.err)

			entries := logs.FilterMessage("execution input blacklist materialized").All()
			if len(entries) != 1 {
				t.Fatalf("terminal logs = %d, want 1", len(entries))
			}
			entry := entries[0]
			if entry.Level != test.wantLevel {
				t.Fatalf("log level = %s, want %s", entry.Level, test.wantLevel)
			}
			fields := entry.ContextMap()
			if got := fields["blacklist.filter.outcome"]; got != test.wantOutcome {
				t.Fatalf("outcome = %#v, want %q", got, test.wantOutcome)
			}
			if test.wantFailureKind == "" {
				if _, ok := fields["failure.kind"]; ok {
					t.Fatalf("completed log must not have failure.kind: %#v", fields)
				}
			} else if got := fields["failure.kind"]; got != test.wantFailureKind {
				t.Fatalf("failure.kind = %#v, want %q", got, test.wantFailureKind)
			}
		})
	}
}

func TestExecutionInputBlacklistObservationRecordsClosedMetricMatrix(t *testing.T) {
	_, counter := installExecutionInputBlacklistObservationTestTelemetry(t)
	roles := []executionArtifactRole{artifactRoleSubdomains, artifactRoleHostPorts, artifactRoleWebsiteURLs}
	terminals := []struct {
		err     error
		outcome string
	}{
		{outcome: "completed"},
		{err: context.Canceled, outcome: "cancelled"},
		{err: status.Error(codes.DataLoss, "corrupt immutable input"), outcome: "failed"},
	}
	stats := executionInputBlacklistStats{examined: 3, excluded: 1, emitted: 2}
	for _, role := range roles {
		for _, terminal := range terminals {
			observation := newExecutionInputBlacklistObservation(role, 23, 31)
			observation.setStats(func() executionInputBlacklistStats { return stats })
			observation.observe(terminal.err)
		}
	}

	if got, want := len(counter.samples), 27; got != want {
		t.Fatalf("metric samples = %d, want all %d closed combinations", got, want)
	}
	seen := map[string]bool{}
	for _, sample := range counter.samples {
		if sample.ctx != context.Background() {
			t.Fatalf("metric context = %#v, want context.Background to prevent trace exemplars", sample.ctx)
		}
		if sample.optionCount != 1 || len(sample.attributes) != 3 {
			t.Fatalf("metric attributes/options = %#v/%d, want exactly role/disposition/outcome", sample.attributes, sample.optionCount)
		}
		for _, forbidden := range []string{"scan.id", "task.id", "target", "hostname", "ip", "url", "pattern", "etag", "digest", "error"} {
			if _, ok := sample.attributes[forbidden]; ok {
				t.Fatalf("metric attributes leaked %q: %#v", forbidden, sample.attributes)
			}
		}
		role := sample.attributes["role"]
		disposition := sample.attributes["disposition"]
		outcome := sample.attributes["outcome"]
		if !isExecutionInputBlacklistMetricRole(role) || !isExecutionInputBlacklistMetricDisposition(disposition) || !isExecutionInputBlacklistMetricOutcome(outcome) {
			t.Fatalf("metric attributes outside closed vocabulary: %#v", sample.attributes)
		}
		key := role + "\x00" + disposition + "\x00" + outcome
		if seen[key] {
			t.Fatalf("duplicate closed metric combination %q", key)
		}
		seen[key] = true
		wantValue := map[string]int64{"examined": 3, "excluded": 1, "emitted": 2}[disposition]
		if sample.value != wantValue {
			t.Fatalf("%s value = %d, want %d", key, sample.value, wantValue)
		}
	}
}

func TestExecutionInputBlacklistObservationCountsRetriesAsWork(t *testing.T) {
	logs, counter := installExecutionInputBlacklistObservationTestTelemetry(t)
	stats := executionInputBlacklistStats{examined: 2, excluded: 1, emitted: 1}
	for attempt := 0; attempt < 2; attempt++ {
		observation := newExecutionInputBlacklistObservation(artifactRoleHostPorts, 23, 31)
		observation.setStats(func() executionInputBlacklistStats { return stats })
		observation.observe(nil)
	}
	if got := len(logs.FilterMessage("execution input blacklist materialized").All()); got != 2 {
		t.Fatalf("retry terminal logs = %d, want 2", got)
	}
	if got := len(counter.samples); got != 6 {
		t.Fatalf("retry metric samples = %d, want 6", got)
	}
	for _, sample := range counter.samples {
		if sample.attributes["outcome"] != "completed" || sample.attributes["role"] != "hostPorts" {
			t.Fatalf("retry sample attributes = %#v", sample.attributes)
		}
	}
}

func TestServerExecutionArtifactResolverExposesFilteredStatsOnlyAsPrivateMetadata(t *testing.T) {
	plan := validExecutionArtifactPlan()
	setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)
	resolver := newExecutionArtifactResolverForTestWithSnapshotSource(
		t,
		plan,
		&executionArtifactTaskReaderStub{},
		executionArtifactDNSCursorStub{records: []string{"blocked.example", "api.blocked.example", "keep.example"}},
		executionArtifactHostPortCursorStub{},
		&executionInputBlacklistSnapshotSourceStub{patterns: []string{"*.blocked.example", "blocked.example"}},
		&executionWordlistSourceStub{},
		executionProviderConfigSourceStub{content: "alienvault: []\n"},
	)
	artifact, err := resolver.AuthorizeExecutionInput(context.Background(), AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch}, subdomainsRequestForPlan(plan))
	if err != nil {
		t.Fatalf("AuthorizeExecutionInput() error = %v", err)
	}
	if artifact.executionInputBlacklistStats == nil {
		t.Fatal("filtered input must carry private aggregate stats metadata")
	}
	if records, err := artifact.Produce(context.Background(), io.Discard); err != nil || records != 1 {
		t.Fatalf("artifact produce records/error = %d/%v, want 1/nil", records, err)
	}
	if got, want := artifact.executionInputBlacklistStats(), (executionInputBlacklistStats{examined: 3, excluded: 2, emitted: 1}); got != want {
		t.Fatalf("private filtered stats = %#v, want %#v", got, want)
	}
}

func TestStreamExecutionInputConsumesPrivateFilteredStatsOnce(t *testing.T) {
	logs, counter := installExecutionInputBlacklistObservationTestTelemetry(t)
	plan := validExecutionArtifactPlan()
	setPlanInputBindingForTest(plan, executionartifact.RoleSubdomainsInput, executionartifact.ContentTypeSubdomains)
	resolver := newExecutionArtifactResolverForTestWithSnapshotSource(
		t,
		plan,
		&executionArtifactTaskReaderStub{},
		executionArtifactDNSCursorStub{records: []string{"blocked.example", "api.blocked.example", "keep.example"}},
		executionArtifactHostPortCursorStub{},
		&executionInputBlacklistSnapshotSourceStub{patterns: []string{"*.blocked.example", "blocked.example"}},
		&executionWordlistSourceStub{},
		executionProviderConfigSourceStub{content: "alienvault: []\n"},
	)
	service := NewExecutionArtifactService(&agentFinderStub{agent: &agentdomain.Agent{ID: 42}}, resolver)
	stream := &executionArtifactFrameStream{ctx: agentAuthContext()}
	if err := service.StreamExecutionInput(subdomainsRequestForPlan(plan), stream); err != nil {
		t.Fatalf("StreamExecutionInput() error = %v", err)
	}
	entries := logs.FilterMessage("execution input blacklist materialized").All()
	if len(entries) != 1 {
		t.Fatalf("terminal aggregate logs = %d, want exactly one", len(entries))
	}
	if fields := entries[0].ContextMap(); fields["blacklist.filter.outcome"] != "completed" || fields["blacklist.filter.examined"] != uint64(3) || fields["blacklist.filter.excluded"] != uint64(2) || fields["blacklist.filter.emitted"] != uint64(1) {
		t.Fatalf("terminal aggregate fields = %#v", fields)
	}
	assertExecutionInputBlacklistMetricSamples(t, counter.samples, "subdomains", "completed", executionInputBlacklistStats{examined: 3, excluded: 2, emitted: 1})
}

func TestExecutionInputBlacklistTelemetryDoesNotLeakIntoPersistentOrPublicSurfaces(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", ".."))
	for _, path := range []string{
		filepath.Join(repoRoot, "contracts"),
		filepath.Join(repoRoot, "frontend"),
		filepath.Join(repoRoot, "server", "internal", "modules", "scan", "dto"),
		filepath.Join(repoRoot, "server", "internal", "modules", "scan", "handler"),
		filepath.Join(repoRoot, "server", "internal", "modules", "scan", "repository"),
	} {
		assertExecutionInputBlacklistTelemetryAbsent(t, path)
	}
}

func assertExecutionInputBlacklistMetricSamples(t *testing.T, samples []executionInputBlacklistMetricSample, role, outcome string, stats executionInputBlacklistStats) {
	t.Helper()
	if got, want := len(samples), 3; got != want {
		t.Fatalf("metric samples = %d, want %d", got, want)
	}
	values := map[string]int64{"examined": int64(stats.examined), "excluded": int64(stats.excluded), "emitted": int64(stats.emitted)}
	for _, sample := range samples {
		if !reflect.DeepEqual(sample.attributes, map[string]string{"role": role, "disposition": sample.attributes["disposition"], "outcome": outcome}) {
			t.Fatalf("metric attributes = %#v, want closed role/disposition/outcome fields", sample.attributes)
		}
		if want, ok := values[sample.attributes["disposition"]]; !ok || sample.value != want {
			t.Fatalf("metric sample = %#v, want disposition value from %#v", sample, values)
		}
	}
}

func isExecutionInputBlacklistMetricRole(value string) bool {
	return value == "subdomains" || value == "hostPorts" || value == "websiteURLs"
}

func isExecutionInputBlacklistMetricDisposition(value string) bool {
	return value == "examined" || value == "excluded" || value == "emitted"
}

func isExecutionInputBlacklistMetricOutcome(value string) bool {
	return value == "completed" || value == "cancelled" || value == "failed"
}

func mapStringValues(values map[string]any) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if stringValue, ok := value.(string); ok {
			result = append(result, stringValue)
		}
	}
	return result
}

func assertExecutionInputBlacklistTelemetryAbsent(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case "node_modules", ".next", "coverage", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".proto", ".ts", ".tsx":
		default:
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, forbidden := range []string{executionInputBlacklistRecordsMetricName, "blacklist.filter."} {
			if strings.Contains(string(contents), forbidden) {
				return &executionInputBlacklistTelemetryLeakError{path: path, value: forbidden}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

type executionInputBlacklistTelemetryLeakError struct {
	path  string
	value string
}

func (err *executionInputBlacklistTelemetryLeakError) Error() string {
	return "execution-input blacklist telemetry leaked into " + err.path + ": " + err.value
}

var _ executionInputBlacklistRecordCounter = (*executionInputBlacklistCounterStub)(nil)
