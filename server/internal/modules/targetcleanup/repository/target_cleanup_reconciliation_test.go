package repository

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	cleanupdomain "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTargetCleanupRepositoryDeletesBoundedAssetsByOldTargetID(t *testing.T) {
	db := openTargetCleanupRepositoryDB(t)
	repo := NewTargetCleanupRepository(db)
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	seedTargetCleanupTombstone(t, db, 1, now)
	seedTargetCleanupActiveTarget(t, db, 2)
	if err := db.Exec("INSERT INTO subdomain (id, target_id) VALUES (1, 1), (2, 1), (3, 1), (4, 2)").Error; err != nil {
		t.Fatalf("seed Subdomains: %v", err)
	}

	deleted, err := repo.DeleteCurrentAssetBatch(context.Background(), 1, cleanupdomain.AssetResourceSubdomain, 2)
	if err != nil || deleted != 2 {
		t.Fatalf("DeleteCurrentAssetBatch(first) = %d, %v", deleted, err)
	}
	assertTargetCleanupAssetIDs(t, db, "subdomain", []int{3}, 1)
	assertTargetCleanupAssetIDs(t, db, "subdomain", []int{4}, 2)
	deleted, err = repo.DeleteCurrentAssetBatch(context.Background(), 1, cleanupdomain.AssetResourceSubdomain, 2)
	if err != nil || deleted != 1 {
		t.Fatalf("DeleteCurrentAssetBatch(second) = %d, %v", deleted, err)
	}
	deleted, err = repo.DeleteCurrentAssetBatch(context.Background(), 1, cleanupdomain.AssetResourceSubdomain, 2)
	if err != nil || deleted != 0 {
		t.Fatalf("DeleteCurrentAssetBatch(empty) = %d, %v", deleted, err)
	}

	if err := db.Exec("INSERT INTO vulnerability (id, target_id, reviewed) VALUES (10, 1, TRUE), (11, 1, FALSE), (12, 2, TRUE)").Error; err != nil {
		t.Fatalf("seed Vulnerabilities: %v", err)
	}
	deleted, err = repo.DeleteCurrentAssetBatch(context.Background(), 1, cleanupdomain.AssetResourceVulnerability, 10)
	if err != nil || deleted != 2 {
		t.Fatalf("DeleteCurrentAssetBatch(vulnerabilities) = %d, %v", deleted, err)
	}
	assertTargetCleanupAssetIDs(t, db, "vulnerability", []int{12}, 2)
}

func TestTargetCleanupRepositoryDeletesEveryAssetTableByTargetIDOnly(t *testing.T) {
	for resourceIndex, resource := range cleanupdomain.OrderedAssetResources() {
		resource := resource
		t.Run(string(resource), func(t *testing.T) {
			db := openTargetCleanupRepositoryDB(t)
			repo := NewTargetCleanupRepository(db)
			now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
			table, ok := targetCleanupAssetTable(resource)
			if !ok {
				t.Fatalf("unknown cleanup resource %q", resource)
			}
			const oldTargetID = 1
			const replacementTargetID = 2
			mustTargetCleanupExec(t, db, "INSERT INTO target (id, name, deleted_at) VALUES (?, ?, ?)", oldTargetID, "reusable.example", now)
			mustTargetCleanupExec(t, db, "INSERT INTO target (id, name, deleted_at) VALUES (?, ?, NULL)", replacementTargetID, "reusable.example", now)

			firstID := resourceIndex*100 + 1
			if resource == cleanupdomain.AssetResourceVulnerability {
				mustTargetCleanupExec(t, db, fmt.Sprintf("INSERT INTO %s (id, target_id, reviewed) VALUES (?, ?, TRUE), (?, ?, FALSE), (?, ?, TRUE), (?, ?, TRUE)", table), firstID, oldTargetID, firstID+1, oldTargetID, firstID+2, oldTargetID, firstID+3, replacementTargetID)
			} else {
				mustTargetCleanupExec(t, db, fmt.Sprintf("INSERT INTO %s (id, target_id) VALUES (?, ?), (?, ?), (?, ?), (?, ?)", table), firstID, oldTargetID, firstID+1, oldTargetID, firstID+2, oldTargetID, firstID+3, replacementTargetID)
			}

			deleted, err := repo.DeleteCurrentAssetBatch(context.Background(), oldTargetID, resource, 2)
			if err != nil || deleted != 2 {
				t.Fatalf("first DeleteCurrentAssetBatch(%s) = %d, %v", resource, deleted, err)
			}
			assertTargetCleanupAssetIDs(t, db, table, []int{firstID + 2}, oldTargetID)
			assertTargetCleanupAssetIDs(t, db, table, []int{firstID + 3}, replacementTargetID)
			deleted, err = repo.DeleteCurrentAssetBatch(context.Background(), oldTargetID, resource, 2)
			if err != nil || deleted != 1 {
				t.Fatalf("second DeleteCurrentAssetBatch(%s) = %d, %v", resource, deleted, err)
			}
			deleted, err = repo.DeleteCurrentAssetBatch(context.Background(), oldTargetID, resource, 2)
			if err != nil || deleted != 0 {
				t.Fatalf("empty DeleteCurrentAssetBatch(%s) = %d, %v", resource, deleted, err)
			}
			assertTargetCleanupAssetIDs(t, db, table, []int{firstID + 3}, replacementTargetID)
		})
	}
}

