package fingerprint

import (
	"context"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/application"
	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
)

func TestOfficialFingerprintHubWebAggregateStorageRoundTrip(t *testing.T) {
	root := fingerprintFixtureRoot(t)
	manifest := loadFullFingerprintFixtureManifest(t, root)
	fixture := manifest.FullFixtures[0]
	contents := readFingerprintFixture(t, root, fixture.Path)
	if actual := fixtureSHA256(contents); actual != fixture.SHA256 {
		t.Fatalf("fixture SHA-256 = %s, want %s", actual, fixture.SHA256)
	}

	records, err := domain.ParseNativeImport(domain.LibraryFingerPrintHub, contents)
	if err != nil {
		t.Fatalf("ParseNativeImport() official aggregate: %v", err)
	}
	if len(records) != fixture.Statistics.RawRecordCount {
		t.Fatalf("raw record count = %d, want %d", len(records), fixture.Statistics.RawRecordCount)
	}
	if reduced := domain.CollapseLastWins(records); len(reduced) != fixture.Statistics.FinalIdentityCount {
		t.Fatalf("final identity count = %d, want %d", len(reduced), fixture.Statistics.FinalIdentityCount)
	}

	store := newFingerprintIntegrationStore(t)
	facade := application.NewFacade(store)
	ctx := context.Background()
	first, err := facade.Import(ctx, domain.LibraryFingerPrintHub, contents)
	if err != nil {
		t.Fatalf("Import() official aggregate: %v", err)
	}
	if first != fixture.ExpectedImport.FirstImport.toApplicationCounts() {
		t.Fatalf("first import = %#v, want %#v", first, fixture.ExpectedImport.FirstImport)
	}

	exported, err := facade.Export(ctx, domain.LibraryFingerPrintHub)
	if err != nil {
		t.Fatalf("Export(): %v", err)
	}
	if _, err := domain.ParseNativeImport(domain.LibraryFingerPrintHub, exported); err != nil {
		t.Fatalf("ParseNativeImport() exported aggregate: %v", err)
	}
	reimport, err := facade.Import(ctx, domain.LibraryFingerPrintHub, exported)
	if err != nil {
		t.Fatalf("Import() exported aggregate: %v", err)
	}
	if reimport != fixture.ExpectedImport.Reimport.toApplicationCounts() {
		t.Fatalf("re-import = %#v, want %#v", reimport, fixture.ExpectedImport.Reimport)
	}
}

func (counts fingerprintImportCounts) toApplicationCounts() application.ImportCounts {
	return application.ImportCounts{
		CreatedCount:   counts.CreatedCount,
		UpdatedCount:   counts.UpdatedCount,
		UnchangedCount: counts.UnchangedCount,
	}
}
