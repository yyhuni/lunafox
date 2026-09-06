package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	serverdatabase "github.com/yyhuni/lunafox/server/internal/database"
	catalogrepository "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	targetcleanupapp "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/application"
	cleanupdomain "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/domain"
	targetcleanuprepository "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	targetCleanupScalePostgresDSNEnv = "LUNAFOX_TARGET_CLEANUP_SCALE_POSTGRES_DSN"
	targetCleanupScaleDatabase       = "lunafox_target_cleanup_scale"
	targetCleanupScaleSmallRows      = 100_000
	targetCleanupScaleLargeRows      = 1_000_000
	targetCleanupScaleBatchSize      = 1_000
	targetCleanupScaleMaxHeapGrowth  = 64 << 20
)

type targetCleanupScaleResourceSpec struct {
	resource        cleanupdomain.AssetResource
	table           string
	indexName       string
	noiseMultiplier int
}

var targetCleanupScaleResources = []targetCleanupScaleResourceSpec{
	{resource: cleanupdomain.AssetResourceSubdomain, table: "subdomain", indexName: "idx_subdomain_target_id", noiseMultiplier: 1},
	{resource: cleanupdomain.AssetResourceHostPortMapping, table: "host_port_mapping", indexName: "idx_hpm_target_id", noiseMultiplier: 1},
	{resource: cleanupdomain.AssetResourceWebsite, table: "website", indexName: "idx_website_target_id", noiseMultiplier: 3},
	{resource: cleanupdomain.AssetResourceEndpoint, table: "endpoint", indexName: "idx_endpoint_target_id", noiseMultiplier: 3},
	{resource: cleanupdomain.AssetResourceDirectory, table: "directory", indexName: "idx_directory_target_id", noiseMultiplier: 1},
	{resource: cleanupdomain.AssetResourceScreenshot, table: "screenshot", indexName: "idx_screenshot_target_id", noiseMultiplier: 1},
	{resource: cleanupdomain.AssetResourceVulnerability, table: "vulnerability", indexName: "idx_vuln_target_id", noiseMultiplier: 3},
}

// TestTargetCleanupScalePostgres is intentionally opt-in. The release gate
// supplies a fresh PostgreSQL 18 database; ordinary Go tests must not create
// or retain million-row fixtures.
func TestTargetCleanupScalePostgres(t *testing.T) {
	db, sqlDB := openTargetCleanupScalePostgres(t)
	ctx := context.Background()
	for _, spec := range targetCleanupScaleResources {
		spec := spec
		t.Run(string(spec.resource), func(t *testing.T) {
			targetCleanupScaleResource(t, ctx, db, sqlDB, spec)
		})
	}
}

func openTargetCleanupScalePostgres(t *testing.T) (*gorm.DB, *sql.DB) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv(targetCleanupScalePostgresDSNEnv))
	if dsn == "" {
		t.Skip("set " + targetCleanupScalePostgresDSNEnv + " to run the PostgreSQL 18 Target cleanup scale gate")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("open Target cleanup scale PostgreSQL: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open Target cleanup scale SQL handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = sqlDB.Close() })

	var databaseName string
	if err := db.Raw("SELECT current_database()").Scan(&databaseName).Error; err != nil {
		t.Fatalf("read Target cleanup scale database name: %v", err)
	}
	if databaseName != targetCleanupScaleDatabase {
		t.Fatalf("refusing destructive Target cleanup scale database %q; want %q", databaseName, targetCleanupScaleDatabase)
	}
	var publicTables int64
	if err := db.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'`).Scan(&publicTables).Error; err != nil {
		t.Fatalf("inspect Target cleanup scale database: %v", err)
	}
	if publicTables != 0 {
		t.Fatalf("Target cleanup scale database must start empty, found %d public tables", publicTables)
	}

	previousFS, previousPath := serverdatabase.MigrationsFS, serverdatabase.MigrationsPath
	serverdatabase.MigrationsFS = migrationsFS
	serverdatabase.MigrationsPath = "migrations"
	t.Cleanup(func() {
		serverdatabase.MigrationsFS = previousFS
		serverdatabase.MigrationsPath = previousPath
	})
	if err := serverdatabase.RunMigrations(sqlDB); err != nil {
		t.Fatalf("apply empty 000001 baseline for Target cleanup scale gate: %v", err)
	}
	t.Cleanup(func() {
		if err := serverdatabase.MigrateDown(sqlDB); err != nil {
			t.Errorf("tear down Target cleanup scale baseline: %v", err)
		}
	})
	return db, sqlDB
}