func TestTargetCleanupRepositoryDeletesOnlyTargetControlPlaneRows(t *testing.T) {
	db := openTargetCleanupRepositoryDB(t)
	repo := NewTargetCleanupRepository(db)
	now := time.Now().UTC()
	seedTargetCleanupTombstone(t, db, 1, now)
	seedTargetCleanupActiveTarget(t, db, 2)
	if err := db.Exec("INSERT INTO organization_target (organization_id, target_id) VALUES (10, 1), (10, 2)").Error; err != nil {
		t.Fatalf("seed organization relations: %v", err)
	}
	if err := db.Exec(`INSERT INTO blacklist_policy (id, scope, target_id) VALUES
		(1, 'global', NULL), (2, 'target', 1), (3, 'target', 2)`).Error; err != nil {
		t.Fatalf("seed policies: %v", err)
	}

	relations, policies, err := repo.DeleteTargetControlPlane(context.Background(), 1)
	if err != nil || relations != 1 || policies != 1 {
		t.Fatalf("DeleteTargetControlPlane() = relations=%d policies=%d err=%v", relations, policies, err)
	}
	assertTargetCleanupCount(t, db, "organization_target", "target_id = ?", 1, 0)
	assertTargetCleanupCount(t, db, "organization_target", "target_id = ?", 2, 1)
	assertTargetCleanupCount(t, db, "blacklist_policy", "id = ?", 1, 1)
	assertTargetCleanupCount(t, db, "blacklist_policy", "id = ?", 2, 0)
	assertTargetCleanupCount(t, db, "blacklist_policy", "id = ?", 3, 1)
}

func TestTargetCleanupRepositoryDueJobsAndDiagnosticsRemainIndependent(t *testing.T) {
	db := openTargetCleanupRepositoryDB(t)
	repo := NewTargetCleanupRepository(db)
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	for _, targetID := range []int{1, 2, 3, 4} {
		seedTargetCleanupTombstone(t, db, targetID, now)
	}
	seedTargetCleanupJob(t, db, 1, 1, "pending", 0, now.Add(-2*time.Minute))
	seedTargetCleanupJob(t, db, 2, 2, "pending", 0, now.Add(-time.Minute))
	seedTargetCleanupJob(t, db, 3, 3, "pending", 0, now.Add(time.Minute))
	seedTargetCleanupJob(t, db, 4, 4, "completed", 2, now.Add(-time.Hour))

	jobs, err := repo.ListDue(context.Background(), now, 10)
	if err != nil || len(jobs) != 2 || jobs[0].ID != 1 || jobs[1].ID != 2 {
		t.Fatalf("ListDue() = %+v, %v", jobs, err)
	}
	nextRetry := now.Add(time.Hour)
	if err := repo.RecordFailure(context.Background(), 1, 3, nextRetry, "database_lock"); err != nil {
		t.Fatalf("RecordFailure(): %v", err)
	}
	if err := repo.Defer(context.Background(), 2, now.Add(5*time.Second)); err != nil {
		t.Fatalf("Defer(): %v", err)
	}
	var first, second model.TargetCleanupJob
	if err := db.Where("id = ?", 1).First(&first).Error; err != nil {
		t.Fatalf("read first job: %v", err)
	}
	if err := db.Where("id = ?", 2).First(&second).Error; err != nil {
		t.Fatalf("read second job: %v", err)
	}
	if first.RetryCount != 3 || first.LastError != "database_lock" || !first.NextRetryAt.Equal(nextRetry) {
		t.Fatalf("first retry diagnostics = %+v", first)
	}
	if second.RetryCount != 0 || second.LastError != "" || !second.NextRetryAt.Equal(now.Add(5*time.Second)) {
		t.Fatalf("budget deferral changed error diagnostics: %+v", second)
	}
	backlog, err := repo.InspectBacklog(context.Background())
	if err != nil || backlog.UnfinishedCount != 3 || backlog.OldestCreatedAt == nil {
		t.Fatalf("InspectBacklog() = %+v, %v", backlog, err)
	}
}

