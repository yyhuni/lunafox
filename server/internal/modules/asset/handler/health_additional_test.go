package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHealthHandlerCheckLivenessAndReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(nil, nil)
	if handler == nil || handler.db != nil || handler.redis != nil {
		t.Fatal("expected health handler without dependencies")
	}

	check := performRootHandlerRequest(t, handler.Check)
	if check.Code != http.StatusOK || !strings.Contains(check.Body.String(), "\"database\":\"not_configured\"") {
		t.Fatalf("unexpected check response: code=%d body=%s", check.Code, check.Body.String())
	}

	live := performRootHandlerRequest(t, handler.Liveness)
	if live.Code != http.StatusOK || !strings.Contains(live.Body.String(), "\"status\":\"alive\"") {
		t.Fatalf("unexpected liveness response: code=%d body=%s", live.Code, live.Body.String())
	}

	ready := performRootHandlerRequest(t, handler.Readiness)
	if ready.Code != http.StatusOK || !strings.Contains(ready.Body.String(), "\"status\":\"ready\"") {
		t.Fatalf("unexpected readiness response: code=%d body=%s", ready.Code, ready.Body.String())
	}
}

func TestScheduledScanFailuresCannotChangeExistingHealthContracts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandler(nil, nil)
	responses := map[string]*httptest.ResponseRecorder{
		"current":   performRootHandlerRequest(t, handler.Check),
		"liveness":  performRootHandlerRequest(t, handler.Liveness),
		"readiness": performRootHandlerRequest(t, handler.Readiness),
	}
	for name, response := range responses {
		if response.Code != http.StatusOK {
			t.Fatalf("%s health status = %d body=%s", name, response.Code, response.Body.String())
		}
		body := strings.ToLower(response.Body.String())
		for _, forbidden := range []string{"scheduler", "scheduledscan", "scheduled_scan", "occurrence", "retention", "backlog"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("%s health response contains scheduler state %q: %s", name, forbidden, response.Body.String())
			}
		}
	}
}

func TestHealthHandlerUnavailableDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newHandlerSQLiteDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	handler := NewHealthHandler(db, nil)
	check := performRootHandlerRequest(t, handler.Check)
	if check.Code != http.StatusServiceUnavailable || !strings.Contains(check.Body.String(), "\"status\":\"unhealthy\"") {
		t.Fatalf("unexpected check response: code=%d body=%s", check.Code, check.Body.String())
	}

	ready := performRootHandlerRequest(t, handler.Readiness)
	if ready.Code != http.StatusServiceUnavailable || !strings.Contains(ready.Body.String(), "\"reason\":\"database_unavailable\"") {
		t.Fatalf("unexpected readiness response: code=%d body=%s", ready.Code, ready.Body.String())
	}
}

func TestHealthHandlerConfiguredDatabaseAndRedisFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("configured database healthy", func(t *testing.T) {
		db := newHandlerSQLiteDB(t)
		handler := NewHealthHandler(db, nil)
		recorder := performRootHandlerRequest(t, handler.Check)
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "\"database\":\"connected\"") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("redis disconnected", func(t *testing.T) {
		client := redis.NewClient(&redis.Options{
			Addr:         "127.0.0.1:1",
			DialTimeout:  10 * time.Millisecond,
			ReadTimeout:  10 * time.Millisecond,
			WriteTimeout: 10 * time.Millisecond,
		})
		t.Cleanup(func() {
			_ = client.Close()
		})

		handler := NewHealthHandler(nil, client)
		recorder := performRootHandlerRequest(t, handler.Check)
		if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "\"redis\":\"disconnected\"") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})
}

