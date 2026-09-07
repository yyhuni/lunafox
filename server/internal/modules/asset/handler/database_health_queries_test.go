package handler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestDatabaseHealthQueryHelpersWithScriptedDB(t *testing.T) {
	t.Run("queryReadOnly parses on and off", func(t *testing.T) {
		handler := &HealthHandler{db: newDatabaseHealthScriptedDB(t,
			databaseHealthQueryResult{
				expect:  "SHOW transaction_read_only",
				columns: []string{"transaction_read_only"},
				rows:    [][]driver.Value{{" ON "}},
			},
		)}

		readOnly, err := handler.queryReadOnly(context.Background())
		if err != nil {
			t.Fatalf("queryReadOnly returned error: %v", err)
		}
		if !readOnly {
			t.Fatal("expected readOnly=true for ON")
		}

		handler = &HealthHandler{db: newDatabaseHealthScriptedDB(t,
			databaseHealthQueryResult{
				expect:  "SHOW transaction_read_only",
				columns: []string{"transaction_read_only"},
				rows:    [][]driver.Value{{"false"}},
			},
		)}

		readOnly, err = handler.queryReadOnly(context.Background())
		if err != nil {
			t.Fatalf("queryReadOnly returned error: %v", err)
		}
		if readOnly {
			t.Fatal("expected readOnly=false for false")
		}
	})

	t.Run("queryUptimeSeconds returns value and zero for null", func(t *testing.T) {
		handler := &HealthHandler{db: newDatabaseHealthScriptedDB(t,
			databaseHealthQueryResult{
				expect:  "pg_postmaster_start_time",
				columns: []string{"uptime"},
				rows:    [][]driver.Value{{int64(7200)}},
			},
		)}
		uptime, err := handler.queryUptimeSeconds(context.Background())
		if err != nil {
			t.Fatalf("queryUptimeSeconds returned error: %v", err)
		}
		if uptime != 7200 {
			t.Fatalf("expected uptime=7200, got %d", uptime)
		}

		handler = &HealthHandler{db: newDatabaseHealthScriptedDB(t,
			databaseHealthQueryResult{
				expect:  "pg_postmaster_start_time",
				columns: []string{"uptime"},
				rows:    [][]driver.Value{{nil}},
			},
		)}
		uptime, err = handler.queryUptimeSeconds(context.Background())
		if err != nil {
			t.Fatalf("queryUptimeSeconds returned error: %v", err)
		}
		if uptime != 0 {
			t.Fatalf("expected uptime=0 for null, got %d", uptime)
		}
	})

	t.Run("queryMaxConnections returns value and zero for null", func(t *testing.T) {
		handler := &HealthHandler{db: newDatabaseHealthScriptedDB(t,
			databaseHealthQueryResult{
				expect:  "SHOW max_connections",
				columns: []string{"max_connections"},
				rows:    [][]driver.Value{{int64(150)}},
			},
		)}
		maxConn, err := handler.queryMaxConnections(context.Background())
		if err != nil {
			t.Fatalf("queryMaxConnections returned error: %v", err)
		}
		if maxConn != 150 {
			t.Fatalf("expected maxConnections=150, got %d", maxConn)
		}

		handler = &HealthHandler{db: newDatabaseHealthScriptedDB(t,
			databaseHealthQueryResult{
				expect:  "SHOW max_connections",
				columns: []string{"max_connections"},
				rows:    [][]driver.Value{{nil}},
			},
		)}
		maxConn, err = handler.queryMaxConnections(context.Background())
		if err != nil {
			t.Fatalf("queryMaxConnections returned error: %v", err)
		}
		if maxConn != 0 {
			t.Fatalf("expected maxConnections=0 for null, got %d", maxConn)
		}
	})

	t.Run("queryDatabaseSizeBytes returns value or unavailable", func(t *testing.T) {
		handler := &HealthHandler{db: newDatabaseHealthScriptedDB(t,
			databaseHealthQueryResult{
				expect:  "pg_database_size",
				columns: []string{"database_size_bytes"},
				rows:    [][]driver.Value{{int64(123456789)}},
			},
		)}
		sizeBytes, ok, err := handler.queryDatabaseSizeBytes(context.Background())
		if err != nil {
			t.Fatalf("queryDatabaseSizeBytes returned error: %v", err)
		}
		if !ok || sizeBytes != 123456789 {
			t.Fatalf("expected database size (123456789, true), got (%d, %v)", sizeBytes, ok)
		}

		handler = &HealthHandler{db: newDatabaseHealthScriptedDB(t,
			databaseHealthQueryResult{
				expect:  "pg_database_size",
				columns: []string{"database_size_bytes"},
				rows:    [][]driver.Value{{nil}},
			},
		)}
		sizeBytes, ok, err = handler.queryDatabaseSizeBytes(context.Background())
		if err != nil {
			t.Fatalf("queryDatabaseSizeBytes returned error: %v", err)
		}
		if ok || sizeBytes != 0 {
			t.Fatalf("expected unavailable database size, got (%d, %v)", sizeBytes, ok)
		}
	})

	t.Run("optional float queries return value or unavailable", func(t *testing.T) {
		cases := []struct {
			name string
			run  func(*HealthHandler) (float64, bool, error)
			sql  string
			want float64
		}{
			{name: "queryQPS", run: func(h *HealthHandler) (float64, bool, error) { return h.queryQPS(context.Background()) }, sql: "FROM pg_stat_database", want: 12.5},
			{name: "queryWalGeneratedMb24h", run: func(h *HealthHandler) (float64, bool, error) { return h.queryWalGeneratedMb24h(context.Background()) }, sql: "FROM pg_stat_wal", want: 256.25},
			{name: "queryCacheHitRate", run: func(h *HealthHandler) (float64, bool, error) { return h.queryCacheHitRate(context.Background()) }, sql: "blks_hit", want: 99.9},
			{name: "queryDeadlocks1h", run: func(h *HealthHandler) (float64, bool, error) { return h.queryDeadlocks1h(context.Background()) }, sql: "deadlocks::double precision", want: 1.75},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				handler := &HealthHandler{db: newDatabaseHealthScriptedDB(t,
					databaseHealthQueryResult{
						expect:  tc.sql,
						columns: []string{"value"},
						rows:    [][]driver.Value{{tc.want}},
					},
				)}
				value, ok, err := tc.run(handler)
				if err != nil {
					t.Fatalf("%s returned error: %v", tc.name, err)
				}
				if !ok || value != tc.want {
					t.Fatalf("expected %s to return (%v, true), got (%v, %v)", tc.name, tc.want, value, ok)
				}

				handler = &HealthHandler{db: newDatabaseHealthScriptedDB(t,
					databaseHealthQueryResult{
						expect:  tc.sql,
						columns: []string{"value"},
						rows:    [][]driver.Value{{nil}},
					},
				)}
				value, ok, err = tc.run(handler)
				if err != nil {
					t.Fatalf("%s returned error: %v", tc.name, err)
				}
				if ok || value != 0 {
					t.Fatalf("expected %s to return unavailable zero, got (%v, %v)", tc.name, value, ok)
				}
			})
		}
	})

	t.Run("count queries return value and zero for null", func(t *testing.T) {
		cases := []struct {
			name string
			run  func(*HealthHandler) (int, error)
			sql  string
			want int
		}{
			{name: "queryLockWaitCount", run: func(h *HealthHandler) (int, error) { return h.queryLockWaitCount(context.Background()) }, sql: "wait_event_type = 'Lock'", want: 6},
			{name: "queryLongTransactionCount", run: func(h *HealthHandler) (int, error) { return h.queryLongTransactionCount(context.Background()) }, sql: "xact_start IS NOT NULL", want: 3},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				handler := &HealthHandler{db: newDatabaseHealthScriptedDB(t,
					databaseHealthQueryResult{
						expect:  tc.sql,
						columns: []string{"count"},
						rows:    [][]driver.Value{{int64(tc.want)}},
					},
				)}
				value, err := tc.run(handler)
				if err != nil {
					t.Fatalf("%s returned error: %v", tc.name, err)
				}
				if value != tc.want {
					t.Fatalf("expected %s=%d, got %d", tc.name, tc.want, value)
				}

				handler = &HealthHandler{db: newDatabaseHealthScriptedDB(t,
					databaseHealthQueryResult{
						expect:  tc.sql,
						columns: []string{"count"},
						rows:    [][]driver.Value{{nil}},
					},
				)}
				value, err = tc.run(handler)
				if err != nil {
					t.Fatalf("%s returned error: %v", tc.name, err)
				}
				if value != 0 {
					t.Fatalf("expected %s=0 for null, got %d", tc.name, value)
				}
			})
		}
	})

	t.Run("queryOldestPendingTaskAgeSec returns value and zero for null", func(t *testing.T) {
		handler := &HealthHandler{db: newDatabaseHealthScriptedDB(t,
			databaseHealthQueryResult{
				expect:  "FROM scan_task st JOIN scan s ON s.id = st.scan_id",
				columns: []string{"age"},
				rows:    [][]driver.Value{{int64(615)}},
			},
		)}
		ageSec, err := handler.queryOldestPendingTaskAgeSec(context.Background())
		if err != nil {
			t.Fatalf("queryOldestPendingTaskAgeSec returned error: %v", err)
		}
		if ageSec != 615 {
			t.Fatalf("expected ageSec=615, got %d", ageSec)
		}

		handler = &HealthHandler{db: newDatabaseHealthScriptedDB(t,
			databaseHealthQueryResult{
				expect:  "s.status IN ('pending', 'running') AND s.deleted_at IS NULL",
				columns: []string{"age"},
				rows:    [][]driver.Value{{nil}},
			},
		)}
		ageSec, err = handler.queryOldestPendingTaskAgeSec(context.Background())
		if err != nil {
			t.Fatalf("queryOldestPendingTaskAgeSec returned error: %v", err)
		}
		if ageSec != 0 {
			t.Fatalf("expected ageSec=0 for null, got %d", ageSec)
		}
	})
}

