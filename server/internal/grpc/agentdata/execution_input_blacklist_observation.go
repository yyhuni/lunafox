package agentdata

import (
	"context"
	"errors"
	"math"
	"sync"

	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const executionInputBlacklistRecordsMetricName = "execution_input_blacklist_records_total"

type executionInputBlacklistStats struct {
	examined uint64
	excluded uint64
	emitted  uint64
}

type executionInputBlacklistRecordCounter interface {
	Add(context.Context, int64, ...metric.AddOption)
}

var executionInputBlacklistRecords executionInputBlacklistRecordCounter = mustExecutionArtifactCounter(executionInputBlacklistRecordsMetricName)

type executionInputBlacklistObservation struct {
	scanID               int
	taskID               int
	role                 string
	stats                func() executionInputBlacklistStats
	transportInterrupted bool
	once                 sync.Once
}

func newExecutionInputBlacklistObservation(role executionArtifactRole, scanID, taskID int) *executionInputBlacklistObservation {
	metricRole, ok := executionInputBlacklistMetricRole(role)
	if !ok || scanID <= 0 || taskID <= 0 {
		return nil
	}
	return &executionInputBlacklistObservation{scanID: scanID, taskID: taskID, role: metricRole}
}

func executionInputBlacklistMetricRole(role executionArtifactRole) (string, bool) {
	switch role {
	case artifactRoleSubdomains:
		return "subdomains", true
	case artifactRoleHostPorts:
		return "hostPorts", true
	case artifactRoleWebsiteURLs:
		return "websiteURLs", true
	case artifactRoleEndpointURLs:
		return "endpointURLs", true
	default:
		return "", false
	}
}

func (observation *executionInputBlacklistObservation) setStats(stats func() executionInputBlacklistStats) {
	if observation != nil {
		observation.stats = stats
	}
}

func (observation *executionInputBlacklistObservation) markTransportInterrupted() {
	if observation != nil {
		observation.transportInterrupted = true
	}
}

// observe is deliberately called only by StreamExecutionInput's one defer.
// Keeping metrics on Background prevents a trace context from becoming a
// high-cardinality exemplar for a Scan or task identity.
func (observation *executionInputBlacklistObservation) observe(streamErr error) {
	if observation == nil {
		return
	}
	observation.once.Do(func() {
		stats := executionInputBlacklistStats{}
		if observation.stats != nil {
			stats = observation.stats()
		}
		terminal := classifyExecutionInputBlacklistTerminal(streamErr, observation.transportInterrupted)
		observation.recordMetrics(terminal.outcome, stats)
		observation.log(terminal, stats)
	})
}

type executionInputBlacklistTerminal struct {
	outcome     string
	failureKind string
	level       executionInputBlacklistLogLevel
}

type executionInputBlacklistLogLevel uint8

const (
	executionInputBlacklistLogInfo executionInputBlacklistLogLevel = iota + 1
	executionInputBlacklistLogWarn
	executionInputBlacklistLogError
)

func classifyExecutionInputBlacklistTerminal(err error, transportInterrupted bool) executionInputBlacklistTerminal {
	if err == nil {
		return executionInputBlacklistTerminal{outcome: "completed", level: executionInputBlacklistLogInfo}
	}
	code := status.Code(err)
	switch {
	case errors.Is(err, context.Canceled) || code == codes.Canceled:
		return executionInputBlacklistTerminal{outcome: "cancelled", failureKind: "context_cancelled", level: executionInputBlacklistLogWarn}
	case errors.Is(err, context.DeadlineExceeded) || code == codes.DeadlineExceeded:
		return executionInputBlacklistTerminal{outcome: "cancelled", failureKind: "deadline_exceeded", level: executionInputBlacklistLogWarn}
	case code == codes.DataLoss:
		return executionInputBlacklistTerminal{outcome: "failed", failureKind: "immutable_input_invalid", level: executionInputBlacklistLogError}
	case transportInterrupted:
		return executionInputBlacklistTerminal{outcome: "failed", failureKind: "transfer_interrupted", level: executionInputBlacklistLogWarn}
	case code == codes.Unavailable:
		return executionInputBlacklistTerminal{outcome: "failed", failureKind: "input_unavailable", level: executionInputBlacklistLogError}
	case code == codes.PermissionDenied || code == codes.FailedPrecondition:
		return executionInputBlacklistTerminal{outcome: "failed", failureKind: "authorization_rejected", level: executionInputBlacklistLogError}
	default:
		return executionInputBlacklistTerminal{outcome: "failed", failureKind: "input_materialization_failed", level: executionInputBlacklistLogError}
	}
}

func (observation *executionInputBlacklistObservation) recordMetrics(outcome string, stats executionInputBlacklistStats) {
	for _, disposition := range []struct {
		name  string
		value uint64
	}{
		{name: "examined", value: stats.examined},
		{name: "excluded", value: stats.excluded},
		{name: "emitted", value: stats.emitted},
	} {
		executionInputBlacklistRecords.Add(
			context.Background(),
			executionInputBlacklistMetricValue(disposition.value),
			metric.WithAttributes(
				attribute.String("role", observation.role),
				attribute.String("disposition", disposition.name),
				attribute.String("outcome", outcome),
			),
		)
	}
}

func executionInputBlacklistMetricValue(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}

func (observation *executionInputBlacklistObservation) log(terminal executionInputBlacklistTerminal, stats executionInputBlacklistStats) {
	fields := []zap.Field{
		zap.Int("scan.id", observation.scanID),
		zap.Int("task.id", observation.taskID),
		zap.String("execution.input.role", observation.role),
		zap.String("blacklist.filter.outcome", terminal.outcome),
		zap.Uint64("blacklist.filter.examined", stats.examined),
		zap.Uint64("blacklist.filter.excluded", stats.excluded),
		zap.Uint64("blacklist.filter.emitted", stats.emitted),
	}
	if terminal.failureKind != "" {
		fields = append(fields, zap.String("failure.kind", terminal.failureKind))
	}
	switch terminal.level {
	case executionInputBlacklistLogWarn:
		pkg.Warn("execution input blacklist materialized", fields...)
	case executionInputBlacklistLogError:
		pkg.Error("execution input blacklist materialized", fields...)
	default:
		pkg.Info("execution input blacklist materialized", fields...)
	}
}
