package installedengines

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	enginemanifest "github.com/yyhuni/lunafox/contracts/enginemanifest"
	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/repositoryname"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type installedEngineQueryRepositoryStub struct {
	records []catalogdomain.Engine
	record  *catalogdomain.Engine
	listErr error
	getErr  error
	getIDs  []string
}

func (stub *installedEngineQueryRepositoryStub) ListInstalledEngines() ([]catalogdomain.Engine, error) {
	if stub.listErr != nil {
		return nil, stub.listErr
	}
	return append([]catalogdomain.Engine(nil), stub.records...), nil
}

func (stub *installedEngineQueryRepositoryStub) GetInstalledEngineByID(engineID string) (*catalogdomain.Engine, error) {
	stub.getIDs = append(stub.getIDs, engineID)
	if stub.getErr != nil {
		return nil, stub.getErr
	}
	if stub.record == nil {
		return nil, catalogdomain.ErrEngineNotFound
	}
	record := cloneEngineRegistration(*stub.record)
	return &record, nil
}

type exactPackageCacheLoaderStub struct {
	entries map[ociartifact.PackageDigest]ExactPackageCacheEntry
	errors  map[ociartifact.PackageDigest]error
	calls   []ociartifact.PackageDigest
}

func (stub *exactPackageCacheLoaderStub) LoadExactPackage(expectedDigest ociartifact.PackageDigest) (ExactPackageCacheEntry, error) {
	stub.calls = append(stub.calls, expectedDigest)
	if err := stub.errors[expectedDigest]; err != nil {
		return ExactPackageCacheEntry{}, err
	}
	entry, found := stub.entries[expectedDigest]
	if !found {
		return ExactPackageCacheEntry{}, os.ErrNotExist
	}
	return entry, nil
}

func TestNewRepositoryBackedQueryRequiresRepositoryAndCache(t *testing.T) {
	repository := &installedEngineQueryRepositoryStub{}
	cache := &exactPackageCacheLoaderStub{}
	var typedNilRepository *installedEngineQueryRepositoryStub
	var typedNilCache *exactPackageCacheLoaderStub
	if _, err := NewRepositoryBackedQuery(nil, cache); err == nil {
		t.Fatal("NewRepositoryBackedQuery() accepted a nil repository")
	}
	if _, err := NewRepositoryBackedQuery(repository, nil); err == nil {
		t.Fatal("NewRepositoryBackedQuery() accepted a nil cache loader")
	}
	if _, err := NewRepositoryBackedQuery(typedNilRepository, cache); err == nil {
		t.Fatal("NewRepositoryBackedQuery() accepted a typed-nil repository")
	}
	if _, err := NewRepositoryBackedQuery(repository, typedNilCache); err == nil {
		t.Fatal("NewRepositoryBackedQuery() accepted a typed-nil cache loader")
	}
}

func TestRepositoryBackedQueryReturnsOneRecordSnapshotWithExactLayout(t *testing.T) {
	fixture := newInstalledEnginePackageFixture(t, repositoryname.FirstPartyEngineIDPortScan, "1.2.3", 'a', 'b', 'c')
	repository := &installedEngineQueryRepositoryStub{record: &fixture.record}
	cache := &exactPackageCacheLoaderStub{entries: map[ociartifact.PackageDigest]ExactPackageCacheEntry{
		fixture.digest: {PackageDigest: fixture.digest, Layout: fixture.layout},
	}}
	query, err := NewRepositoryBackedQuery(repository, cache)
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := query.GetInstalledEnginePackage(fixture.record.EngineID)
	if err != nil {
		t.Fatalf("GetInstalledEnginePackage() error = %v", err)
	}
	if len(repository.getIDs) != 1 || repository.getIDs[0] != fixture.record.EngineID {
		t.Fatalf("repository calls = %v", repository.getIDs)
	}
	if len(cache.calls) != 1 || cache.calls[0] != fixture.digest {
		t.Fatalf("cache calls = %v, want only %s", cache.calls, fixture.digest)
	}
	if resolved.Registration.PackageDigest != string(fixture.digest) ||
		resolved.Layout.Definition.EngineDefinition.EngineID != fixture.record.EngineID ||
		resolved.Layout.Definition.PackageManifest.EngineVersion != fixture.record.PackageVersion {
		t.Fatalf("resolved package = %#v", resolved)
	}

	resolved.Registration.Manifest[0] = 'x'
	resolved.Layout.Definition.PackageManifest.RuntimeImage.Refs[0] = "mutated"
	resolved.Layout.LocaleResources["en"]["engine"].(map[string]any)["displayName"] = "mutated"
	if bytes.Equal(resolved.Registration.Manifest, fixture.record.Manifest) ||
		fixture.layout.Definition.PackageManifest.RuntimeImage.Refs[0] == "mutated" ||
		fixture.layout.LocaleResources["en"]["engine"].(map[string]any)["displayName"] == "mutated" {
		t.Fatal("resolved package aliases repository or cache-owned data")
	}
}

