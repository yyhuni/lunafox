package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/application"
	model "github.com/yyhuni/lunafox/server/internal/modules/loginvisual/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type queryTraceRecorder struct {
	logger.Interface
	errors []error
}

func (recorder *queryTraceRecorder) LogMode(_ logger.LogLevel) logger.Interface { return recorder }

func (recorder *queryTraceRecorder) Trace(_ context.Context, _ time.Time, _ func() (string, int64), err error) {
	recorder.errors = append(recorder.errors, err)
}

func TestDiscoverabilityRepositoryPersistsOneRecordPerAccount(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.LoginVisualDiscovery{}); err != nil {
		t.Fatalf("migrate discovery table: %v", err)
	}
	repository := NewDiscoverabilityRepository(db)

	unlocked, err := repository.IsUnlocked(context.Background(), 7)
	if err != nil || unlocked {
		t.Fatalf("initial discoverability = %t, %v; want false, nil", unlocked, err)
	}
	if err := repository.Unlock(context.Background(), 7); err != nil {
		t.Fatalf("first unlock: %v", err)
	}
	if err := repository.Unlock(context.Background(), 7); err != nil {
		t.Fatalf("repeated unlock: %v", err)
	}

	unlocked, err = repository.IsUnlocked(context.Background(), 7)
	if err != nil || !unlocked {
		t.Fatalf("persisted discoverability = %t, %v; want true, nil", unlocked, err)
	}
	var count int64
	if err := db.Model(&model.LoginVisualDiscovery{}).Where("user_id = ?", 7).Count(&count).Error; err != nil {
		t.Fatalf("count discovery records: %v", err)
	}
	if count != 1 {
		t.Fatalf("record count = %d, want 1", count)
	}
}

func TestDiscoverabilityRepositoryMissingRecordDoesNotEmitRecordNotFound(t *testing.T) {
	traceRecorder := &queryTraceRecorder{Interface: logger.Default.LogMode(logger.Error)}
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: traceRecorder})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.LoginVisualDiscovery{}); err != nil {
		t.Fatalf("migrate discovery table: %v", err)
	}
	traceRecorder.errors = nil
	repository := NewDiscoverabilityRepository(db)

	unlocked, err := repository.IsUnlocked(context.Background(), 9)
	if err != nil || unlocked {
		t.Fatalf("missing discoverability = %t, %v; want false, nil", unlocked, err)
	}
	for _, traceErr := range traceRecorder.errors {
		if errors.Is(traceErr, gorm.ErrRecordNotFound) {
			t.Fatal("missing discoverability must not emit gorm.ErrRecordNotFound")
		}
	}
}

func TestDiscoverabilityRepositoryRejectsMissingAccountIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	repository := NewDiscoverabilityRepository(db)

	if err := repository.Unlock(context.Background(), 0); !errors.Is(err, application.ErrPermissionDenied) {
		t.Fatalf("unlock error = %v, want permission denied", err)
	}
}
