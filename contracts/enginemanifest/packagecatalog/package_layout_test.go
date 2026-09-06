package packagecatalog

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
)

func TestRequiredEnginePackagePathsAreCanonicalFourFileLayout(t *testing.T) {
	want := []string{
		"package.json",
		"engine.json",
		"locales/en.json",
		"locales/zh.json",
	}
	got := RequiredEnginePackagePaths()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RequiredEnginePackagePaths() = %#v, want %#v", got, want)
	}
	got[0] = "mutated"
	if reflect.DeepEqual(RequiredEnginePackagePaths(), got) {
		t.Fatal("RequiredEnginePackagePaths returned mutable shared state")
	}
	if localePaths := enginecontract.RequiredLocaleResourcePaths(); !reflect.DeepEqual(localePaths, want[2:]) {
		t.Fatalf("RequiredLocaleResourcePaths() = %#v, want %#v", localePaths, want[2:])
	}
	locales := enginecontract.RequiredLocales()
	if wantLocales := []string{"en", "zh"}; !reflect.DeepEqual(locales, wantLocales) {
		t.Fatalf("RequiredLocales() = %#v, want %#v", locales, wantLocales)
	}
	locales[0] = "mutated"
	if reflect.DeepEqual(enginecontract.RequiredLocales(), locales) {
		t.Fatal("RequiredLocales returned mutable shared state")
	}
}

func TestDecodeEnginePackageLayoutAcceptsExactLayoutAndDerivedLocaleKeys(t *testing.T) {
	entries := validEnginePackageLayoutEntries()
	layout, err := DecodeEnginePackageLayout(entries, "demo.lfengine.tar.gz")
	if err != nil {
		t.Fatalf("DecodeEnginePackageLayout() error = %v", err)
	}
	if layout.Definition.PackageManifest.EngineID != "engine.lunafox.subdomain_discovery" {
		t.Fatalf("package engineId = %q", layout.Definition.PackageManifest.EngineID)
	}
	if layout.Definition.EngineDefinition.Execution.EngineAPIMajor != 2 {
		t.Fatalf("engineApiMajor = %d", layout.Definition.EngineDefinition.Execution.EngineAPIMajor)
	}
	for _, locale := range []string{"en", "zh"} {
		if _, exists := layout.LocaleResources[locale]; !exists {
			t.Fatalf("validated locale %q is missing from result", locale)
		}
	}

	// A validated view must not retain the caller's mutable payload buffers.
	copy(entries[2].Payload, []byte("mutated"))
	if displayName, ok := enginecontract.ResolveLocaleString(layout.LocaleResources["en"], enginecontract.EngineDisplayNameLocalizationKey); !ok || displayName != "Demo" {
		t.Fatalf("validated locale view aliases input payload: displayName = %q, ok = %v", displayName, ok)
	}
}

func TestDecodeEnginePackageLayoutRejectsEveryMissingRequiredFile(t *testing.T) {
	for _, missingPath := range RequiredEnginePackagePaths() {
		t.Run(missingPath, func(t *testing.T) {
			entries := validEnginePackageLayoutEntries()
			for index, entry := range entries {
				if entry.Path == missingPath {
					entries = append(entries[:index], entries[index+1:]...)
					break
				}
			}
			_, err := DecodeEnginePackageLayout(entries, "package")
			if err == nil || !strings.Contains(err.Error(), "missing required package v2 file") {
				t.Fatalf("DecodeEnginePackageLayout() error = %v, want missing file rejection", err)
			}
		})
	}
}

