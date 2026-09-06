package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/installedengines"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type ScanWorkflowSnapshotQuery interface {
	GetScanWorkflowByIDContext(ctx context.Context, scanWorkflowID string) (*catalogdomain.ManagedScanWorkflow, error)
}

type scanCreateWorkflowReader struct{ workflows ScanWorkflowSnapshotQuery }

// NewScanCreateWorkflowReader returns the database workflow snapshot reader for scan creation.
func NewScanCreateWorkflowReader(workflows ScanWorkflowSnapshotQuery) scanapp.ScanCreateWorkflowReader {
	return scanCreateWorkflowReader{workflows: workflows}
}

func (reader scanCreateWorkflowReader) GetScanWorkflowManifest(ctx context.Context, scanWorkflowID string) (scanapp.ScanCreateWorkflowManifest, error) {
	if reader.workflows == nil {
		return scanapp.ScanCreateWorkflowManifest{}, fmt.Errorf("scan workflow repository is not configured")
	}
	workflow, err := reader.workflows.GetScanWorkflowByIDContext(ctx, scanWorkflowID)
	if err != nil {
		return scanapp.ScanCreateWorkflowManifest{}, err
	}
	return workflowToScanCreateWorkflowManifest(*workflow), nil
}

type scanCreateEnginePackageReader struct {
	query installedengines.Query
}

// NewScanCreateEnginePackageReader returns the default engine catalog reader for scan creation.
func NewScanCreateEnginePackageReader(query installedengines.Query) scanapp.ScanCreateEnginePackageReader {
	return scanCreateEnginePackageReader{query: query}
}

func (reader scanCreateEnginePackageReader) listEnginePackagesByID() (map[string]scanapp.ScanCreateEnginePackage, error) {
	loadedPackages, err := reader.query.ListInstalledEnginePackages()
	if err != nil {
		return nil, err
	}
	enginePackages := make(map[string]scanapp.ScanCreateEnginePackage, len(loadedPackages))
	for _, loadedPackage := range loadedPackages {
		definition := loadedPackage.Layout.Definition.EngineDefinition
		packageManifest := loadedPackage.Layout.Definition.PackageManifest
		engineID := strings.TrimSpace(definition.EngineID)
		enginePackages[engineID] = scanapp.ScanCreateEnginePackage{
			Package: scanapp.PlanTaskPackage{
				Identity: scanapp.PlanTaskPackageIdentity{
					EngineID: engineID, PackageDigest: loadedPackage.Registration.PackageDigest,
				},
				PackageVersion:   packageManifest.EngineVersion,
				Definition:       definition,
				RuntimeImageRefs: append([]string(nil), packageManifest.RuntimeImage.Refs...),
			},
		}
	}
	return enginePackages, nil
}

func (reader scanCreateEnginePackageReader) ListEnginePackagesByID() (map[string]scanapp.ScanCreateEnginePackage, error) {
	if reader.query == nil {
		return nil, fmt.Errorf("installed Engine Package v2 query is not configured")
	}
	return reader.listEnginePackagesByID()
}

func (reader scanCreateEnginePackageReader) LoadEnginePackage(ctx context.Context, engineID string) (scanapp.ScanCreateEnginePackage, error) {
	if reader.query == nil {
		return scanapp.ScanCreateEnginePackage{}, fmt.Errorf("installed Engine Package v2 query is not configured")
	}
	if err := ctx.Err(); err != nil {
		return scanapp.ScanCreateEnginePackage{}, err
	}
	resolved, err := reader.query.GetInstalledEnginePackage(strings.TrimSpace(engineID))
	if err != nil {
		if errors.Is(err, installedengines.ErrEnginePackageNotFound) {
			return scanapp.ScanCreateEnginePackage{}, fmt.Errorf("%w: engine %q", scanapp.ErrCreateScanWorkflowEngineUnavailable, engineID)
		}
		return scanapp.ScanCreateEnginePackage{}, err
	}
	if err := ctx.Err(); err != nil {
		return scanapp.ScanCreateEnginePackage{}, err
	}
	definition := resolved.Layout.Definition.EngineDefinition
	packageManifest := resolved.Layout.Definition.PackageManifest
	if definition.EngineID != engineID || packageManifest.EngineID != engineID || resolved.Registration.EngineID != engineID {
		return scanapp.ScanCreateEnginePackage{}, fmt.Errorf("installed Engine Package v2 identity mismatch for %q", engineID)
	}
	return scanapp.ScanCreateEnginePackage{
		Package: scanapp.PlanTaskPackage{
			Identity:         scanapp.PlanTaskPackageIdentity{EngineID: engineID, PackageDigest: resolved.Registration.PackageDigest},
			PackageVersion:   packageManifest.EngineVersion,
			Definition:       definition,
			RuntimeImageRefs: append([]string(nil), packageManifest.RuntimeImage.Refs...),
		},
	}, nil
}

type scanCreateEngineRegistrationReader struct {
	repository catalogdomain.InstalledEngineQueryRepository
}

func NewScanCreateEngineRegistrationReader(repository catalogdomain.InstalledEngineQueryRepository) scanapp.ScanCreateEngineRegistrationReader {
	return scanCreateEngineRegistrationReader{repository: repository}
}

func (reader scanCreateEngineRegistrationReader) EngineRegistrationExists(ctx context.Context, engineID string) (bool, error) {
	engineID = strings.TrimSpace(engineID)
	if reader.repository == nil || engineID == "" {
		return false, fmt.Errorf("engine registration lookup is not configured")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	registration, err := reader.repository.GetInstalledEngineByID(engineID)
	if err != nil {
		if errors.Is(err, catalogdomain.ErrEngineNotFound) {
			return false, nil
		}
		return false, err
	}
	if registration == nil || registration.EngineID != engineID {
		return false, fmt.Errorf("engine registration identity mismatch for %q", engineID)
	}
	return true, nil
}

func workflowToScanCreateWorkflowManifest(workflow catalogdomain.ManagedScanWorkflow) scanapp.ScanCreateWorkflowManifest {
	stages := make([]scanapp.ScanCreateWorkflowStage, 0, len(workflow.Stages))
	for _, stage := range workflow.Stages {
		steps := make([]scanapp.ScanCreateWorkflowStep, 0, len(stage.Steps))
		for _, step := range stage.Steps {
			steps = append(steps, scanapp.ScanCreateWorkflowStep{
				StepID:   step.StepID,
				EngineID: step.EngineID,
			})
		}
		stages = append(stages, scanapp.ScanCreateWorkflowStage{
			StageID: stage.StageID,
			Steps:   steps,
		})
	}
	return scanapp.ScanCreateWorkflowManifest{
		ScanWorkflowID: workflow.ScanWorkflowID,
		Stages:         stages,
	}
}
