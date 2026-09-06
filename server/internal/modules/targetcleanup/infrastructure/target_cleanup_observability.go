package infrastructure

import (
	"context"
	"sync/atomic"
	"time"

	targetcleanupapp "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/application"
	cleanupdomain "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/zap"
)

var targetCleanupMeter = otel.Meter("lunafox.target.cleanup")

type targetCleanupEventLogger interface {
	Info(string, ...zap.Field)
	Error(string, ...zap.Field)
}

type targetCleanupMetricRecorder interface {
	RecordRun(targetcleanupapp.TargetCleanupRunEvent)
	RecordBacklog(cleanupdomain.CleanupBacklog, time.Time)
}

type targetCleanupRunObserver struct {
	logger  targetCleanupEventLogger
	metrics targetCleanupMetricRecorder
	now     func() time.Time
}

func NewTargetCleanupRunObserver() targetcleanupapp.TargetCleanupRunObserver {
	return newTargetCleanupRunObserver(targetCleanupPackageLogger{}, newTargetCleanupOTelMetrics(targetCleanupMeter), time.Now)
}

func newTargetCleanupRunObserver(
	logger targetCleanupEventLogger,
	metrics targetCleanupMetricRecorder,
	now func() time.Time,
) *targetCleanupRunObserver {
	if logger == nil {
		logger = targetCleanupPackageLogger{}
	}
	if metrics == nil {
		metrics = noopTargetCleanupMetrics{}
	}
	if now == nil {
		now = time.Now
	}
	return &targetCleanupRunObserver{logger: logger, metrics: metrics, now: now}
}

func (observer *targetCleanupRunObserver) RunStarted(job cleanupdomain.CleanupJob, startedAt time.Time) {
	if observer == nil {
		return
	}
	observer.logger.Info("Target cleanup run started",
		zap.Int("target_cleanup.job.id", job.ID),
		zap.Int("target_cleanup.target.id", job.TargetID),
		zap.Int("target_cleanup.retry_count", job.RetryCount),
		zap.Time("target_cleanup.started_at", startedAt.UTC()),
	)
}

func (observer *targetCleanupRunObserver) RunFinished(event targetcleanupapp.TargetCleanupRunEvent) {
	if observer == nil {
		return
	}
	duration := event.FinishedAt.Sub(event.StartedAt)
	if duration < 0 {
		duration = 0
	}
	fields := []zap.Field{
		zap.Int("target_cleanup.job.id", event.Job.ID),
		zap.Int("target_cleanup.target.id", event.Job.TargetID),
		zap.String("target_cleanup.outcome", boundedTargetCleanupOutcome(event.Outcome)),
		zap.Duration("target_cleanup.duration", duration),
		zap.Int("target_cleanup.retry_count", event.RetryCount),
		zap.Int("target_cleanup.asset_batches", event.Result.AssetBatches),
		zap.Int64("target_cleanup.asset_rows_deleted", event.Result.Counts.TotalAssetRows()),
		zap.Int("target_cleanup.schedules_deleted", event.Result.Counts.Schedules),
		zap.Int64("target_cleanup.organization_relations_deleted", event.Result.Counts.OrganizationRelations),
		zap.Int64("target_cleanup.target_policies_deleted", event.Result.Counts.TargetPolicies),
		zap.Int("target_cleanup.scans_cancelled", event.Result.Counts.Scans),
		zap.Int("target_cleanup.tasks_cancelled", event.Result.Counts.Tasks),
		zap.Int("target_cleanup.agent_notifications", event.Result.Counts.AgentNotifications),
		zap.Int("target_cleanup.agent_notification_failures", event.Result.Counts.AgentNotificationFails),
	}
	if event.NextRetryAt != nil {
		fields = append(fields, zap.Time("target_cleanup.next_retry_at", event.NextRetryAt.UTC()))
	}
	if failureClass := boundedTargetCleanupFailureClass(event.FailureClass); failureClass != "" {
		fields = append(fields, zap.String("target_cleanup.error_class", failureClass))
	}
	if boundedTargetCleanupOutcome(event.Outcome) == "failed" {
		observer.logger.Error("Target cleanup run finished", fields...)
	} else {
		observer.logger.Info("Target cleanup run finished", fields...)
	}
	observer.metrics.RecordRun(event)
}