func targetCleanupScaleResource(t *testing.T, ctx context.Context, db *gorm.DB, sqlDB *sql.DB, spec targetCleanupScaleResourceSpec) {
	t.Helper()

	emptyTargetID := targetCleanupScaleCreateTarget(t, db, fmt.Sprintf("scale-empty-%s", spec.table))
	emptyTrace := targetCleanupScaleTombstoneAndTrace(t, db, emptyTargetID)
	targetCleanupScaleAssertSynchronousDeleteTrace(t, emptyTrace)
	emptySignature := targetCleanupScaleTraceSignature(emptyTrace)

	smallNoiseRows := targetCleanupScaleNoiseRows(spec, targetCleanupScaleSmallRows)
	smallNoiseTargetID := targetCleanupScaleCreateTarget(t, db, fmt.Sprintf("scale-small-noise-%s", spec.table))
	targetCleanupScaleInsertRows(t, ctx, sqlDB, spec, smallNoiseTargetID, smallNoiseRows)
	smallTargetID := targetCleanupScaleCreateTarget(t, db, fmt.Sprintf("scale-small-%s", spec.table))
	targetCleanupScaleInsertRows(t, ctx, sqlDB, spec, smallTargetID, targetCleanupScaleSmallRows)
	smallTrace := targetCleanupScaleTombstoneAndTrace(t, db, smallTargetID)
	targetCleanupScaleAssertSynchronousDeleteTrace(t, smallTrace)
	if got := targetCleanupScaleTraceSignature(smallTrace); !equalStrings(got, emptySignature) {
		t.Fatalf("synchronous DELETE trace changed with %d %s rows:\nempty=%q\nsmall=%q", targetCleanupScaleSmallRows, spec.table, emptySignature, got)
	}
	if _, err := sqlDB.ExecContext(ctx, "ANALYZE "+spec.table); err != nil {
		t.Fatalf("analyze %s small fixture: %v", spec.table, err)
	}
	smallExplain := targetCleanupScaleExplainCandidates(t, ctx, sqlDB, spec, smallTargetID)
	targetCleanupScaleAssertPlan(t, fmt.Sprintf("%s/100k", spec.table), spec, smallExplain)
	smallBatch, err := targetCleanupScaleMeasureBatch(func() (int64, error) {
		return targetcleanuprepository.NewTargetCleanupRepository(db).DeleteCurrentAssetBatch(ctx, smallTargetID, spec.resource, targetCleanupScaleBatchSize)
	})
	if err != nil {
		t.Fatalf("run 100k %s cleanup batch: %v", spec.table, err)
	}
	if smallBatch.deleted != targetCleanupScaleBatchSize {
		t.Fatalf("100k %s batch deleted=%d, want %d", spec.table, smallBatch.deleted, targetCleanupScaleBatchSize)
	}
	smallRemaining := targetCleanupScaleCountRows(t, ctx, sqlDB, spec.table, smallTargetID)
	if smallRemaining != targetCleanupScaleSmallRows-targetCleanupScaleBatchSize {
		t.Fatalf("100k %s remaining=%d, want %d", spec.table, smallRemaining, targetCleanupScaleSmallRows-targetCleanupScaleBatchSize)
	}
	if noiseRows := targetCleanupScaleCountRows(t, ctx, sqlDB, spec.table, smallNoiseTargetID); noiseRows != smallNoiseRows {
		t.Fatalf("100k %s cleanup crossed Target scope: noise rows=%d, want %d", spec.table, noiseRows, smallNoiseRows)
	}
	t.Logf("SCALE_METRIC resource=%s scale=100k fixture_rows=%d batch_deleted=%d remaining=%d duration=%s heap_growth_bytes=%d explain_rows=%d explain_shared_read_blocks=%d explain_shared_hit_blocks=%d explain_indexes=%q explain_nodes=%q",
		spec.resource, targetCleanupScaleSmallRows, smallBatch.deleted, smallRemaining, smallBatch.duration, smallBatch.heapGrowth,
		smallExplain.rows, smallExplain.sharedReadBlocks, smallExplain.sharedHitBlocks, strings.Join(smallExplain.indexes, ","), strings.Join(smallExplain.nodes, ","))
	targetCleanupScaleTruncate(t, ctx, sqlDB, spec.table)

	largeNoiseRows := targetCleanupScaleNoiseRows(spec, targetCleanupScaleLargeRows)
	largeNoiseTargetID := targetCleanupScaleCreateTarget(t, db, fmt.Sprintf("scale-large-noise-%s", spec.table))
	targetCleanupScaleInsertRows(t, ctx, sqlDB, spec, largeNoiseTargetID, largeNoiseRows)
	largeTargetID := targetCleanupScaleCreateTarget(t, db, fmt.Sprintf("scale-large-%s", spec.table))
	targetCleanupScaleInsertRows(t, ctx, sqlDB, spec, largeTargetID, targetCleanupScaleLargeRows)
	largeTrace := targetCleanupScaleTombstoneAndTrace(t, db, largeTargetID)
	targetCleanupScaleAssertSynchronousDeleteTrace(t, largeTrace)
	if got := targetCleanupScaleTraceSignature(largeTrace); !equalStrings(got, emptySignature) {
		t.Fatalf("synchronous DELETE trace changed with %d %s rows:\nempty=%q\nlarge=%q", targetCleanupScaleLargeRows, spec.table, emptySignature, got)
	}
	if _, err := sqlDB.ExecContext(ctx, "ANALYZE "+spec.table); err != nil {
		t.Fatalf("analyze %s large fixture: %v", spec.table, err)
	}
	largeExplain := targetCleanupScaleExplainCandidates(t, ctx, sqlDB, spec, largeTargetID)
	targetCleanupScaleAssertPlan(t, fmt.Sprintf("%s/1m", spec.table), spec, largeExplain)
	largeBatch, workerResult := targetCleanupScaleMeasureWorker(t, ctx, db, largeTargetID)
	if workerResult.AssetBatches != 1 || !workerResult.Deferred || workerResult.Completed {
		t.Fatalf("1m %s worker result=%+v, want one deferred bounded batch", spec.table, workerResult)
	}
	if largeBatch.deleted != targetCleanupScaleBatchSize {
		t.Fatalf("1m %s worker batch deleted=%d, want %d", spec.table, largeBatch.deleted, targetCleanupScaleBatchSize)
	}
	largeRemaining := targetCleanupScaleCountRows(t, ctx, sqlDB, spec.table, largeTargetID)
	if largeRemaining != targetCleanupScaleLargeRows-targetCleanupScaleBatchSize {
		t.Fatalf("1m %s remaining=%d, want %d", spec.table, largeRemaining, targetCleanupScaleLargeRows-targetCleanupScaleBatchSize)
	}
	if noiseRows := targetCleanupScaleCountRows(t, ctx, sqlDB, spec.table, largeNoiseTargetID); noiseRows != largeNoiseRows {
		t.Fatalf("1m %s cleanup crossed Target scope: noise rows=%d, want %d", spec.table, noiseRows, largeNoiseRows)
	}
	if largeBatch.heapGrowth > targetCleanupScaleMaxHeapGrowth {
		t.Fatalf("1m %s worker heap growth=%d, want <=%d", spec.table, largeBatch.heapGrowth, targetCleanupScaleMaxHeapGrowth)
	}
	t.Logf("SCALE_METRIC resource=%s scale=1m fixture_rows=%d batch_deleted=%d remaining=%d duration=%s heap_growth_bytes=%d explain_rows=%d explain_shared_read_blocks=%d explain_shared_hit_blocks=%d explain_indexes=%q explain_nodes=%q",
		spec.resource, targetCleanupScaleLargeRows, largeBatch.deleted, largeRemaining, largeBatch.duration, largeBatch.heapGrowth,
		largeExplain.rows, largeExplain.sharedReadBlocks, largeExplain.sharedHitBlocks, strings.Join(largeExplain.indexes, ","), strings.Join(largeExplain.nodes, ","))

	// A duration boundary defers before starting a batch. This is a deterministic
	// boundary check; it is not a machine-dependent end-to-end cleanup deadline.
	boundaryJob := targetCleanupScaleFindJob(t, db, largeTargetID)
	boundaryService, err := targetcleanupapp.NewTargetCleanupReconciliationService(
		targetcleanuprepository.NewTargetCleanupRepository(db),
		targetCleanupScaleScheduleCleaner{},
		targetCleanupScaleScanCanceller{},
		targetCleanupScalePublisher{},
	)
	if err != nil {
		t.Fatalf("create duration-boundary cleanup service: %v", err)
	}
	boundaryStart := time.Now().UTC()
	clockCalls := 0
	boundaryService.WithClock(func() time.Time {
		clockCalls++
		if clockCalls == 1 {
			return boundaryStart
		}
		return boundaryStart.Add(time.Second)
	})
	boundaryResult, err := boundaryService.Reconcile(ctx, boundaryJob, targetcleanupapp.TargetCleanupRunOptions{
		AssetBatchSize: targetCleanupScaleBatchSize, MaxAssetBatches: 100, MaxRunDuration: time.Nanosecond,
	})
	if err != nil || !boundaryResult.Deferred || boundaryResult.AssetBatches != 0 {
		t.Fatalf("%s duration-boundary result=%+v err=%v, want deferred before first batch", spec.table, boundaryResult, err)
	}
	if got := targetCleanupScaleCountRows(t, ctx, sqlDB, spec.table, largeTargetID); got != largeRemaining {
		t.Fatalf("%s duration boundary changed rows: got=%d want=%d", spec.table, got, largeRemaining)
	}

	targetCleanupScaleTruncate(t, ctx, sqlDB, spec.table)
}