func TestDatabaseHealthOnlineSnapshotWithScriptedDB(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(newDatabaseHealthScriptedDB(t,
		databaseHealthQueryResult{
			expect:  "SHOW max_connections",
			columns: []string{"max_connections"},
			rows:    [][]driver.Value{{int64(100)}},
		},
		databaseHealthQueryResult{
			expect:  "pg_is_in_recovery",
			columns: []string{"role"},
			rows:    [][]driver.Value{{"primary"}},
		},
		databaseHealthQueryResult{
			expect:  "SHOW server_version",
			columns: []string{"server_version"},
			rows:    [][]driver.Value{{"16.2"}},
		},
		databaseHealthQueryResult{
			expect:  "SHOW transaction_read_only",
			columns: []string{"transaction_read_only"},
			rows:    [][]driver.Value{{"off"}},
		},
		databaseHealthQueryResult{
			expect:  "pg_postmaster_start_time",
			columns: []string{"uptime"},
			rows:    [][]driver.Value{{int64(86400)}},
		},
		databaseHealthQueryResult{
			expect:  "pg_database_size",
			columns: []string{"database_size_bytes"},
			rows:    [][]driver.Value{{int64(987654321)}},
		},
		databaseHealthQueryResult{
			expect:  "wait_event_type = 'Lock'",
			columns: []string{"count"},
			rows:    [][]driver.Value{{int64(0)}},
		},
		databaseHealthQueryResult{
			expect:  "deadlocks::double precision",
			columns: []string{"deadlocks"},
			rows:    [][]driver.Value{{float64(0)}},
		},
		databaseHealthQueryResult{
			expect:  "xact_start IS NOT NULL",
			columns: []string{"count"},
			rows:    [][]driver.Value{{int64(1)}},
		},
		databaseHealthQueryResult{
			expect:  "FROM scan_task",
			columns: []string{"age"},
			rows:    [][]driver.Value{{int64(30)}},
		},
		databaseHealthQueryResult{
			expect:  "FROM pg_stat_database",
			columns: []string{"qps"},
			rows:    [][]driver.Value{{float64(12.5)}},
		},
		databaseHealthQueryResult{
			expect:  "FROM pg_stat_wal",
			columns: []string{"wal"},
			rows:    [][]driver.Value{{float64(256.25)}},
		},
		databaseHealthQueryResult{
			expect:  "blks_hit",
			columns: []string{"cache_hit_rate"},
			rows:    [][]driver.Value{{float64(98.7)}},
		},
	), nil)

	recorder := performRootHandlerRequest(t, handler.DatabaseHealth)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status code 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var snapshot databaseHealthSnapshotResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if snapshot.Status != dbHealthStatusOnline {
		t.Fatalf("expected status online, got %s", snapshot.Status)
	}
	if strings.Contains(recorder.Body.String(), `"region"`) {
		t.Fatalf("database health response must not expose deployment region: %s", recorder.Body.String())
	}
	if snapshot.Role != "primary" || snapshot.Version != "16.2" || snapshot.ReadOnly {
		t.Fatalf("unexpected role/version/readOnly snapshot: %+v", snapshot)
	}
	if snapshot.UptimeSeconds != 86400 {
		t.Fatalf("expected uptimeSeconds=86400, got %d", snapshot.UptimeSeconds)
	}
	if snapshot.DatabaseSizeBytes == nil || *snapshot.DatabaseSizeBytes != 987654321 {
		t.Fatalf("expected databaseSizeBytes=987654321, got %v", snapshot.DatabaseSizeBytes)
	}
	if snapshot.CoreSignals.ConnectionsMax != 100 {
		t.Fatalf("expected connectionsMax=100, got %d", snapshot.CoreSignals.ConnectionsMax)
	}
	if snapshot.CoreSignals.LockWaitCount != 0 || snapshot.CoreSignals.Deadlocks1h != 0 || snapshot.CoreSignals.LongTransactionCount != 1 || snapshot.CoreSignals.OldestPendingTaskAgeSec != 30 {
		t.Fatalf("unexpected core signals: %+v", snapshot.CoreSignals)
	}
	if snapshot.OptionalSignals.QPS == nil || *snapshot.OptionalSignals.QPS != 12.5 {
		t.Fatalf("expected qps=12.5, got %v", snapshot.OptionalSignals.QPS)
	}
	if snapshot.OptionalSignals.WalGeneratedMb24h == nil || *snapshot.OptionalSignals.WalGeneratedMb24h != 256.25 {
		t.Fatalf("expected walGeneratedMb24h=256.25, got %v", snapshot.OptionalSignals.WalGeneratedMb24h)
	}
	if snapshot.OptionalSignals.CacheHitRate == nil || *snapshot.OptionalSignals.CacheHitRate != 98.7 {
		t.Fatalf("expected cacheHitRate=98.7, got %v", snapshot.OptionalSignals.CacheHitRate)
	}
	if snapshot.Findings == nil {
		t.Fatalf("expected findings array to be present")
	}
	if len(snapshot.Findings) != 0 {
		t.Fatalf("expected healthy scripted snapshot to have no findings, got %+v", snapshot.Findings)
	}
	if len(snapshot.UnavailableSignals) != 0 || len(snapshot.Alerts) != 0 {
		t.Fatalf("expected no unavailable signals or alerts, got unavailable=%+v alerts=%+v", snapshot.UnavailableSignals, snapshot.Alerts)
	}
}

