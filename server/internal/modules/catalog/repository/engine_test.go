package repository

import (
	"context"
	"errors"
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEngineRepositoryUpsertReplacesCurrentEngineAtomicallyWhenConfirmed(t *testing.T) {
	repository := newEngineRepositoryForTest(t)
	first := testInstalledEngine("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "docker.io/yyhuni/lunafox-engine-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if err := repository.UpsertInstalledEngine(context.Background(), &first, true); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	replacement := first
	replacement.PackageVersion = "2.0.0"
	replacement.PackageDigest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	replacement.ArtifactRef = "ghcr.io/yyhuni/lunafox-engine-port-scan@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	replacement.Manifest = []byte(`{"engineId":"engine.lunafox.port_scan","manifestVersion":"engine.v5"}`)
	if err := repository.UpsertInstalledEngine(context.Background(), &replacement, true); err != nil {
		t.Fatalf("replacement upsert: %v", err)
	}
	engines, err := repository.ListInstalledEngines()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(engines) != 1 ||
		engines[0].PackageDigest != replacement.PackageDigest ||
		engines[0].PackageVersion != replacement.PackageVersion ||
		engines[0].ArtifactRef != replacement.ArtifactRef ||
		string(engines[0].Manifest) != string(replacement.Manifest) {
		t.Fatalf("unexpected current engine state: %#v", engines)
	}
}

func TestEngineRepositoryRejectsUnconfirmedReplacementWithoutChangingCurrentRecord(t *testing.T) {
	repository := newEngineRepositoryForTest(t)
	first := testInstalledEngine(
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"registry.example/team/engine@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	)
	if err := repository.UpsertInstalledEngine(context.Background(), &first, false); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	replacement := first
	replacement.PackageDigest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	replacement.ArtifactRef = "registry.example/team/engine@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	err := repository.UpsertInstalledEngine(context.Background(), &replacement, false)
	var conflict *catalogdomain.EngineReplacementConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("unconfirmed replacement error = %v, want EngineReplacementConflictError", err)
	}
	if conflict.EngineID != first.EngineID || conflict.CurrentPackageDigest != first.PackageDigest || conflict.ProposedPackageDigest != replacement.PackageDigest {
		t.Fatalf("unexpected replacement conflict: %#v", conflict)
	}
	current, err := repository.GetInstalledEngineByID(first.EngineID)
	if err != nil {
		t.Fatal(err)
	}
	if current.PackageDigest != first.PackageDigest || current.ArtifactRef != first.ArtifactRef {
		t.Fatalf("unconfirmed replacement changed current record: %#v", current)
	}
}

func TestEngineRepositorySamePackageRegistrationIsStrictNoOp(t *testing.T) {
	repository := newEngineRepositoryForTest(t)
	engine := testInstalledEngine(
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"docker.io/yyhuni/lunafox-engine-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	)
	if err := repository.UpsertInstalledEngine(context.Background(), &engine, true); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	before, err := repository.GetInstalledEngineByID(engine.EngineID)
	if err != nil {
		t.Fatal(err)
	}

	if err := repository.UpsertInstalledEngine(context.Background(), &engine, true); err != nil {
		t.Fatalf("idempotent upsert: %v", err)
	}
	after, err := repository.GetInstalledEngineByID(engine.EngineID)
	if err != nil {
		t.Fatal(err)
	}
	if after.ID != before.ID || !after.CreatedAt.Equal(before.CreatedAt) || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("idempotent upsert changed persistent identity or timestamps: before=%#v after=%#v", before, after)
	}
}

func TestEngineRepositorySameDigestRejectsPackageFactDrift(t *testing.T) {
	repository := newEngineRepositoryForTest(t)
	engine := testInstalledEngine(
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"docker.io/yyhuni/lunafox-engine-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	)
	if err := repository.UpsertInstalledEngine(context.Background(), &engine, true); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	drifted := engine
	drifted.PackageVersion = "9.9.9"
	drifted.Manifest = []byte(`{"engineId":"engine.lunafox.port_scan","unexpected":true}`)
	if err := repository.UpsertInstalledEngine(context.Background(), &drifted, true); err == nil {
		t.Fatal("same package digest accepted conflicting package-derived facts")
	}
	current, err := repository.GetInstalledEngineByID(engine.EngineID)
	if err != nil {
		t.Fatal(err)
	}
	if current.PackageVersion != engine.PackageVersion || string(current.Manifest) != string(engine.Manifest) {
		t.Fatalf("conflicting replay mutated current registration: %#v", current)
	}
}

func TestEngineRepositorySameDigestDoesNotLoseJSONIntegerPrecision(t *testing.T) {
	repository := newEngineRepositoryForTest(t)
	engine := testInstalledEngine(
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"docker.io/yyhuni/lunafox-engine-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	)
	engine.Manifest = []byte(`{"default":9007199254740992}`)
	if err := repository.UpsertInstalledEngine(context.Background(), &engine, true); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	drifted := engine
	drifted.Manifest = []byte(`{"default":9007199254740993}`)
	if err := repository.UpsertInstalledEngine(context.Background(), &drifted, true); err == nil {
		t.Fatal("same package digest treated distinct int64 JSON values as equal")
	}
	current, err := repository.GetInstalledEngineByID(engine.EngineID)
	if err != nil {
		t.Fatal(err)
	}
	if string(current.Manifest) != string(engine.Manifest) {
		t.Fatalf("integer precision conflict mutated current registration: %#v", current)
	}
}

func TestEngineRepositorySameDigestMayUpdateSuccessfulRegistryLocation(t *testing.T) {
	repository := newEngineRepositoryForTest(t)
	engine := testInstalledEngine(
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"docker.io/yyhuni/lunafox-engine-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	)
	if err := repository.UpsertInstalledEngine(context.Background(), &engine, true); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	replayed := engine
	replayed.ArtifactRef = "ghcr.io/yyhuni/lunafox-engine-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if err := repository.UpsertInstalledEngine(context.Background(), &replayed, true); err != nil {
		t.Fatalf("alternate Registry location upsert: %v", err)
	}
	current, err := repository.GetInstalledEngineByID(engine.EngineID)
	if err != nil {
		t.Fatal(err)
	}
	if current.ArtifactRef != replayed.ArtifactRef || current.PackageDigest != engine.PackageDigest {
		t.Fatalf("unexpected replayed registration: %#v", current)
	}
}

func TestEngineRepositorySameDigestRejectsDifferentArtifactManifestIdentity(t *testing.T) {
	repository := newEngineRepositoryForTest(t)
	engine := testInstalledEngine(
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"docker.io/yyhuni/lunafox-engine-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	)
	if err := repository.UpsertInstalledEngine(context.Background(), &engine, true); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	replayed := engine
	replayed.ArtifactRef = "ghcr.io/yyhuni/lunafox-engine-port-scan@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if err := repository.UpsertInstalledEngine(context.Background(), &replayed, true); err == nil {
		t.Fatal("same package digest accepted a different OCI artifact manifest identity")
	}
	current, err := repository.GetInstalledEngineByID(engine.EngineID)
	if err != nil {
		t.Fatal(err)
	}
	if current.ArtifactRef != engine.ArtifactRef {
		t.Fatalf("conflicting artifact identity mutated current registration: %#v", current)
	}
}

func TestEngineRepositoryAcceptsAnyCanonicalArtifactRepository(t *testing.T) {
	for _, artifactRef := range []string{
		"registry.example/team/engine@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"docker.io/yyhuni/not-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	} {
		t.Run(artifactRef, func(t *testing.T) {
			repository := newEngineRepositoryForTest(t)
			engine := testInstalledEngine(
				"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				artifactRef,
			)
			if err := repository.UpsertInstalledEngine(context.Background(), &engine, false); err != nil {
				t.Fatalf("repository rejected canonical package artifact repository: %v", err)
			}
			engines, err := repository.ListInstalledEngines()
			if err != nil {
				t.Fatal(err)
			}
			if len(engines) != 1 {
				t.Fatalf("canonical artifact reference did not write registration: %#v", engines)
			}
		})
	}
}

func TestEngineRepositoryKeepsUniquePackageDigest(t *testing.T) {
	repository := newEngineRepositoryForTest(t)
	engine := testInstalledEngine("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "docker.io/yyhuni/lunafox-engine-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if err := repository.UpsertInstalledEngine(context.Background(), &engine, true); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	conflict := engine
	conflict.EngineID = "engine.lunafox.website_discovery"
	conflict.ArtifactRef = "docker.io/yyhuni/lunafox-engine-website-discovery@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if err := repository.UpsertInstalledEngine(context.Background(), &conflict, true); err == nil {
		t.Fatal("expected package digest unique constraint violation")
	}
	engines, err := repository.ListInstalledEngines()
	if err != nil {
		t.Fatal(err)
	}
	if len(engines) != 1 || engines[0].EngineID != engine.EngineID || engines[0].ArtifactRef != engine.ArtifactRef {
		t.Fatalf("unique constraint failure changed current registrations: %#v", engines)
	}
}

func TestEngineRepositoryCancelledContextDoesNotWrite(t *testing.T) {
	repository := newEngineRepositoryForTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	engine := testInstalledEngine(
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"docker.io/yyhuni/lunafox-engine-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	)
	if err := repository.UpsertInstalledEngine(ctx, &engine, true); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled upsert error = %v, want context.Canceled", err)
	}
	engines, err := repository.ListInstalledEngines()
	if err != nil {
		t.Fatal(err)
	}
	if len(engines) != 0 {
		t.Fatalf("cancelled upsert wrote registrations: %#v", engines)
	}
}

func TestEngineRepositoryMissingRecordReturnsDomainNotFound(t *testing.T) {
	repository := newEngineRepositoryForTest(t)
	_, err := repository.GetInstalledEngineByID("engine.lunafox.missing")
	if !errors.Is(err, catalogdomain.ErrEngineNotFound) {
		t.Fatalf("expected domain engine-not-found error, got %v", err)
	}
}

func newEngineRepositoryForTest(t *testing.T) *EngineRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Engine{}); err != nil {
		t.Fatalf("migrate engine: %v", err)
	}
	return NewEngineRepository(db)
}

func testInstalledEngine(packageDigest, artifactRef string) catalogdomain.Engine {
	return catalogdomain.Engine{
		EngineID: "engine.lunafox.port_scan", Publisher: "lunafox", PackageVersion: "1.0.0",
		ArtifactRef: artifactRef, PackageDigest: packageDigest,
		Manifest: []byte(`{"engineId":"engine.lunafox.port_scan"}`),
	}
}