func targetCleanupScaleCreateTarget(t *testing.T, db *gorm.DB, name string) int {
	t.Helper()
	var targetID int
	if err := db.Raw("INSERT INTO target (name, type) VALUES (?, 'domain') RETURNING id", name).Scan(&targetID).Error; err != nil {
		t.Fatalf("create scale Target %q: %v", name, err)
	}
	return targetID
}

func targetCleanupScaleNoiseRows(spec targetCleanupScaleResourceSpec, rows int) int {
	multiplier := spec.noiseMultiplier
	if multiplier <= 0 {
		multiplier = 1
	}
	return rows * multiplier
}

func targetCleanupScaleTombstoneAndTrace(t *testing.T, db *gorm.DB, targetID int) []string {
	t.Helper()
	trace := &targetCleanupScaleSQLTrace{}
	traced := db.Session(&gorm.Session{Logger: trace})
	if _, err := catalogrepository.NewTargetRepository(traced).TombstoneAndEnsureCleanup(context.Background(), targetID); err != nil {
		t.Fatalf("tombstone scale Target %d: %v", targetID, err)
	}
	return trace.statementsSnapshot()
}

func targetCleanupScaleFindJob(t *testing.T, db *gorm.DB, targetID int) cleanupdomain.CleanupJob {
	t.Helper()
	jobs, err := targetcleanuprepository.NewTargetCleanupRepository(db).ListDue(context.Background(), time.Now().UTC().Add(time.Hour), 100)
	if err != nil {
		t.Fatalf("list scale cleanup Job for Target %d: %v", targetID, err)
	}
	for _, job := range jobs {
		if job.TargetID == targetID {
			return job
		}
	}
	t.Fatalf("scale cleanup Job for Target %d not found", targetID)
	return cleanupdomain.CleanupJob{}
}