func TestDatabaseHealthHandlesDBConnectorError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(&gorm.DB{Config: &gorm.Config{ConnPool: failingDBConnector{err: fmt.Errorf("sql handle failed")}}}, nil)
	recorder := performRootHandlerRequest(t, handler.DatabaseHealth)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "\"status\":\"offline\"") || !strings.Contains(body, "Cannot acquire SQL handle") {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestDatabaseHealthHandlesProbeFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newHandlerSQLiteDB(t)
	db.Error = fmt.Errorf("probe failed")
	handler := NewHealthHandler(db, nil)

	recorder := performRootHandlerRequest(t, handler.DatabaseHealth)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "\"status\":\"offline\"") || !strings.Contains(body, "Probe failed") {
		t.Fatalf("unexpected response body: %s", body)
	}
}

type databaseHealthQueryResult struct {
	expect  string
	columns []string
	rows    [][]driver.Value
	err     error
}

type databaseHealthScript struct {
	mu      sync.Mutex
	queries []databaseHealthQueryResult
	index   int
}

var (
	databaseHealthDriverOnce sync.Once
	databaseHealthScriptsMu  sync.Mutex
	databaseHealthScripts    = make(map[string]*databaseHealthScript)
)

func newDatabaseHealthScriptedDB(t *testing.T, queries ...databaseHealthQueryResult) *gorm.DB {
	t.Helper()

	databaseHealthDriverOnce.Do(func() {
		sql.Register("asset_database_health_scripted", &databaseHealthDriver{})
	})

	dsn := fmt.Sprintf("database-health-%d", time.Now().UnixNano())
	databaseHealthScriptsMu.Lock()
	databaseHealthScripts[dsn] = &databaseHealthScript{queries: queries}
	databaseHealthScriptsMu.Unlock()

	sqlDB, err := sql.Open("asset_database_health_scripted", dsn)
	if err != nil {
		t.Fatalf("open scripted sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
		databaseHealthScriptsMu.Lock()
		delete(databaseHealthScripts, dsn)
		databaseHealthScriptsMu.Unlock()
	})

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm db: %v", err)
	}
	return db
}

