package repository

import (
	"context"
	"errors"
	"testing"

	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExecutionCatalogQueriesHonorCanceledContext(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.Wordlist{}, &model.SubfinderProviderSettings{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	wordlists := NewWordlistRepository(db)
	if _, err := wordlists.GetByIDContext(ctx, 7); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetByIDContext error = %v, want context.Canceled", err)
	}
	settings := NewSubfinderProviderSettingsRepository(db)
	if _, err := settings.GetInstanceContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetInstanceContext error = %v, want context.Canceled", err)
	}
}
