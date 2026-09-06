package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type fingerprintFixtureManifest struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Source        fingerprintSource    `json:"source"`
	FullFixtures  []fingerprintFixture `json:"fullFixtures"`
}

type fingerprintSource struct {
	Repository  string `json:"repository"`
	Revision    string `json:"revision"`
	ExtractedAt string `json:"extractedAt"`
	Purpose     string `json:"purpose"`
}

type fingerprintFixture struct {
	Library        string                    `json:"library"`
	Path           string                    `json:"path"`
	SourcePath     string                    `json:"sourcePath"`
	Format         string                    `json:"format"`
	FormatVersion  string                    `json:"formatVersion"`
	SHA256         string                    `json:"sha256"`
	Statistics     fingerprintFixtureStats   `json:"statistics"`
	ExpectedImport fingerprintExpectedImport `json:"expectedImport"`
}

type fingerprintFixtureStats struct {
	RawRecordCount                    int `json:"rawRecordCount"`
	FinalIdentityCount                int `json:"finalIdentityCount"`
	SameNameAndRuleDuplicateRecords   int `json:"sameNameAndRuleDuplicateRecords"`
	NativeIDCount                     int `json:"nativeIDCount"`
	DuplicateNativeIDGroups           int `json:"duplicateNativeIDGroups"`
	DifferentHTTPRuleVariantsRetained int `json:"differentHTTPRuleVariantsRetained"`
	SameRuleMetadataVariantsCollapsed int `json:"sameRuleMetadataVariantsCollapsed"`
}

type fingerprintExpectedImport struct {
	Outcome      string                  `json:"outcome"`
	FirstImport  fingerprintImportCounts `json:"firstImport"`
	Reimport     fingerprintImportCounts `json:"reimport"`
	NativeExport string                  `json:"nativeExport"`
}

type fingerprintImportCounts struct {
	CreatedCount   int `json:"createdCount"`
	UpdatedCount   int `json:"updatedCount"`
	UnchangedCount int `json:"unchangedCount"`
}

type smallFingerprintFixtureManifest struct {
	SchemaVersion int                       `json:"schemaVersion"`
	Fixtures      []smallFingerprintFixture `json:"fixtures"`
}

type smallFingerprintFixture struct {
	ID         string                             `json:"id"`
	Library    string                             `json:"library"`
	Path       string                             `json:"path"`
	Outcome    string                             `json:"outcome"`
	Scenario   string                             `json:"scenario"`
	Diagnostic *smallFingerprintFixtureDiagnostic `json:"diagnostic"`
}

type smallFingerprintFixtureDiagnostic struct {
	Kind        string `json:"kind"`
	RecordIndex int    `json:"recordIndex"`
	FieldPath   string `json:"fieldPath"`
	Reason      string `json:"reason"`
}

func fingerprintFixtureRoot(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate fingerprint fixture test source")
	}
	return filepath.Join(filepath.Dir(sourceFile), "testdata")
}

func readFingerprintFixture(t *testing.T, root, relativePath string) []byte {
	t.Helper()
	path := managedFixturePath(t, root, relativePath)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read managed fixture %q: %v", relativePath, err)
	}
	return contents
}

func managedFixturePath(t *testing.T, root, relativePath string) string {
	t.Helper()
	cleanPath := filepath.Clean(relativePath)
	if relativePath == "" || filepath.IsAbs(relativePath) || cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		t.Fatalf("fixture path must remain under project testdata: %q", relativePath)
	}
	path := filepath.Join(root, cleanPath)
	relativeToRoot, err := filepath.Rel(root, path)
	if err != nil || relativeToRoot == ".." || strings.HasPrefix(relativeToRoot, ".."+string(filepath.Separator)) {
		t.Fatalf("fixture path escaped project testdata: %q", relativePath)
	}
	return path
}

func loadFullFingerprintFixtureManifest(t *testing.T, root string) fingerprintFixtureManifest {
	t.Helper()
	contents := readFingerprintFixture(t, root, "manifest.json")
	var manifest fingerprintFixtureManifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatalf("parse fingerprint fixture manifest: %v", err)
	}
	if manifest.SchemaVersion != 1 {
		t.Fatalf("fixture manifest schemaVersion = %d, want 1", manifest.SchemaVersion)
	}
	if manifest.Source.Repository == "" || manifest.Source.Revision == "" || manifest.Source.ExtractedAt == "" || manifest.Source.Purpose == "" {
		t.Fatalf("fixture manifest source metadata is incomplete: %+v", manifest.Source)
	}
	if len(manifest.FullFixtures) != 1 {
		t.Fatalf("fixture manifest contains %d full fixtures, want 1", len(manifest.FullFixtures))
	}
	return manifest
}

func loadSmallFingerprintFixtureManifest(t *testing.T, root string) smallFingerprintFixtureManifest {
	t.Helper()
	contents := readFingerprintFixture(t, root, "small/manifest.json")
	var manifest smallFingerprintFixtureManifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatalf("parse small fingerprint fixture manifest: %v", err)
	}
	if manifest.SchemaVersion != 1 || len(manifest.Fixtures) == 0 {
		t.Fatalf("small fixture manifest is incomplete: schemaVersion=%d fixtures=%d", manifest.SchemaVersion, len(manifest.Fixtures))
	}
	return manifest
}

func fixtureSHA256(contents []byte) string {
	digest := sha256.Sum256(contents)
	return hex.EncodeToString(digest[:])
}
