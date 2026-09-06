package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	serverdatabase "github.com/yyhuni/lunafox/server/internal/database"
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	searchhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/search"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	globalAssetSearchScalePostgresDSNEnv         = "LUNAFOX_GLOBAL_ASSET_SEARCH_SCALE_POSTGRES_DSN"
	globalAssetSearchScaleEvidenceDirEnv         = "LUNAFOX_GLOBAL_ASSET_SEARCH_SCALE_EVIDENCE_DIR"
	globalAssetSearchScaleDatabase               = "lunafox_global_asset_search_scale"
	globalAssetSearchScaleRowsPerAsset           = 1_000_000
	globalAssetSearchScaleProbeRows              = 100
	globalAssetSearchScaleWideRows               = 100
	globalAssetSearchScaleTombstoneRows          = 100
	globalAssetSearchScaleSamples                = 100
	globalAssetSearchScaleWarmupRequests         = 5
	globalAssetSearchScaleNormalPageSize         = 10
	globalAssetSearchScaleWidePageSize           = 100
	globalAssetSearchScaleNormalP99Limit         = 2 * time.Second
	globalAssetSearchScaleWidePageLimit          = 10 * time.Second
	globalAssetSearchScaleWideBodyBytes          = 64 << 10
	globalAssetSearchScaleWideHeaderBytes        = 8 << 10
	globalAssetSearchScaleMinimumWidePayloadSize = 7 << 20
)

var globalAssetSearchScaleSpecs = []globalAssetSearchScaleAssetSpec{
	{assetType: assetapp.GlobalAssetSearchAssetTypeWebsite, table: "website"},
	{assetType: assetapp.GlobalAssetSearchAssetTypeEndpoint, table: "endpoint"},
}

type globalAssetSearchScaleAssetSpec struct {
	assetType assetapp.GlobalAssetSearchAssetType
	table     string
}

type globalAssetSearchScaleFixture struct {
	assetType  assetapp.GlobalAssetSearchAssetType
	table      string
	probeURL   string
	probeHost  string
	probeTitle string
}

type globalAssetSearchScaleQueryFamily struct {
	name                 string
	query                string
	where                string
	args                 []any
	expectedResultCount  int
	acceptableIndexNames []string
}

type globalAssetSearchScaleEvidence struct {
	Schema            string                                   `json:"schema"`
	Database          string                                   `json:"database"`
	PostgreSQLVersion string                                   `json:"postgresqlVersion"`
	GoVersion         string                                   `json:"goVersion"`
	CPUs              int                                      `json:"cpus"`
	Fixture           []globalAssetSearchScaleFixtureEvidence  `json:"fixture"`
	Latency           []globalAssetSearchScaleLatencyEvidence  `json:"latency"`
	WidePages         []globalAssetSearchScaleWidePageEvidence `json:"widePages"`
	Plans             []globalAssetSearchScalePlanEvidence     `json:"plans"`
}

type globalAssetSearchScaleFixtureEvidence struct {
	AssetType     string `json:"assetType"`
	Table         string `json:"table"`
	Rows          int    `json:"rows"`
	ActiveRows    int    `json:"activeRows"`
	TombstoneRows int    `json:"tombstoneRows"`
	WideRows      int    `json:"wideRows"`
}

type globalAssetSearchScaleLatencyEvidence struct {
	AssetType    string `json:"assetType"`
	Family       string `json:"family"`
	Samples      int    `json:"samples"`
	PayloadBytes int    `json:"payloadBytes"`
	P50          string `json:"p50"`
	P95          string `json:"p95"`
	P99          string `json:"p99"`
	Max          string `json:"max"`
}

type globalAssetSearchScaleWidePageEvidence struct {
	AssetType         string `json:"assetType"`
	Rows              int    `json:"rows"`
	ResponseBytes     int    `json:"responseBytes"`
	ResponseBodyBytes int    `json:"responseBodyBytes"`
	ResponseHeadBytes int    `json:"responseHeaderBytes"`
	TableStorageBytes int64  `json:"tableStorageBytes"`
	TotalStorageBytes int64  `json:"totalStorageBytes"`
	Duration          string `json:"duration"`
}

type globalAssetSearchScalePlanEvidence struct {
	AssetType    string   `json:"assetType"`
	Family       string   `json:"family"`
	File         string   `json:"file"`
	Indexes      []string `json:"indexes"`
	TopLevelRows int      `json:"topLevelRows"`
	TempRead     int64    `json:"tempReadBlocks"`
	TempWritten  int64    `json:"tempWrittenBlocks"`
}

