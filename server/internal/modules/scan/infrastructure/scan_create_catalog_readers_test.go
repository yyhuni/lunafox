package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	"github.com/yyhuni/lunafox/server/internal/installedengines"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type scanCreateQueryStub struct {
	packages []installedengines.ResolvedInstalledEnginePackage
}

func (stub scanCreateQueryStub) ListInstalledEnginePackages() ([]installedengines.ResolvedInstalledEnginePackage, error) {
	return append([]installedengines.ResolvedInstalledEnginePackage(nil), stub.packages...), nil
}

func (stub scanCreateQueryStub) GetInstalledEnginePackage(engineID string) (installedengines.ResolvedInstalledEnginePackage, error) {
	for _, candidate := range stub.packages {
		if candidate.Registration.EngineID == engineID {
			return candidate, nil
		}
	}
	return installedengines.ResolvedInstalledEnginePackage{}, fmt.Errorf("not found")
}

func TestScanWorkflowToScanCreateWorkflowManifestMapsCreateProjection(t *testing.T) {
	workflow := catalogdomain.ManagedScanWorkflow{
		ScanWorkflowID: "subdomain_discovery",
		Stages: []scanworkflow.Stage{{
			StageID: "discovery",
			Steps: []scanworkflow.Step{{
				StepID:   "subdomain_discovery",
				EngineID: "engine.lunafox.subdomain_discovery",
			}},
		}},
	}

	result := workflowToScanCreateWorkflowManifest(workflow)

	if result.ScanWorkflowID != "subdomain_discovery" {
		t.Fatalf("unexpected scan workflow projection: %+v", result)
	}
	if len(result.Stages) != 1 || len(result.Stages[0].Steps) != 1 {
		t.Fatalf("expected projected stage and step, got %+v", result)
	}
	step := result.Stages[0].Steps[0]
	if step.StepID != "subdomain_discovery" || step.EngineID != "engine.lunafox.subdomain_discovery" {
		t.Fatalf("unexpected projected step: %+v", step)
	}
}

func TestScanCreateCatalogReaderAdaptersSatisfyApplicationPorts(t *testing.T) {
	var _ scanapp.ScanCreateWorkflowReader = scanCreateWorkflowReader{}
	var _ scanapp.ScanCreateEnginePackageReader = scanCreateEnginePackageReader{query: scanCreateQueryStub{}}
}

func TestScanCreateEnginePackageReaderHonorsCanceledOperationContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reader := scanCreateEnginePackageReader{query: scanCreateQueryStub{}}

	if _, err := reader.LoadEnginePackage(ctx, "engine.lunafox.port_scan"); !errors.Is(err, context.Canceled) {
		t.Fatalf("LoadEnginePackage() error = %v, want context canceled", err)
	}
}

func TestScanCreateEnginePackageReaderProjectsExactPackage(t *testing.T) {
	definition := enginecontract.EngineDefinition{
		ManifestVersion: enginecontract.SupportedRootManifestVersion,
		EngineID:        "engine.lunafox.subdomain_discovery", Publisher: "lunafox",
		Execution: engineexecution.ExecutionDefinition{
			EngineAPIMajor: 2, SupportedTargetTypes: []string{engineexecution.TargetTypeDomain},
			ConfigSections: []engineexecution.ConfigSectionDefinition{{ID: "recon", Params: []engineexecution.ParamDefinition{{Key: "threads", Type: engineexecution.ParamTypeInteger, Default: 10}}}},
		},
	}
	reader := scanCreateEnginePackageReader{query: scanCreateQueryStub{
		packages: []installedengines.ResolvedInstalledEnginePackage{
			{
				Registration: catalogdomain.Engine{EngineID: definition.EngineID, PackageDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
				Layout: enginepackagecatalog.EnginePackageLayout{Definition: enginepackagecatalog.EnginePackageDefinition{
					PackageManifest:  packagemanifest.PackageManifest{PackageFormatVersion: packagemanifest.PackageFormatVersion, EngineID: definition.EngineID, EngineVersion: "1.0.0", RuntimeImage: packagemanifest.RuntimeImage{Refs: []string{"docker.io/lunafox/lunafox-engine-runtime-subdomain-discovery@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}},
					EngineDefinition: definition,
				}},
			},
		},
	}}

	packages, err := reader.ListEnginePackagesByID()
	if err != nil {
		t.Fatalf("ListEnginePackagesByID failed: %v", err)
	}

	subdomain := packages["engine.lunafox.subdomain_discovery"]
	if subdomain.Package.Identity.PackageDigest == "" || len(subdomain.Package.RuntimeImageRefs) != 1 || subdomain.Package.Definition.Execution.EngineAPIMajor != 2 {
		t.Fatalf("expected exact Engine Package v2 projection, got %+v", subdomain.Package)
	}
	if got := subdomain.Package.Definition.Execution.SupportedTargetTypes; len(got) != 1 || got[0] != engineexecution.TargetTypeDomain {
		t.Fatalf("expected exact domain-only applicability, got %+v", got)
	}
}

func TestScanCreateEnginePackageReaderPreservesDualInputsWithoutLegacyProfile(t *testing.T) {
	definition := enginecontract.EngineDefinition{
		ManifestVersion: enginecontract.SupportedRootManifestVersion,
		EngineID:        "engine.lunafox.port_scan",
		Publisher:       "lunafox",
		Execution: engineexecution.ExecutionDefinition{
			EngineAPIMajor:       2,
			SupportedTargetTypes: []string{engineexecution.TargetTypeDomain},
			ConfigSections: []engineexecution.ConfigSectionDefinition{{
				ID: "scan", Params: []engineexecution.ParamDefinition{{Key: "threads", Type: engineexecution.ParamTypeInteger, Default: 10}},
			}},
		},
	}
	reader := scanCreateEnginePackageReader{query: scanCreateQueryStub{
		packages: []installedengines.ResolvedInstalledEnginePackage{
			{
				Registration: catalogdomain.Engine{EngineID: definition.EngineID, PackageDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
				Layout: enginepackagecatalog.EnginePackageLayout{Definition: enginepackagecatalog.EnginePackageDefinition{
					PackageManifest: packagemanifest.PackageManifest{
						PackageFormatVersion: packagemanifest.PackageFormatVersion,
						EngineID:             definition.EngineID,
						EngineVersion:        "1.0.0",
						RuntimeImage: packagemanifest.RuntimeImage{Refs: []string{
							"docker.io/lunafox/lunafox-engine-runtime-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
						}},
					},
					EngineDefinition: definition,
				}},
			},
		},
	}}

	_, err := reader.ListEnginePackagesByID()
	if err != nil {
		t.Fatalf("ListEnginePackagesByID rejected legal dual inputs before PlanTask: %v", err)
	}
}
