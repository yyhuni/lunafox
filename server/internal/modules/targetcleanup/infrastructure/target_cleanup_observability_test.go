package infrastructure

import (
	"context"
	"strings"
	"testing"
	"time"

	targetcleanupapp "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/application"
	cleanupdomain "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/domain"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

type targetCleanupLogEvent struct {
	level   string
	message string
	fields  []zap.Field
}

type targetCleanupLoggerCapture struct {
	events []targetCleanupLogEvent
}

func (capture *targetCleanupLoggerCapture) Info(message string, fields ...zap.Field) {
	capture.events = append(capture.events, targetCleanupLogEvent{level: "info", message: message, fields: append([]zap.Field(nil), fields...)})
}

func (capture *targetCleanupLoggerCapture) Error(message string, fields ...zap.Field) {
	capture.events = append(capture.events, targetCleanupLogEvent{level: "error", message: message, fields: append([]zap.Field(nil), fields...)})
}

type targetCleanupMetricsCapture struct {
	runs     []targetcleanupapp.TargetCleanupRunEvent
	backlogs []cleanupdomain.CleanupBacklog
}

func (capture *targetCleanupMetricsCapture) RecordRun(event targetcleanupapp.TargetCleanupRunEvent) {
	capture.runs = append(capture.runs, event)
}

func (capture *targetCleanupMetricsCapture) RecordBacklog(backlog cleanupdomain.CleanupBacklog, _ time.Time) {
	capture.backlogs = append(capture.backlogs, backlog)
}

func TestTargetCleanupRunObserverUsesOneStartAndOneBoundedSummary(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	logger := &targetCleanupLoggerCapture{}
	metrics := &targetCleanupMetricsCapture{}
	observer := newTargetCleanupRunObserver(logger, metrics, func() time.Time { return now })
	job := cleanupdomain.CleanupJob{ID: 17, TargetID: 23, RetryCount: 2}
	observer.RunStarted(job, now)
	nextRetry := now.Add(time.Minute)
	observer.RunFinished(targetcleanupapp.TargetCleanupRunEvent{
		Job: job, StartedAt: now, FinishedAt: now.Add(time.Second), Outcome: "failed", RetryCount: 3,
		NextRetryAt: &nextRetry, FailureClass: "raw token=do-not-log",
		Result: targetcleanupapp.TargetCleanupReconciliationResult{Counts: cleanupdomain.CleanupCounts{
			AssetRows: map[cleanupdomain.AssetResource]int64{cleanupdomain.AssetResourceSubdomain: 2},
		}},
	})
	observer.BacklogObserved(cleanupdomain.CleanupBacklog{UnfinishedCount: 4})

	if len(logger.events) != 2 || logger.events[0].level != "info" || logger.events[1].level != "error" {
		t.Fatalf("log events = %+v, want one start and one failed summary", logger.events)
	}
	if len(metrics.runs) != 1 || len(metrics.backlogs) != 1 {
		t.Fatalf("metric observations = runs=%d backlogs=%d", len(metrics.runs), len(metrics.backlogs))
	}
	for _, event := range logger.events {
		for _, field := range event.fields {
			if strings.Contains(strings.ToLower(field.Key), "name") || strings.Contains(field.String, "do-not-log") {
				t.Fatalf("unsafe log field %q=%q", field.Key, field.String)
			}
		}
	}
	if fieldValue(logger.events[1].fields, "target_cleanup.error_class") != "database_or_reconciliation" {
		t.Fatalf("unexpected bounded error class fields: %+v", logger.events[1].fields)
	}
}

func TestTargetCleanupMetricAttributesUseClosedVocabulary(t *testing.T) {
	for _, test := range []struct {
		outcome string
		failure string
		want    map[string]string
	}{
		{outcome: "completed", want: map[string]string{"outcome": "completed"}},
		{outcome: "deferred", want: map[string]string{"outcome": "deferred"}},
		{outcome: "failed", failure: "database_lock", want: map[string]string{"outcome": "failed", "error_class": "database_lock"}},
		{outcome: "unknown", failure: "target=23", want: map[string]string{"outcome": "failed", "error_class": "database_or_reconciliation"}},
	} {
		attributes := targetCleanupMetricAttributes(test.outcome, test.failure)
		got := make(map[string]string, len(attributes))
		for _, attribute := range attributes {
			got[string(attribute.Key)] = attribute.Value.AsString()
		}
		if len(got) != len(test.want) {
			t.Fatalf("attributes(%q, %q) = %#v, want %#v", test.outcome, test.failure, got, test.want)
		}
		for key, want := range test.want {
			if got[key] != want {
				t.Fatalf("attributes(%q, %q) = %#v, want %#v", test.outcome, test.failure, got, test.want)
			}
		}
	}
}

func TestTargetCleanupRunObserverEmitsBoundedSummariesForAllOutcomes(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	for _, outcome := range []string{"completed", "deferred", "failed"} {
		t.Run(outcome, func(t *testing.T) {
			logger := &targetCleanupLoggerCapture{}
			observer := newTargetCleanupRunObserver(logger, &targetCleanupMetricsCapture{}, func() time.Time { return now })
			job := cleanupdomain.CleanupJob{ID: 17, TargetID: 23}
			observer.RunStarted(job, now)
			observer.RunFinished(targetcleanupapp.TargetCleanupRunEvent{
				Job: job, StartedAt: now, FinishedAt: now.Add(time.Second), Outcome: outcome,
				FailureClass: "database_lock",
			})
			if len(logger.events) != 2 || logger.events[0].message != "Target cleanup run started" || logger.events[1].message != "Target cleanup run finished" {
				t.Fatalf("%s logs = %+v, want exactly one start and one summary", outcome, logger.events)
			}
			if got := fieldValue(logger.events[1].fields, "target_cleanup.outcome"); got != outcome {
				t.Fatalf("%s summary outcome = %q", outcome, got)
			}
		})
	}
}

func TestTargetCleanupOTelMetricsExposeOnlyClosedAttributesAndBacklogGauges(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown target cleanup meter provider: %v", err)
		}
	})
	metrics := newTargetCleanupOTelMetrics(provider.Meter("target-cleanup-observability-test"))
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	metrics.RecordRun(targetcleanupapp.TargetCleanupRunEvent{
		StartedAt: now, FinishedAt: now.Add(2 * time.Second), Outcome: "failed", FailureClass: "database_lock",
		Result: targetcleanupapp.TargetCleanupReconciliationResult{Counts: cleanupdomain.CleanupCounts{
			AssetRows:              map[cleanupdomain.AssetResource]int64{cleanupdomain.AssetResourceSubdomain: 3},
			Schedules:              1,
			OrganizationRelations:  1,
			TargetPolicies:         1,
			Scans:                  1,
			Tasks:                  2,
			AgentNotifications:     2,
			AgentNotificationFails: 1,
		}},
	})
	metrics.RecordBacklog(cleanupdomain.CleanupBacklog{UnfinishedCount: 4, OldestCreatedAt: timePointer(now.Add(-90 * time.Second))}, now)

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &collected); err != nil {
		t.Fatalf("collect target cleanup metrics: %v", err)
	}
	if len(collected.ScopeMetrics) == 0 {
		t.Fatal("target cleanup metrics were not collected")
	}
	gauges := map[string]int64{}
	for _, scope := range collected.ScopeMetrics {
		for _, observed := range scope.Metrics {
			switch data := observed.Data.(type) {
			case metricdata.Sum[int64]:
				for _, point := range data.DataPoints {
					assertTargetCleanupMetricAttributeSet(t, point.Attributes)
				}
			case metricdata.Histogram[float64]:
				for _, point := range data.DataPoints {
					assertTargetCleanupMetricAttributeSet(t, point.Attributes)
				}
			case metricdata.Gauge[int64]:
				for _, point := range data.DataPoints {
					assertTargetCleanupMetricAttributeSet(t, point.Attributes)
					gauges[observed.Name] = point.Value
				}
			default:
				t.Fatalf("unexpected target cleanup metric aggregation %T for %s", observed.Data, observed.Name)
			}
		}
	}
	if gauges["target_cleanup_unfinished_jobs"] != 4 || gauges["target_cleanup_oldest_unfinished_age_seconds"] != 90 {
		t.Fatalf("backlog gauges = %#v, want unfinished=4 oldest_age=90", gauges)
	}
}

