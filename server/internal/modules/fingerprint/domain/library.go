// Package domain defines the supported native fingerprint libraries and their
// persistence-independent import contracts.
package domain

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Library identifies one supported native fingerprint format.
type Library string

const LibraryFingerPrintHub Library = "fingerprinthub"

var supportedLibraries = [...]Library{
	LibraryFingerPrintHub,
}

const fingerprintResourcePrefix = "fingerprintLibraries"

// ParseLibrary validates the route-level library discriminator.
func ParseLibrary(value string) (Library, error) {
	library := Library(strings.ToLower(strings.TrimSpace(value)))
	if !library.IsSupported() {
		return "", fmt.Errorf("unsupported fingerprint library %q", value)
	}
	return library, nil
}

// IsSupported reports whether the library is part of the fixed fingerprint
// library collection.
func (library Library) IsSupported() bool {
	for _, supported := range supportedLibraries {
		if library == supported {
			return true
		}
	}
	return false
}

// Values returns the accepted route values in their stable API order.
func Values() []Library {
	return append([]Library(nil), supportedLibraries[:]...)
}

// CanonicalName creates the only externally visible identity for a persisted
// fingerprint. resourceID is an immutable server-generated UUID, never a
// standalone response field.
func (library Library) CanonicalName(resourceID string) string {
	return fmt.Sprintf("%s/%s/fingerprints/%s", fingerprintResourcePrefix, library, resourceID)
}

// ParseCanonicalName validates and decomposes a canonical fingerprint name.
func ParseCanonicalName(name string) (Library, string, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 4 || parts[0] != fingerprintResourcePrefix || parts[2] != "fingerprints" || strings.TrimSpace(parts[3]) == "" {
		return "", "", fmt.Errorf("invalid fingerprint resource name")
	}

	library, err := ParseLibrary(parts[1])
	if err != nil {
		return "", "", fmt.Errorf("invalid fingerprint resource name: %w", err)
	}
	if _, err := uuid.Parse(parts[3]); err != nil {
		return "", "", fmt.Errorf("invalid fingerprint resource name: resource id must be UUID")
	}
	return library, parts[3], nil
}

// FileExtension returns the native export extension for the library.
func (library Library) FileExtension() string {
	return "json"
}

// CanonicalFilename is the native filename shared by the HTTP download and
// Agent platform resource. It is a library contract, not a client-side label.
func (library Library) CanonicalFilename() string {
	if library == LibraryFingerPrintHub {
		return "fingerprinthub_web.json"
	}
	return ""
}

// NativeContentType returns the media type used for native exports.
func (library Library) NativeContentType() string {
	return "application/json"
}
