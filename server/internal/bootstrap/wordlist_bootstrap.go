package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	"gorm.io/gorm"
)

type defaultWordlistManifest struct {
	Wordlists []defaultWordlistManifestEntry `json:"wordlists"`
}

type defaultWordlistManifestEntry struct {
	FileName    string   `json:"fileName"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Required    bool     `json:"required"`
}

type defaultWordlistImport struct {
	entry      defaultWordlistManifestEntry
	sourcePath string
	fileSize   int64
	lineCount  int
	fileHash   string
}

// RunWordlistBootstrap copies the immutable default wordlists into shared
// storage and records them through the Catalog boundary. A repeated run only
// accepts the exact previously imported set; it never repairs or overwrites a
// missing or changed resource because that state may contain user intent.
func RunWordlistBootstrap(ctx context.Context, databaseConfig *config.DatabaseConfig, basePath, manifestPath, sourceDirectory string) error {
	if ctx == nil {
		return errors.New("wordlist bootstrap context is required")
	}
	if databaseConfig == nil {
		return errors.New("wordlist bootstrap database configuration is required")
	}
	basePath = strings.TrimSpace(basePath)
	if basePath == "" || !filepath.IsAbs(basePath) {
		return errors.New("WORDLISTS_BASE_PATH must be an absolute path")
	}
	imports, err := readDefaultWordlistImports(manifestPath, sourceDirectory)
	if err != nil {
		return err
	}

	db, err := database.NewDatabase(databaseConfig)
	if err != nil {
		return fmt.Errorf("connect database for wordlist bootstrap: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("access database connection for wordlist bootstrap: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	return bootstrapDefaultWordlists(ctx, db, basePath, imports)
}

func bootstrapDefaultWordlists(ctx context.Context, db *gorm.DB, basePath string, imports []defaultWordlistImport) error {
	if ctx == nil {
		return errors.New("wordlist bootstrap context is required")
	}
	if db == nil {
		return errors.New("wordlist bootstrap database is required")
	}

	repository := catalogrepo.NewWordlistRepository(db)
	existing, err := repository.ListAllContext(ctx)
	if err != nil {
		return fmt.Errorf("inspect existing wordlist catalog: %w", err)
	}
	existingByFileName := make(map[string]catalogdomain.Wordlist, len(existing))
	for _, wordlist := range existing {
		existingByFileName[wordlist.FileName] = wordlist
	}

	matching := 0
	for _, item := range imports {
		if _, exists := existingByFileName[item.entry.FileName]; exists {
			matching++
		}
	}
	if matching > 0 {
		if matching != len(imports) {
			return errors.New("default wordlist bootstrap state is partial; explicit repair or reset is required")
		}
		for _, item := range imports {
			if err := validatePersistedDefaultWordlist(basePath, item, existingByFileName[item.entry.FileName]); err != nil {
				return err
			}
		}
		// Older public Compose bootstraps created this root as 0755. Once the
		// complete default set is proven intact, tighten it in place so Agent's
		// shared materialization check accepts the existing volume.
		if err := catalogapp.RestrictWordlistStorageDirectory(basePath); err != nil {
			return fmt.Errorf("restrict persisted default wordlist storage: %w", err)
		}
		return nil
	}
	if len(existing) != 0 {
		return errors.New("wordlist catalog contains user data without the default bootstrap set; explicit migration or reset is required")
	}
	if err := requireUninitializedWordlistStorage(basePath); err != nil {
		return err
	}

	createdPaths := make([]string, 0, len(imports))
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		service := catalogapp.NewWordlistCommandService(
			catalogrepo.NewWordlistRepository(tx),
			basePath,
			catalogapp.NewLocalWordlistFileStore(),
		)
		for _, item := range imports {
			file, err := os.Open(item.sourcePath)
			if err != nil {
				return fmt.Errorf("open default wordlist %s: %w", item.entry.FileName, err)
			}
			wordlist, createErr := service.CreateWordlist(
				ctx,
				item.entry.FileName,
				item.entry.Description,
				item.entry.Tags,
				item.entry.FileName,
				file,
			)
			closeErr := file.Close()
			if createErr != nil {
				return fmt.Errorf("import default wordlist %s: %w", item.entry.FileName, createErr)
			}
			createdPaths = append(createdPaths, wordlist.FilePath)
			if closeErr != nil {
				return fmt.Errorf("close default wordlist %s: %w", item.entry.FileName, closeErr)
			}
		}
		return nil
	})
	if err == nil {
		return nil
	}
	var cleanupErrors []error
	for _, path := range createdPaths {
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("remove rolled-back default wordlist %s: %w", filepath.Base(path), removeErr))
		}
	}
	return errors.Join(append([]error{err}, cleanupErrors...)...)
}

func requireUninitializedWordlistStorage(basePath string) error {
	_, err := os.Lstat(basePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect default wordlist storage: %w", err)
	}
	// The directory itself is the durable evidence that a previous import was
	// attempted. Treat even an empty directory as state instead of reseeding it.
	return errors.New("default wordlist storage already exists without catalog state; explicit repair or reset is required")
}

func validatePersistedDefaultWordlist(basePath string, item defaultWordlistImport, wordlist catalogdomain.Wordlist) error {
	expectedPath := filepath.Join(basePath, item.entry.FileName)
	if wordlist.Description != item.entry.Description ||
		!slices.Equal(wordlist.Tags, item.entry.Tags) ||
		wordlist.FilePath != expectedPath ||
		wordlist.FileSize != item.fileSize ||
		wordlist.LineCount != item.lineCount ||
		wordlist.FileHash != item.fileHash {
		return fmt.Errorf("default wordlist %s catalog metadata drifted; explicit repair or reset is required", item.entry.FileName)
	}
	metadata, err := inspectDefaultWordlistFile(expectedPath)
	if err != nil {
		return fmt.Errorf("validate persisted default wordlist %s: %w", item.entry.FileName, err)
	}
	if metadata.FileSize != item.fileSize || metadata.LineCount != item.lineCount || metadata.FileHash != item.fileHash {
		return fmt.Errorf("default wordlist %s file content drifted; explicit repair or reset is required", item.entry.FileName)
	}
	return nil
}

func readDefaultWordlistImports(manifestPath, sourceDirectory string) ([]defaultWordlistImport, error) {
	manifestPath = strings.TrimSpace(manifestPath)
	sourceDirectory = strings.TrimSpace(sourceDirectory)
	if manifestPath == "" || !filepath.IsAbs(manifestPath) {
		return nil, errors.New("wordlist bootstrap manifest path must be absolute")
	}
	if sourceDirectory == "" || !filepath.IsAbs(sourceDirectory) {
		return nil, errors.New("wordlist bootstrap source directory must be absolute")
	}
	manifestInfo, err := os.Lstat(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("inspect wordlist bootstrap manifest: %w", err)
	}
	if !manifestInfo.Mode().IsRegular() {
		return nil, errors.New("wordlist bootstrap manifest must be a regular file")
	}
	sourceInfo, err := os.Stat(sourceDirectory)
	if err != nil {
		return nil, fmt.Errorf("inspect wordlist bootstrap source directory: %w", err)
	}
	if !sourceInfo.IsDir() {
		return nil, errors.New("wordlist bootstrap source path must be a directory")
	}

	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("open wordlist bootstrap manifest: %w", err)
	}
	defer func() { _ = manifestFile.Close() }()
	decoder := json.NewDecoder(io.LimitReader(manifestFile, 1<<20))
	decoder.DisallowUnknownFields()
	var manifest defaultWordlistManifest
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode wordlist bootstrap manifest: %w", err)
	}
	if len(manifest.Wordlists) == 0 {
		return nil, errors.New("wordlist bootstrap manifest must contain at least one default wordlist")
	}

	imports := make([]defaultWordlistImport, 0, len(manifest.Wordlists))
	seen := make(map[string]struct{}, len(manifest.Wordlists))
	for _, entry := range manifest.Wordlists {
		fileName, err := catalogdomain.ValidateWordlistFileName(entry.FileName)
		if err != nil {
			return nil, fmt.Errorf("wordlist bootstrap manifest fileName: %w", err)
		}
		if !entry.Required {
			return nil, fmt.Errorf("wordlist bootstrap manifest marks %s as optional", fileName)
		}
		if _, exists := seen[fileName]; exists {
			return nil, fmt.Errorf("wordlist bootstrap manifest repeats %s", fileName)
		}
		seen[fileName] = struct{}{}
		sourcePath := filepath.Join(sourceDirectory, fileName)
		fileInfo, err := os.Lstat(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("inspect default wordlist %s: %w", fileName, err)
		}
		if !fileInfo.Mode().IsRegular() || fileInfo.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("default wordlist %s must be a regular non-symlink file", fileName)
		}
		if fileInfo.Size() == 0 {
			return nil, fmt.Errorf("default wordlist %s must not be empty", fileName)
		}
		entry.FileName = fileName
		entry.Description = catalogdomain.NormalizeWordlistDescription(entry.Description)
		entry.Tags = catalogdomain.NormalizeWordlistTags(entry.Tags)
		metadata, err := inspectDefaultWordlistFile(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("inspect default wordlist %s metadata: %w", fileName, err)
		}
		imports = append(imports, defaultWordlistImport{
			entry:      entry,
			sourcePath: sourcePath,
			fileSize:   metadata.FileSize,
			lineCount:  metadata.LineCount,
			fileHash:   metadata.FileHash,
		})
	}
	return imports, nil
}

func inspectDefaultWordlistFile(path string) (*catalogapp.WordlistFileMetadata, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("wordlist must be a regular non-symlink file")
	}
	metadata, changed, err := catalogapp.NewLocalWordlistFileStore().RefreshMetadata(path, -1, time.Time{})
	if err != nil {
		return nil, err
	}
	if !changed || metadata == nil {
		return nil, errors.New("wordlist metadata inspection returned no result")
	}
	return metadata, nil
}