func (observer *targetCleanupRunObserver) BacklogObserved(backlog cleanupdomain.CleanupBacklog) {
	if observer == nil {
		return
	}
	observer.metrics.RecordBacklog(backlog, observer.now().UTC())
}

type targetCleanupPackageLogger struct{}

func (targetCleanupPackageLogger) Info(message string, fields ...zap.Field) {
	pkg.Info(message, fields...)
}

func (targetCleanupPackageLogger) Error(message string, fields ...zap.Field) {
	pkg.Error(message, fields...)
}

type noopTargetCleanupMetrics struct{}

func (noopTargetCleanupMetrics) RecordRun(targetcleanupapp.TargetCleanupRunEvent) {}
func (noopTargetCleanupMetrics) RecordBacklog(cleanupdomain.CleanupBacklog, time.Time) {
}

type targetCleanupOTelMetrics struct {
	runs                metric.Int64Counter
	duration            metric.Float64Histogram
	deletedRows         metric.Int64Counter
	scansCancelled      metric.Int64Counter
	tasksCancelled      metric.Int64Counter
	agentNotifications  metric.Int64Counter
	retries             metric.Int64Counter
	unfinishedJobs      atomic.Int64
	oldestUnfinishedAge atomic.Int64
	gaugeRegistrations  []metric.Registration
}

func newTargetCleanupOTelMetrics(meter metric.Meter) *targetCleanupOTelMetrics {
	metrics := &targetCleanupOTelMetrics{
		runs:               mustTargetCleanupCounter(meter, "target_cleanup_runs_total"),
		duration:           mustTargetCleanupHistogram(meter, "target_cleanup_run_duration_seconds"),
		deletedRows:        mustTargetCleanupCounter(meter, "target_cleanup_deleted_rows_total"),
		scansCancelled:     mustTargetCleanupCounter(meter, "target_cleanup_scans_cancelled_total"),
		tasksCancelled:     mustTargetCleanupCounter(meter, "target_cleanup_tasks_cancelled_total"),
		agentNotifications: mustTargetCleanupCounter(meter, "target_cleanup_agent_notifications_total"),
		retries:            mustTargetCleanupCounter(meter, "target_cleanup_retries_total"),
	}
	unfinished, unfinishedErr := meter.Int64ObservableGauge("target_cleanup_unfinished_jobs")
	oldestAge, oldestAgeErr := meter.Int64ObservableGauge("target_cleanup_oldest_unfinished_age_seconds")
	if unfinishedErr == nil && oldestAgeErr == nil {
		registration, err := meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
			attributes := metric.WithAttributes(attribute.String("status", "pending"))
			observer.ObserveInt64(unfinished, metrics.unfinishedJobs.Load(), attributes)
			observer.ObserveInt64(oldestAge, metrics.oldestUnfinishedAge.Load(), attributes)
			return nil
		}, unfinished, oldestAge)
		if err == nil {
			metrics.gaugeRegistrations = append(metrics.gaugeRegistrations, registration)
		}
	}
	return metrics
}

func mustTargetCleanupCounter(meter metric.Meter, name string) metric.Int64Counter {
	counter, err := meter.Int64Counter(name)
	if err != nil {
		return metricnoop.Int64Counter{}
	}
	return counter
}

func mustTargetCleanupHistogram(meter metric.Meter, name string) metric.Float64Histogram {
	histogram, err := meter.Float64Histogram(name)
	if err != nil {
		return metricnoop.Float64Histogram{}
	}
	return histogram
}

