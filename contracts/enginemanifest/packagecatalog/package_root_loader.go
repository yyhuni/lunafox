package packagecatalog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LoadEnginePackageLayoutFromRoot is the filesystem adapter for the same
// closed four-file contract used by archive validators. It does not follow
// symlinks, infer missing files from sibling package roots, or perform
// unbounded reads from a damaged expanded cache.
func LoadEnginePackageLayoutFromRoot(packageRoot string, maxPayloadBytes int64) (EnginePackageLayout, error) {
	packageRoot = strings.TrimSpace(packageRoot)
	if packageRoot == "" {
		return EnginePackageLayout{}, fmt.Errorf("package v2 root is required")
	}
	if maxPayloadBytes <= 0 {
		return EnginePackageLayout{}, fmt.Errorf("package v2 payload size limit must be positive")
	}
	rootInfo, err := os.Lstat(packageRoot)
	if err != nil {
		return EnginePackageLayout{}, fmt.Errorf("inspect package v2 root %q: %w", packageRoot, err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return EnginePackageLayout{}, fmt.Errorf("package v2 root %q must be a real directory", packageRoot)
	}

	requiredPaths := RequiredEnginePackagePaths()
	required := make(map[string]struct{}, len(requiredPaths))
	for _, requiredPath := range requiredPaths {
		required[requiredPath] = struct{}{}
	}
	entries := make([]PackageLayoutEntry, 0, len(requiredPaths))
	var totalPayloadBytes int64
	err = filepath.WalkDir(packageRoot, func(currentPath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relativePath, err := filepath.Rel(packageRoot, currentPath)
		if err != nil {
			return fmt.Errorf("resolve package v2 path %q: %w", currentPath, err)
		}
		if relativePath == "." {
			return nil
		}
		canonicalPath := filepath.ToSlash(relativePath)
		if entry.IsDir() {
			if canonicalPath != "locales" {
				return fmt.Errorf("unexpected package v2 directory %q", canonicalPath)
			}
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspect package v2 file %q: %w", canonicalPath, err)
		}
		layoutEntry := PackageLayoutEntry{Path: canonicalPath, Mode: info.Mode()}
		if err := ValidatePackageLayoutEntry(layoutEntry); err != nil {
			return err
		}
		if _, exists := required[canonicalPath]; !exists {
			return fmt.Errorf("unexpected package v2 file %q", canonicalPath)
		}
		if info.Size() < 0 || info.Size() > maxPayloadBytes-totalPayloadBytes {
			return fmt.Errorf("package v2 payload exceeds maximum size %d", maxPayloadBytes)
		}
		payload, err := readPackageLayoutFile(currentPath, info.Size(), maxPayloadBytes-totalPayloadBytes)
		if err != nil {
			return fmt.Errorf("read package v2 file %q: %w", canonicalPath, err)
		}
		totalPayloadBytes += int64(len(payload))
		entries = append(entries, PackageLayoutEntry{
			Path:    canonicalPath,
			Mode:    info.Mode(),
			Payload: payload,
		})
		return nil
	})
	if err != nil {
		return EnginePackageLayout{}, fmt.Errorf("walk package v2 root %q: %w", packageRoot, err)
	}
	return DecodeEnginePackageLayout(entries, packageRoot)
}

func readPackageLayoutFile(path string, expectedSize, maxBytes int64) (payload []byte, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close package v2 file: %w", closeErr)
		}
	}()
	payload, err = io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > maxBytes {
		return nil, fmt.Errorf("payload exceeds remaining size limit %d", maxBytes)
	}
	if int64(len(payload)) != expectedSize {
		return nil, fmt.Errorf("file size changed during validation: got %d want %d", len(payload), expectedSize)
	}
	return payload, nil
}