func TestRepositoryBackedQueryMapsOnlyRepositoryNotFound(t *testing.T) {
	tests := []struct {
		name          string
		repositoryErr error
		wantNotFound  bool
	}{
		{name: "domain not found", repositoryErr: fmt.Errorf("lookup: %w", catalogdomain.ErrEngineNotFound), wantNotFound: true},
		{name: "database unavailable", repositoryErr: errors.New("database unavailable")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &installedEngineQueryRepositoryStub{getErr: test.repositoryErr}
			query, err := NewRepositoryBackedQuery(repository, &exactPackageCacheLoaderStub{})
			if err != nil {
				t.Fatal(err)
			}
			_, err = query.GetInstalledEnginePackage(repositoryname.FirstPartyEngineIDPortScan)
			if errors.Is(err, ErrEnginePackageNotFound) != test.wantNotFound {
				t.Fatalf("GetInstalledEnginePackage() error = %v, wantNotFound=%v", err, test.wantNotFound)
			}
			if !test.wantNotFound && !errors.Is(err, test.repositoryErr) {
				t.Fatalf("technical repository error was not preserved: %v", err)
			}
		})
	}
}

func TestRepositoryBackedQueryMapsUnknownCanonicalEngineToRepositoryNotFound(t *testing.T) {
	repository := &installedEngineQueryRepositoryStub{getErr: catalogdomain.ErrEngineNotFound}
	query, err := NewRepositoryBackedQuery(repository, &exactPackageCacheLoaderStub{})
	if err != nil {
		t.Fatal(err)
	}
	const unknownEngineID = "engine.lunafox.unknown"
	_, err = query.GetInstalledEnginePackage(unknownEngineID)
	if !errors.Is(err, ErrEnginePackageNotFound) {
		t.Fatalf("GetInstalledEnginePackage() error = %v, want repository not-found classification", err)
	}
	if !reflect.DeepEqual(repository.getIDs, []string{unknownEngineID}) {
		t.Fatalf("repository lookups = %#v, want unknown canonical ID lookup", repository.getIDs)
	}
}

func TestRepositoryBackedQueryRejectsRepositoryResultForAnotherEngine(t *testing.T) {
	other := newInstalledEnginePackageFixture(t, repositoryname.FirstPartyEngineIDWebsiteDiscovery, "1.2.3", 'd', 'e', 'f')
	cache := &exactPackageCacheLoaderStub{}
	query, err := NewRepositoryBackedQuery(&installedEngineQueryRepositoryStub{record: &other.record}, cache)
	if err != nil {
		t.Fatal(err)
	}

	_, err = query.GetInstalledEnginePackage(repositoryname.FirstPartyEngineIDPortScan)
	if err == nil || !strings.Contains(err.Error(), "repository identity mismatch") {
		t.Fatalf("GetInstalledEnginePackage() error = %v", err)
	}
	if len(cache.calls) != 0 {
		t.Fatalf("cache was read after repository identity drift: %v", cache.calls)
	}
}

