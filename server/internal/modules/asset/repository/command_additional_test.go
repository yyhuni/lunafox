package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type repositoryWriteResult struct {
	expect  string
	columns []string
	rows    [][]driver.Value
	err     error
	inspect func(string) error
}

type repositoryWriteScript struct {
	mu      sync.Mutex
	results []repositoryWriteResult
	index   int
}

var (
	repositoryWriteDriverOnce sync.Once
	repositoryWriteScriptsMu  sync.Mutex
	repositoryWriteScripts    = make(map[string]*repositoryWriteScript)
)

func newRepositoryWriteScriptedDB(t *testing.T, results ...repositoryWriteResult) *gorm.DB {
	t.Helper()

	repositoryWriteDriverOnce.Do(func() {
		sql.Register("asset_repository_write_scripted", &repositoryWriteDriver{})
	})

	dsn := fmt.Sprintf("repository-write-%s", strings.ReplaceAll(t.Name(), "/", "_"))
	repositoryWriteScriptsMu.Lock()
	repositoryWriteScripts[dsn] = &repositoryWriteScript{results: results}
	repositoryWriteScriptsMu.Unlock()

	sqlDB, err := sql.Open("asset_repository_write_scripted", dsn)
	if err != nil {
		t.Fatalf("open scripted sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
		repositoryWriteScriptsMu.Lock()
		delete(repositoryWriteScripts, dsn)
		repositoryWriteScriptsMu.Unlock()
	})

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm db: %v", err)
	}
	return db
}

type repositoryWriteDriver struct{}

func (d *repositoryWriteDriver) Open(name string) (driver.Conn, error) {
	repositoryWriteScriptsMu.Lock()
	script := repositoryWriteScripts[name]
	repositoryWriteScriptsMu.Unlock()
	if script == nil {
		return nil, fmt.Errorf("missing scripted database for dsn %s", name)
	}
	return &repositoryWriteConn{script: script}, nil
}

type repositoryWriteConn struct {
	script *repositoryWriteScript
}

func (c *repositoryWriteConn) Prepare(query string) (driver.Stmt, error) {
	return &repositoryWriteStmt{script: c.script, query: query}, nil
}

func (c *repositoryWriteConn) Close() error {
	return nil
}

func (c *repositoryWriteConn) Begin() (driver.Tx, error) {
	return &repositoryWriteTx{}, nil
}

func (c *repositoryWriteConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return &repositoryWriteTx{}, nil
}

func (c *repositoryWriteConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	return c.script.query(query)
}

func (c *repositoryWriteConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	result, err := c.script.next(query)
	if err != nil {
		return nil, err
	}
	if result.err != nil {
		return nil, result.err
	}
	return driver.RowsAffected(1), nil
}

func (c *repositoryWriteConn) CheckNamedValue(*driver.NamedValue) error {
	return nil
}

type repositoryWriteStmt struct {
	script *repositoryWriteScript
	query  string
}

func (s *repositoryWriteStmt) Close() error {
	return nil
}

func (s *repositoryWriteStmt) NumInput() int {
	return -1
}

func (s *repositoryWriteStmt) Exec([]driver.Value) (driver.Result, error) {
	result, err := s.script.next(s.query)
	if err != nil {
		return nil, err
	}
	if result.err != nil {
		return nil, result.err
	}
	return driver.RowsAffected(1), nil
}

func (s *repositoryWriteStmt) Query([]driver.Value) (driver.Rows, error) {
	return s.script.query(s.query)
}

func (s *repositoryWriteStmt) ExecContext(_ context.Context, _ []driver.NamedValue) (driver.Result, error) {
	return s.Exec(nil)
}

func (s *repositoryWriteStmt) QueryContext(_ context.Context, _ []driver.NamedValue) (driver.Rows, error) {
	return s.Query(nil)
}

type repositoryWriteTx struct{}

func (t *repositoryWriteTx) Commit() error {
	return nil
}

func (t *repositoryWriteTx) Rollback() error {
	return nil
}

type repositoryWriteRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (r *repositoryWriteRows) Columns() []string {
	return r.columns
}

func (r *repositoryWriteRows) Close() error {
	return nil
}

func (r *repositoryWriteRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

func (s *repositoryWriteScript) next(query string) (repositoryWriteResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.index >= len(s.results) {
		return repositoryWriteResult{}, fmt.Errorf("unexpected query #%d: %s", s.index+1, compactHostPortSQL(query))
	}

	result := s.results[s.index]
	s.index++
	if result.expect != "" && !strings.Contains(compactHostPortSQL(query), compactHostPortSQL(result.expect)) {
		return repositoryWriteResult{}, fmt.Errorf("query #%d mismatch: got %q want substring %q", s.index, compactHostPortSQL(query), compactHostPortSQL(result.expect))
	}
	if result.inspect != nil {
		if err := result.inspect(query); err != nil {
			return repositoryWriteResult{}, fmt.Errorf("query #%d inspection failed: %w", s.index, err)
		}
	}
	return result, nil
}

func (s *repositoryWriteScript) query(query string) (driver.Rows, error) {
	result, err := s.next(query)
	if err != nil {
		return nil, err
	}
	if result.err != nil {
		return nil, result.err
	}

	columns := result.columns
	if len(columns) == 0 {
		columns = []string{"id"}
	}
	rows := result.rows
	if len(rows) == 0 {
		rows = [][]driver.Value{{int64(1)}}
	}
	return &repositoryWriteRows{columns: columns, rows: rows}, nil
}

func closeRepositorySQLDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}
}

func TestRepositoryCommandErrorsWithClosedSQLite(t *testing.T) {
	status := 200
	status16 := int16(204)
	length := 128
	vhost := true

	t.Run("directory", func(t *testing.T) {
		db := newAssetRepositoryDB(t)
		closeRepositorySQLDB(t, db)
		repo := NewDirectoryRepository(db)
		if _, err := repo.BatchCreate([]assetdomain.Directory{{TargetID: 1, URL: "https://example.com/a"}}); err == nil {
			t.Fatal("expected batch create error on closed db")
		}
		if _, err := repo.BatchUpsert([]assetdomain.Directory{{TargetID: 1, URL: "https://example.com/a"}}); err == nil {
			t.Fatal("expected batch upsert error on closed db")
		}
	})

	t.Run("subdomain", func(t *testing.T) {
		db := newAssetRepositoryDB(t)
		closeRepositorySQLDB(t, db)
		repo := NewSubdomainRepository(db)
		if _, err := repo.BatchCreate([]assetdomain.Subdomain{{TargetID: 1, DNSName: "api.example.com"}}); err == nil {
			t.Fatal("expected batch create error on closed db")
		}
	})

	t.Run("website", func(t *testing.T) {
		db := newAssetRepositoryDB(t)
		closeRepositorySQLDB(t, db)
		repo := NewWebsiteRepository(db)
		if _, err := repo.BatchCreate([]assetdomain.Website{{TargetID: 1, URL: "https://example.com", Host: "example.com"}}); err == nil {
			t.Fatal("expected batch create error on closed db")
		}
		if _, err := repo.BatchUpsert([]assetdomain.Website{{TargetID: 1, URL: "https://example.com", Host: "example.com", StatusCode: &status, ContentLength: &length, Vhost: &vhost}}); err == nil {
			t.Fatal("expected batch upsert error on closed db")
		}
	})

	t.Run("endpoint", func(t *testing.T) {
		db := newAssetRepositoryDB(t)
		closeRepositorySQLDB(t, db)
		repo := NewEndpointRepository(db)
		if _, err := repo.BatchCreate([]assetdomain.Endpoint{{TargetID: 1, URL: "https://example.com/api", Host: "example.com"}}); err == nil {
			t.Fatal("expected batch create error on closed db")
		}
		if _, err := repo.BatchUpsert([]assetdomain.Endpoint{{TargetID: 1, URL: "https://example.com/api", Host: "example.com", StatusCode: &status, ContentLength: &length, Vhost: &vhost}}); err == nil {
			t.Fatal("expected batch upsert error on closed db")
		}
	})

	t.Run("screenshot", func(t *testing.T) {
		db := newAssetRepositoryDB(t)
		closeRepositorySQLDB(t, db)
		repo := NewScreenshotRepository(db)
		if _, err := repo.BatchUpsert([]assetdomain.Screenshot{{TargetID: 1, URL: "https://example.com", StatusCode: &status16}}); err == nil {
			t.Fatal("expected batch upsert error on closed db")
		}
	})

	t.Run("host port", func(t *testing.T) {
		db := newAssetRepositoryDB(t)
		closeRepositorySQLDB(t, db)
		repo := NewHostPortRepository(db)
		if _, err := repo.BatchUpsert([]assetdomain.HostPort{{TargetID: 1, Host: "a.example.com", IP: "1.1.1.1", Port: 443}}); err == nil {
			t.Fatal("expected batch upsert error on closed db")
		}
	})
}

