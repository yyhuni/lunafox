package scanwiring

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	catalogmodel "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type scanWiringTargetTestModel struct {
	ID            int        `gorm:"primaryKey;autoIncrement"`
	Name          string     `gorm:"column:name;size:300"`
	Type          string     `gorm:"column:type;size:20"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime"`
	LastScannedAt *time.Time `gorm:"column:last_scanned_at"`
	DeletedAt     *time.Time `gorm:"column:deleted_at"`
}

func (scanWiringTargetTestModel) TableName() string { return "target" }

func TestScanTargetLookupAdapterEnsureQuickTargetsCreatesMissingAndReusesExisting(t *testing.T) {
	dbName := url.QueryEscape(fmt.Sprintf("%s_%p", t.Name(), t))
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&scanWiringTargetTestModel{}); err != nil {
		t.Fatalf("migrate target: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE blacklist_policy (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scope TEXT NOT NULL,
			target_id INTEGER,
			patterns TEXT NOT NULL,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("migrate blacklist policy: %v", err)
	}
	if err := db.Create(&catalogmodel.Target{Name: "test.com", Type: "domain"}).Error; err != nil {
		t.Fatalf("insert existing target: %v", err)
	}

	targetRepo := catalogrepo.NewTargetRepository(db)
	targetCommandService := catalogapp.NewTargetCommandService(targetRepo, nil)
	adapter := newScanTargetLookupAdapter(targetRepo, targetCommandService)
	result, err := adapter.EnsureQuickTargets(context.Background(), []string{" Test.COM ", "new.example.com", "new.example.com", "***"})
	if err != nil {
		t.Fatalf("ensure quick targets: %v", err)
	}

	if len(result.Targets) != 2 {
		t.Fatalf("expected 2 resolved targets, got %+v", result.Targets)
	}
	if result.Targets[0].Name != "test.com" || result.Targets[1].Name != "new.example.com" {
		t.Fatalf("unexpected target order or names: %+v", result.Targets)
	}
	if result.TargetStats.Created != 1 || result.TargetStats.Skipped != 1 || result.TargetStats.Failed != 1 {
		t.Fatalf("unexpected target stats: %+v", result.TargetStats)
	}
	if len(result.Errors) != 1 || result.Errors[0].Input != "***" {
		t.Fatalf("expected invalid target error, got %+v", result.Errors)
	}

	var count int64
	if err := db.Model(&catalogmodel.Target{}).Where("name = ?", "new.example.com").Count(&count).Error; err != nil {
		t.Fatalf("count created target: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one created target row, got %d", count)
	}
}
