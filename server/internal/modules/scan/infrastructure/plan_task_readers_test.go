package infrastructure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	"github.com/yyhuni/lunafox/server/internal/installedengines"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type planTaskWordlistCatalogStub struct {
	value    *domain.Wordlist
	err      error
	requests []int
}

func (stub *planTaskWordlistCatalogStub) GetByID(id int) (*domain.Wordlist, error) {
	stub.requests = append(stub.requests, id)
	return stub.value, stub.err
}

func TestConfigResourceResolverPinsExactFileMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "words.txt")
	content := []byte("one\ntwo\n")
	if err := os.WriteFile(path, content, 0o640); err != nil {
		t.Fatalf("write wordlist: %v", err)
	}
	hash := sha256.Sum256(content)
	resourceName := resourcenames.Wordlist(7)
	catalog := &planTaskWordlistCatalogStub{value: &domain.Wordlist{ID: 7, FileName: "words.txt", FilePath: path, FileSize: int64(len(content)), LineCount: 2, FileHash: hex.EncodeToString(hash[:])}}
	reader, err := NewConfigResourceResolver(catalog)
	if err != nil {
		t.Fatalf("NewConfigResourceResolver failed: %v", err)
	}
	metadata, err := reader.ResolveConfigResource(context.Background(), scanapp.ConfigResourceResolveRequest{
		Field: "configuration.steps[\"step\"].engineConfig.scan.wordlist", ResourceKind: "wordlist", ResourceName: resourceName,
	})
	if err != nil {
		t.Fatalf("ResolveConfigResource failed: %v", err)
	}
	if metadata.Resource != resourceName || metadata.Basename != "words.txt" || metadata.SizeBytes != int64(len(content)) || metadata.LineCount != 2 || metadata.SHA256 != "sha256:"+hex.EncodeToString(hash[:]) {
		t.Fatalf("unexpected pinned metadata: %#v", metadata)
	}
	if len(catalog.requests) != 1 || catalog.requests[0] != 7 {
		t.Fatalf("Catalog requests = %#v", catalog.requests)
	}
}

func TestConfigResourceResolverClassifiesCatalogFailures(t *testing.T) {
	tests := []struct {
		name      string
		catalog   *planTaskWordlistCatalogStub
		wantCause scanapp.ConfigResourceValidationCause
	}{
		{name: "missing row", catalog: &planTaskWordlistCatalogStub{err: domain.ErrWordlistNotFound}, wantCause: scanapp.ConfigResourceUnavailable},
		{name: "wrapped missing row", catalog: &planTaskWordlistCatalogStub{err: fmt.Errorf("lookup: %w", domain.ErrWordlistNotFound)}, wantCause: scanapp.ConfigResourceUnavailable},
		{name: "database unavailable", catalog: &planTaskWordlistCatalogStub{err: errors.New("database connection refused")}, wantCause: scanapp.ConfigResourceValidationUnavailable},
		{name: "nil row", catalog: &planTaskWordlistCatalogStub{}, wantCause: scanapp.ConfigResourceUnavailable},
		{name: "identity mismatch", catalog: &planTaskWordlistCatalogStub{value: &domain.Wordlist{ID: 8, FileName: "other.txt"}}, wantCause: scanapp.ConfigResourceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver, _ := NewConfigResourceResolver(test.catalog)
			_, err := resolver.ResolveConfigResource(context.Background(), wordlistResolveRequest(resourcenames.Wordlist(7)))
			assertConfigResourceCause(t, err, test.wantCause)
		})
	}
}

func TestConfigResourceResolverRejectsInvalidLogicalNameBeforeCatalogLookup(t *testing.T) {
	catalog := &planTaskWordlistCatalogStub{}
	resolver, _ := NewConfigResourceResolver(catalog)
	for _, resourceName := range []string{"", " words", "words.txt", "wordlists/0", "wordlists/7/extra", "wordlists/x", "../wordlists/7", "bad\x00name"} {
		t.Run(fmt.Sprintf("%q", resourceName), func(t *testing.T) {
			_, err := resolver.ResolveConfigResource(context.Background(), wordlistResolveRequest(resourceName))
			var inputErr *scanapp.ConfigResourceInputError
			if !errors.As(err, &inputErr) {
				t.Fatalf("error = %v, want ConfigResourceInputError", err)
			}
		})
	}
	if len(catalog.requests) != 0 {
		t.Fatalf("invalid canonical resources reached Catalog: %#v", catalog.requests)
	}
}

