package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestGetIPAggregationWithScriptedDB(t *testing.T) {
	createdAtOne := time.Date(2026, 3, 18, 13, 0, 0, 0, time.FixedZone("CET", 3600))
	createdAtTwo := time.Date(2026, 3, 18, 9, 30, 0, 0, time.FixedZone("EST", -5*3600))

	repo := NewHostPortRepository(newHostPortQueryScriptedDB(t,
		hostPortQueryResult{
			expect:  "COUNT(*)",
			columns: []string{"count"},
			rows:    [][]driver.Value{{int64(2)}},
		},
		hostPortQueryResult{
			expect:  "MIN(created_at)",
			columns: []string{"ip", "created_at"},
			rows: [][]driver.Value{
				{"2.2.2.2", createdAtOne},
				{"1.1.1.1", createdAtTwo},
			},
		},
	))

	rows, total, err := repo.GetIPAggregation(42, 1, 10, "", "createdAt desc")
	if err != nil {
		t.Fatalf("GetIPAggregation returned error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total=2, got %d", total)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].IP != "2.2.2.2" || rows[1].IP != "1.1.1.1" {
		t.Fatalf("unexpected rows order: %+v", rows)
	}
	if rows[0].CreatedAt != createdAtOne.UTC() || rows[1].CreatedAt != createdAtTwo.UTC() {
		t.Fatalf("expected UTC-normalized timestamps, got %+v", rows)
	}
	if rows[0].CreatedAt.Location() != time.UTC || rows[1].CreatedAt.Location() != time.UTC {
		t.Fatalf("expected UTC locations, got %+v", rows)
	}
}

func TestGetIPAggregationReturnsCountError(t *testing.T) {
	expectedErr := errors.New("count failed")
	repo := NewHostPortRepository(newHostPortQueryScriptedDB(t, hostPortQueryResult{
		expect: "COUNT(*)",
		err:    expectedErr,
	}))

	rows, total, err := repo.GetIPAggregation(42, 1, 10, "", "createdAt desc")
	if !errors.Is(err, expectedErr) || rows != nil || total != 0 {
		t.Fatalf("unexpected result rows=%v total=%d err=%v", rows, total, err)
	}
}

func TestGetHostsAndPortsByIPReturnsQueryError(t *testing.T) {
	expectedErr := errors.New("scan failed")
	repo := NewHostPortRepository(newHostPortQueryScriptedDB(t, hostPortQueryResult{
		expect: "distinct host, port",
		err:    expectedErr,
	}))

	hosts, ports, err := repo.GetHostsAndPortsByIP(42, "1.1.1.1", "")
	if !errors.Is(err, expectedErr) || hosts != nil || ports != nil {
		t.Fatalf("unexpected result hosts=%v ports=%v err=%v", hosts, ports, err)
	}
}

func TestHostPortExactHostFilterConditionPreservesContainsSearch(t *testing.T) {
	exactCondition, exactArgs := hostPortFilterCondition(scope.ParsedFilter{Field: "host", Operator: "==", Value: " API.Acme.COM "})
	if exactCondition != "LOWER(host) = ?" || len(exactArgs) != 1 || exactArgs[0] != "api.acme.com" {
		t.Fatalf("unexpected exact host condition=%q args=%v", exactCondition, exactArgs)
	}

	containsCondition, containsArgs := hostPortFilterCondition(scope.ParsedFilter{Field: "host", Operator: "=", Value: "api"})
	if containsCondition != "host ILIKE ?" || len(containsArgs) != 1 || containsArgs[0] != "%api%" {
		t.Fatalf("unexpected contains host condition=%q args=%v", containsCondition, containsArgs)
	}
}

type hostPortQueryResult struct {
	expect  string
	columns []string
	rows    [][]driver.Value
	err     error
}

type hostPortQueryScript struct {
	mu      sync.Mutex
	queries []hostPortQueryResult
	index   int
}

