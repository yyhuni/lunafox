package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestApplyMigratesLegacyReferencesWithoutRenamingFiles(t *testing.T) {
	db := openMigrationTestDB(t)
	createLegacyMigrationTables(t, db)
	path, content := createMigrationWordlistFile(t, "dns.txt", "one\ntwo\n")
	insertLegacyWordlist(t, db, 7, "dns.txt", path, content)
	if err := db.Exec(`INSERT INTO scan (id, configuration) VALUES (?, ?)`, 12, `{"steps":{"directory":{"enabled":true,"engineConfig":{"scan":{"enabled":true,"wordlist":"dns.txt"}}}}}`).Error; err != nil {
		t.Fatalf("insert scan: %v", err)
	}
	if err := db.Exec(`INSERT INTO scheduled_scan (id, configuration) VALUES (?, ?)`, 15, `{"steps":{"directory":{"enabled":true,"engineConfig":{"scan":{"enabled":true,"wordlist":"dns.txt"}}}}}`).Error; err != nil {
		t.Fatalf("insert scheduled scan: %v", err)
	}

	migration := NewWordlistResourceIdentityMigration(db)
	report, err := migration.Apply(context.Background())
	if err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if report.WordlistCount != 1 || report.VerifiedFileCount != 1 || !report.NeedsColumnRename {
		t.Fatalf("unexpected migration report: %+v", report)
	}
	if report.UpdatedScanCount != 1 || report.UpdatedScheduleCount != 1 || report.UpdatedPlanCount != 0 {
		t.Fatalf("unexpected update counts: %+v", report)
	}
	columns := migrationFixtureColumns(t, db)
	if columns["name"] || !columns["file_name"] {
		t.Fatal("migration did not replace legacy wordlist.name with file_name")
	}
	var fileName string
	if err := db.Raw(`SELECT file_name FROM wordlist WHERE id = ?`, 7).Scan(&fileName).Error; err != nil {
		t.Fatalf("read migrated file name: %v", err)
	}
	if fileName != "dns.txt" {
		t.Fatalf("file name = %q, want dns.txt", fileName)
	}
	assertMigrationJSONResource(t, db, scanTable, 12)
	assertMigrationJSONResource(t, db, scheduledScanTable, 15)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read original wordlist file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("migration changed file content: %q", got)
	}
	if filepath.Base(path) != "dns.txt" {
		t.Fatalf("migration renamed file: %s", path)
	}

	again, err := migration.Apply(context.Background())
	if err != nil {
		t.Fatalf("rerun migration: %v", err)
	}
	if again.NeedsColumnRename || again.UpdatedScanCount != 0 || again.UpdatedScheduleCount != 0 || again.UpdatedPlanCount != 0 {
		t.Fatalf("repeatable migration changed current snapshot: %+v", again)
	}
}

func TestApplyFailsBeforeMutatingWhenCatalogFileIntegrityDoesNotMatch(t *testing.T) {
	db := openMigrationTestDB(t)
	createLegacyMigrationTables(t, db)
	path, content := createMigrationWordlistFile(t, "dns.txt", "one\ntwo\n")
	insertLegacyWordlist(t, db, 7, "dns.txt", path, content)
	if err := db.Exec(`UPDATE wordlist SET file_hash = ? WHERE id = ?`, strings.Repeat("0", 64), 7).Error; err != nil {
		t.Fatalf("corrupt metadata: %v", err)
	}
	legacyConfig := `{"steps":{"directory":{"enabled":true,"engineConfig":{"scan":{"enabled":true,"wordlist":"dns.txt"}}}}}`
	if err := db.Exec(`INSERT INTO scan (id, configuration) VALUES (?, ?)`, 12, legacyConfig).Error; err != nil {
		t.Fatalf("insert scan: %v", err)
	}

	_, err := NewWordlistResourceIdentityMigration(db).Apply(context.Background())
	if err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("apply error = %v, want integrity failure", err)
	}
	columns := migrationFixtureColumns(t, db)
	if !columns["name"] || columns["file_name"] {
		t.Fatal("failed preflight changed the wordlist schema")
	}
	var configuration string
	if err := db.Raw(`SELECT configuration FROM scan WHERE id = ?`, 12).Scan(&configuration).Error; err != nil {
		t.Fatalf("read scan after failed migration: %v", err)
	}
	if configuration != legacyConfig {
		t.Fatalf("failed preflight changed scan configuration: %s", configuration)
	}
}

