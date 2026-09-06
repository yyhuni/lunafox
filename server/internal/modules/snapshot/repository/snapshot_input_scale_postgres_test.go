package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	snapshotScaleSchema              = "snapshot_input_scale"
	snapshotScaleDatabase            = "lunafox_snapshot_scale"
	snapshotScaleMaxHeapGrowth       = 96 << 20
	snapshotScaleMaxTempBlocks       = 262_144
	snapshotScaleMaxStartup          = 2 * time.Minute
	snapshotScaleMaxQueryOrIteration = 3 * time.Minute
)

func TestSnapshotInputScalePostgres(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LUNAFOX_POSTGRES_SCALE_DSN"))
	if dsn == "" {
		t.Skip("set LUNAFOX_POSTGRES_SCALE_DSN to run the opt-in PostgreSQL scale verification")
	}
	if !strings.Contains(dsn, "search_path="+snapshotScaleSchema) {
		t.Fatalf("scale DSN must pin search_path=%s", snapshotScaleSchema)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL scale database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open PostgreSQL SQL handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	var databaseName string
	if err := sqlDB.QueryRowContext(ctx, "SELECT current_database()").Scan(&databaseName); err != nil {
		t.Fatalf("read PostgreSQL database identity: %v", err)
	}
	if databaseName != snapshotScaleDatabase {
		t.Fatalf("refusing destructive scale fixture in database %q; want %q", databaseName, snapshotScaleDatabase)
	}

	resetSnapshotScaleSchema(t, ctx, db)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = db.WithContext(cleanupCtx).Exec("DROP SCHEMA IF EXISTS " + snapshotScaleSchema + " CASCADE").Error
	})

	for _, scale := range []struct {
		name   string
		scanID int
		rows   int
	}{
		{name: "100k", scanID: 100_001, rows: 100_000},
		{name: "1m", scanID: 1_000_001, rows: 1_000_000},
	} {
		t.Run(scale.name, func(t *testing.T) {
			insertSnapshotScaleRows(t, ctx, sqlDB, scale.scanID, scale.rows)
			if _, err := sqlDB.ExecContext(ctx, "ANALYZE subdomain_snapshot"); err != nil {
				t.Fatalf("analyze subdomain snapshot: %v", err)
			}
			if _, err := sqlDB.ExecContext(ctx, "ANALYZE host_port_mapping_snapshot"); err != nil {
				t.Fatalf("analyze HostPort snapshot: %v", err)
			}

			subdomainExplain := explainSnapshotScaleQuery(t, ctx, sqlDB,
				"SELECT dns_name FROM subdomain_snapshot WHERE scan_id = $1 AND dns_name <> '' ORDER BY dns_name ASC",
				scale.scanID,
			)
			hostPortExplain := explainSnapshotScaleQuery(t, ctx, sqlDB,
				"SELECT host, ip, port FROM host_port_mapping_snapshot WHERE scan_id = $1 ORDER BY host ASC, ip ASC, port ASC",
				scale.scanID,
			)
			assertSnapshotScaleExplain(t, "subdomain", scale.rows, subdomainExplain)
			assertSnapshotScaleExplain(t, "host_port", scale.rows, hostPortExplain)
			assertSnapshotScalePartitionPruning(t, subdomainExplain, "subdomain_snapshot", scale.scanID)
			assertSnapshotScalePartitionPruning(t, hostPortExplain, "host_port_mapping_snapshot", scale.scanID)

			subdomainCursor := measureSnapshotCursor(t, func(visit func()) error {
				return NewSubdomainSnapshotRepository(db).ForEachDNSNameByScanID(ctx, scale.scanID, func(string) error {
					visit()
					return nil
				})
			})
			hostPortCursor := measureSnapshotCursor(t, func(visit func()) error {
				return NewHostPortSnapshotRepository(db).ForEachHostPortByScanID(ctx, scale.scanID, func(snapshotdomain.HostPortInputEvidence) error {
					visit()
					return nil
				})
			})
			assertSnapshotCursorMeasurement(t, "subdomain", scale.rows, subdomainCursor)
			assertSnapshotCursorMeasurement(t, "host_port", scale.rows, hostPortCursor)

			logSnapshotScaleMeasurement(t, scale.name, "subdomain", scale.rows, subdomainExplain, subdomainCursor)
			logSnapshotScaleMeasurement(t, scale.name, "host_port", scale.rows, hostPortExplain, hostPortCursor)
			if scale.name == "1m" {
				measureConcurrentSnapshotCursors(t, ctx, db, scale.scanID, scale.rows)
			}
			measureSnapshotPartitionCleanup(t, ctx, sqlDB, "subdomain_snapshot", scale.scanID)
			measureSnapshotPartitionCleanup(t, ctx, sqlDB, "host_port_mapping_snapshot", scale.scanID)
			measureBoundedSnapshotScaleTaskDeletion(t, ctx, sqlDB, scale.scanID)
		})
	}
}