var (
	hostPortQueryDriverOnce sync.Once
	hostPortQueryScriptsMu  sync.Mutex
	hostPortQueryScripts    = make(map[string]*hostPortQueryScript)
)

func newHostPortQueryScriptedDB(t *testing.T, queries ...hostPortQueryResult) *gorm.DB {
	t.Helper()

	hostPortQueryDriverOnce.Do(func() {
		sql.Register("asset_host_port_query_scripted", &hostPortQueryDriver{})
	})

	dsn := fmt.Sprintf("host-port-query-%d", time.Now().UnixNano())
	hostPortQueryScriptsMu.Lock()
	hostPortQueryScripts[dsn] = &hostPortQueryScript{queries: queries}
	hostPortQueryScriptsMu.Unlock()

	sqlDB, err := sql.Open("asset_host_port_query_scripted", dsn)
	if err != nil {
		t.Fatalf("open scripted sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
		hostPortQueryScriptsMu.Lock()
		delete(hostPortQueryScripts, dsn)
		hostPortQueryScriptsMu.Unlock()
	})

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm db: %v", err)
	}
	return db
}

type hostPortQueryDriver struct{}

func (d *hostPortQueryDriver) Open(name string) (driver.Conn, error) {
	hostPortQueryScriptsMu.Lock()
	script := hostPortQueryScripts[name]
	hostPortQueryScriptsMu.Unlock()
	if script == nil {
		return nil, fmt.Errorf("missing scripted database for dsn %s", name)
	}
	return &hostPortQueryConn{script: script}, nil
}

type hostPortQueryConn struct {
	script *hostPortQueryScript
}

func (c *hostPortQueryConn) Prepare(query string) (driver.Stmt, error) {
	return &hostPortQueryStmt{script: c.script, query: query}, nil
}

func (c *hostPortQueryConn) Close() error {
	return nil
}

func (c *hostPortQueryConn) Begin() (driver.Tx, error) {
	return nil, fmt.Errorf("transactions are not supported in scripted driver")
}

func (c *hostPortQueryConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	return c.script.query(query)
}

func (c *hostPortQueryConn) CheckNamedValue(*driver.NamedValue) error {
	return nil
}

type hostPortQueryStmt struct {
	script *hostPortQueryScript
	query  string
}

func (s *hostPortQueryStmt) Close() error {
	return nil
}

func (s *hostPortQueryStmt) NumInput() int {
	return -1
}

func (s *hostPortQueryStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, fmt.Errorf("exec is not supported in scripted driver")
}

func (s *hostPortQueryStmt) Query([]driver.Value) (driver.Rows, error) {
	return s.script.query(s.query)
}

func (s *hostPortQueryStmt) QueryContext(_ context.Context, _ []driver.NamedValue) (driver.Rows, error) {
	return s.script.query(s.query)
}

type hostPortQueryRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (r *hostPortQueryRows) Columns() []string {
	return r.columns
}

func (r *hostPortQueryRows) Close() error {
	return nil
}

func (r *hostPortQueryRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

func (s *hostPortQueryScript) query(query string) (driver.Rows, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.index >= len(s.queries) {
		return nil, fmt.Errorf("unexpected query #%d: %s", s.index+1, compactHostPortSQL(query))
	}

	result := s.queries[s.index]
	s.index++
	if result.expect != "" && !strings.Contains(compactHostPortSQL(query), compactHostPortSQL(result.expect)) {
		return nil, fmt.Errorf("query #%d mismatch: got %q want substring %q", s.index, compactHostPortSQL(query), compactHostPortSQL(result.expect))
	}
	if result.err != nil {
		return nil, result.err
	}

	return &hostPortQueryRows{
		columns: result.columns,
		rows:    result.rows,
	}, nil
}

func compactHostPortSQL(sqlText string) string {
	return strings.ToLower(strings.Join(strings.Fields(sqlText), " "))
}
