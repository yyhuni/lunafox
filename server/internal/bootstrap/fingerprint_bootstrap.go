package bootstrap

import (
	"bytes"
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
	"gorm.io/gorm"
)

type fingerprintBootstrapService interface {
	Import(context.Context, domain.Library, []byte) (fingerprintapp.ImportCounts, error)
	Export(context.Context, domain.Library) ([]byte, error)
	Statistics(context.Context) (fingerprintapp.LibraryStatistics, error)
	CurrentGeneration(context.Context, domain.Library) (int64, error)
}

type repositoryFingerprintBootstrapService struct {
	facade     *fingerprintapp.Facade
	repository *fingerprintrepo.FingerprintRepository
}

func newRepositoryFingerprintBootstrapService(db *gorm.DB) *repositoryFingerprintBootstrapService {
	repository := fingerprintrepo.NewFingerprintRepository(db)
	return &repositoryFingerprintBootstrapService{
		facade:     fingerprintapp.NewFacade(repository),
		repository: repository,
	}
}

func (service *repositoryFingerprintBootstrapService) Import(ctx context.Context, library domain.Library, contents []byte) (fingerprintapp.ImportCounts, error) {
	return service.facade.Import(ctx, library, contents)
}

func (service *repositoryFingerprintBootstrapService) Export(ctx context.Context, library domain.Library) ([]byte, error) {
	return service.facade.Export(ctx, library)
}

func (service *repositoryFingerprintBootstrapService) Statistics(ctx context.Context) (fingerprintapp.LibraryStatistics, error) {
	return service.facade.Statistics(ctx)
}

func (service *repositoryFingerprintBootstrapService) CurrentGeneration(ctx context.Context, library domain.Library) (int64, error) {
	return service.repository.CurrentGeneration(ctx, library)
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

	return importFingerprintBootstrapCorpus(ctx, contents, newRepositoryFingerprintBootstrapService(db))
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
	expectedRecords, err := domain.ParseNativeImport(domain.LibraryFingerPrintHub, contents)
	if err != nil {
		return fmt.Errorf("import FingerprintHub bootstrap corpus: %w", err)
	}
	expectedRecords = domain.CollapseLastWins(expectedRecords)
	generation, err := service.CurrentGeneration(ctx, domain.LibraryFingerPrintHub)
	if err != nil {
		return fmt.Errorf("read FingerprintHub bootstrap generation: %w", err)
	}
	statistics, err := service.Statistics(ctx)
	if err != nil {
		return fmt.Errorf("read FingerprintHub bootstrap statistics: %w", err)
	}
	if generation == 0 {
		if statistics.FingerPrintHub != 0 {
			return errors.New("FingerprintHub bootstrap records exist without initialization state; explicit repair or reset is required")
		}
		counts, err := service.Import(ctx, domain.LibraryFingerPrintHub, contents)
		if err != nil {
			return fmt.Errorf("import FingerprintHub bootstrap corpus: %w", err)
		}
		if counts.CreatedCount != len(expectedRecords) || counts.UpdatedCount != 0 || counts.UnchangedCount != 0 {
			return errors.New("FingerprintHub bootstrap import encountered existing records; explicit repair or reset is required")
		}
		statistics, err = service.Statistics(ctx)
		if err != nil {
			return fmt.Errorf("read FingerprintHub bootstrap statistics: %w", err)
		}
		if statistics.FingerPrintHub == 0 {
			return errors.New("fingerprint bootstrap completed with zero FingerprintHub records")
		}
		if statistics.FingerPrintHub != int64(len(expectedRecords)) {
			return fmt.Errorf("fingerprint bootstrap completed with %d FingerprintHub records, want %d", statistics.FingerPrintHub, len(expectedRecords))
		}
		generation, err = service.CurrentGeneration(ctx, domain.LibraryFingerPrintHub)
		if err != nil {
			return fmt.Errorf("read FingerprintHub bootstrap generation: %w", err)
		}
		if generation <= 0 {
			return errors.New("fingerprint bootstrap completed without initialization state")
		}
		return nil
	}
	if statistics.FingerPrintHub == 0 {
		return errors.New("FingerprintHub bootstrap corpus is missing from initialized state; explicit repair or reset is required")
	}

	currentContents, err := service.Export(ctx, domain.LibraryFingerPrintHub)
	if err != nil {
		return fmt.Errorf("export existing FingerprintHub bootstrap state: %w", err)
	}
	currentRecords, err := domain.ParseNativeImport(domain.LibraryFingerPrintHub, currentContents)
	if err != nil {
		return fmt.Errorf("validate existing FingerprintHub bootstrap state: %w", err)
	}
	currentRecords = domain.CollapseLastWins(currentRecords)
	if int64(len(currentRecords)) != statistics.FingerPrintHub {
		return errors.New("FingerprintHub bootstrap statistics disagree with stored records; explicit repair or reset is required")
	}
	currentByIdentity := make(map[string]domain.ImportedRecord, len(currentRecords))
	for _, record := range currentRecords {
		currentByIdentity[record.IdentityKey] = record
	}
	for _, expected := range expectedRecords {
		current, exists := currentByIdentity[expected.IdentityKey]
		if !exists {
			return errors.New("FingerprintHub bootstrap corpus is partially missing; explicit repair or reset is required")
		}
		if !bytes.Equal(current.ContentHash, expected.ContentHash) {
			return errors.New("FingerprintHub bootstrap corpus drifted; explicit repair or reset is required")
		}
	}
	return nil
}
