package fingerprint

import (
	"context"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/application"
	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/repository"
	fingerprintmodel "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFingerprintHubFacadeImportsExportsAndMaintainsSingletonStatistics(t *testing.T) {
	store := newFingerprintIntegrationStore(t)
	facade := application.NewFacade(store)
	ctx := context.Background()
	contents := []byte(`[
  {"id":"example-one","info":{"name":"Example One","severity":"info"},"http":[{"matchers":[{"type":"word","words":["one"]}]}]},
  {"id":"example-two","info":{"name":"Example Two"},"http":[{"matchers":[{"type":"word","words":["two"]}]}]}
]`)
	counts, err := facade.Import(ctx, domain.LibraryFingerPrintHub, contents)
	if err != nil || counts != (application.ImportCounts{CreatedCount: 2}) {
		t.Fatalf("Import() = %#v, %v", counts, err)
	}
	statistics, err := facade.Statistics(ctx)
	if err != nil || statistics.FingerPrintHub != 2 {
		t.Fatalf("Statistics() = %#v, %v", statistics, err)
	}
	exported, err := facade.Export(ctx, domain.LibraryFingerPrintHub)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if _, err := domain.ParseNativeImport(domain.LibraryFingerPrintHub, exported); err != nil {
		t.Fatalf("exported aggregate no longer parses: %v", err)
	}
	records, err := store.ListAll(ctx, domain.LibraryFingerPrintHub)
	if err != nil || len(records) != 2 {
		t.Fatalf("ListAll() = %#v, %v", records, err)
	}
	deleted, err := facade.Delete(ctx, domain.LibraryFingerPrintHub, []string{records[0].CanonicalName()})
	if err != nil || deleted != 1 {
		t.Fatalf("Delete() = %d, %v", deleted, err)
	}
	if generation, err := store.CurrentGeneration(ctx, domain.LibraryFingerPrintHub); err != nil || generation != 2 {
		t.Fatalf("CurrentGeneration() = %d, %v, want 2", generation, err)
	}
}

func TestFingerprintHubFacadeRejectsRetiredLibraryBeforeStorage(t *testing.T) {
	store := newFingerprintIntegrationStore(t)
	if _, err := application.NewFacade(store).Import(context.Background(), domain.Library("ehole"), []byte(`[]`)); err == nil {
		t.Fatal("Import() accepted retired library")
	}
	if generation, err := store.CurrentGeneration(context.Background(), domain.LibraryFingerPrintHub); err != nil || generation != 0 {
		t.Fatalf("CurrentGeneration() = %d, %v, want 0", generation, err)
	}
}

func TestFingerprintHubClearAndDeletionDoNotReseedTheLibrary(t *testing.T) {
	contents := []byte(`[
  {"id":"clear-check","info":{"name":"Clear Check","severity":"info"},"http":[{"matchers":[{"type":"word","words":["clear"]}]}]}
]`)

	t.Run("clear keeps the library empty", func(t *testing.T) {
		store := newFingerprintIntegrationStore(t)
		facade := application.NewFacade(store)
		if _, err := facade.Import(context.Background(), domain.LibraryFingerPrintHub, contents); err != nil {
			t.Fatalf("import fingerprint: %v", err)
		}
		if _, err := facade.Clear(context.Background(), domain.LibraryFingerPrintHub); err != nil {
			t.Fatalf("clear fingerprint library: %v", err)
		}
		statistics, err := facade.Statistics(context.Background())
		if err != nil || statistics.FingerPrintHub != 0 {
			t.Fatalf("statistics after clear = %#v, %v; want empty library", statistics, err)
		}
	})

	t.Run("user deletion keeps the library empty", func(t *testing.T) {
		store := newFingerprintIntegrationStore(t)
		facade := application.NewFacade(store)
		if _, err := facade.Import(context.Background(), domain.LibraryFingerPrintHub, contents); err != nil {
			t.Fatalf("import fingerprint: %v", err)
		}
		records, err := store.ListAll(context.Background(), domain.LibraryFingerPrintHub)
		if err != nil || len(records) != 1 {
			t.Fatalf("list imported fingerprints = %#v, %v", records, err)
		}
		if _, err := facade.Delete(context.Background(), domain.LibraryFingerPrintHub, []string{records[0].CanonicalName()}); err != nil {
			t.Fatalf("delete fingerprint: %v", err)
		}
		statistics, err := facade.Statistics(context.Background())
		if err != nil || statistics.FingerPrintHub != 0 {
			t.Fatalf("statistics after deletion = %#v, %v; want empty library", statistics, err)
		}
	})
}

func newFingerprintIntegrationStore(t *testing.T) *repository.FingerprintRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&fingerprintmodel.FingerPrintHubFingerprint{},
		&fingerprintmodel.FingerprintLibraryState{},
		&fingerprintmodel.FingerprintLibraryArtifact{},
	); err != nil {
		t.Fatalf("migrate fingerprint store: %v", err)
	}
	return repository.NewFingerprintRepository(db)
}