func TestRepositoryEndpointAndWebsiteBatchUpsertSuccessWithScriptedPostgres(t *testing.T) {
	status := 200
	length := 256
	vhost := true

	t.Run("website", func(t *testing.T) {
		repo := NewWebsiteRepository(newRepositoryWriteScriptedDB(t, repositoryWriteResult{
			expect:  `insert into "website"`,
			columns: []string{"id"},
			rows:    [][]driver.Value{{int64(1)}},
		}))

		affected, err := repo.BatchUpsert([]assetdomain.Website{{
			TargetID:      1,
			URL:           "https://example.com",
			Host:          "example.com",
			Title:         "home",
			StatusCode:    &status,
			ContentLength: &length,
			Vhost:         &vhost,
		}})
		if err != nil || affected != 1 {
			t.Fatalf("unexpected batch upsert result affected=%d err=%v", affected, err)
		}
	})

	t.Run("endpoint", func(t *testing.T) {
		repo := NewEndpointRepository(newRepositoryWriteScriptedDB(t, repositoryWriteResult{
			expect:  `insert into "endpoint"`,
			columns: []string{"id"},
			rows:    [][]driver.Value{{int64(1)}},
		}))

		affected, err := repo.BatchUpsert([]assetdomain.Endpoint{{
			TargetID:      1,
			URL:           "https://example.com/api",
			Host:          "example.com",
			Title:         "api",
			StatusCode:    &status,
			ContentLength: &length,
			Vhost:         &vhost,
		}})
		if err != nil || affected != 1 {
			t.Fatalf("unexpected batch upsert result affected=%d err=%v", affected, err)
		}
	})
}

func TestWebsiteRepositoryTechnologyUpsertOnlyReplacesTech(t *testing.T) {
	repo := NewWebsiteRepository(newRepositoryWriteScriptedDB(t, repositoryWriteResult{
		expect: `insert into "website"`,
		inspect: func(query string) error {
			const conflictPrefix = `on conflict ("url","target_id") do update set `
			normalized := compactHostPortSQL(query)
			index := strings.Index(normalized, conflictPrefix)
			if index < 0 {
				return fmt.Errorf("missing Website technology conflict clause: %q", normalized)
			}
			updates := normalized[index+len(conflictPrefix):]
			if beforeReturning, _, found := strings.Cut(updates, " returning "); found {
				updates = beforeReturning
			}
			if updates != `"tech"="excluded"."tech"` {
				return fmt.Errorf("technology conflict update = %q, want only tech", updates)
			}
			return nil
		},
	}))

	affected, err := repo.BatchUpsertTechnologyContext(context.Background(), 7, []assetdomain.WebsiteTechnology{{
		URL:  "https://api.example.com",
		Host: "api.example.com",
		Tech: []string{"nginx"},
	}})
	if err != nil || affected != 1 {
		t.Fatalf("BatchUpsertTechnologyContext() = %d, %v", affected, err)
	}
}