func TestTargetCleanupRepositoryCompletionRequiresEveryCurrentConditionToBeAbsent(t *testing.T) {
	cases := []struct {
		name string
		seed func(*testing.T, *gorm.DB)
	}{
		{name: "target-scoped schedule", seed: func(t *testing.T, db *gorm.DB) {
			mustTargetCleanupExec(t, db, "INSERT INTO scheduled_scan (id, target_id) VALUES (1, 1)")
		}},
		{name: "schedule occurrence", seed: func(t *testing.T, db *gorm.DB) {
			mustTargetCleanupExec(t, db, "INSERT INTO scheduled_scan (id, target_id) VALUES (1, 1)")
			mustTargetCleanupExec(t, db, "INSERT INTO scheduled_scan_occurrence (id, scheduled_scan_id) VALUES (1, 1)")
		}},
		{name: "organization relation", seed: func(t *testing.T, db *gorm.DB) {
			mustTargetCleanupExec(t, db, "INSERT INTO organization_target (organization_id, target_id) VALUES (1, 1)")
		}},
		{name: "Target policy", seed: func(t *testing.T, db *gorm.DB) {
			mustTargetCleanupExec(t, db, "INSERT INTO blacklist_policy (id, scope, target_id) VALUES (1, 'target', 1)")
		}},
		{name: "active pending Scan", seed: func(t *testing.T, db *gorm.DB) {
			mustTargetCleanupExec(t, db, "INSERT INTO scan (id, target_id, status) VALUES (1, 1, 'pending')")
		}},
	}
	for _, resource := range cleanupdomain.OrderedAssetResources() {
		resource := resource
		cases = append(cases, struct {
			name string
			seed func(*testing.T, *gorm.DB)
		}{name: string(resource), seed: func(t *testing.T, db *gorm.DB) {
			table, ok := targetCleanupAssetTable(resource)
			if !ok {
				t.Fatalf("unknown resource %q", resource)
			}
			mustTargetCleanupExec(t, db, fmt.Sprintf("INSERT INTO %s (id, target_id) VALUES (1, 1)", table))
		}})
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			db := openTargetCleanupRepositoryDB(t)
			repo := NewTargetCleanupRepository(db)
			now := time.Now().UTC()
			seedTargetCleanupTombstone(t, db, 1, now)
			seedTargetCleanupJob(t, db, 1, 1, "pending", 0, now)
			test.seed(t, db)

			completed, err := repo.MarkCompletedIfClear(context.Background(), 1, 1, now)
			if err != nil || completed {
				t.Fatalf("MarkCompletedIfClear() = %v, %v; remaining %s must block completion", completed, err, test.name)
			}
			assertTargetCleanupJobStatus(t, db, 1, "pending")
		})
	}
}