func assertTargetCleanupMetricAttributeSet(t *testing.T, attributes attribute.Set) {
	t.Helper()
	allowed := map[string]struct{}{"outcome": {}, "resource": {}, "error_class": {}, "status": {}}
	allowedValues := map[string]map[string]struct{}{
		"outcome": {
			"completed": {}, "deferred": {}, "failed": {}, "delivered": {},
		},
		"resource": {
			"subdomain": {}, "host_port_mapping": {}, "website": {}, "endpoint": {}, "directory": {}, "screenshot": {}, "vulnerability": {},
			"scheduled_scan": {}, "organization_target": {}, "blacklist_policy": {},
		},
		"error_class": {
			"timeout": {}, "canceled": {}, "database_lock": {}, "database_timeout": {}, "database_or_reconciliation": {}, "diagnostic_persist_failed": {},
		},
		"status": {"pending": {}},
	}
	for _, value := range attributes.ToSlice() {
		key := string(value.Key)
		if _, ok := allowed[key]; !ok {
			t.Fatalf("metric attribute %q is not in the closed target cleanup vocabulary", value.Key)
		}
		if _, ok := allowedValues[key][value.Value.AsString()]; !ok {
			t.Fatalf("metric attribute %q=%q is not a bounded target cleanup value", value.Key, value.Value.AsString())
		}
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func fieldValue(fields []zap.Field, key string) string {
	for _, field := range fields {
		if field.Key == key {
			return field.String
		}
	}
	return ""
}