// TestGlobalAssetSearchScalePostgres is intentionally opt-in. The release
// gate provisions a fresh PostgreSQL 18 database; ordinary Go tests must not
// create or retain this million-row fixture.
func TestGlobalAssetSearchScalePostgres(t *testing.T) {
	db, sqlDB, evidenceDir, evidence := openGlobalAssetSearchScalePostgres(t)
	t.Cleanup(func() { writeGlobalAssetSearchScaleEvidence(t, evidenceDir, evidence) })

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	fixtures := make([]globalAssetSearchScaleFixture, 0, len(globalAssetSearchScaleSpecs))
	for _, spec := range globalAssetSearchScaleSpecs {
		fixture := seedGlobalAssetSearchScaleFixture(t, ctx, sqlDB, spec)
		fixtures = append(fixtures, fixture)
		evidence.Fixture = append(evidence.Fixture, globalAssetSearchScaleFixtureEvidence{
			AssetType:     string(fixture.assetType),
			Table:         fixture.table,
			Rows:          globalAssetSearchScaleRowsPerAsset,
			ActiveRows:    globalAssetSearchScaleRowsPerAsset - globalAssetSearchScaleTombstoneRows,
			TombstoneRows: globalAssetSearchScaleTombstoneRows,
			WideRows:      globalAssetSearchScaleWideRows,
		})
	}
	assertGlobalAssetSearchScaleIndexInventory(t, ctx, sqlDB)

	gin.SetMode(gin.TestMode)
	service := assetapp.NewGlobalAssetSearchService(assetrepo.NewWebsiteRepository(db), assetrepo.NewEndpointRepository(db))
	handler := searchhandler.NewGlobalAssetSearchHandler(service)
	router := gin.New()
	router.GET("/v1/assets:search", handler.Search)

	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(string(fixture.assetType), func(t *testing.T) {
			for _, family := range globalAssetSearchScaleQueryFamilies(fixture) {
				family := family
				t.Run(family.name, func(t *testing.T) {
					latency := measureGlobalAssetSearchScaleQuery(t, router, fixture, family)
					evidence.Latency = append(evidence.Latency, latency)
					plan := collectGlobalAssetSearchScalePlan(t, ctx, sqlDB, evidenceDir, fixture, family)
					assertGlobalAssetSearchScalePlan(t, fixture, family, plan)
					evidence.Plans = append(evidence.Plans, plan.evidence)
				})
			}
			evidence.WidePages = append(evidence.WidePages, measureGlobalAssetSearchScaleWidePage(t, ctx, sqlDB, router, fixture))
		})
	}
}

func openGlobalAssetSearchScalePostgres(t *testing.T) (*gorm.DB, *sql.DB, string, *globalAssetSearchScaleEvidence) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv(globalAssetSearchScalePostgresDSNEnv))
	if dsn == "" {
		t.Skip("set " + globalAssetSearchScalePostgresDSNEnv + " to run the PostgreSQL 18 global asset search scale gate")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("open global asset search scale PostgreSQL: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open global asset search scale SQL handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(16)
	sqlDB.SetMaxIdleConns(16)
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var databaseName string
	if err := sqlDB.QueryRowContext(ctx, "SELECT current_database()").Scan(&databaseName); err != nil {
		t.Fatalf("read global asset search scale database name: %v", err)
	}
	if databaseName != globalAssetSearchScaleDatabase {
		t.Fatalf("refusing destructive global asset search scale database %q; want %q", databaseName, globalAssetSearchScaleDatabase)
	}
	var publicTables int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'`).Scan(&publicTables); err != nil {
		t.Fatalf("inspect global asset search scale database: %v", err)
	}
	if publicTables != 0 {
		t.Fatalf("global asset search scale database must start empty, found %d public tables", publicTables)
	}

	previousFS, previousPath := serverdatabase.MigrationsFS, serverdatabase.MigrationsPath
	serverdatabase.MigrationsFS = migrationsFS
	serverdatabase.MigrationsPath = "migrations"
	t.Cleanup(func() {
		serverdatabase.MigrationsFS = previousFS
		serverdatabase.MigrationsPath = previousPath
	})
	if err := serverdatabase.RunMigrations(sqlDB); err != nil {
		t.Fatalf("apply empty baseline for global asset search scale: %v", err)
	}
	t.Cleanup(func() {
		if err := serverdatabase.MigrateDown(sqlDB); err != nil {
			t.Errorf("tear down global asset search scale baseline: %v", err)
		}
	})

	evidenceDir := strings.TrimSpace(os.Getenv(globalAssetSearchScaleEvidenceDirEnv))
	if evidenceDir == "" {
		evidenceDir = t.TempDir()
	}
	if err := os.MkdirAll(filepath.Join(evidenceDir, "plans"), 0o755); err != nil {
		t.Fatalf("create global asset search scale evidence directory: %v", err)
	}
	var postgresVersion string
	if err := sqlDB.QueryRowContext(ctx, "SHOW server_version").Scan(&postgresVersion); err != nil {
		t.Fatalf("read PostgreSQL scale version: %v", err)
	}
	evidence := &globalAssetSearchScaleEvidence{
		Schema:            "lunafox.global-asset-search-scale.v1",
		Database:          databaseName,
		PostgreSQLVersion: postgresVersion,
		GoVersion:         runtime.Version(),
		CPUs:              runtime.NumCPU(),
	}
	return db, sqlDB, evidenceDir, evidence
}