func TestRepositoryBackedQueryPreservesMissingAndCorruptCacheErrors(t *testing.T) {
	fixture := newInstalledEnginePackageFixture(t, repositoryname.FirstPartyEngineIDPortScan, "1.2.3", 'a', 'b', 'c')
	tests := []struct {
		name     string
		cacheErr error
	}{
		{name: "missing", cacheErr: os.ErrNotExist},
		{name: "corrupt", cacheErr: errors.New("archive digest mismatch")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query, err := NewRepositoryBackedQuery(
				&installedEngineQueryRepositoryStub{record: &fixture.record},
				&exactPackageCacheLoaderStub{errors: map[ociartifact.PackageDigest]error{fixture.digest: test.cacheErr}},
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = query.GetInstalledEnginePackage(fixture.record.EngineID)
			if !errors.Is(err, test.cacheErr) || errors.Is(err, ErrEnginePackageNotFound) {
				t.Fatalf("GetInstalledEnginePackage() error = %v", err)
			}
		})
	}
}

func TestRepositoryBackedQueryNeverSelectsSiblingDigest(t *testing.T) {
	selected := newInstalledEnginePackageFixture(t, repositoryname.FirstPartyEngineIDPortScan, "1.2.3", 'a', 'b', 'c')
	sibling := newInstalledEnginePackageFixture(t, repositoryname.FirstPartyEngineIDWebsiteDiscovery, "1.2.3", 'd', 'e', 'f')
	cache := &exactPackageCacheLoaderStub{entries: map[ociartifact.PackageDigest]ExactPackageCacheEntry{
		sibling.digest: {PackageDigest: sibling.digest, Layout: sibling.layout},
	}}
	query, err := NewRepositoryBackedQuery(&installedEngineQueryRepositoryStub{record: &selected.record}, cache)
	if err != nil {
		t.Fatal(err)
	}

	_, err = query.GetInstalledEnginePackage(selected.record.EngineID)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("GetInstalledEnginePackage() error = %v, want exact selected digest missing", err)
	}
	if len(cache.calls) != 1 || cache.calls[0] != selected.digest {
		t.Fatalf("cache calls = %v, query tried a sibling digest", cache.calls)
	}
}