func TestDecodeEnginePackageLayoutRejectsUnexpectedLegacyAndExecutableEntries(t *testing.T) {
	unexpectedPaths := []string{
		"checksums.txt",
		"artifacts/schemas/config.schema.json",
		"artifacts/results/asset.subdomain.v1.schema.json",
		"artifacts/result-schemas/asset.subdomain.v1.json",
		"artifacts/runtimes/demo.runtime.json",
		"artifacts/runtimes/demo.runtime.v2.json",
		"runtime.v1.json",
		"runtime.v2.json",
		"runtime-bundles/demo.tar.gz",
		"bin/engine",
		"engine.runtime.json",
		"locale/en.json",
		"locales/fr.json",
	}
	for _, unexpectedPath := range unexpectedPaths {
		t.Run(unexpectedPath, func(t *testing.T) {
			entries := append(validEnginePackageLayoutEntries(), PackageLayoutEntry{
				Path:    unexpectedPath,
				Mode:    0o644,
				Payload: []byte("unexpected"),
			})
			_, err := DecodeEnginePackageLayout(entries, "package")
			if err == nil || !strings.Contains(err.Error(), "unexpected package v2 file") {
				t.Fatalf("DecodeEnginePackageLayout() error = %v, want unexpected file rejection", err)
			}
		})
	}

	entries := validEnginePackageLayoutEntries()
	entries[1].Mode = 0o755
	_, err := DecodeEnginePackageLayout(entries, "package")
	if err == nil || !strings.Contains(err.Error(), "must not be executable") {
		t.Fatalf("DecodeEnginePackageLayout() error = %v, want executable rejection", err)
	}
}

func TestDecodeEnginePackageLayoutRejectsUnsafeNonCanonicalAndDuplicatePaths(t *testing.T) {
	unsafePaths := []string{
		"../package.json",
		"/package.json",
		"./package.json",
		"locales/../engine.json",
		`locales\en.json`,
		" package.json",
		"C:/package.json",
		"C:package.json",
		"package.json\x00",
		"package.json\n",
	}
	for _, unsafePath := range unsafePaths {
		t.Run(unsafePath, func(t *testing.T) {
			entries := validEnginePackageLayoutEntries()
			entries[0].Path = unsafePath
			_, err := DecodeEnginePackageLayout(entries, "package")
			if err == nil || !strings.Contains(err.Error(), "canonical safe relative path") {
				t.Fatalf("DecodeEnginePackageLayout() error = %v, want unsafe path rejection", err)
			}
		})
	}

	entries := append(validEnginePackageLayoutEntries(), validEnginePackageLayoutEntries()[0])
	_, err := DecodeEnginePackageLayout(entries, "package")
	if err == nil || !strings.Contains(err.Error(), "duplicate package v2 file") {
		t.Fatalf("DecodeEnginePackageLayout() error = %v, want duplicate path rejection", err)
	}
}

func TestDecodeEnginePackageLayoutRejectsNonRegularEntries(t *testing.T) {
	entries := validEnginePackageLayoutEntries()
	entries[2].Mode = fs.ModeSymlink | 0o777

	_, err := DecodeEnginePackageLayout(entries, "package")
	if err == nil || !strings.Contains(err.Error(), "must be a regular file") {
		t.Fatalf("DecodeEnginePackageLayout() error = %v, want non-regular entry rejection", err)
	}
}