func resetSnapshotScaleSchema(t *testing.T, ctx context.Context, db *gorm.DB) {
	t.Helper()
	statements := []string{
		"DROP SCHEMA IF EXISTS " + snapshotScaleSchema + " CASCADE",
		"CREATE SCHEMA " + snapshotScaleSchema,
		`CREATE TABLE subdomain_snapshot (
			id BIGSERIAL NOT NULL,
			scan_id INTEGER NOT NULL,
			dns_name VARCHAR(1000) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (scan_id, id)
		) PARTITION BY RANGE (scan_id)`,
		"CREATE UNIQUE INDEX unique_subdomain_dns_name_per_scan_snapshot ON subdomain_snapshot(scan_id, dns_name)",
		`CREATE TABLE host_port_mapping_snapshot (
			id BIGSERIAL NOT NULL,
			scan_id INTEGER NOT NULL,
			host VARCHAR(1000) NOT NULL,
			ip INET NOT NULL,
			port INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (scan_id, id)
		) PARTITION BY RANGE (scan_id)`,
		"CREATE INDEX idx_hpm_snap_scan_effective_host_port ON host_port_mapping_snapshot(scan_id, (COALESCE(NULLIF(LOWER(BTRIM(host)), ''), HOST(ip))), port)",
		"CREATE UNIQUE INDEX unique_scan_host_ip_port_snapshot ON host_port_mapping_snapshot(scan_id, host, ip, port)",
		`CREATE TABLE scan_task (id BIGSERIAL PRIMARY KEY, scan_id INTEGER NOT NULL)`,
	}
	for _, scanID := range []int{100_001, 110_001, 1_000_001, 1_010_001} {
		partitionRange, err := scanHistoryRangeForID(scanID)
		if err != nil {
			t.Fatalf("resolve scale partition range: %v", err)
		}
		for _, parent := range []string{"subdomain_snapshot", "host_port_mapping_snapshot"} {
			statements = append(statements, fmt.Sprintf(
				"CREATE TABLE %s PARTITION OF %s FOR VALUES FROM (%d) TO (%d)",
				scanHistoryPartitionName(parent, partitionRange.Start), parent, partitionRange.Start, partitionRange.End,
			))
		}
	}
	for _, statement := range statements {
		if err := db.WithContext(ctx).Exec(statement).Error; err != nil {
			t.Fatalf("prepare PostgreSQL scale schema: %v", err)
		}
	}
}

func insertSnapshotScaleRows(t *testing.T, ctx context.Context, db *sql.DB, scanID, rows int) {
	t.Helper()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO subdomain_snapshot (scan_id, dns_name)
		SELECT $1::INTEGER, 'host-' || LPAD(value::text, 7, '0') || '.scan-' || ($1::INTEGER)::text || '.example.com'
		FROM generate_series(1, $2::INTEGER) AS value
	`, scanID, rows); err != nil {
		t.Fatalf("insert %d subdomain snapshots: %v", rows, err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO host_port_mapping_snapshot (scan_id, host, ip, port)
		SELECT
			$1::INTEGER,
			CASE WHEN value % 4 = 0 THEN '' ELSE 'host-' || LPAD(value::text, 7, '0') || '.scan-' || ($1::INTEGER)::text || '.example.com' END,
			INET '10.0.0.0' + (((value - 1) % 16000000)::BIGINT),
			(ARRAY[80, 443, 8080, 8443])[(value % 4) + 1]
		FROM generate_series(1, $2::INTEGER) AS value
	`, scanID, rows); err != nil {
		t.Fatalf("insert %d HostPort snapshots: %v", rows, err)
	}
}

