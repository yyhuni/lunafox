package packagemanifest

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CleanPackageRelativePath returns a clean path that stays inside a engine package.
func CleanPackageRelativePath(relativePath string) (string, error) {
	cleanPath := filepath.Clean(strings.TrimSpace(relativePath))
	if cleanPath == "" || cleanPath == "." || filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) || cleanPath == ".." {
		return "", fmt.Errorf("engine package artifact path %q must stay within package", relativePath)
	}
	return cleanPath, nil
}