func TestTargetCleanupRepositoryCompletionPreservesHistoryAndOrganizationSchedules(t *testing.T) {
	db := openTargetCleanupRepositoryDB(t)
	repo := NewTargetCleanupRepository(db)
	now := time.Now().UTC()
	seedTargetCleanupTombstone(t, db, 1, now)
	seedTargetCleanupJob(t, db, 1, 1, "pending", 0, now)
	for _, statement := range []string{
		"INSERT INTO scheduled_scan (id, target_id) VALUES (1, NULL)",
		"INSERT INTO scan (id, target_id, status) VALUES (1, 1, 'cancelled')",
		"INSERT INTO scan_task (id, scan_id) VALUES (1, 1)",
		"INSERT INTO task_progress_log (id, scan_task_id) VALUES (1, 1)",
		"INSERT INTO scan_blacklist_snapshot (scan_id) VALUES (1)",
		"INSERT INTO subdomain_snapshot (id, scan_id) VALUES (1, 1)",
		"INSERT INTO host_port_mapping_snapshot (id, scan_id) VALUES (1, 1)",
		"INSERT INTO website_snapshot (id, scan_id) VALUES (1, 1)",
		"INSERT INTO endpoint_snapshot (id, scan_id) VALUES (1, 1)",
		"INSERT INTO directory_snapshot (id, scan_id) VALUES (1, 1)",
		"INSERT INTO screenshot_snapshot (id, scan_id) VALUES (1, 1)",
		"INSERT INTO vulnerability_snapshot (id, scan_id) VALUES (1, 1)",
	} {
		mustTargetCleanupExec(t, db, statement)
	}

	completed, err := repo.MarkCompletedIfClear(context.Background(), 1, 1, now)
	if err != nil || !completed {
		t.Fatalf("MarkCompletedIfClear() = %v, %v; retained history must not block completion", completed, err)
	}
	assertTargetCleanupJobStatus(t, db, 1, "completed")
	for _, table := range []string{
		"scan", "scan_task", "task_progress_log", "scan_blacklist_snapshot",
		"subdomain_snapshot", "host_port_mapping_snapshot", "website_snapshot", "endpoint_snapshot",
		"directory_snapshot", "screenshot_snapshot", "vulnerability_snapshot",
	} {
		assertTargetCleanupCount(t, db, table, "1 = ?", 1, 1)
	}
}

func TestTargetCleanupRepositoryNeverDeletesScanOwnedHistory(t *testing.T) {
	source, err := os.ReadFile("target_cleanup_reconciliation.go")
	if err != nil {
		t.Fatalf("read cleanup repository source: %v", err)
	}
	lowerSource := strings.ToLower(string(source))
	for _, table := range []string{
		"scan", "scan_task", "task_progress_log", "scan_blacklist_snapshot",
		"subdomain_snapshot", "host_port_mapping_snapshot", "website_snapshot", "endpoint_snapshot",
		"directory_snapshot", "screenshot_snapshot", "vulnerability_snapshot",
	} {
		if strings.Contains(lowerSource, "delete from "+table) {
			t.Fatalf("Target cleanup persistence must not delete retained history table %q", table)
		}
	}
}

func openTargetCleanupRepositoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	for _, statement := range []string{
		"CREATE TABLE target (id INTEGER PRIMARY KEY, name TEXT, deleted_at DATETIME)",
		`CREATE TABLE target_cleanup_job (
			id INTEGER PRIMARY KEY, target_id INTEGER NOT NULL, status TEXT NOT NULL, retry_count INTEGER NOT NULL,
			next_retry_at DATETIME NOT NULL, last_error TEXT NOT NULL, completed_at DATETIME, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL
		)`,
		"CREATE TABLE organization_target (organization_id INTEGER NOT NULL, target_id INTEGER NOT NULL)",
		"CREATE TABLE blacklist_policy (id INTEGER PRIMARY KEY, scope TEXT NOT NULL, target_id INTEGER)",
		"CREATE TABLE scheduled_scan (id INTEGER PRIMARY KEY, target_id INTEGER)",
		"CREATE TABLE scheduled_scan_occurrence (id INTEGER PRIMARY KEY, scheduled_scan_id INTEGER)",
		"CREATE TABLE scan (id INTEGER PRIMARY KEY, target_id INTEGER NOT NULL, status TEXT NOT NULL)",
		"CREATE TABLE scan_task (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL)",
		"CREATE TABLE task_progress_log (id INTEGER PRIMARY KEY, scan_task_id INTEGER NOT NULL)",
		"CREATE TABLE scan_blacklist_snapshot (scan_id INTEGER PRIMARY KEY)",
		"CREATE TABLE subdomain_snapshot (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL)",
		"CREATE TABLE host_port_mapping_snapshot (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL)",
		"CREATE TABLE website_snapshot (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL)",
		"CREATE TABLE endpoint_snapshot (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL)",
		"CREATE TABLE directory_snapshot (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL)",
		"CREATE TABLE screenshot_snapshot (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL)",
		"CREATE TABLE vulnerability_snapshot (id INTEGER PRIMARY KEY, scan_id INTEGER NOT NULL)",
		"CREATE TABLE subdomain (id INTEGER PRIMARY KEY, target_id INTEGER NOT NULL)",
		"CREATE TABLE host_port_mapping (id INTEGER PRIMARY KEY, target_id INTEGER NOT NULL)",
		"CREATE TABLE website (id INTEGER PRIMARY KEY, target_id INTEGER NOT NULL)",
		"CREATE TABLE endpoint (id INTEGER PRIMARY KEY, target_id INTEGER NOT NULL)",
		"CREATE TABLE directory (id INTEGER PRIMARY KEY, target_id INTEGER NOT NULL)",
		"CREATE TABLE screenshot (id INTEGER PRIMARY KEY, target_id INTEGER NOT NULL)",
		"CREATE TABLE vulnerability (id INTEGER PRIMARY KEY, target_id INTEGER NOT NULL, reviewed BOOLEAN NOT NULL DEFAULT FALSE)",
	} {
		mustTargetCleanupExec(t, db, statement)
	}
	return db
}

func seedTargetCleanupTombstone(t *testing.T, db *gorm.DB, targetID int, deletedAt time.Time) {
	t.Helper()
	mustTargetCleanupExec(t, db, "INSERT INTO target (id, deleted_at) VALUES (?, ?)", targetID, deletedAt)
}

func seedTargetCleanupActiveTarget(t *testing.T, db *gorm.DB, targetID int) {
	t.Helper()
	mustTargetCleanupExec(t, db, "INSERT INTO target (id, deleted_at) VALUES (?, NULL)", targetID)
}

func seedTargetCleanupJob(t *testing.T, db *gorm.DB, id, targetID int, status string, retryCount int, nextRetryAt time.Time) {
	t.Helper()
	mustTargetCleanupExec(t, db, `INSERT INTO target_cleanup_job
		(id, target_id, status, retry_count, next_retry_at, last_error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, '', ?, ?)`, id, targetID, status, retryCount, nextRetryAt, nextRetryAt, nextRetryAt)
}

func assertTargetCleanupAssetIDs(t *testing.T, db *gorm.DB, table string, want []int, targetID int) {
	t.Helper()
	var ids []int
	if err := db.Table(table).Where("target_id = ?", targetID).Order("id ASC").Pluck("id", &ids).Error; err != nil {
		t.Fatalf("read %s IDs: %v", table, err)
	}
	if fmt.Sprint(ids) != fmt.Sprint(want) {
		t.Fatalf("%s target %d IDs = %v, want %v", table, targetID, ids, want)
	}
}

func assertTargetCleanupCount(t *testing.T, db *gorm.DB, table, condition string, argument any, want int64) {
	t.Helper()
	var count int64
	if err := db.Table(table).Where(condition, argument).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != want {
		t.Fatalf("%s count = %d, want %d", table, count, want)
	}
}

func assertTargetCleanupJobStatus(t *testing.T, db *gorm.DB, jobID int, want string) {
	t.Helper()
	var status string
	if err := db.Table("target_cleanup_job").Select("status").Where("id = ?", jobID).Scan(&status).Error; err != nil {
		t.Fatalf("read Job %d: %v", jobID, err)
	}
	if status != want {
		t.Fatalf("Job %d status = %q, want %q", jobID, status, want)
	}
}

func mustTargetCleanupExec(t *testing.T, db *gorm.DB, statement string, arguments ...any) {
	t.Helper()
	if err := db.Exec(statement, arguments...).Error; err != nil {
		t.Fatalf("execute %q: %v", statement, err)
	}
}