type snapshotExplainDocument struct {
	Plan          snapshotExplainNode `json:"Plan"`
	PlanningTime  float64             `json:"Planning Time"`
	ExecutionTime float64             `json:"Execution Time"`
}

type snapshotExplainNode struct {
	NodeType          string                `json:"Node Type"`
	RelationName      string                `json:"Relation Name"`
	IndexName         string                `json:"Index Name"`
	ActualStartupTime float64               `json:"Actual Startup Time"`
	ActualTotalTime   float64               `json:"Actual Total Time"`
	ActualRows        float64               `json:"Actual Rows"`
	SharedHitBlocks   int64                 `json:"Shared Hit Blocks"`
	SharedReadBlocks  int64                 `json:"Shared Read Blocks"`
	TempReadBlocks    int64                 `json:"Temp Read Blocks"`
	TempWrittenBlocks int64                 `json:"Temp Written Blocks"`
	Plans             []snapshotExplainNode `json:"Plans"`
}

type snapshotExplainMeasurement struct {
	Startup       time.Duration
	Total         time.Duration
	Rows          int64
	SharedHit     int64
	SharedRead    int64
	TempRead      int64
	TempWritten   int64
	NodeTypes     []string
	IndexNames    []string
	RelationNames []string
	PlanningTime  time.Duration
	ExecutionTime time.Duration
}

func explainSnapshotScaleQuery(t *testing.T, ctx context.Context, db *sql.DB, query string, scanID int) snapshotExplainMeasurement {
	t.Helper()
	var raw string
	if err := db.QueryRowContext(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+query, scanID).Scan(&raw); err != nil {
		t.Fatalf("run PostgreSQL scale EXPLAIN: %v", err)
	}
	var documents []snapshotExplainDocument
	if err := json.Unmarshal([]byte(raw), &documents); err != nil || len(documents) != 1 {
		t.Fatalf("decode PostgreSQL scale EXPLAIN: documents=%d err=%v", len(documents), err)
	}
	measurement := snapshotExplainMeasurement{
		Startup:       milliseconds(documents[0].Plan.ActualStartupTime),
		Total:         milliseconds(documents[0].Plan.ActualTotalTime),
		Rows:          int64(math.Round(documents[0].Plan.ActualRows)),
		PlanningTime:  milliseconds(documents[0].PlanningTime),
		ExecutionTime: milliseconds(documents[0].ExecutionTime),
	}
	collectSnapshotExplainFacts(documents[0].Plan, &measurement)
	sort.Strings(measurement.NodeTypes)
	sort.Strings(measurement.IndexNames)
	sort.Strings(measurement.RelationNames)
	return measurement
}

func collectSnapshotExplainFacts(node snapshotExplainNode, measurement *snapshotExplainMeasurement) {
	measurement.SharedHit = max(measurement.SharedHit, node.SharedHitBlocks)
	measurement.SharedRead = max(measurement.SharedRead, node.SharedReadBlocks)
	measurement.TempRead = max(measurement.TempRead, node.TempReadBlocks)
	measurement.TempWritten = max(measurement.TempWritten, node.TempWrittenBlocks)
	if node.NodeType != "" {
		measurement.NodeTypes = append(measurement.NodeTypes, node.NodeType)
	}
	if node.IndexName != "" {
		measurement.IndexNames = append(measurement.IndexNames, node.IndexName)
	}
	if node.RelationName != "" {
		measurement.RelationNames = append(measurement.RelationNames, node.RelationName)
	}
	for _, child := range node.Plans {
		collectSnapshotExplainFacts(child, measurement)
	}
}

func assertSnapshotScalePartitionPruning(t *testing.T, measurement snapshotExplainMeasurement, parent string, scanID int) {
	t.Helper()
	partitionRange, err := scanHistoryRangeForID(scanID)
	if err != nil {
		t.Fatalf("resolve query partition range: %v", err)
	}
	want := scanHistoryPartitionName(parent, partitionRange.Start)
	for _, relation := range measurement.RelationNames {
		if relation == want {
			return
		}
	}
	t.Fatalf("%s EXPLAIN did not reach expected pruned child %q; relations=%q", parent, want, measurement.RelationNames)
}

