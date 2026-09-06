package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
	fingerprintapp "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/application"
	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
	fingerprintrepo "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/repository"
)

type fingerprintBootstrapService interface {
	Import(context.Context, domain.Library, []byte) (fingerprintapp.ImportCounts, error)
	Statistics(context.Context) (fingerprintapp.LibraryStatistics, error)
}

// RunFingerprintBootstrap imports the installer-staged corpus through the
// FingerprintHub application boundary. It deliberately opens only a database
// connection and never starts HTTP, workers, schedulers, or the Server runtime.
func RunFingerprintBootstrap(ctx context.Context, databaseConfig *config.DatabaseConfig, corpusPath string) error {
	if ctx == nil {
		return errors.New("fingerprint bootstrap context is required")
	}
	if databaseConfig == nil {
		return errors.New("fingerprint bootstrap database configuration is required")
	}

	contents, err := readFingerprintBootstrapCorpus(corpusPath)
	if err != nil {
		return err
	}
	db, err := database.NewDatabase(databaseConfig)
	if err != nil {
		return fmt.Errorf("connect database for fingerprint bootstrap: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("access database connection for fingerprint bootstrap: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	return importFingerprintBootstrapCorpus(ctx, contents, fingerprintapp.NewFacade(fingerprintrepo.NewFingerprintRepository(db)))
}

func readFingerprintBootstrapCorpus(corpusPath string) ([]byte, error) {
	path := strings.TrimSpace(corpusPath)
	if path == "" {
		return nil, errors.New("fingerprint bootstrap corpus path is required")
	}
	if !filepath.IsAbs(path) {
		return nil, errors.New("fingerprint bootstrap corpus path must be absolute")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fingerprint bootstrap corpus: %w", err)
	}
	return contents, nil
}

func importFingerprintBootstrapCorpus(ctx context.Context, contents []byte, service fingerprintBootstrapService) error {
	if ctx == nil {
		return errors.New("fingerprint bootstrap context is required")
	}
	if service == nil {
		return errors.New("fingerprint bootstrap service is required")
	}
	if _, err := service.Import(ctx, domain.LibraryFingerPrintHub, contents); err != nil {
		return fmt.Errorf("import FingerprintHub bootstrap corpus: %w", err)
	}
	statistics, err := service.Statistics(ctx)
	if err != nil {
		return fmt.Errorf("read FingerprintHub bootstrap statistics: %w", err)
	}
	if statistics.FingerPrintHub <= 0 {
		return errors.New("fingerprint bootstrap completed with zero FingerprintHub records")
	}
	return nil
}
