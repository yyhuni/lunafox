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
	fingerprintrepo "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/repository"
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
		facade := newFingerprintBootstrapFacade(t)
		if err := importFingerprintBootstrapCorpus(context.Background(), contents, facade); err != nil {
			t.Fatalf("import bootstrap corpus: %v", err)
		}
		statistics, err := facade.Statistics(context.Background())
		if err != nil || statistics.FingerPrintHub != 1 {
			t.Fatalf("statistics = %#v, %v; want one imported record", statistics, err)
		}
	})

	t.Run("invalid native corpus fails before a successful bootstrap", func(t *testing.T) {
		path := writeFingerprintBootstrapCorpus(t, `{"not":"an aggregate"}`)
		contents, err := readFingerprintBootstrapCorpus(path)
		if err != nil {
			t.Fatalf("read bootstrap corpus: %v", err)
		}
		facade := newFingerprintBootstrapFacade(t)
		err = importFingerprintBootstrapCorpus(context.Background(), contents, facade)
		if err == nil || !strings.Contains(err.Error(), "import FingerprintHub bootstrap corpus") {
			t.Fatalf("import error = %v, want native import failure", err)
		}
		statistics, statsErr := facade.Statistics(context.Background())
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

func newFingerprintBootstrapFacade(t *testing.T) *fingerprintapp.Facade {
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
	return fingerprintapp.NewFacade(fingerprintrepo.NewFingerprintRepository(db))
}

type stubFingerprintBootstrapService struct {
	importCalls int
}

func (service *stubFingerprintBootstrapService) Import(_ context.Context, _ domain.Library, _ []byte) (fingerprintapp.ImportCounts, error) {
	service.importCalls++
	return fingerprintapp.ImportCounts{}, nil
}

func (service *stubFingerprintBootstrapService) Statistics(context.Context) (fingerprintapp.LibraryStatistics, error) {
	return fingerprintapp.LibraryStatistics{}, nil
}