func milliseconds(value float64) time.Duration {
	return time.Duration(value * float64(time.Millisecond))
}

type snapshotCursorMeasurement struct {
	Rows           int
	FirstRow       time.Duration
	Total          time.Duration
	PeakHeapGrowth uint64
}

func measureSnapshotCursor(t *testing.T, run func(func()) error) snapshotCursorMeasurement {
	t.Helper()
	runtime.GC()
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	baseline := memory.HeapAlloc
	peak := baseline
	measurement := snapshotCursorMeasurement{}
	startedAt := time.Now()
	err := run(func() {
		measurement.Rows++
		if measurement.Rows == 1 {
			measurement.FirstRow = time.Since(startedAt)
		}
		if measurement.Rows%4096 == 0 {
			runtime.ReadMemStats(&memory)
			peak = max(peak, memory.HeapAlloc)
		}
	})
	measurement.Total = time.Since(startedAt)
	if err != nil {
		t.Fatalf("stream PostgreSQL snapshot cursor: %v", err)
	}
	runtime.ReadMemStats(&memory)
	peak = max(peak, memory.HeapAlloc)
	if peak > baseline {
		measurement.PeakHeapGrowth = peak - baseline
	}
	return measurement
}

func assertSnapshotScaleExplain(t *testing.T, kind string, rows int, measurement snapshotExplainMeasurement) {
	t.Helper()
	if measurement.Rows != int64(rows) {
		t.Fatalf("%s EXPLAIN rows = %d, want %d", kind, measurement.Rows, rows)
	}
	if measurement.Startup > snapshotScaleMaxStartup || measurement.ExecutionTime > snapshotScaleMaxQueryOrIteration {
		t.Fatalf("%s EXPLAIN exceeded scale budget: startup=%s execution=%s", kind, measurement.Startup, measurement.ExecutionTime)
	}
	if measurement.TempWritten > snapshotScaleMaxTempBlocks || measurement.TempRead > snapshotScaleMaxTempBlocks {
		t.Fatalf("%s EXPLAIN temp I/O exceeded %d blocks: read=%d written=%d", kind, snapshotScaleMaxTempBlocks, measurement.TempRead, measurement.TempWritten)
	}
}

func assertSnapshotCursorMeasurement(t *testing.T, kind string, rows int, measurement snapshotCursorMeasurement) {
	t.Helper()
	if measurement.Rows != rows {
		t.Fatalf("%s cursor rows = %d, want %d", kind, measurement.Rows, rows)
	}
	if measurement.Total > snapshotScaleMaxQueryOrIteration {
		t.Fatalf("%s cursor exceeded scale budget: %s", kind, measurement.Total)
	}
	if measurement.PeakHeapGrowth > snapshotScaleMaxHeapGrowth {
		t.Fatalf("%s cursor peak heap growth = %d, want <= %d", kind, measurement.PeakHeapGrowth, snapshotScaleMaxHeapGrowth)
	}
}

func logSnapshotScaleMeasurement(t *testing.T, scale, kind string, rows int, explain snapshotExplainMeasurement, cursor snapshotCursorMeasurement) {
	t.Helper()
	t.Logf("SCALE_METRIC scale=%s kind=%s rows=%d first_row=%s cursor_total=%s peak_heap_growth_bytes=%d explain_startup=%s explain_total=%s planning=%s execution=%s shared_hit_blocks=%d shared_read_blocks=%d temp_read_blocks=%d temp_written_blocks=%d nodes=%q indexes=%q",
		scale, kind, rows, cursor.FirstRow, cursor.Total, cursor.PeakHeapGrowth,
		explain.Startup, explain.Total, explain.PlanningTime, explain.ExecutionTime,
		explain.SharedHit, explain.SharedRead, explain.TempRead, explain.TempWritten,
		strings.Join(explain.NodeTypes, ","), strings.Join(explain.IndexNames, ","),
	)
}