type databaseHealthDriver struct{}

func (d *databaseHealthDriver) Open(name string) (driver.Conn, error) {
	databaseHealthScriptsMu.Lock()
	script := databaseHealthScripts[name]
	databaseHealthScriptsMu.Unlock()
	if script == nil {
		return nil, fmt.Errorf("missing scripted database for dsn %s", name)
	}
	return &databaseHealthConn{script: script}, nil
}

type databaseHealthConn struct {
	script *databaseHealthScript
}

func (c *databaseHealthConn) Prepare(query string) (driver.Stmt, error) {
	return &databaseHealthStmt{script: c.script, query: query}, nil
}

func (c *databaseHealthConn) Close() error {
	return nil
}

func (c *databaseHealthConn) Begin() (driver.Tx, error) {
	return nil, fmt.Errorf("transactions are not supported in scripted driver")
}

func (c *databaseHealthConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	return c.script.query(query)
}

func (c *databaseHealthConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	return c.script.exec(query)
}

func (c *databaseHealthConn) CheckNamedValue(*driver.NamedValue) error {
	return nil
}

type databaseHealthStmt struct {
	script *databaseHealthScript
	query  string
}

func (s *databaseHealthStmt) Close() error {
	return nil
}

func (s *databaseHealthStmt) NumInput() int {
	return -1
}

