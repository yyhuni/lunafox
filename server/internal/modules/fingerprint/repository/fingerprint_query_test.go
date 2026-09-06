package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
	persistence "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fingerprintQueryTraceRecorder struct {
	logger.Interface
	errors []error
}

func (recorder *fingerprintQueryTraceRecorder) LogMode(_ logger.LogLevel) logger.Interface {
	return recorder
}

func (recorder *fingerprintQueryTraceRecorder) Trace(_ context.Context, _ time.Time, _ func() (string, int64), err error) {
	recorder.errors = append(recorder.errors, err)
}

func TestFingerprintRepositoryMissingArtifactIsAnEmptyResultWithoutRecordNotFound(t *testing.T) {
	traceRecorder := &fingerprintQueryTraceRecorder{Interface: logger.Default.LogMode(logger.Error)}
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: traceRecorder})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&persistence.FingerprintLibraryArtifact{}); err != nil {
		t.Fatalf("migrate artifact table: %v", err)
	}
	traceRecorder.errors = nil

	artifact, err := NewFingerprintRepository(db).FindArtifact(context.Background(), domain.LibraryFingerPrintHub, 1)
	if err != nil || artifact != nil {
		t.Fatalf("missing artifact = %#v, %v; want nil artifact and nil error", artifact, err)
	}
	for _, traceErr := range traceRecorder.errors {
		if errors.Is(traceErr, gorm.ErrRecordNotFound) {
			t.Fatal("missing artifact must not emit gorm.ErrRecordNotFound")
		}
	}
}

func TestFingerprintRepositoryFindArtifactPropagatesQueryErrors(t *testing.T) {
	traceRecorder := &fingerprintQueryTraceRecorder{Interface: logger.Default.LogMode(logger.Error)}
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: traceRecorder})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	artifact, err := NewFingerprintRepository(db).FindArtifact(context.Background(), domain.LibraryFingerPrintHub, 1)
	if artifact != nil || err == nil {
		t.Fatalf("missing table = %#v, %v; want query error", artifact, err)
	}
	for _, traceErr := range traceRecorder.errors {
		if traceErr != nil && !errors.Is(traceErr, gorm.ErrRecordNotFound) {
			return
		}
	}
	t.Fatal("real query error was not observed by GORM logger")
}