func measureSnapshotPartitionCleanup(t *testing.T, ctx context.Context, db *sql.DB, parent string, scanID int) {
	t.Helper()
	partitionRange, err := scanHistoryRangeForID(scanID)
	if err != nil {
		t.Fatalf("resolve cleanup partition range: %v", err)
	}
	partition := scanHistoryPartitionName(parent, partitionRange.Start)
	var bytes int64
	if err := db.QueryRowContext(ctx, `SELECT pg_total_relation_size($1::regclass)`, partition).Scan(&bytes); err != nil {
		t.Fatalf("measure %s bytes: %v", partition, err)
	}
	if bytes <= 0 {
		t.Fatalf("%s must contain measurable data before detach/drop, got %d bytes", partition, bytes)
	}
	before := snapshotScaleCurrentLSN(t, ctx, db)
	startedAt := time.Now()
	if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s DETACH PARTITION %s", parent, partition)); err != nil {
		t.Fatalf("detach %s: %v", partition, err)
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s", partition)); err != nil {
		t.Fatalf("drop %s: %v", partition, err)
	}
	duration := time.Since(startedAt)
	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, partition).Scan(&exists); err != nil || exists {
		t.Fatalf("verify %s dropped: exists=%t err=%v", partition, exists, err)
	}
	after := snapshotScaleCurrentLSN(t, ctx, db)
	walBytes := snapshotScaleWALDiff(t, ctx, db, after, before)
	t.Logf("PARTITION_CLEANUP_METRIC parent=%s partition=%s rows_scope_scan_id=%d bytes=%d duration=%s wal_bytes=%d operation=detach_drop", parent, partition, scanID, bytes, duration, walBytes)
}