func TestDatabaseHealthCoverage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("nil db", func(t *testing.T) {
		handler := NewHealthHandler(nil, nil)
		recorder := performRootHandlerRequest(t, handler.DatabaseHealth)
		if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "database_unavailable") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("sqlite degraded path", func(t *testing.T) {
		db := newHandlerSQLiteDB(t)
		handler := NewHealthHandler(db, nil)

		recorder := performRootHandlerRequest(t, handler.DatabaseHealth)
		body := recorder.Body.String()
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		if !strings.Contains(body, "\"status\":\"degraded\"") || !strings.Contains(body, "unavailableSignals") || strings.Contains(body, `"region"`) {
			t.Fatalf("unexpected body: %s", body)
		}
	})

	t.Run("maintenance mode", func(t *testing.T) {
		t.Setenv("DB_MAINTENANCE_MODE", "true")
		db := newHandlerSQLiteDB(t)
		handler := NewHealthHandler(db, nil)

		recorder := performRootHandlerRequest(t, handler.DatabaseHealth)
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "\"status\":\"maintenance\"") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("closed db degraded path", func(t *testing.T) {
		db := newHandlerSQLiteDB(t)
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatalf("get sql db: %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close sql db: %v", err)
		}

		handler := NewHealthHandler(db, nil)
		recorder := performRootHandlerRequest(t, handler.DatabaseHealth)
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "\"status\":\"degraded\"") || !strings.Contains(recorder.Body.String(), "sql: database is closed") {
			t.Fatalf("unexpected response: code=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})
}

func TestDatabaseHealthHelpers(t *testing.T) {
	handler := &HealthHandler{}

	snapshot := baseSnapshotForStatusTest()
	handler.addUnavailable(&snapshot, "qps", signalScopeOptional, reasonQueryFailed, "query failed")
	if handler.countUnavailable(snapshot.UnavailableSignals, signalScopeOptional) != 1 {
		t.Fatalf("unexpected unavailable count: %+v", snapshot.UnavailableSignals)
	}

	if classifyReason(errors.New("permission denied")) != reasonPermissionDenied {
		t.Fatal("expected permission denied classification")
	}
	if classifyReason(errors.New("context deadline timeout")) != reasonTimeout {
		t.Fatal("expected timeout classification")
	}
	if classifyReason(errors.New("table does not exist")) != reasonUnsupported {
		t.Fatal("expected unsupported classification")
	}
	if classifyReason(errors.New("query execution failed")) != reasonQueryFailed {
		t.Fatal("expected query failed classification")
	}
	if classifyReason(nil) != reasonUnknown {
		t.Fatal("expected nil error to classify as unknown")
	}

	if optionalString("  ") != nil {
		t.Fatal("expected blank optional string to be nil")
	}
	if value := optionalString(" value "); value == nil || *value != "value" {
		t.Fatalf("unexpected optional string value: %v", value)
	}
	if value := floatPtr(1.5); value == nil || *value != 1.5 {
		t.Fatalf("unexpected float pointer: %v", value)
	}
	if clampFloat(-1, 0, 10) != 0 || clampFloat(99, 0, 10) != 10 || clampFloat(5, 0, 10) != 5 {
		t.Fatal("unexpected clampFloat result")
	}
	if maxFloat64(1, 2) != 2 || maxInt64(1, 2) != 2 || maxInt(1, 2) != 2 {
		t.Fatal("unexpected max helper result")
	}
	if maxFloat64(5, 2) != 5 || maxInt64(5, 2) != 5 || maxInt(5, 2) != 5 {
		t.Fatal("unexpected max helper reverse-order result")
	}

	handler.finalizeSnapshotStatus(nil)
}

func TestFinalizeSnapshotStatusAdditionalAlerts(t *testing.T) {
	handler := &HealthHandler{}
	snapshot := baseSnapshotForStatusTest()
	snapshot.CoreSignals.ConnectionUsagePercent = 90
	snapshot.CoreSignals.ProbeLatencyMs = 600
	snapshot.ReadOnly = true

	handler.finalizeSnapshotStatus(&snapshot)

	if snapshot.Status != dbHealthStatusDegraded {
		t.Fatalf("expected degraded status, got %s", snapshot.Status)
	}
	if !containsAlertTitle(snapshot.Alerts, "Established connection count is high") {
		t.Fatalf("expected established connection count alert, got %+v", snapshot.Alerts)
	}
	if !containsAlertTitle(snapshot.Alerts, "Probe latency high") {
		t.Fatalf("expected probe latency alert, got %+v", snapshot.Alerts)
	}
	if !containsAlertTitle(snapshot.Alerts, "Primary is read-only") {
		t.Fatalf("expected read-only alert, got %+v", snapshot.Alerts)
	}
}

type failingDBConnector struct {
	err error
}

func (c failingDBConnector) GetDBConn() (*sql.DB, error) {
	return nil, c.err
}

func (c failingDBConnector) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, c.err
}

func (c failingDBConnector) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	return nil, c.err
}

func (c failingDBConnector) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, c.err
}

func (c failingDBConnector) QueryRowContext(context.Context, string, ...interface{}) *sql.Row {
	return &sql.Row{}
}

func TestHealthHandlerCheckDatabaseHandleError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(&gorm.DB{Config: &gorm.Config{ConnPool: failingDBConnector{err: errors.New("db handle unavailable")}}}, nil)
	recorder := performRootHandlerRequest(t, handler.Check)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "\"database\":\"error\"") || !strings.Contains(body, "db handle unavailable") {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func performRootHandlerRequest(t *testing.T, handlerFunc gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/healthChecks/current", nil)
	handlerFunc(ctx)
	ctx.Writer.WriteHeaderNow()
	return recorder
}

func newHandlerSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:handler-root-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	return db
}