func (s *databaseHealthStmt) Exec([]driver.Value) (driver.Result, error) {
	return s.script.exec(s.query)
}

func (s *databaseHealthStmt) Query([]driver.Value) (driver.Rows, error) {
	return s.script.query(s.query)
}

func (s *databaseHealthStmt) ExecContext(_ context.Context, _ []driver.NamedValue) (driver.Result, error) {
	return s.script.exec(s.query)
}

func (s *databaseHealthStmt) QueryContext(_ context.Context, _ []driver.NamedValue) (driver.Rows, error) {
	return s.script.query(s.query)
}

type databaseHealthRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (r *databaseHealthRows) Columns() []string {
	return r.columns
}

func (r *databaseHealthRows) Close() error {
	return nil
}

func (r *databaseHealthRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

func (s *databaseHealthScript) query(query string) (driver.Rows, error) {
	result, err := s.next(query)
	if err != nil {
		return nil, err
	}
	if result.err != nil {
		return nil, result.err
	}

	return &databaseHealthRows{
		columns: result.columns,
		rows:    result.rows,
	}, nil
}

func (s *databaseHealthScript) exec(query string) (driver.Result, error) {
	result, err := s.next(query)
	if err != nil {
		return nil, err
	}
	if result.err != nil {
		return nil, result.err
	}
	return driver.RowsAffected(1), nil
}

func (s *databaseHealthScript) next(query string) (databaseHealthQueryResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.index >= len(s.queries) {
		return databaseHealthQueryResult{}, fmt.Errorf("unexpected query #%d: %s", s.index+1, compactDatabaseHealthSQL(query))
	}

	result := s.queries[s.index]
	s.index++
	if result.expect != "" && !strings.Contains(compactDatabaseHealthSQL(query), compactDatabaseHealthSQL(result.expect)) {
		return databaseHealthQueryResult{}, fmt.Errorf("query #%d mismatch: got %q want substring %q", s.index, compactDatabaseHealthSQL(query), compactDatabaseHealthSQL(result.expect))
	}
	return result, nil
}

func compactDatabaseHealthSQL(sqlText string) string {
	return strings.ToLower(strings.Join(strings.Fields(sqlText), " "))
}