func measureBoundedSnapshotScaleTaskDeletion(t *testing.T, ctx context.Context, db *sql.DB, scanID int) {
	t.Helper()
	const batchSize = 1_000
	const taskCount = 2_500
	if _, err := db.ExecContext(ctx, `INSERT INTO scan_task (scan_id) SELECT $1 FROM generate_series(1, $2)`, scanID, taskCount); err != nil {
		t.Fatalf("insert scale scan tasks: %v", err)
	}
	before := snapshotScaleCurrentLSN(t, ctx, db)
	startedAt := time.Now()
	deleted := int64(0)
	batches := 0
	for {
		result, err := db.ExecContext(ctx, `
			WITH task_batch AS (
				SELECT id
				FROM scan_task
				WHERE scan_id = $1
				ORDER BY id
				LIMIT $2
			)
			DELETE FROM scan_task AS task
			USING task_batch
			WHERE task.id = task_batch.id`, scanID, batchSize)
		if err != nil {
			t.Fatalf("batch-delete scan tasks: %v", err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			t.Fatalf("read batch deletion affected rows: %v", err)
		}
		if rowsAffected > batchSize {
			t.Fatalf("batch deletion removed %d rows, exceeds %d", rowsAffected, batchSize)
		}
		deleted += rowsAffected
		if rowsAffected == 0 {
			break
		}
		batches++
	}
	if deleted != taskCount || batches != 3 {
		t.Fatalf("bounded task deletion removed=%d batches=%d, want removed=%d batches=3", deleted, batches, taskCount)
	}
	after := snapshotScaleCurrentLSN(t, ctx, db)
	walBytes := snapshotScaleWALDiff(t, ctx, db, after, before)
	t.Logf("TASK_CLEANUP_METRIC scan_id=%d rows=%d batches=%d batch_size=%d duration=%s wal_bytes=%d operation=bounded_delete", scanID, deleted, batches, batchSize, time.Since(startedAt), walBytes)
}

func snapshotScaleCurrentLSN(t *testing.T, ctx context.Context, db *sql.DB) string {
	t.Helper()
	var lsn string
	if err := db.QueryRowContext(ctx, `SELECT pg_current_wal_lsn()`).Scan(&lsn); err != nil {
		t.Fatalf("read current WAL LSN: %v", err)
	}
	return lsn
}

func snapshotScaleWALDiff(t *testing.T, ctx context.Context, db *sql.DB, after, before string) int64 {
	t.Helper()
	var bytes float64
	if err := db.QueryRowContext(ctx, `SELECT pg_wal_lsn_diff($1, $2)`, after, before).Scan(&bytes); err != nil {
		t.Fatalf("measure WAL delta: %v", err)
	}
	if bytes < 0 {
		t.Fatalf("WAL delta must not be negative: %f", bytes)
	}
	return int64(bytes)
}

type concurrentSnapshotCursorResult struct {
	Name     string
	Rows     int
	FirstRow time.Duration
	Total    time.Duration
	Err      error
}

func measureConcurrentSnapshotCursors(t *testing.T, ctx context.Context, db *gorm.DB, scanID, expectedRows int) {
	t.Helper()
	type cursorJob struct {
		name string
		run  func(func()) error
	}
	jobs := []cursorJob{
		{name: "subdomain-a", run: func(visit func()) error {
			return NewSubdomainSnapshotRepository(db).ForEachDNSNameByScanID(ctx, scanID, func(string) error { visit(); return nil })
		}},
		{name: "subdomain-b", run: func(visit func()) error {
			return NewSubdomainSnapshotRepository(db).ForEachDNSNameByScanID(ctx, scanID, func(string) error { visit(); return nil })
		}},
		{name: "host-port-a", run: func(visit func()) error {
			return NewHostPortSnapshotRepository(db).ForEachHostPortByScanID(ctx, scanID, func(snapshotdomain.HostPortInputEvidence) error { visit(); return nil })
		}},
		{name: "host-port-b", run: func(visit func()) error {
			return NewHostPortSnapshotRepository(db).ForEachHostPortByScanID(ctx, scanID, func(snapshotdomain.HostPortInputEvidence) error { visit(); return nil })
		}},
	}

	runtime.GC()
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	baseline := memory.HeapAlloc
	peak := baseline
	stopSampling := make(chan struct{})
	samplingDone := make(chan struct{})
	go func() {
		defer close(samplingDone)
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				runtime.ReadMemStats(&memory)
				peak = max(peak, memory.HeapAlloc)
			case <-stopSampling:
				return
			}
		}
	}()

	startedAt := time.Now()
	results := make(chan concurrentSnapshotCursorResult, len(jobs))
	var wait sync.WaitGroup
	for _, job := range jobs {
		wait.Add(1)
		go func(job cursorJob) {
			defer wait.Done()
			result := concurrentSnapshotCursorResult{Name: job.name}
			started := time.Now()
			result.Err = job.run(func() {
				result.Rows++
				if result.Rows == 1 {
					result.FirstRow = time.Since(started)
				}
			})
			result.Total = time.Since(started)
			results <- result
		}(job)
	}
	wait.Wait()
	wallTime := time.Since(startedAt)
	close(stopSampling)
	<-samplingDone
	close(results)
	runtime.ReadMemStats(&memory)
	peak = max(peak, memory.HeapAlloc)
	peakGrowth := uint64(0)
	if peak > baseline {
		peakGrowth = peak - baseline
	}

	for result := range results {
		if result.Err != nil {
			t.Fatalf("concurrent %s cursor: %v", result.Name, result.Err)
		}
		if result.Rows != expectedRows {
			t.Fatalf("concurrent %s cursor rows = %d, want %d", result.Name, result.Rows, expectedRows)
		}
		if result.FirstRow > snapshotScaleMaxStartup || result.Total > snapshotScaleMaxQueryOrIteration {
			t.Fatalf("concurrent %s cursor exceeded scale budget: first=%s total=%s", result.Name, result.FirstRow, result.Total)
		}
		t.Logf("CONCURRENCY_METRIC cursor=%s rows=%d first_row=%s total=%s", result.Name, result.Rows, result.FirstRow, result.Total)
	}
	if peakGrowth > 4*snapshotScaleMaxHeapGrowth {
		t.Fatalf("concurrent cursor peak heap growth = %d, want <= %d", peakGrowth, 4*snapshotScaleMaxHeapGrowth)
	}
	t.Logf("CONCURRENCY_METRIC cursors=%d rows_per_cursor=%d wall=%s peak_heap_growth_bytes=%d", len(jobs), expectedRows, wallTime, peakGrowth)
}
