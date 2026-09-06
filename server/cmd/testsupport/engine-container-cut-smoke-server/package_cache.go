package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/runtimeimage"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/server/internal/engineinstall"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

const (
	packageReceiptSchema          = "lunafox.engine-package-build-results.v1"
	packageReceiptModeDevelopment = "development"
	packageReceiptModeProduction  = "production"
)

type packageBuildResults struct {
	SchemaVersion string                 `json:"schemaVersion"`
	Mode          string                 `json:"mode"`
	Packages      []packageBuildArtifact `json:"packages"`
}

type packageBuildArtifact struct {
	EngineID           string   `json:"engineId"`
	EngineVersion      string   `json:"engineVersion"`
	ArchivePath        string   `json:"archivePath"`
	PackageDigest      string   `json:"packageDigest"`
	RuntimeImageDigest string   `json:"runtimeImageDigest"`
	RuntimeImageRefs   []string `json:"runtimeImageRefs"`
}

type packageMapReader struct {
	packages           map[string]scanapp.PlanTaskPackage
	cacheEmptyObserved bool
	receiptMode        string
}

func (reader *packageMapReader) LoadExactPackage(_ context.Context, identity scanapp.PlanTaskPackageIdentity) (scanapp.PlanTaskPackage, error) {
	if reader == nil {
		return scanapp.PlanTaskPackage{}, fmt.Errorf("package reader is nil")
	}
	value, ok := reader.packages[identity.EngineID]
	if !ok || value.Identity.PackageDigest != identity.PackageDigest {
		return scanapp.PlanTaskPackage{}, fmt.Errorf("exact package %s/%s is unavailable", identity.EngineID, identity.PackageDigest)
	}
	return value, nil
}

func (reader *packageMapReader) ListEnginePackagesByID() (map[string]scanapp.ScanCreateEnginePackage, error) {
	if reader == nil {
		return nil, fmt.Errorf("package reader is nil")
	}
	packages := make(map[string]scanapp.ScanCreateEnginePackage, len(reader.packages))
	for engineID, pkg := range reader.packages {
		packages[engineID] = scanapp.ScanCreateEnginePackage{Package: pkg}
	}
	return packages, nil
}

func (reader *packageMapReader) LoadEnginePackage(_ context.Context, engineID string) (scanapp.ScanCreateEnginePackage, error) {
	if reader == nil {
		return scanapp.ScanCreateEnginePackage{}, fmt.Errorf("package reader is nil")
	}
	pkg, ok := reader.packages[engineID]
	if !ok {
		return scanapp.ScanCreateEnginePackage{}, fmt.Errorf("package %q is unavailable", engineID)
	}
	return scanapp.ScanCreateEnginePackage{Package: pkg}, nil
}

func loadPackageReceipt(ctx context.Context, receiptPath, cacheRoot string) (*packageMapReader, int, error) {
	if ctx == nil {
		return nil, 0, fmt.Errorf("package load context is required")
	}
	payload, err := os.ReadFile(receiptPath)
	if err != nil {
		return nil, 0, fmt.Errorf("read package build receipt: %w", err)
	}
	var receipt packageBuildResults
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return nil, 0, fmt.Errorf("decode package build receipt: %w", err)
	}
	if err := validatePackageReceiptEnvelope(receipt); err != nil {
		return nil, 0, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, 0, fmt.Errorf("package build receipt contains trailing JSON")
		}
		return nil, 0, fmt.Errorf("decode trailing package build receipt JSON: %w", err)
	}
	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		return nil, 0, fmt.Errorf("inspect package cache cold-start: %w", err)
	}
	if len(entries) != 0 {
		return nil, 0, fmt.Errorf("package cache must be empty before smoke, found %d entries", len(entries))
	}
	if err := os.MkdirAll(cacheRoot, 0o750); err != nil {
		return nil, 0, err
	}
	installer := engineinstall.CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 64 << 20}
	reader := &packageMapReader{
		packages:           make(map[string]scanapp.PlanTaskPackage, len(receipt.Packages)),
		cacheEmptyObserved: true,
		receiptMode:        receipt.Mode,
	}
	// Source discovery is validated by the outer smoke before this isolated
	// Server starts. Duplicating that catalog here would make a new Engine fail
	// before the receipt-derived identity set can be verified end to end.
	for _, artifact := range receipt.Packages {
		if artifact.EngineID == "" || artifact.EngineVersion == "" || artifact.ArchivePath == "" || len(artifact.RuntimeImageRefs) == 0 {
			return nil, 0, fmt.Errorf("package receipt has incomplete artifact %q", artifact.EngineID)
		}
		if _, exists := reader.packages[artifact.EngineID]; exists {
			return nil, 0, fmt.Errorf("package receipt contains duplicate engine %q", artifact.EngineID)
		}
		candidates, err := runtimeimage.ParseFirstPartyCandidates(artifact.EngineID, artifact.RuntimeImageRefs)
		if err != nil {
			return nil, 0, fmt.Errorf("package %s Runtime Image refs: %w", artifact.EngineID, err)
		}
		if string(candidates.RuntimeImageDigest) != artifact.RuntimeImageDigest {
			return nil, 0, fmt.Errorf("package %s Runtime Image digest mismatch", artifact.EngineID)
		}
		digest, err := ociartifact.ParsePackageDigest(artifact.PackageDigest)
		if err != nil {
			return nil, 0, fmt.Errorf("package %s digest: %w", artifact.EngineID, err)
		}
		file, err := os.Open(filepath.Clean(artifact.ArchivePath))
		if err != nil {
			return nil, 0, fmt.Errorf("open package %s archive: %w", artifact.EngineID, err)
		}
		info, statErr := file.Stat()
		if statErr != nil {
			_ = file.Close()
			return nil, 0, fmt.Errorf("stat package %s archive: %w", artifact.EngineID, statErr)
		}
		cached, stageErr := installer.StageAndPromotePackage(ctx, file, digest, info.Size(), func(_ context.Context, layout packagecatalog.EnginePackageLayout) error {
			if layout.Definition.EngineDefinition.EngineID != artifact.EngineID || layout.Definition.PackageManifest.EngineID != artifact.EngineID {
				return fmt.Errorf("package identity mismatch for %s", artifact.EngineID)
			}
			if layout.Definition.PackageManifest.EngineVersion != artifact.EngineVersion {
				return fmt.Errorf("package version mismatch for %s", artifact.EngineID)
			}
			if len(layout.Definition.PackageManifest.RuntimeImage.Refs) != len(artifact.RuntimeImageRefs) {
				return fmt.Errorf("package Runtime Image refs mismatch for %s", artifact.EngineID)
			}
			for index, ref := range artifact.RuntimeImageRefs {
				if layout.Definition.PackageManifest.RuntimeImage.Refs[index] != ref {
					return fmt.Errorf("package Runtime Image ref mismatch for %s", artifact.EngineID)
				}
			}
			return nil
		})
		_ = file.Close()
		if stageErr != nil {
			return nil, 0, fmt.Errorf("stage package %s: %w", artifact.EngineID, stageErr)
		}
		if cached.PackageDigest != digest {
			return nil, 0, fmt.Errorf("package cache digest mismatch for %s", artifact.EngineID)
		}
		definition := cached.Layout.Definition.EngineDefinition
		reader.packages[artifact.EngineID] = scanapp.PlanTaskPackage{
			Identity:         scanapp.PlanTaskPackageIdentity{EngineID: artifact.EngineID, PackageDigest: artifact.PackageDigest},
			PackageVersion:   artifact.EngineVersion,
			Definition:       definition,
			RuntimeImageRefs: append([]string(nil), cached.Layout.Definition.PackageManifest.RuntimeImage.Refs...),
		}
	}
	return reader, len(reader.packages), nil
}