func targetCleanupScaleInsertRows(t *testing.T, ctx context.Context, db *sql.DB, spec targetCleanupScaleResourceSpec, targetID, rows int) {
	t.Helper()
	var query string
	switch spec.resource {
	case cleanupdomain.AssetResourceSubdomain:
		query = `INSERT INTO subdomain (target_id, dns_name)
			SELECT $1::INTEGER, 'subdomain-' || LPAD(value::text, 7, '0') || '-' || $1::text || '.example'
			FROM generate_series(1, $2::INTEGER) AS value`
	case cleanupdomain.AssetResourceHostPortMapping:
		query = `INSERT INTO host_port_mapping (target_id, host, ip, port)
			SELECT $1::INTEGER, 'host-' || LPAD(value::text, 7, '0') || '-' || $1::text || '.example',
				INET '192.0.2.0' + (value - 1)::BIGINT, 443
			FROM generate_series(1, $2::INTEGER) AS value`
	case cleanupdomain.AssetResourceWebsite:
		query = `INSERT INTO website (target_id, url)
			SELECT $1::INTEGER, 'https://website-' || LPAD(value::text, 7, '0') || '-' || $1::text || '.example/'
			FROM generate_series(1, $2::INTEGER) AS value`
	case cleanupdomain.AssetResourceEndpoint:
		query = `INSERT INTO endpoint (target_id, url)
			SELECT $1::INTEGER, 'https://endpoint-' || LPAD(value::text, 7, '0') || '-' || $1::text || '.example/api'
			FROM generate_series(1, $2::INTEGER) AS value`
	case cleanupdomain.AssetResourceDirectory:
		query = `INSERT INTO directory (target_id, url)
			SELECT $1::INTEGER, 'https://directory-' || LPAD(value::text, 7, '0') || '-' || $1::text || '.example/'
			FROM generate_series(1, $2::INTEGER) AS value`
	case cleanupdomain.AssetResourceScreenshot:
		query = `INSERT INTO screenshot (target_id, url)
			SELECT $1::INTEGER, 'https://screenshot-' || LPAD(value::text, 7, '0') || '-' || $1::text || '.example/'
			FROM generate_series(1, $2::INTEGER) AS value`
	case cleanupdomain.AssetResourceVulnerability:
		query = `INSERT INTO vulnerability (target_id, url, vuln_type, severity, source, reviewed)
			SELECT $1::INTEGER, 'https://vulnerability-' || LPAD(value::text, 7, '0') || '-' || $1::text || '.example/',
				'scale', 'low', 'target-cleanup-scale', (value % 2 = 0)
			FROM generate_series(1, $2::INTEGER) AS value`
	default:
		t.Fatalf("no scale fixture for resource %q", spec.resource)
	}
	if _, err := db.ExecContext(ctx, query, targetID, rows); err != nil {
		t.Fatalf("insert %d %s rows for Target %d: %v", rows, spec.table, targetID, err)
	}
}