func TestRepositoryBackedQueryRejectsDatabaseCacheIdentityDrift(t *testing.T) {
	base := newInstalledEnginePackageFixture(t, repositoryname.FirstPartyEngineIDPortScan, "1.2.3", 'a', 'b', 'c')
	other := newInstalledEnginePackageFixture(t, repositoryname.FirstPartyEngineIDWebsiteDiscovery, "1.2.3", 'd', 'e', 'f')
	tests := []struct {
		name   string
		mutate func(*catalogdomain.Engine, *ExactPackageCacheEntry)
		want   string
	}{
		{
			name: "cache digest",
			mutate: func(_ *catalogdomain.Engine, entry *ExactPackageCacheEntry) {
				entry.PackageDigest = other.digest
			},
			want: "cache digest mismatch",
		},
		{
			name: "engine identity",
			mutate: func(_ *catalogdomain.Engine, entry *ExactPackageCacheEntry) {
				entry.Layout = other.layout
			},
			want: "engineId mismatch",
		},
		{
			name: "publisher",
			mutate: func(_ *catalogdomain.Engine, entry *ExactPackageCacheEntry) {
				entry.Layout.Definition.EngineDefinition.Publisher = "other"
			},
			want: "publisher mismatch",
		},
		{
			name: "version",
			mutate: func(_ *catalogdomain.Engine, entry *ExactPackageCacheEntry) {
				entry.Layout.Definition.PackageManifest.EngineVersion = "9.9.9"
			},
			want: "version mismatch",
		},
		{
			name: "database manifest projection",
			mutate: func(record *catalogdomain.Engine, _ *ExactPackageCacheEntry) {
				definition := base.layout.Definition.EngineDefinition
				definition.Execution.EngineAPIMajor = 3
				record.Manifest = mustMarshalJSONTest(t, definition)
			},
			want: "unsupported Engine API major",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := cloneEngineRegistration(base.record)
			entry := ExactPackageCacheEntry{PackageDigest: base.digest, Layout: clonePackageLayoutTest(t, base.layout)}
			test.mutate(&record, &entry)
			query, err := NewRepositoryBackedQuery(
				&installedEngineQueryRepositoryStub{record: &record},
				&exactPackageCacheLoaderStub{entries: map[ociartifact.PackageDigest]ExactPackageCacheEntry{base.digest: entry}},
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = query.GetInstalledEnginePackage(record.EngineID)
			if err == nil || !strings.Contains(err.Error(), test.want) || errors.Is(err, ErrEnginePackageNotFound) {
				t.Fatalf("GetInstalledEnginePackage() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestRepositoryBackedQueryRejectsNonCanonicalRegistrationBeforeCacheRead(t *testing.T) {
	base := newInstalledEnginePackageFixture(t, repositoryname.FirstPartyEngineIDPortScan, "1.2.3", 'a', 'b', 'c')
	tests := []struct {
		name   string
		mutate func(*catalogdomain.Engine)
	}{
		{name: "engine ID", mutate: func(record *catalogdomain.Engine) { record.EngineID = " engine.lunafox.port_scan" }},
		{name: "artifact tag", mutate: func(record *catalogdomain.Engine) {
			record.ArtifactRef = "docker.io/yyhuni/lunafox-engine-port-scan:latest"
		}},
		{name: "package digest", mutate: func(record *catalogdomain.Engine) { record.PackageDigest = strings.ToUpper(record.PackageDigest) }},
		{name: "publisher", mutate: func(record *catalogdomain.Engine) { record.Publisher = " lunafox" }},
		{name: "package version", mutate: func(record *catalogdomain.Engine) { record.PackageVersion = "1.2.3 " }},
		{name: "manifest", mutate: func(record *catalogdomain.Engine) { record.Manifest = []byte(`{"manifestVersion":"engine.v4"}`) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := cloneEngineRegistration(base.record)
			test.mutate(&record)
			cache := &exactPackageCacheLoaderStub{}
			query := &RepositoryBackedQuery{repository: &installedEngineQueryRepositoryStub{}, cache: cache}
			_, err := query.loadRecord(record)
			if err == nil {
				t.Fatal("loadRecord() accepted a non-canonical registration")
			}
			if len(cache.calls) != 0 {
				t.Fatalf("cache was read before registration validation: %v", cache.calls)
			}
		})
	}
}

func TestRepositoryBackedQueryListFailsWithoutPartialCatalog(t *testing.T) {
	first := newInstalledEnginePackageFixture(t, repositoryname.FirstPartyEngineIDPortScan, "1.2.3", 'a', 'b', 'c')
	second := newInstalledEnginePackageFixture(t, repositoryname.FirstPartyEngineIDWebsiteDiscovery, "1.2.3", 'd', 'e', 'f')
	cacheErr := errors.New("second archive corrupt")
	query, err := NewRepositoryBackedQuery(
		&installedEngineQueryRepositoryStub{records: []catalogdomain.Engine{first.record, second.record}},
		&exactPackageCacheLoaderStub{
			entries: map[ociartifact.PackageDigest]ExactPackageCacheEntry{
				first.digest: {PackageDigest: first.digest, Layout: first.layout},
			},
			errors: map[ociartifact.PackageDigest]error{second.digest: cacheErr},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	packages, err := query.ListInstalledEnginePackages()
	if packages != nil || !errors.Is(err, cacheErr) {
		t.Fatalf("ListInstalledEnginePackages() = %#v, %v", packages, err)
	}
}

type installedEnginePackageFixture struct {
	record catalogdomain.Engine
	digest ociartifact.PackageDigest
	layout enginepackagecatalog.EnginePackageLayout
}

func newInstalledEnginePackageFixture(
	t *testing.T,
	engineID string,
	version string,
	artifactDigestByte byte,
	packageDigestByte byte,
	runtimeDigestByte byte,
) installedEnginePackageFixture {
	t.Helper()
	packageRepository, err := repositoryname.FirstPartyOCIRepositoryName(engineID)
	if err != nil {
		t.Fatal(err)
	}
	runtimeRepository, err := repositoryname.FirstPartyRuntimeImageRepositoryName(engineID)
	if err != nil {
		t.Fatal(err)
	}
	targetTypes := `["domain","ip","cidr"]`
	if engineID == repositoryname.FirstPartyEngineIDSubdomainDiscovery {
		targetTypes = `["domain"]`
	}
	packagePayload := []byte(fmt.Sprintf(`{
  "packageFormatVersion":"lunafox.engine-package.v2",
  "engineId":%q,
  "engineVersion":%q,
  "runtimeImage":{"refs":["docker.io/yyhuni/%s@sha256:%s"]}
}`, engineID, version, runtimeRepository, strings.Repeat(string(runtimeDigestByte), 64)))
	enginePayload := []byte(fmt.Sprintf(`{
  "manifestVersion":"engine.v5",
  "engineId":%q,
  "publisher":"lunafox",
  "execution":{
    "engineApiMajor":2,
    "supportedTargetTypes":%s,
    "configSections":[{
      "id":"scan",
      "defaultEnabled":true,
      "params":[{"key":"timeout","type":"integer","default":30,"minimum":1}]
    }]
  }
}`, engineID, targetTypes))
	definition, err := enginepackagecatalog.DecodeEnginePackageDefinition(packagePayload, enginePayload, "package.json", "engine.json")
	if err != nil {
		t.Fatal(err)
	}
	locales := map[string]map[string]any{
		"en": validLocaleResourceTest(),
		"zh": validLocaleResourceTest(),
	}
	digest := ociartifact.PackageDigest("sha256:" + strings.Repeat(string(packageDigestByte), 64))
	manifestPayload := mustMarshalJSONTest(t, definition.EngineDefinition)
	var indented bytes.Buffer
	if err := json.Indent(&indented, manifestPayload, "", "  "); err != nil {
		t.Fatal(err)
	}
	return installedEnginePackageFixture{
		digest: digest,
		record: catalogdomain.Engine{
			EngineID:       engineID,
			Publisher:      "lunafox",
			PackageVersion: version,
			ArtifactRef: fmt.Sprintf(
				"docker.io/yyhuni/%s@sha256:%s",
				packageRepository,
				strings.Repeat(string(artifactDigestByte), 64),
			),
			PackageDigest: string(digest),
			Manifest:      indented.Bytes(),
		},
		layout: enginepackagecatalog.EnginePackageLayout{
			Definition:      definition,
			LocaleResources: locales,
		},
	}
}

func validLocaleResourceTest() map[string]any {
	return map[string]any{
		"engine": map[string]any{"displayName": "Engine", "description": "Description"},
		"sections": map[string]any{
			"scan": map[string]any{
				"name":        "Scan",
				"description": "Scan settings",
				"params": map[string]any{
					"timeout": map[string]any{"description": "Timeout"},
				},
			},
		},
	}
}

func clonePackageLayoutTest(t *testing.T, layout enginepackagecatalog.EnginePackageLayout) enginepackagecatalog.EnginePackageLayout {
	t.Helper()
	locales := make(map[string]map[string]any, len(layout.LocaleResources))
	for locale, resource := range layout.LocaleResources {
		cloned, err := cloneLocaleResource(resource)
		if err != nil {
			t.Fatal(err)
		}
		locales[locale] = cloned
	}
	packageManifest := layout.Definition.PackageManifest
	packageManifest.RuntimeImage.Refs = append([]string(nil), packageManifest.RuntimeImage.Refs...)
	return enginepackagecatalog.EnginePackageLayout{
		Definition: enginepackagecatalog.EnginePackageDefinition{
			PackageManifest:  packageManifest,
			EngineDefinition: enginemanifest.CloneEngineDefinition(layout.Definition.EngineDefinition),
		},
		LocaleResources: locales,
	}
}

func mustMarshalJSONTest(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}
