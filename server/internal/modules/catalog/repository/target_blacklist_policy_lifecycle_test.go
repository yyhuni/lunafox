package repository

import (
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTargetCreateRollsBackWhenLocalBlacklistPolicyCannotPersist(t *testing.T) {
	db := openTargetBlacklistLifecycleDB(t, false)
	repository := NewTargetRepository(db)
	target := &catalogdomain.Target{Name: "rollback.example", Type: catalogdomain.TargetTypeDomain}
	if err := repository.Create(target); err == nil {
		t.Fatal("Create succeeded without blacklist_policy table")
	}
	var count int64
	if err := db.Model(&model.Target{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("policy write failure left %d targets committed", count)
	}
}

func TestTargetCreateCreatesLocalBlacklistPolicyInTheSameTransaction(t *testing.T) {
	db := openTargetBlacklistLifecycleDB(t, true)
	repository := NewTargetRepository(db)
	target := &catalogdomain.Target{Name: "created.example", Type: catalogdomain.TargetTypeDomain}
	if err := repository.Create(target); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if target.ID <= 0 {
		t.Fatal("Create did not assign target identity")
	}
	assertTargetBlacklistPolicy(t, db, target.ID, 1)
}

func TestTargetBatchCreateInitializesOnlyInsertedTargetPolicies(t *testing.T) {
	db := openTargetBlacklistLifecycleDB(t, true)
	if err := db.Exec(`CREATE UNIQUE INDEX target_blacklist_lifecycle_name_key ON target(name)`).Error; err != nil {
		t.Fatal(err)
	}
	var existingID int
	if err := db.Raw(`INSERT INTO target (name, type) VALUES (?, ?) RETURNING id`, "existing.example", catalogdomain.TargetTypeDomain).Scan(&existingID).Error; err != nil {
		t.Fatal(err)
	}
	repository := NewTargetRepository(db)
	created, err := repository.BatchCreateIgnoreConflicts([]catalogdomain.Target{
		{Name: "existing.example", Type: catalogdomain.TargetTypeDomain},
		{Name: "new.example", Type: catalogdomain.TargetTypeDomain},
	})
	if err != nil {
		t.Fatalf("BatchCreateIgnoreConflicts() error = %v", err)
	}
	if created != 1 {
		t.Fatalf("created = %d, want 1", created)
	}
	assertTargetBlacklistPolicy(t, db, existingID, 0)
	var newID int
	if err := db.Raw(`SELECT id FROM target WHERE name = ?`, "new.example").Scan(&newID).Error; err != nil {
		t.Fatal(err)
	}
	assertTargetBlacklistPolicy(t, db, newID, 1)
}

func openTargetBlacklistLifecycleDB(t *testing.T, includePolicyTable bool) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		CREATE TABLE target (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME,
			last_scanned_at DATETIME,
			deleted_at DATETIME
		)
	`).Error; err != nil {
		t.Fatal(err)
	}
	if includePolicyTable {
		if err := db.Exec(`
			CREATE TABLE blacklist_policy (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				scope TEXT NOT NULL,
				target_id INTEGER,
				patterns JSON NOT NULL,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(scope, target_id)
			)
		`).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func assertTargetBlacklistPolicy(t *testing.T, db *gorm.DB, targetID, want int) {
	t.Helper()
	var count int
	if err := db.Raw(`SELECT COUNT(*) FROM blacklist_policy WHERE scope = 'target' AND target_id = ? AND patterns = '[]'`, targetID).Scan(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("target %d local blacklist policies = %d, want %d", targetID, count, want)
	}
}