func TestDecodeEnginePackageLayoutRejectsForbiddenPackageManifestSiblings(t *testing.T) {
	fields := map[string]string{
		"files":                  `[]`,
		"checksums":              `"checksums.txt"`,
		"locales":                `{"en":"translations/en.json"}`,
		"localePaths":            `["translations/en.json","translations/zh.json"]`,
		"packageDigest":          `"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
		"artifactManifestDigest": `"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
		"runtimeImageDigest":     `"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
	}
	for field, value := range fields {
		t.Run(field, func(t *testing.T) {
			entries := validEnginePackageLayoutEntries()
			entries[0].Payload = addJSONFieldForPackageLayoutTest(entries[0].Payload, field, value)
			_, err := DecodeEnginePackageLayout(entries, "package")
			if err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("DecodeEnginePackageLayout() error = %v, want forbidden sibling rejection", err)
			}
		})
	}
}

func TestDecodeEnginePackageLayoutRejectsForbiddenEngineManifestSiblings(t *testing.T) {
	fields := map[string]string{
		"runtimeRef":       `"runtime.subdomain_discovery"`,
		"runtimeSource":    `"runtime.subdomain_discovery"`,
		"configSchema":     `{"ref":"schema.engine.subdomain_discovery"}`,
		"configSchemaRef":  `"schema.engine.subdomain_discovery"`,
		"configSchemaPath": `"artifacts/schemas/subdomain_discovery.schema.json"`,
		"$id":              `"schema.engine.subdomain_discovery"`,
		"install":          `{"command":"bin/engine"}`,
	}
	for field, value := range fields {
		t.Run(field, func(t *testing.T) {
			entries := validEnginePackageLayoutEntries()
			entries[1].Payload = addJSONFieldForPackageLayoutTest(entries[1].Payload, field, value)
			_, err := DecodeEnginePackageLayout(entries, "package")
			if err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("DecodeEnginePackageLayout() error = %v, want forbidden sibling rejection", err)
			}
		})
	}
}

func TestDecodeEnginePackageLayoutRejectsInvalidOrIncompleteLocales(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{name: "array", payload: `[]`, want: "must be an object"},
		{name: "null", payload: `null`, want: "must be an object"},
		{name: "missing derived param key", payload: `{"engine":{"displayName":"Demo","description":"Demo"},"sections":{"scan":{"name":"Scan","description":"Scan","params":{}}}}`, want: "sections.scan.params.wordlist.description"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries := validEnginePackageLayoutEntries()
			entries[2].Payload = []byte(test.payload)
			_, err := DecodeEnginePackageLayout(entries, "package")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("DecodeEnginePackageLayout() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestLoadEnginePackageLayoutFromRootUsesSameClosedContract(t *testing.T) {
	root := writeEnginePackageLayoutRoot(t, validEnginePackageLayoutEntries())

	layout, err := LoadEnginePackageLayoutFromRoot(root, 1<<20)
	if err != nil {
		t.Fatalf("LoadEnginePackageLayoutFromRoot() error = %v", err)
	}
	if layout.Definition.EngineDefinition.EngineID != "engine.lunafox.subdomain_discovery" {
		t.Fatalf("loaded engineId = %q", layout.Definition.EngineDefinition.EngineID)
	}
}

func TestLoadEnginePackageLayoutFromRootRejectsLegacyRuntimeManifestPath(t *testing.T) {
	root := writeEnginePackageLayoutRoot(t, validEnginePackageLayoutEntries())
	legacyPath := filepath.Join(root, "artifacts", "runtimes", "demo.runtime.json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, []byte(`{"manifestVersion":"runtime.v1","engineApiMajor":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadEnginePackageLayoutFromRoot(root, 1<<20); err == nil || !strings.Contains(err.Error(), "unexpected package v2 directory") {
		t.Fatalf("LoadEnginePackageLayoutFromRoot() error = %v, want legacy runtime manifest rejection", err)
	}
}

func TestLoadEnginePackageLayoutFromRootRejectsUnexpectedDirectoriesAndFiles(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, string)
		want  string
	}{
		{
			name: "unexpected file",
			setup: func(t *testing.T, root string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, "checksums.txt"), []byte("sum"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: "unexpected package v2 file",
		},
		{
			name: "unexpected directory",
			setup: func(t *testing.T, root string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(root, "artifacts"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			want: "unexpected package v2 directory",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := writeEnginePackageLayoutRoot(t, validEnginePackageLayoutEntries())
			test.setup(t, root)
			_, err := LoadEnginePackageLayoutFromRoot(root, 1<<20)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("LoadEnginePackageLayoutFromRoot() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestLoadEnginePackageLayoutFromRootBoundsReadsAndRejectsMembershipFirst(t *testing.T) {
	t.Run("bounded required payload", func(t *testing.T) {
		root := writeEnginePackageLayoutRoot(t, validEnginePackageLayoutEntries())
		_, err := LoadEnginePackageLayoutFromRoot(root, 1)
		if err == nil || !strings.Contains(err.Error(), "payload exceeds maximum size") {
			t.Fatalf("LoadEnginePackageLayoutFromRoot() error = %v, want bounded-read rejection", err)
		}
	})

	t.Run("unexpected member before read", func(t *testing.T) {
		root := writeEnginePackageLayoutRoot(t, validEnginePackageLayoutEntries())
		if err := os.WriteFile(filepath.Join(root, "0-unexpected.json"), make([]byte, 1024), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadEnginePackageLayoutFromRoot(root, 1)
		if err == nil || !strings.Contains(err.Error(), "unexpected package v2 file") {
			t.Fatalf("LoadEnginePackageLayoutFromRoot() error = %v, want membership rejection before payload read", err)
		}
	})
}

func TestLoadEnginePackageLayoutFromRootRejectsExecutableAndSymlinkEntries(t *testing.T) {
	t.Run("executable", func(t *testing.T) {
		root := writeEnginePackageLayoutRoot(t, validEnginePackageLayoutEntries())
		if err := os.Chmod(filepath.Join(root, "engine.json"), 0o755); err != nil {
			t.Fatal(err)
		}
		_, err := LoadEnginePackageLayoutFromRoot(root, 1<<20)
		if err == nil || !strings.Contains(err.Error(), "must not be executable") {
			t.Fatalf("LoadEnginePackageLayoutFromRoot() error = %v", err)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		root := writeEnginePackageLayoutRoot(t, validEnginePackageLayoutEntries())
		enginePath := filepath.Join(root, "engine.json")
		if err := os.Remove(enginePath); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(root, "package.json"), enginePath); err != nil {
			t.Fatal(err)
		}
		_, err := LoadEnginePackageLayoutFromRoot(root, 1<<20)
		if err == nil || !strings.Contains(err.Error(), "must be a regular file") {
			t.Fatalf("LoadEnginePackageLayoutFromRoot() error = %v", err)
		}
	})
}

func TestLoadEnginePackageLayoutFromRootRejectsSymlinkRoot(t *testing.T) {
	realRoot := writeEnginePackageLayoutRoot(t, validEnginePackageLayoutEntries())
	linkRoot := filepath.Join(t.TempDir(), "package-link")
	if err := os.Symlink(realRoot, linkRoot); err != nil {
		t.Fatal(err)
	}

	_, err := LoadEnginePackageLayoutFromRoot(linkRoot, 1<<20)
	if err == nil || !strings.Contains(err.Error(), "must be a real directory") {
		t.Fatalf("LoadEnginePackageLayoutFromRoot() error = %v", err)
	}
}

func validEnginePackageLayoutEntries() []PackageLayoutEntry {
	locale := []byte(`{
  "engine":{"displayName":"Demo","description":"Demo engine"},
  "sections":{"scan":{"name":"Scan","description":"Scan settings","params":{"wordlist":{"description":"Wordlist"}}}}
}`)
	return []PackageLayoutEntry{
		{
			Path: "package.json", Mode: 0o644,
			Payload: validPackageManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		},
		{
			Path: "engine.json", Mode: 0o644,
			Payload: validEngineManifestForCatalogTest("engine.lunafox.subdomain_discovery"),
		},
		{Path: "locales/en.json", Mode: 0o644, Payload: append([]byte(nil), locale...)},
		{Path: "locales/zh.json", Mode: 0o644, Payload: append([]byte(nil), locale...)},
	}
}

func addJSONFieldForPackageLayoutTest(payload []byte, field, value string) []byte {
	text := strings.TrimSpace(string(payload))
	return []byte(strings.TrimSuffix(text, "}") + `,"` + field + `":` + value + `}`)
}

func writeEnginePackageLayoutRoot(t *testing.T, entries []PackageLayoutEntry) string {
	t.Helper()
	root := t.TempDir()
	for _, entry := range entries {
		filePath := filepath.Join(root, filepath.FromSlash(entry.Path))
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filePath, entry.Payload, entry.Mode.Perm()); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