func validatePackageReceiptEnvelope(receipt packageBuildResults) error {
	if receipt.SchemaVersion != packageReceiptSchema {
		return fmt.Errorf("package build receipt has unexpected schema %q", receipt.SchemaVersion)
	}
	switch receipt.Mode {
	case packageReceiptModeDevelopment, packageReceiptModeProduction:
	default:
		return fmt.Errorf("package build receipt has unsupported mode %q", receipt.Mode)
	}
	if len(receipt.Packages) == 0 {
		return fmt.Errorf("package build receipt must contain at least one package")
	}
	return nil
}

func (reader *packageMapReader) packageForEngine(engineID string) (scanapp.PlanTaskPackage, bool) {
	if reader == nil {
		return scanapp.PlanTaskPackage{}, false
	}
	value, ok := reader.packages[engineID]
	return value, ok
}

// OverrideRuntimeImageForSmoke replaces only the detached package projection
// used to compile smoke plans. The archived package and receipt remain
// untouched; requiring the exact source ref here makes accidental use of an
// arbitrary test image fail before any Scan is created.
func (reader *packageMapReader) OverrideRuntimeImageForSmoke(engineID, sourceRef, fixtureRef string) error {
	if reader == nil {
		return fmt.Errorf("package reader is nil")
	}
	pkg, ok := reader.packages[engineID]
	if !ok {
		return fmt.Errorf("package %q is unavailable", engineID)
	}
	if len(pkg.RuntimeImageRefs) == 0 || pkg.RuntimeImageRefs[0] != sourceRef {
		return fmt.Errorf("smoke fixture source ref does not match package %q", engineID)
	}
	if _, err := ociartifact.ParseDigestReference(sourceRef); err != nil {
		return fmt.Errorf("smoke fixture source ref is invalid: %w", err)
	}
	fixture, err := ociartifact.ParseDigestReference(fixtureRef)
	if err != nil {
		return fmt.Errorf("smoke fixture image ref is invalid: %w", err)
	}
	if fixtureRef != fixture.String() {
		return fmt.Errorf("smoke fixture image ref is not canonical")
	}
	if fixture.Digest == "" {
		return fmt.Errorf("smoke fixture image digest is required")
	}
	pkg.RuntimeImageRefs = []string{fixture.String()}
	reader.packages[engineID] = pkg
	return nil
}

// packageReceiptIDs is used only for deterministic evidence and diagnostics.
func (reader *packageMapReader) packageReceiptIDs() []string {
	ids := make([]string, 0, len(reader.packages))
	for id := range reader.packages {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

var _ scanapp.PlanTaskPackageReader = (*packageMapReader)(nil)
var _ scanapp.ScanCreateEnginePackageReader = (*packageMapReader)(nil)