func seedGlobalAssetSearchScaleFixture(t *testing.T, ctx context.Context, db *sql.DB, spec globalAssetSearchScaleAssetSpec) globalAssetSearchScaleFixture {
	t.Helper()
	activeTargetID := createGlobalAssetSearchScaleTarget(t, ctx, db, spec.table+"-global-search-scale-active", nil)
	deletedAt := time.Date(2026, 8, 7, 8, 0, 0, 0, time.UTC)
	tombstoneTargetID := createGlobalAssetSearchScaleTarget(t, ctx, db, spec.table+"-global-search-scale-tombstone", &deletedAt)

	noiseRows := globalAssetSearchScaleRowsPerAsset - globalAssetSearchScaleProbeRows - globalAssetSearchScaleWideRows - globalAssetSearchScaleTombstoneRows
	insertGlobalAssetSearchScaleNoiseRows(t, ctx, db, spec, activeTargetID, noiseRows)
	fixture := insertGlobalAssetSearchScaleProbeRows(t, ctx, db, spec, activeTargetID)
	insertGlobalAssetSearchScaleWideRows(t, ctx, db, spec, activeTargetID)
	insertGlobalAssetSearchScaleTombstoneRows(t, ctx, db, spec, tombstoneTargetID)
	if _, err := db.ExecContext(ctx, "ANALYZE "+spec.table); err != nil {
		t.Fatalf("analyze %s global search scale fixture: %v", spec.table, err)
	}

	var totalRows, activeRows, tombstoneRows int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+spec.table).Scan(&totalRows); err != nil {
		t.Fatalf("count %s global search scale rows: %v", spec.table, err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+spec.table+" WHERE target_id = $1", activeTargetID).Scan(&activeRows); err != nil {
		t.Fatalf("count active %s global search scale rows: %v", spec.table, err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+spec.table+" WHERE target_id = $1", tombstoneTargetID).Scan(&tombstoneRows); err != nil {
		t.Fatalf("count tombstone %s global search scale rows: %v", spec.table, err)
	}
	if totalRows != globalAssetSearchScaleRowsPerAsset || activeRows+tombstoneRows != totalRows || tombstoneRows != globalAssetSearchScaleTombstoneRows {
		t.Fatalf("unexpected %s scale fixture counts: total=%d active=%d tombstone=%d", spec.table, totalRows, activeRows, tombstoneRows)
	}
	return fixture
}

func createGlobalAssetSearchScaleTarget(t *testing.T, ctx context.Context, db *sql.DB, name string, deletedAt *time.Time) int {
	t.Helper()
	var id int
	if err := db.QueryRowContext(ctx, `INSERT INTO target (name, type, deleted_at) VALUES ($1, 'domain', $2) RETURNING id`, name, deletedAt).Scan(&id); err != nil {
		t.Fatalf("create global search scale Target %q: %v", name, err)
	}
	return id
}

func insertGlobalAssetSearchScaleNoiseRows(t *testing.T, ctx context.Context, db *sql.DB, spec globalAssetSearchScaleAssetSpec, targetID, rows int) {
	t.Helper()
	query := fmt.Sprintf(`
		INSERT INTO %s (target_id, url, host, title, status_code, tech, created_at)
		SELECT $1::INTEGER,
			'https://%s-noise-' || LPAD(value::text, 7, '0') || '.example/asset/' || value::text,
			'%s-noise-' || LPAD(value::text, 7, '0') || '.example',
			'Noise asset ' || value::text,
			200,
			ARRAY['noise-tech']::varchar(100)[],
			TIMESTAMPTZ '2026-08-07 09:00:00+00' - (value * INTERVAL '1 second')
		FROM generate_series(1, $2::INTEGER) AS value`, spec.table, spec.table, spec.table)
	if _, err := db.ExecContext(ctx, query, targetID, rows); err != nil {
		t.Fatalf("insert %d %s global search noise rows: %v", rows, spec.table, err)
	}
}

func insertGlobalAssetSearchScaleProbeRows(t *testing.T, ctx context.Context, db *sql.DB, spec globalAssetSearchScaleAssetSpec, targetID int) globalAssetSearchScaleFixture {
	t.Helper()
	query := fmt.Sprintf(`
		INSERT INTO %s (target_id, url, host, title, status_code, tech, created_at)
		SELECT $1::INTEGER,
			'https://%s-probe-global-search-' || LPAD(value::text, 6, '0') || '.example/record/' || LPAD(value::text, 6, '0'),
			'%s-probe-global-search-' || LPAD(value::text, 6, '0') || '.example',
			'Global Search Probe ' || LPAD(value::text, 6, '0'),
			201,
			ARRAY['global-search-probe-tech']::varchar(100)[],
			TIMESTAMPTZ '2026-08-07 10:00:00+00' - (value * INTERVAL '1 second')
		FROM generate_series(1, %d) AS value`, spec.table, spec.table, spec.table, globalAssetSearchScaleProbeRows)
	if _, err := db.ExecContext(ctx, query, targetID); err != nil {
		t.Fatalf("insert %s global search probe rows: %v", spec.table, err)
	}
	return globalAssetSearchScaleFixture{
		assetType:  spec.assetType,
		table:      spec.table,
		probeURL:   "https://" + spec.table + "-probe-global-search-000001.example/record/000001",
		probeHost:  spec.table + "-probe-global-search-000001.example",
		probeTitle: "Global Search Probe 000001",
	}
}

func insertGlobalAssetSearchScaleWideRows(t *testing.T, ctx context.Context, db *sql.DB, spec globalAssetSearchScaleAssetSpec, targetID int) {
	t.Helper()
	query := "INSERT INTO " + spec.table + " (target_id, url, host, title, status_code, tech, response_headers, response_body, created_at) VALUES ($1, $2, $3, $4, 202, $5::varchar(100)[], $6, $7, $8)"
	for value := 1; value <= globalAssetSearchScaleWideRows; value++ {
		body := globalAssetSearchScalePayload(spec.table+"-body", value, globalAssetSearchScaleWideBodyBytes)
		headers := globalAssetSearchScalePayload(spec.table+"-headers", value, globalAssetSearchScaleWideHeaderBytes)
		createdAt := time.Date(2026, 8, 7, 11, 0, 0, 0, time.UTC).Add(-time.Duration(value) * time.Second)
		if _, err := db.ExecContext(ctx, query,
			targetID,
			fmt.Sprintf("https://%s-wide-global-search-%06d.example/wide/%06d", spec.table, value, value),
			fmt.Sprintf("%s-wide-global-search-%06d.example", spec.table, value),
			fmt.Sprintf("Global Search Wide %06d", value),
			pq.Array([]string{"global-search-wide-tech"}),
			headers,
			body,
			createdAt,
		); err != nil {
			t.Fatalf("insert %s global search wide row %d: %v", spec.table, value, err)
		}
	}
}

func globalAssetSearchScalePayload(prefix string, value, size int) string {
	var builder strings.Builder
	builder.Grow(size)
	for fragment := 0; builder.Len() < size; fragment++ {
		digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d", prefix, value, fragment)))
		builder.WriteString(hex.EncodeToString(digest[:]))
	}
	return builder.String()[:size]
}

func insertGlobalAssetSearchScaleTombstoneRows(t *testing.T, ctx context.Context, db *sql.DB, spec globalAssetSearchScaleAssetSpec, targetID int) {
	t.Helper()
	query := fmt.Sprintf(`
		INSERT INTO %s (target_id, url, host, title, status_code, tech, created_at)
		SELECT $1::INTEGER,
			'https://%s-probe-global-search-tombstone-' || LPAD(value::text, 6, '0') || '.example/record/' || LPAD(value::text, 6, '0'),
			'%s-probe-global-search-tombstone-' || LPAD(value::text, 6, '0') || '.example',
			'Global Search Tombstone ' || LPAD(value::text, 6, '0'),
			201,
			ARRAY['global-search-probe-tech']::varchar(100)[],
			TIMESTAMPTZ '2026-08-07 12:00:00+00' - (value * INTERVAL '1 second')
		FROM generate_series(1, %d) AS value`, spec.table, spec.table, spec.table, globalAssetSearchScaleTombstoneRows)
	if _, err := db.ExecContext(ctx, query, targetID); err != nil {
		t.Fatalf("insert %s global search tombstone rows: %v", spec.table, err)
	}
}

func assertGlobalAssetSearchScaleIndexInventory(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	for _, spec := range globalAssetSearchScaleSpecs {
		rows, err := db.QueryContext(ctx, `SELECT indexname, indexdef FROM pg_indexes WHERE schemaname = 'public' AND tablename = $1`, spec.table)
		if err != nil {
			t.Fatalf("list %s scale indexes: %v", spec.table, err)
		}
		definitions := map[string]string{}
		for rows.Next() {
			var name, definition string
			if err := rows.Scan(&name, &definition); err != nil {
				_ = rows.Close()
				t.Fatalf("scan %s scale index: %v", spec.table, err)
			}
			definitions[name] = strings.ToLower(definition)
		}
		if err := rows.Close(); err != nil {
			t.Fatalf("close %s scale index rows: %v", spec.table, err)
		}
		for name, required := range map[string]string{
			"idx_" + spec.table + "_url_trgm":    "using gin (url gin_trgm_ops)",
			"idx_" + spec.table + "_host_trgm":   "using gin (host gin_trgm_ops)",
			"idx_" + spec.table + "_title_trgm":  "using gin (title gin_trgm_ops)",
			"idx_" + spec.table + "_tech_gin":    "using gin (tech)",
			"idx_" + spec.table + "_url":         "(url)",
			"idx_" + spec.table + "_host":        "(host)",
			"idx_" + spec.table + "_title":       "(title)",
			"idx_" + spec.table + "_status_code": "(status_code)",
		} {
			definition, ok := definitions[name]
			if !ok || !strings.Contains(definition, required) {
				t.Fatalf("%s must retain scale-search index %q (%q), got %q", spec.table, name, required, definition)
			}
		}
		for name, definition := range definitions {
			if strings.Contains(definition, "using gin") && (strings.Contains(definition, "response_body") || strings.Contains(definition, "response_headers")) {
				t.Fatalf("%s must not create response-evidence GIN index %q: %s", spec.table, name, definition)
			}
		}
	}
}

func globalAssetSearchScaleQueryFamilies(fixture globalAssetSearchScaleFixture) []globalAssetSearchScaleQueryFamily {
	contains := globalAssetSearchScaleContainsPattern("probe-global-search")
	return []globalAssetSearchScaleQueryFamily{
		{
			name:                 "plain-url",
			query:                fixture.probeURL,
			where:                "asset.url = $1",
			args:                 []any{fixture.probeURL},
			expectedResultCount:  1,
			acceptableIndexNames: []string{"idx_" + fixture.table + "_url", "unique_" + fixture.table + "_url_target"},
		},
		{
			name:                 "url-exact",
			query:                `url=="` + fixture.probeURL + `"`,
			where:                "asset.url = $1",
			args:                 []any{fixture.probeURL},
			expectedResultCount:  1,
			acceptableIndexNames: []string{"idx_" + fixture.table + "_url", "unique_" + fixture.table + "_url_target"},
		},
		{
			name:                 "host-contains",
			query:                `host="probe-global-search"`,
			where:                "asset.host ILIKE $1 ESCAPE '\\'",
			args:                 []any{contains},
			expectedResultCount:  globalAssetSearchScaleNormalPageSize,
			acceptableIndexNames: []string{"idx_" + fixture.table + "_host_trgm"},
		},
		{
			name:                 "host-exact",
			query:                `host=="` + fixture.probeHost + `"`,
			where:                "asset.host = $1",
			args:                 []any{fixture.probeHost},
			expectedResultCount:  1,
			acceptableIndexNames: []string{"idx_" + fixture.table + "_host"},
		},
		{
			name:                 "title-contains",
			query:                `title="Global Search Probe"`,
			where:                "asset.title ILIKE $1 ESCAPE '\\'",
			args:                 []any{globalAssetSearchScaleContainsPattern("Global Search Probe")},
			expectedResultCount:  globalAssetSearchScaleNormalPageSize,
			acceptableIndexNames: []string{"idx_" + fixture.table + "_title_trgm"},
		},
		{
			name:                 "title-exact",
			query:                `title=="` + fixture.probeTitle + `"`,
			where:                "asset.title = $1",
			args:                 []any{fixture.probeTitle},
			expectedResultCount:  1,
			acceptableIndexNames: []string{"idx_" + fixture.table + "_title"},
		},
		{
			name:                 "status-code",
			query:                `statusCode="201"`,
			where:                "asset.status_code = $1",
			args:                 []any{201},
			expectedResultCount:  globalAssetSearchScaleNormalPageSize,
			acceptableIndexNames: []string{"idx_" + fixture.table + "_status_code"},
		},
		{
			name:                 "tech",
			query:                `tech="global-search-probe-tech"`,
			where:                "asset.tech @> $1::varchar(100)[]",
			args:                 []any{pq.Array([]string{"global-search-probe-tech"})},
			expectedResultCount:  globalAssetSearchScaleNormalPageSize,
			acceptableIndexNames: []string{"idx_" + fixture.table + "_tech_gin"},
		},
		{
			name:                 "and",
			query:                `host="probe-global-search" && tech="global-search-probe-tech"`,
			where:                "asset.host ILIKE $1 ESCAPE '\\' AND asset.tech @> $2::varchar(100)[]",
			args:                 []any{contains, pq.Array([]string{"global-search-probe-tech"})},
			expectedResultCount:  globalAssetSearchScaleNormalPageSize,
			acceptableIndexNames: []string{"idx_" + fixture.table + "_host_trgm", "idx_" + fixture.table + "_tech_gin"},
		},
	}
}

func globalAssetSearchScaleContainsPattern(value string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
	return "%" + escaped + "%"
}

func measureGlobalAssetSearchScaleQuery(t *testing.T, router http.Handler, fixture globalAssetSearchScaleFixture, family globalAssetSearchScaleQueryFamily) globalAssetSearchScaleLatencyEvidence {
	t.Helper()
	for iteration := 0; iteration < globalAssetSearchScaleWarmupRequests; iteration++ {
		globalAssetSearchScaleRunHandlerRequest(t, router, fixture.assetType, family.query, globalAssetSearchScaleNormalPageSize, family.expectedResultCount)
	}
	durations := make([]time.Duration, 0, globalAssetSearchScaleSamples)
	payloadBytes := 0
	for iteration := 0; iteration < globalAssetSearchScaleSamples; iteration++ {
		duration, bytes := globalAssetSearchScaleRunHandlerRequest(t, router, fixture.assetType, family.query, globalAssetSearchScaleNormalPageSize, family.expectedResultCount)
		durations = append(durations, duration)
		payloadBytes = max(payloadBytes, bytes)
	}
	sort.Slice(durations, func(left, right int) bool { return durations[left] < durations[right] })
	p99 := globalAssetSearchScalePercentile(durations, 0.99)
	if p99 > globalAssetSearchScaleNormalP99Limit {
		t.Fatalf("%s %s handler-to-complete-JSON p99=%s exceeds %s", fixture.assetType, family.name, p99, globalAssetSearchScaleNormalP99Limit)
	}
	return globalAssetSearchScaleLatencyEvidence{
		AssetType:    string(fixture.assetType),
		Family:       family.name,
		Samples:      len(durations),
		PayloadBytes: payloadBytes,
		P50:          globalAssetSearchScalePercentile(durations, 0.50).String(),
		P95:          globalAssetSearchScalePercentile(durations, 0.95).String(),
		P99:          p99.String(),
		Max:          durations[len(durations)-1].String(),
	}
}

func globalAssetSearchScalePercentile(sorted []time.Duration, percentile float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func globalAssetSearchScaleRunHandlerRequest(t *testing.T, router http.Handler, assetType assetapp.GlobalAssetSearchAssetType, query string, pageSize, expectedResultCount int) (time.Duration, int) {
	t.Helper()
	values := url.Values{}
	values.Set("q", query)
	values.Set("assetType", string(assetType))
	values.Set("pageSize", fmt.Sprintf("%d", pageSize))
	request := httptest.NewRequest(http.MethodGet, "/v1/assets:search?"+values.Encode(), nil)
	recorder := httptest.NewRecorder()
	startedAt := time.Now()
	router.ServeHTTP(recorder, request)
	duration := time.Since(startedAt)
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s handler query %q returned %d: %s", assetType, query, recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.Bytes()
	if !json.Valid(body) {
		t.Fatalf("%s handler query %q returned invalid JSON", assetType, query)
	}
	var response struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode %s handler query %q: %v", assetType, query, err)
	}
	if len(response.Results) != expectedResultCount {
		t.Fatalf("%s handler query %q returned %d result(s), want %d", assetType, query, len(response.Results), expectedResultCount)
	}
	return duration, len(body)
}

type globalAssetSearchScalePlanDocument struct {
	Plan          globalAssetSearchScalePlanNode `json:"Plan"`
	PlanningTime  float64                        `json:"Planning Time"`
	ExecutionTime float64                        `json:"Execution Time"`
}

type globalAssetSearchScalePlanNode struct {
	NodeType          string                           `json:"Node Type"`
	RelationName      string                           `json:"Relation Name"`
	IndexName         string                           `json:"Index Name"`
	ActualRows        float64                          `json:"Actual Rows"`
	TempReadBlocks    int64                            `json:"Temp Read Blocks"`
	TempWrittenBlocks int64                            `json:"Temp Written Blocks"`
	Plans             []globalAssetSearchScalePlanNode `json:"Plans"`
}

type globalAssetSearchScaleCollectedPlan struct {
	evidence     globalAssetSearchScalePlanEvidence
	assetSeqScan bool
}

func collectGlobalAssetSearchScalePlan(t *testing.T, ctx context.Context, db *sql.DB, evidenceDir string, fixture globalAssetSearchScaleFixture, family globalAssetSearchScaleQueryFamily) globalAssetSearchScaleCollectedPlan {
	t.Helper()
	query := "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) SELECT asset.* FROM " + fixture.table + " AS asset JOIN target AS active_target ON active_target.id = asset.target_id AND active_target.deleted_at IS NULL WHERE " + family.where + " ORDER BY asset.created_at DESC, asset.id DESC LIMIT $" + fmt.Sprintf("%d", len(family.args)+1)
	args := append(append([]any(nil), family.args...), globalAssetSearchScaleNormalPageSize+1)
	var raw string
	if err := db.QueryRowContext(ctx, query, args...).Scan(&raw); err != nil {
		t.Fatalf("collect %s %s JSON EXPLAIN: %v", fixture.assetType, family.name, err)
	}
	var documents []globalAssetSearchScalePlanDocument
	if err := json.Unmarshal([]byte(raw), &documents); err != nil || len(documents) != 1 {
		t.Fatalf("decode %s %s JSON EXPLAIN: documents=%d err=%v", fixture.assetType, family.name, len(documents), err)
	}
	collected := globalAssetSearchScaleCollectedPlan{
		evidence: globalAssetSearchScalePlanEvidence{
			AssetType:    string(fixture.assetType),
			Family:       family.name,
			File:         filepath.ToSlash(filepath.Join("plans", string(fixture.assetType)+"-"+family.name+".json")),
			TopLevelRows: int(documents[0].Plan.ActualRows),
		},
	}
	collectGlobalAssetSearchScalePlanFacts(documents[0].Plan, fixture.table, &collected)
	sort.Strings(collected.evidence.Indexes)
	planPath := filepath.Join(evidenceDir, collected.evidence.File)
	if err := os.WriteFile(planPath, append([]byte(raw), '\n'), 0o644); err != nil {
		t.Fatalf("write %s %s raw JSON EXPLAIN: %v", fixture.assetType, family.name, err)
	}
	return collected
}

func collectGlobalAssetSearchScalePlanFacts(node globalAssetSearchScalePlanNode, assetTable string, collected *globalAssetSearchScaleCollectedPlan) {
	collected.evidence.TempRead += node.TempReadBlocks
	collected.evidence.TempWritten += node.TempWrittenBlocks
	if node.IndexName != "" {
		collected.evidence.Indexes = append(collected.evidence.Indexes, node.IndexName)
	}
	if node.RelationName == assetTable && node.NodeType == "Seq Scan" {
		collected.assetSeqScan = true
	}
	for _, child := range node.Plans {
		collectGlobalAssetSearchScalePlanFacts(child, assetTable, collected)
	}
}

func assertGlobalAssetSearchScalePlan(t *testing.T, fixture globalAssetSearchScaleFixture, family globalAssetSearchScaleQueryFamily, plan globalAssetSearchScaleCollectedPlan) {
	t.Helper()
	if plan.evidence.TopLevelRows > globalAssetSearchScaleNormalPageSize+1 {
		t.Fatalf("%s %s EXPLAIN top-level rows=%d exceeds pageSize+1=%d", fixture.assetType, family.name, plan.evidence.TopLevelRows, globalAssetSearchScaleNormalPageSize+1)
	}
	if plan.assetSeqScan {
		t.Fatalf("%s %s EXPLAIN used an asset-table Seq Scan: indexes=%q", fixture.assetType, family.name, plan.evidence.Indexes)
	}
	if plan.evidence.TempRead != 0 || plan.evidence.TempWritten != 0 {
		t.Fatalf("%s %s EXPLAIN used temporary I/O: temp_read=%d temp_written=%d", fixture.assetType, family.name, plan.evidence.TempRead, plan.evidence.TempWritten)
	}
	for _, expected := range family.acceptableIndexNames {
		if containsGlobalAssetSearchScaleString(plan.evidence.Indexes, expected) {
			return
		}
	}
	t.Fatalf("%s %s EXPLAIN did not use a relevant index %q: got %q", fixture.assetType, family.name, family.acceptableIndexNames, plan.evidence.Indexes)
}

func containsGlobalAssetSearchScaleString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func measureGlobalAssetSearchScaleWidePage(t *testing.T, ctx context.Context, db *sql.DB, router http.Handler, fixture globalAssetSearchScaleFixture) globalAssetSearchScaleWidePageEvidence {
	t.Helper()
	values := url.Values{}
	// Plain q is an exact observed-URL lookup. This structured predicate selects
	// every wide fixture without changing that public search contract.
	values.Set("q", `statusCode="202"`)
	values.Set("assetType", string(fixture.assetType))
	values.Set("pageSize", fmt.Sprintf("%d", globalAssetSearchScaleWidePageSize))
	request := httptest.NewRequest(http.MethodGet, "/v1/assets:search?"+values.Encode(), nil)
	recorder := httptest.NewRecorder()
	startedAt := time.Now()
	router.ServeHTTP(recorder, request)
	duration := time.Since(startedAt)
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s wide page returned %d: %s", fixture.assetType, recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.Bytes()
	var response struct {
		Results []struct {
			ResponseBody    string `json:"responseBody"`
			ResponseHeaders string `json:"responseHeaders"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode %s wide page: %v", fixture.assetType, err)
	}
	if len(response.Results) != globalAssetSearchScaleWideRows {
		t.Fatalf("%s wide page results=%d, want %d", fixture.assetType, len(response.Results), globalAssetSearchScaleWideRows)
	}
	var responseBodyBytes, responseHeaderBytes int
	for _, item := range response.Results {
		if len(item.ResponseBody) < globalAssetSearchScaleWideBodyBytes || len(item.ResponseHeaders) < globalAssetSearchScaleWideHeaderBytes {
			t.Fatalf("%s wide page did not preserve complete evidence: body=%d headers=%d", fixture.assetType, len(item.ResponseBody), len(item.ResponseHeaders))
		}
		responseBodyBytes += len(item.ResponseBody)
		responseHeaderBytes += len(item.ResponseHeaders)
	}
	if len(body) < globalAssetSearchScaleMinimumWidePayloadSize {
		t.Fatalf("%s wide page response bytes=%d, want at least %d", fixture.assetType, len(body), globalAssetSearchScaleMinimumWidePayloadSize)
	}
	if duration > globalAssetSearchScaleWidePageLimit {
		t.Fatalf("%s wide page handler-to-complete-JSON duration=%s exceeds %s", fixture.assetType, duration, globalAssetSearchScaleWidePageLimit)
	}
	var tableStorage, totalStorage int64
	if err := db.QueryRowContext(ctx, "SELECT pg_relation_size($1::regclass), pg_total_relation_size($1::regclass)", fixture.table).Scan(&tableStorage, &totalStorage); err != nil {
		t.Fatalf("read %s wide page storage: %v", fixture.assetType, err)
	}
	return globalAssetSearchScaleWidePageEvidence{
		AssetType:         string(fixture.assetType),
		Rows:              len(response.Results),
		ResponseBytes:     len(body),
		ResponseBodyBytes: responseBodyBytes,
		ResponseHeadBytes: responseHeaderBytes,
		TableStorageBytes: tableStorage,
		TotalStorageBytes: totalStorage,
		Duration:          duration.String(),
	}
}

func writeGlobalAssetSearchScaleEvidence(t *testing.T, evidenceDir string, evidence *globalAssetSearchScaleEvidence) {
	t.Helper()
	if evidence == nil || evidenceDir == "" {
		return
	}
	encoded, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		t.Errorf("encode global asset search scale evidence: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(evidenceDir, "summary.json"), append(encoded, '\n'), 0o644); err != nil {
		t.Errorf("write global asset search scale evidence: %v", err)
	}
}