func targetCleanupScaleCountRows(t *testing.T, ctx context.Context, db *sql.DB, table string, targetID int) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE target_id = $1", targetID).Scan(&count); err != nil {
		t.Fatalf("count %s rows for Target %d: %v", table, targetID, err)
	}
	return count
}

func targetCleanupScaleTruncate(t *testing.T, ctx context.Context, db *sql.DB, table string) {
	t.Helper()
	// This is isolated fixture teardown only. Production Target cleanup never
	// uses TRUNCATE because the asset tables are shared by many Targets.
	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE ONLY "+table+" RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate isolated %s fixture: %v", table, err)
	}
}

type targetCleanupScaleExplainDocument struct {
	Plan          targetCleanupScaleExplainNode `json:"Plan"`
	PlanningTime  float64                       `json:"Planning Time"`
	ExecutionTime float64                       `json:"Execution Time"`
}

type targetCleanupScaleExplainNode struct {
	NodeType          string                          `json:"Node Type"`
	RelationName      string                          `json:"Relation Name"`
	IndexName         string                          `json:"Index Name"`
	ActualRows        float64                         `json:"Actual Rows"`
	SharedHitBlocks   int64                           `json:"Shared Hit Blocks"`
	SharedReadBlocks  int64                           `json:"Shared Read Blocks"`
	TempReadBlocks    int64                           `json:"Temp Read Blocks"`
	TempWrittenBlocks int64                           `json:"Temp Written Blocks"`
	Plans             []targetCleanupScaleExplainNode `json:"Plans"`
}

type targetCleanupScaleExplainMeasurement struct {
	rows             int64
	planningTime     time.Duration
	executionTime    time.Duration
	sharedHitBlocks  int64
	sharedReadBlocks int64
	tempReadBlocks   int64
	tempWritten      int64
	indexes          []string
	nodes            []string
}

