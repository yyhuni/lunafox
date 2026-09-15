package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	fingerprintapp "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/application"
	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
	fingerprintmodel "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const validFingerprintBootstrapCorpus = `[
  {"id":"bootstrap-example","info":{"name":"Bootstrap Example","severity":"info"},"http":[{"matchers":[{"type":"word","words":["bootstrap"]}]}]}
]`

func TestFingerprintBootstrapCorpusUsesNativeImportAndRequiresNonzeroRecords(t *testing.T) {
	t.Run("valid local corpus imports through the application facade", func(t *testing.T) {
		path := writeFingerprintBootstrapCorpus(t, validFingerprintBootstrapCorpus)
		contents, err := readFingerprintBootstrapCorpus(path)
		if err != nil {
			t.Fatalf("read bootstrap corpus: %v", err)
		}
		service := newFingerprintBootstrapFacade(t)
		if err := importFingerprintBootstrapCorpus(context.Background(), contents, service); err != nil {
			t.Fatalf("import bootstrap corpus: %v", err)
		}
		if err := importFingerprintBootstrapCorpus(context.Background(), contents, service); err != nil {
			t.Fatalf("repeat bootstrap corpus: %v", err)
		}
		statistics, err := service.Statistics(context.Background())
		if err != nil || statistics.FingerPrintHub != 1 {
			t.Fatalf("statistics = %#v, %v; want one imported record", statistics, err)
		}
		generation, err := service.CurrentGeneration(context.Background(), domain.LibraryFingerPrintHub)
		if err != nil || generation != 1 {
			t.Fatalf("generation = %d, %v; want unchanged generation 1", generation, err)
		}
	})

	t.Run("invalid native corpus fails before a successful bootstrap", func(t *testing.T) {
		path := writeFingerprintBootstrapCorpus(t, `{"not":"an aggregate"}`)
		contents, err := readFingerprintBootstrapCorpus(path)
		if err != nil {
			t.Fatalf("read bootstrap corpus: %v", err)
		}
		service := newFingerprintBootstrapFacade(t)
		err = importFingerprintBootstrapCorpus(context.Background(), contents, service)
		if err == nil || !strings.Contains(err.Error(), "import FingerprintHub bootstrap corpus") {
			t.Fatalf("import error = %v, want native import failure", err)
		}
		statistics, statsErr := service.Statistics(context.Background())
		if statsErr != nil || statistics.FingerPrintHub != 0 {
			t.Fatalf("statistics after rejected import = %#v, %v", statistics, statsErr)
		}
	})

	t.Run("empty aggregate fails native import validation", func(t *testing.T) {
		path := writeFingerprintBootstrapCorpus(t, `[]`)
		contents, err := readFingerprintBootstrapCorpus(path)
		if err != nil {
			t.Fatalf("read bootstrap corpus: %v", err)
		}
		err = importFingerprintBootstrapCorpus(context.Background(), contents, newFingerprintBootstrapFacade(t))
		if err == nil || !strings.Contains(err.Error(), "import FingerprintHub bootstrap corpus") {
			t.Fatalf("import error = %v, want empty aggregate rejection", err)
		}
	})

	t.Run("successful import with zero statistics fails post-import verification", func(t *testing.T) {
		service := &stubFingerprintBootstrapService{}
		err := importFingerprintBootstrapCorpus(context.Background(), []byte(validFingerprintBootstrapCorpus), service)
		if err == nil || !strings.Contains(err.Error(), "zero FingerprintHub records") {
			t.Fatalf("import error = %v, want zero-record rejection", err)
		}
		if service.importCalls != 1 {
			t.Fatalf("import calls = %d, want 1", service.importCalls)
		}
	})
}

func TestFingerprintBootstrapRejectsMissingOrDriftedSeedWithoutRepair(t *testing.T) {
	ctx := context.Background()
	t.Run("partial corpus", func(t *testing.T) {
		contents := []byte(`[
  {"id":"bootstrap-one","info":{"name":"Bootstrap One","severity":"info"},"http":[{"matchers":[{"type":"word","words":["one"]}]}]},
  {"id":"bootstrap-two","info":{"name":"Bootstrap Two","severity":"info"},"http":[{"matchers":[{"type":"word","words":["two"]}]}]}
]`)
		service := newFingerprintBootstrapFacade(t)
		if err := importFingerprintBootstrapCorpus(ctx, contents, service); err != nil {
			t.Fatalf("initial bootstrap: %v", err)
		}
		records, err := service.repository.ListAll(ctx, domain.LibraryFingerPrintHub)
		if err != nil || len(records) != 2 {
			t.Fatalf("stored records = %+v, %v", records, err)
		}
		if _, err := service.repository.DeleteByResourceIDs(ctx, domain.LibraryFingerPrintHub, []string{records[0].ResourceID}); err != nil {
			t.Fatalf("delete bootstrap record fixture: %v", err)
		}
		err = importFingerprintBootstrapCorpus(ctx, contents, service)
		if err == nil || !strings.Contains(err.Error(), "partially missing") {
			t.Fatalf("partial corpus error = %v", err)
		}
		statistics, statsErr := service.Statistics(ctx)
		if statsErr != nil || statistics.FingerPrintHub != 1 {
			t.Fatalf("partial corpus was repaired: %#v, %v", statistics, statsErr)
		}
	})

	t.Run("changed corpus record", func(t *testing.T) {
		service := newFingerprintBootstrapFacade(t)
		if err := importFingerprintBootstrapCorpus(ctx, []byte(validFingerprintBootstrapCorpus), service); err != nil {
			t.Fatalf("initial bootstrap: %v", err)
		}
		changed := strings.Replace(validFingerprintBootstrapCorpus, "Bootstrap Example", "User Changed", 1)
		if _, err := service.Import(ctx, domain.LibraryFingerPrintHub, []byte(changed)); err != nil {
			t.Fatalf("change stored record fixture: %v", err)
		}
		err := importFingerprintBootstrapCorpus(ctx, []byte(validFingerprintBootstrapCorpus), service)
		if err == nil || !strings.Contains(err.Error(), "drifted") {
			t.Fatalf("drift error = %v", err)
		}
		exported, exportErr := service.Export(ctx, domain.LibraryFingerPrintHub)
		if exportErr != nil || !strings.Contains(string(exported), "User Changed") {
			t.Fatalf("drifted user record was overwritten: %s, %v", exported, exportErr)
		}
	})

	t.Run("cleared initialized corpus", func(t *testing.T) {
		service := newFingerprintBootstrapFacade(t)
		if err := importFingerprintBootstrapCorpus(ctx, []byte(validFingerprintBootstrapCorpus), service); err != nil {
			t.Fatalf("initial bootstrap: %v", err)
		}
		if _, err := service.facade.Clear(ctx, domain.LibraryFingerPrintHub); err != nil {
			t.Fatalf("clear bootstrap fixture: %v", err)
		}
		err := importFingerprintBootstrapCorpus(ctx, []byte(validFingerprintBootstrapCorpus), service)
		if err == nil || !strings.Contains(err.Error(), "missing from initialized state") {
			t.Fatalf("cleared corpus error = %v", err)
		}
		statistics, statsErr := service.Statistics(ctx)
		if statsErr != nil || statistics.FingerPrintHub != 0 {
			t.Fatalf("cleared corpus was reseeded: %#v, %v", statistics, statsErr)
		}
	})
}

func TestFingerprintBootstrapPreservesAdditionalUserRecords(t *testing.T) {
	ctx := context.Background()
	service := newFingerprintBootstrapFacade(t)
	contents := []byte(validFingerprintBootstrapCorpus)
	if err := importFingerprintBootstrapCorpus(ctx, contents, service); err != nil {
		t.Fatalf("initial bootstrap: %v", err)
	}
	extra := []byte(`[
  {"id":"user-extra","info":{"name":"User Extra","severity":"low"},"http":[{"matchers":[{"type":"word","words":["user-extra"]}]}]}
]`)
	if _, err := service.Import(ctx, domain.LibraryFingerPrintHub, extra); err != nil {
		t.Fatalf("import user record: %v", err)
	}
	if err := importFingerprintBootstrapCorpus(ctx, contents, service); err != nil {
		t.Fatalf("repeat bootstrap with user record: %v", err)
	}
	statistics, err := service.Statistics(ctx)
	if err != nil || statistics.FingerPrintHub != 2 {
		t.Fatalf("user record was not preserved: %#v, %v", statistics, err)
	}
}

func TestReadFingerprintBootstrapCorpusRejectsNonlocalOrUnreadableInputs(t *testing.T) {
	if _, err := readFingerprintBootstrapCorpus("https://example.test/web_fingerprint_v4.json"); err == nil || !strings.Contains(err.Error(), "must be absolute") {
		t.Fatalf("network-looking corpus path error = %v, want local-path rejection", err)
	}

	missingPath := filepath.Join(t.TempDir(), "missing.json")
	if _, err := readFingerprintBootstrapCorpus(missingPath); err == nil || !strings.Contains(err.Error(), "read fingerprint bootstrap corpus") {
		t.Fatalf("unreadable corpus error = %v, want file read failure", err)
	}
}

func TestReleasedFingerprintBootstrapCorpusIsAReadableNativeAggregate(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve bootstrap test path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	corpusPath := filepath.Join(repoRoot, "resources", "fingerprints", "web_fingerprint_v4.json")
	contents, err := readFingerprintBootstrapCorpus(corpusPath)
	if err != nil {
		t.Fatalf("read release-tree corpus: %v", err)
	}
	if err := importFingerprintBootstrapCorpus(context.Background(), contents, newFingerprintBootstrapFacade(t)); err != nil {
		t.Fatalf("release-tree corpus bootstrap: %v", err)
	}
}

func TestRunFingerprintBootstrapRejectsMissingInputsBeforeDatabaseConnection(t *testing.T) {
	if err := RunFingerprintBootstrap(nil, nil, ""); err == nil {
		t.Fatal("expected missing bootstrap inputs to fail")
	}
}

func writeFingerprintBootstrapCorpus(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "web_fingerprint_v4.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write bootstrap corpus: %v", err)
	}
	return path
}

func newFingerprintBootstrapFacade(t *testing.T) *repositoryFingerprintBootstrapService {
	t.Helper()
	dsn := fmt.Sprintf("file:fingerprint-bootstrap-%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open bootstrap database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("access bootstrap database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(
		&fingerprintmodel.FingerPrintHubFingerprint{},
		&fingerprintmodel.FingerprintLibraryState{},
		&fingerprintmodel.FingerprintLibraryArtifact{},
	); err != nil {
		t.Fatalf("migrate bootstrap database: %v", err)
	}
	return newRepositoryFingerprintBootstrapService(db)
}

type stubFingerprintBootstrapService struct {
	importCalls int
}

func (service *stubFingerprintBootstrapService) Import(_ context.Context, _ domain.Library, _ []byte) (fingerprintapp.ImportCounts, error) {
	service.importCalls++
	return fingerprintapp.ImportCounts{CreatedCount: 1}, nil
}

func (service *stubFingerprintBootstrapService) Export(context.Context, domain.Library) ([]byte, error) {
	return []byte(validFingerprintBootstrapCorpus), nil
}

func (service *stubFingerprintBootstrapService) Statistics(context.Context) (fingerprintapp.LibraryStatistics, error) {
	return fingerprintapp.LibraryStatistics{}, nil
}

func (service *stubFingerprintBootstrapService) CurrentGeneration(context.Context, domain.Library) (int64, error) {
	return 0, nil
}