func TestPreflightRejectsUnresolvedConfiguredFilename(t *testing.T) {
	db := openMigrationTestDB(t)
	createLegacyMigrationTables(t, db)
	path, content := createMigrationWordlistFile(t, "dns.txt", "one\n")
	insertLegacyWordlist(t, db, 7, "dns.txt", path, content)
	if err := db.Exec(`INSERT INTO scan (id, configuration) VALUES (?, ?)`, 12, `{"wordlist":"missing.txt"}`).Error; err != nil {
		t.Fatalf("insert scan: %v", err)
	}

	_, err := NewWordlistResourceIdentityMigration(db).Preflight(context.Background())
	if err == nil || !strings.Contains(err.Error(), "cannot be resolved") {
		t.Fatalf("preflight error = %v, want unresolved resource", err)
	}
	if !migrationFixtureColumns(t, db)["name"] {
		t.Fatal("preflight must not rename the schema")
	}
}

func openMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "-")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func createLegacyMigrationTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE wordlist (id INTEGER PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', file_path TEXT NOT NULL, file_size INTEGER NOT NULL, line_count INTEGER NOT NULL, file_hash TEXT NOT NULL)`,
		`CREATE UNIQUE INDEX unique_wordlist_name ON wordlist(name)`,
		`CREATE TABLE scan (id INTEGER PRIMARY KEY, configuration JSON NOT NULL)`,
		`CREATE TABLE scheduled_scan (id INTEGER PRIMARY KEY, configuration JSON NOT NULL)`,
		`CREATE TABLE scan_task (id INTEGER PRIMARY KEY, resolved_execution_plan BLOB NOT NULL DEFAULT '')`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create migration fixture table: %v", err)
		}
	}
}

func createMigrationWordlistFile(t *testing.T, name, value string) (string, []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	content := []byte(value)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write wordlist fixture: %v", err)
	}
	return path, content
}

func insertLegacyWordlist(t *testing.T, db *gorm.DB, id int, name, path string, content []byte) {
	t.Helper()
	digest := sha256.Sum256(content)
	if err := db.Exec(
		`INSERT INTO wordlist (id, name, file_path, file_size, line_count, file_hash) VALUES (?, ?, ?, ?, ?, ?)`,
		id,
		name,
		path,
		len(content),
		countLines(content),
		hex.EncodeToString(digest[:]),
	).Error; err != nil {
		t.Fatalf("insert wordlist: %v", err)
	}
}

func assertMigrationJSONResource(t *testing.T, db *gorm.DB, table string, id int) {
	t.Helper()
	var configuration string
	if err := db.Raw(`SELECT configuration FROM `+table+` WHERE id = ?`, id).Scan(&configuration).Error; err != nil {
		t.Fatalf("read %s configuration: %v", table, err)
	}
	if !strings.Contains(configuration, `"wordlist":"wordlists/7"`) {
		t.Fatalf("%s configuration was not migrated: %s", table, configuration)
	}
}

func migrationFixtureColumns(t *testing.T, db *gorm.DB) map[string]bool {
	t.Helper()
	type column struct {
		Name string
	}
	var columns []column
	if err := db.Raw(`PRAGMA table_info(wordlist)`).Scan(&columns).Error; err != nil {
		t.Fatalf("inspect migration fixture columns: %v", err)
	}
	result := make(map[string]bool, len(columns))
	for _, column := range columns {
		result[column.Name] = true
	}
	return result
}