func TestConfigResourceResolverRejectsUnsafeOrUnavailableFiles(t *testing.T) {
	content := []byte("one\ntwo\n")
	tests := []struct {
		name string
		path func(*testing.T) string
	}{
		{name: "missing", path: func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing.txt") }},
		{name: "directory", path: func(t *testing.T) string { return t.TempDir() }},
		{name: "symlink", path: func(t *testing.T) string {
			dir := t.TempDir()
			target := filepath.Join(dir, "target.txt")
			if err := os.WriteFile(target, content, 0o640); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(dir, "link.txt")
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			return link
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := test.path(t)
			catalog := &planTaskWordlistCatalogStub{value: &domain.Wordlist{
				ID: 7, FileName: "words.txt", FilePath: path, FileSize: int64(len(content)), LineCount: 2,
				FileHash: hashWordlistContent(content),
			}}
			resolver, _ := NewConfigResourceResolver(catalog)
			_, err := resolver.ResolveConfigResource(context.Background(), wordlistResolveRequest(resourcenames.Wordlist(7)))
			assertConfigResourceCause(t, err, scanapp.ConfigResourceUnavailable)
			if strings.Contains(err.Error(), path) {
				t.Fatalf("public error leaked Server path %q: %v", path, err)
			}
		})
	}
}

func TestConfigResourceResolverRejectsUnreadableFileWhenEnforcedByOS(t *testing.T) {
	path := filepath.Join(t.TempDir(), "words.txt")
	content := []byte("one\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	if file, err := os.Open(path); err == nil {
		_ = file.Close()
		t.Skip("current test identity can read mode-000 files")
	}
	catalog := &planTaskWordlistCatalogStub{value: &domain.Wordlist{
		ID: 7, FileName: "words.txt", FilePath: path, FileSize: int64(len(content)), LineCount: 1, FileHash: hashWordlistContent(content),
	}}
	resolver, _ := NewConfigResourceResolver(catalog)
	_, err := resolver.ResolveConfigResource(context.Background(), wordlistResolveRequest(resourcenames.Wordlist(7)))
	assertConfigResourceCause(t, err, scanapp.ConfigResourceUnavailable)
}

func TestConfigResourceResolverRejectsEveryCatalogMetadataDrift(t *testing.T) {
	content := []byte("one\ntwo\n")
	tests := []struct {
		name   string
		mutate func(*domain.Wordlist)
	}{
		{name: "size", mutate: func(wordlist *domain.Wordlist) { wordlist.FileSize++ }},
		{name: "line count", mutate: func(wordlist *domain.Wordlist) { wordlist.LineCount++ }},
		{name: "digest", mutate: func(wordlist *domain.Wordlist) { wordlist.FileHash = strings.Repeat("0", 64) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "words.txt")
			if err := os.WriteFile(path, content, 0o640); err != nil {
				t.Fatal(err)
			}
			wordlist := &domain.Wordlist{
				ID: 7, FileName: "words.txt", FilePath: path, FileSize: int64(len(content)), LineCount: 2, FileHash: hashWordlistContent(content),
			}
			test.mutate(wordlist)
			resolver, _ := NewConfigResourceResolver(&planTaskWordlistCatalogStub{value: wordlist})
			_, err := resolver.ResolveConfigResource(context.Background(), wordlistResolveRequest(resourcenames.Wordlist(7)))
			assertConfigResourceCause(t, err, scanapp.ConfigResourceUnavailable)
		})
	}
}

func TestConfigResourceResolverRejectsUnsupportedKindAndMissingDependency(t *testing.T) {
	if _, err := NewConfigResourceResolver(nil); err == nil {
		t.Fatal("expected nil Catalog to fail construction")
	}
	resolver, _ := NewConfigResourceResolver(&planTaskWordlistCatalogStub{})
	_, err := resolver.ResolveConfigResource(context.Background(), scanapp.ConfigResourceResolveRequest{
		Field: "configuration.field", ResourceKind: "unknown", ResourceName: resourcenames.Wordlist(7),
	})
	assertConfigResourceCause(t, err, scanapp.ConfigResourceInternal)
}

func wordlistResolveRequest(resourceName string) scanapp.ConfigResourceResolveRequest {
	return scanapp.ConfigResourceResolveRequest{
		Field:        `configuration.steps["step"].engineConfig.scan.wordlist`,
		ResourceKind: engineexecution.ConfigResourceKindWordlist,
		ResourceName: resourceName,
	}
}

func hashWordlistContent(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func assertConfigResourceCause(t *testing.T, err error, want scanapp.ConfigResourceValidationCause) {
	t.Helper()
	var typed *scanapp.ConfigResourceValidationError
	if !errors.As(err, &typed) || typed.ConfigResourceValidationCause() != string(want) {
		t.Fatalf("error = %v, want ConfigResourceValidationError cause %q", err, want)
	}
}

func TestPlanTaskExactPackageReaderRequiresPinnedDigestMatch(t *testing.T) {
	query := scanCreateQueryStub{packages: []installedengines.ResolvedInstalledEnginePackage{{
		Registration: domain.Engine{EngineID: "engine.lunafox.port_scan", PackageDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}}}
	reader, err := NewPlanTaskExactPackageReader(query)
	if err != nil {
		t.Fatalf("NewPlanTaskExactPackageReader failed: %v", err)
	}
	// The stub intentionally lacks a valid v2 layout; the reader must fail
	// closed instead of falling back to the legacy catalog.
	if _, err := reader.LoadExactPackage(context.Background(), scanapp.PlanTaskPackageIdentity{EngineID: "engine.lunafox.port_scan", PackageDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}); err == nil {
		t.Fatal("expected incomplete exact package projection to fail")
	}
}

func TestPlanTaskReadersHonorCanceledOperationContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resolver, err := NewConfigResourceResolver(&planTaskWordlistCatalogStub{})
	if err != nil {
		t.Fatalf("NewConfigResourceResolver failed: %v", err)
	}
	if _, err := resolver.ResolveConfigResource(ctx, wordlistResolveRequest("words")); !errors.Is(err, context.Canceled) {
		t.Fatalf("ResolveConfigResource() error = %v, want context canceled", err)
	}

	reader, err := NewPlanTaskExactPackageReader(scanCreateQueryStub{})
	if err != nil {
		t.Fatalf("NewPlanTaskExactPackageReader failed: %v", err)
	}
	if _, err := reader.LoadExactPackage(ctx, scanapp.PlanTaskPackageIdentity{
		EngineID: "engine.lunafox.port_scan", PackageDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("LoadExactPackage() error = %v, want context canceled", err)
	}
}