func targetCleanupScaleExplainCandidates(t *testing.T, ctx context.Context, db *sql.DB, spec targetCleanupScaleResourceSpec, targetID int) targetCleanupScaleExplainMeasurement {
	t.Helper()
	var raw string
	query := fmt.Sprintf("EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) SELECT id FROM %s WHERE target_id = $1 ORDER BY id ASC LIMIT $2", spec.table)
	if err := db.QueryRowContext(ctx, query, targetID, targetCleanupScaleBatchSize).Scan(&raw); err != nil {
		t.Fatalf("EXPLAIN %s candidate query: %v", spec.table, err)
	}
	var documents []targetCleanupScaleExplainDocument
	if err := json.Unmarshal([]byte(raw), &documents); err != nil || len(documents) != 1 {
		t.Fatalf("decode %s candidate EXPLAIN: documents=%d err=%v", spec.table, len(documents), err)
	}
	measurement := targetCleanupScaleExplainMeasurement{
		rows:          int64(documents[0].Plan.ActualRows),
		planningTime:  targetCleanupScaleMilliseconds(documents[0].PlanningTime),
		executionTime: targetCleanupScaleMilliseconds(documents[0].ExecutionTime),
	}
	collectTargetCleanupScaleExplainFacts(documents[0].Plan, &measurement)
	sort.Strings(measurement.indexes)
	sort.Strings(measurement.nodes)
	return measurement
}

func collectTargetCleanupScaleExplainFacts(node targetCleanupScaleExplainNode, measurement *targetCleanupScaleExplainMeasurement) {
	measurement.sharedHitBlocks = max(measurement.sharedHitBlocks, node.SharedHitBlocks)
	measurement.sharedReadBlocks = max(measurement.sharedReadBlocks, node.SharedReadBlocks)
	measurement.tempReadBlocks = max(measurement.tempReadBlocks, node.TempReadBlocks)
	measurement.tempWritten = max(measurement.tempWritten, node.TempWrittenBlocks)
	if node.IndexName != "" {
		measurement.indexes = append(measurement.indexes, node.IndexName)
	}
	if node.NodeType != "" {
		measurement.nodes = append(measurement.nodes, node.NodeType)
	}
	for _, child := range node.Plans {
		collectTargetCleanupScaleExplainFacts(child, measurement)
	}
}

func targetCleanupScaleAssertPlan(t *testing.T, label string, spec targetCleanupScaleResourceSpec, measurement targetCleanupScaleExplainMeasurement) {
	t.Helper()
	if measurement.rows != targetCleanupScaleBatchSize {
		t.Fatalf("%s EXPLAIN candidate rows=%d, want %d", label, measurement.rows, targetCleanupScaleBatchSize)
	}
	if !containsString(measurement.indexes, spec.indexName) {
		t.Fatalf("%s EXPLAIN did not use bounded index %q: indexes=%q nodes=%q", label, spec.indexName, measurement.indexes, measurement.nodes)
	}
	if containsString(measurement.nodes, "Seq Scan") || measurement.tempReadBlocks != 0 || measurement.tempWritten != 0 {
		t.Fatalf("%s EXPLAIN materialization was not bounded: nodes=%q temp_read=%d temp_written=%d", label, measurement.nodes, measurement.tempReadBlocks, measurement.tempWritten)
	}
}

type targetCleanupScaleBatchMeasurement struct {
	deleted    int64
	duration   time.Duration
	heapGrowth uint64
}

func targetCleanupScaleMeasureBatch(run func() (int64, error)) (targetCleanupScaleBatchMeasurement, error) {
	measurement := targetCleanupScaleBatchMeasurement{}
	duration, heapGrowth, err := targetCleanupScaleMeasureRun(func() error {
		var err error
		measurement.deleted, err = run()
		return err
	})
	measurement.duration = duration
	measurement.heapGrowth = heapGrowth
	return measurement, err
}

func targetCleanupScaleMilliseconds(value float64) time.Duration {
	return time.Duration(value * float64(time.Millisecond))
}

func targetCleanupScaleMeasureWorker(t *testing.T, ctx context.Context, db *gorm.DB, targetID int) (targetCleanupScaleBatchMeasurement, targetcleanupapp.TargetCleanupReconciliationResult) {
	t.Helper()
	job := targetCleanupScaleFindJob(t, db, targetID)
	service, err := targetcleanupapp.NewTargetCleanupReconciliationService(
		targetcleanuprepository.NewTargetCleanupRepository(db),
		targetCleanupScaleScheduleCleaner{},
		targetCleanupScaleScanCanceller{},
		targetCleanupScalePublisher{},
	)
	if err != nil {
		t.Fatalf("create scale cleanup service: %v", err)
	}
	var result targetcleanupapp.TargetCleanupReconciliationResult
	measurement := targetCleanupScaleBatchMeasurement{}
	measurement.duration, measurement.heapGrowth, err = targetCleanupScaleMeasureRun(func() error {
		result, err = service.Reconcile(ctx, job, targetcleanupapp.TargetCleanupRunOptions{
			AssetBatchSize: targetCleanupScaleBatchSize, MaxAssetBatches: 1, MaxRunDuration: 5 * time.Minute,
		})
		return err
	})
	if err != nil {
		t.Fatalf("run bounded scale cleanup Worker: %v", err)
	}
	measurement.deleted = result.Counts.TotalAssetRows()
	return measurement, result
}

