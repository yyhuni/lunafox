package packagecatalog

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
	"unicode"

	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
)

const (
	PackageManifestPath = "package.json"
	EngineManifestPath  = "engine.json"
)

// PackageLayoutEntry is the passive input to package-v2 layout validation.
// Archive and filesystem readers remain responsible for bounded reads and for
// translating their native entry metadata into this contract shape.
type PackageLayoutEntry struct {
	Path    string
	Mode    fs.FileMode
	Payload []byte
}

// EnginePackageLayout is a detached, validated four-file package view. It
// deliberately carries no archive, artifact-manifest, or Runtime Image digest
// sibling identity.
type EnginePackageLayout struct {
	Definition      EnginePackageDefinition
	LocaleResources map[string]map[string]any
}

// RequiredEnginePackagePaths returns the exact closed package layout in
// canonical archive order.
func RequiredEnginePackagePaths() []string {
	paths := []string{PackageManifestPath, EngineManifestPath}
	paths = append(paths, enginecontract.RequiredLocaleResourcePaths()...)
	return paths
}

// DecodeEnginePackageLayout validates entry safety and exact presence before
// decoding any authoring payload, then delegates package.json and engine.json
// semantics to their contracts-owned strict v2 decoders.
func DecodeEnginePackageLayout(entries []PackageLayoutEntry, source string) (EnginePackageLayout, error) {
	requiredPaths := RequiredEnginePackagePaths()
	required := make(map[string]struct{}, len(requiredPaths))
	for _, requiredPath := range requiredPaths {
		required[requiredPath] = struct{}{}
	}

	payloads := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		if err := ValidatePackageLayoutEntry(entry); err != nil {
			return EnginePackageLayout{}, fmt.Errorf("validate package v2 layout %q: %w", source, err)
		}
		if _, exists := required[entry.Path]; !exists {
			return EnginePackageLayout{}, fmt.Errorf("validate package v2 layout %q: unexpected package v2 file %q", source, entry.Path)
		}
		if _, duplicate := payloads[entry.Path]; duplicate {
			return EnginePackageLayout{}, fmt.Errorf("validate package v2 layout %q: duplicate package v2 file %q", source, entry.Path)
		}
		payloads[entry.Path] = append([]byte(nil), entry.Payload...)
	}
	for _, requiredPath := range requiredPaths {
		if _, exists := payloads[requiredPath]; !exists {
			return EnginePackageLayout{}, fmt.Errorf("validate package v2 layout %q: missing required package v2 file %q", source, requiredPath)
		}
	}

	definition, err := DecodeEnginePackageDefinition(
		payloads[PackageManifestPath],
		payloads[EngineManifestPath],
		packageLayoutSource(source, PackageManifestPath),
		packageLayoutSource(source, EngineManifestPath),
	)
	if err != nil {
		return EnginePackageLayout{}, err
	}

	requiredLocales := enginecontract.RequiredLocales()
	locales := make(map[string]map[string]any, len(requiredLocales))
	for _, locale := range requiredLocales {
		localePath, err := enginecontract.LocaleResourcePath(locale)
		if err != nil {
			return EnginePackageLayout{}, err
		}
		resource, err := enginecontract.DecodeLocaleResource(payloads[localePath], packageLayoutSource(source, localePath))
		if err != nil {
			return EnginePackageLayout{}, err
		}
		if err := enginecontract.ValidateLocaleResourceKeys(definition.EngineDefinition, locale, resource); err != nil {
			return EnginePackageLayout{}, err
		}
		locales[locale] = resource
	}

	return EnginePackageLayout{
		Definition:      cloneEnginePackageDefinition(definition),
		LocaleResources: locales,
	}, nil
}

// ValidatePackageLayoutEntry lets bounded archive and filesystem adapters
// apply the same path and file-type rule before reading or writing payloads.
// Exact membership and package semantics remain owned by
// DecodeEnginePackageLayout.
func ValidatePackageLayoutEntry(entry PackageLayoutEntry) error {
	if !isCanonicalSafePackagePath(entry.Path) {
		return fmt.Errorf("package v2 file path %q must be a canonical safe relative path", entry.Path)
	}
	if !entry.Mode.IsRegular() {
		return fmt.Errorf("package v2 file %q must be a regular file", entry.Path)
	}
	if entry.Mode.Perm()&0o111 != 0 {
		return fmt.Errorf("package v2 file %q must not be executable", entry.Path)
	}
	return nil
}

func isCanonicalSafePackagePath(value string) bool {
	if value == "" || value != strings.TrimSpace(value) || strings.Contains(value, "\\") || path.IsAbs(value) {
		return false
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 || hasWindowsDrivePrefix(value) {
		return false
	}
	cleaned := path.Clean(value)
	if cleaned != value || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return false
	}
	return true
}

func hasWindowsDrivePrefix(value string) bool {
	if len(value) < 2 || value[1] != ':' {
		return false
	}
	first := value[0]
	return first >= 'a' && first <= 'z' || first >= 'A' && first <= 'Z'
}

func packageLayoutSource(source, entryPath string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return entryPath
	}
	return source + ":" + entryPath
}
