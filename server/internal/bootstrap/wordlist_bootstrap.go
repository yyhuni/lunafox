package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
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
}

// RunWordlistBootstrap copies the immutable default wordlists into shared
// storage and records them through the Catalog boundary. It deliberately
// rejects existing entries: bootstrap is fresh-install-only, so silently
// mixing a previous catalog with a new release resource set is unsafe.
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

	service := catalogapp.NewWordlistCommandService(
		catalogrepo.NewWordlistRepository(db),
		basePath,
		catalogapp.NewLocalWordlistFileStore(),
	)
	for _, item := range imports {
		file, err := os.Open(item.sourcePath)
		if err != nil {
			return fmt.Errorf("open default wordlist %s: %w", item.entry.FileName, err)
		}
		_, createErr := service.CreateWordlist(
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
		if closeErr != nil {
			return fmt.Errorf("close default wordlist %s: %w", item.entry.FileName, closeErr)
		}
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
		imports = append(imports, defaultWordlistImport{entry: entry, sourcePath: sourcePath})
	}
	return imports, nil
}