func targetCleanupScaleMeasureRun(run func() error) (time.Duration, uint64, error) {
	runtime.GC()
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	baseline := memory.HeapAlloc
	var mu sync.Mutex
	peak := baseline
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(2 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				runtime.ReadMemStats(&memory)
				mu.Lock()
				peak = max(peak, memory.HeapAlloc)
				mu.Unlock()
			case <-stop:
				return
			}
		}
	}()
	startedAt := time.Now()
	err := run()
	duration := time.Since(startedAt)
	close(stop)
	<-done
	runtime.ReadMemStats(&memory)
	mu.Lock()
	peak = max(peak, memory.HeapAlloc)
	mu.Unlock()
	if peak < baseline {
		return duration, 0, err
	}
	return duration, peak - baseline, err
}

type targetCleanupScaleScheduleCleaner struct{}

func (targetCleanupScaleScheduleCleaner) DeleteTargetScopedSchedules(context.Context, int) (int, error) {
	return 0, nil
}

type targetCleanupScaleScanCanceller struct{}

func (targetCleanupScaleScanCanceller) CancelNextActiveScan(context.Context, int, time.Time) (*targetcleanupapp.TargetScanCancellation, error) {
	return nil, nil
}

type targetCleanupScalePublisher struct{}

func (targetCleanupScalePublisher) TrySendTaskCancel(int, int, int) bool { return true }

type targetCleanupScaleSQLTrace struct {
	mu         sync.Mutex
	statements []string
}

func (trace *targetCleanupScaleSQLTrace) LogMode(gormlogger.LogLevel) gormlogger.Interface {
	return trace
}
func (trace *targetCleanupScaleSQLTrace) Info(context.Context, string, ...interface{})  {}
func (trace *targetCleanupScaleSQLTrace) Warn(context.Context, string, ...interface{})  {}
func (trace *targetCleanupScaleSQLTrace) Error(context.Context, string, ...interface{}) {}
func (trace *targetCleanupScaleSQLTrace) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	statement, _ := fc()
	trace.mu.Lock()
	trace.statements = append(trace.statements, statement)
	trace.mu.Unlock()
}

func (trace *targetCleanupScaleSQLTrace) statementsSnapshot() []string {
	trace.mu.Lock()
	defer trace.mu.Unlock()
	return append([]string(nil), trace.statements...)
}

var targetCleanupScaleSQLLiteral = regexp.MustCompile(`'([^']|'')*'|\b[0-9]+\b`)

func targetCleanupScaleTraceSignature(statements []string) []string {
	signature := make([]string, 0, len(statements))
	for _, statement := range statements {
		normalized := strings.Join(strings.Fields(strings.ToLower(statement)), " ")
		signature = append(signature, targetCleanupScaleSQLLiteral.ReplaceAllString(normalized, "?"))
	}
	return signature
}

func targetCleanupScaleAssertSynchronousDeleteTrace(t *testing.T, statements []string) {
	t.Helper()
	if len(statements) == 0 {
		t.Fatal("synchronous Target DELETE produced no SQL trace")
	}
	for _, statement := range statements {
		lower := strings.ToLower(statement)
		forbidden := []string{
			"subdomain", "host_port_mapping", "website", "endpoint", "directory", "screenshot", "vulnerability",
			"scheduled_scan", "organization_target", "blacklist_policy", "scan_task", "task_progress_log", "_snapshot",
			" delete ", " truncate ", " count(",
		}
		for _, fragment := range forbidden {
			if strings.Contains(lower, fragment) {
				t.Fatalf("synchronous Target DELETE touched related work %q in %q", fragment, statement)
			}
		}
	}
	if strings.Join(targetCleanupScaleTraceSignature(statements), " | ") == "" {
		t.Fatal("synchronous Target DELETE trace normalized to empty")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