func (metrics *targetCleanupOTelMetrics) RecordRun(event targetcleanupapp.TargetCleanupRunEvent) {
	if metrics == nil {
		return
	}
	outcome := boundedTargetCleanupOutcome(event.Outcome)
	failureClass := boundedTargetCleanupFailureClass(event.FailureClass)
	attributes := targetCleanupMetricAttributes(outcome, failureClass)
	metrics.runs.Add(context.Background(), 1, metric.WithAttributes(attributes...))
	duration := event.FinishedAt.Sub(event.StartedAt)
	if duration < 0 {
		duration = 0
	}
	metrics.duration.Record(context.Background(), duration.Seconds(), metric.WithAttributes(attributes...))
	counts := event.Result.Counts
	for _, resource := range cleanupdomain.OrderedAssetResources() {
		if rows := counts.AssetRows[resource]; rows > 0 {
			metrics.deletedRows.Add(context.Background(), rows, metric.WithAttributes(attribute.String("resource", string(resource))))
		}
	}
	metrics.addDeletedRows("scheduled_scan", int64(counts.Schedules))
	metrics.addDeletedRows("organization_target", counts.OrganizationRelations)
	metrics.addDeletedRows("blacklist_policy", counts.TargetPolicies)
	if counts.Scans > 0 {
		metrics.scansCancelled.Add(context.Background(), int64(counts.Scans))
	}
	if counts.Tasks > 0 {
		metrics.tasksCancelled.Add(context.Background(), int64(counts.Tasks))
	}
	if counts.AgentNotifications > counts.AgentNotificationFails {
		metrics.agentNotifications.Add(context.Background(), int64(counts.AgentNotifications-counts.AgentNotificationFails), metric.WithAttributes(attribute.String("outcome", "delivered")))
	}
	if counts.AgentNotificationFails > 0 {
		metrics.agentNotifications.Add(context.Background(), int64(counts.AgentNotificationFails), metric.WithAttributes(attribute.String("outcome", "failed")))
	}
	if outcome == "failed" {
		metrics.retries.Add(context.Background(), 1, metric.WithAttributes(targetCleanupMetricAttributes("failed", failureClass)...))
	}
}

func (metrics *targetCleanupOTelMetrics) RecordBacklog(backlog cleanupdomain.CleanupBacklog, observedAt time.Time) {
	if metrics == nil {
		return
	}
	metrics.unfinishedJobs.Store(backlog.UnfinishedCount)
	ageSeconds := int64(0)
	if backlog.OldestCreatedAt != nil {
		age := observedAt.Sub(backlog.OldestCreatedAt.UTC())
		if age > 0 {
			ageSeconds = int64(age.Seconds())
		}
	}
	metrics.oldestUnfinishedAge.Store(ageSeconds)
}

func (metrics *targetCleanupOTelMetrics) addDeletedRows(resource string, rows int64) {
	if rows > 0 {
		metrics.deletedRows.Add(context.Background(), rows, metric.WithAttributes(attribute.String("resource", resource)))
	}
}

func targetCleanupMetricAttributes(outcome, failureClass string) []attribute.KeyValue {
	attributes := []attribute.KeyValue{attribute.String("outcome", boundedTargetCleanupOutcome(outcome))}
	if bounded := boundedTargetCleanupFailureClass(failureClass); bounded != "" {
		attributes = append(attributes, attribute.String("error_class", bounded))
	}
	return attributes
}

func boundedTargetCleanupOutcome(outcome string) string {
	switch outcome {
	case "completed", "deferred", "failed":
		return outcome
	default:
		return "failed"
	}
}

func boundedTargetCleanupFailureClass(failureClass string) string {
	switch failureClass {
	case "":
		return ""
	case "timeout", "canceled", "database_lock", "database_timeout", "database_or_reconciliation", "diagnostic_persist_failed":
		return failureClass
	default:
		return "database_or_reconciliation"
	}
}
