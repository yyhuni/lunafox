package engineinstall

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

// VerifiedEngineInstaller completes the package, cache, Runtime Image, and
// identity checks before registration may become catalog-visible.
type VerifiedEngineInstaller interface {
	Install(context.Context, ociartifact.ArtifactCandidates) (VerifiedEngineInstallation, error)
}

var _ VerifiedEngineInstaller = (*EnginePackageInstaller)(nil)

// EngineRegistrationService commits only fully verified Engine Package v2
// facts to the current catalog.
type EngineRegistrationService struct {
	installer  VerifiedEngineInstaller
	repository catalogdomain.InstalledEngineCommandRepository
}

func NewEngineRegistrationService(
	installer VerifiedEngineInstaller,
	repository catalogdomain.InstalledEngineCommandRepository,
) (*EngineRegistrationService, error) {
	if isNilRegistrationDependency(installer) || isNilRegistrationDependency(repository) {
		return nil, fmt.Errorf("verified Engine Package v2 installer and installed-engine command repository are required")
	}
	return &EngineRegistrationService{installer: installer, repository: repository}, nil
}

// Install discovers identity from the verified package and atomically makes it
// current only when the caller explicitly permits a different-digest update.
func (service *EngineRegistrationService) Install(
	ctx context.Context,
	candidates ociartifact.ArtifactCandidates,
	allowReplacement bool,
) (*catalogdomain.Engine, error) {
	if service == nil {
		return nil, fmt.Errorf("Engine Package v2 registration service is required")
	}
	if ctx == nil {
		return nil, fmt.Errorf("Engine Package v2 registration context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	verified, err := service.installer.Install(ctx, candidates)
	if err != nil {
		return nil, fmt.Errorf("install verified Engine Package v2: %w", err)
	}
	return service.register(ctx, verified, allowReplacement)
}

func (service *EngineRegistrationService) register(
	ctx context.Context,
	verified VerifiedEngineInstallation,
	allowReplacement bool,
) (*catalogdomain.Engine, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	record := catalogdomain.Engine{
		EngineID:       verified.EngineID,
		Publisher:      verified.Publisher,
		PackageVersion: verified.PackageVersion,
		ArtifactRef:    verified.ArtifactRef,
		PackageDigest:  string(verified.PackageDigest),
		Manifest:       append(json.RawMessage(nil), verified.EngineManifestProjection...),
	}
	repositoryRecord := cloneEngineRegistration(record)
	if err := service.repository.UpsertInstalledEngine(ctx, &repositoryRecord, allowReplacement); err != nil {
		return nil, fmt.Errorf("register verified Engine Package v2 %q: %w", verified.EngineID, err)
	}
	result := cloneEngineRegistration(record)
	return &result, nil
}

// InstallInventory registers packages in declaration order. The caller must
// explicitly allow replacement before a different package digest may replace
// an existing current Engine record. A later failure does not roll back
// registrations already committed for earlier engines.
func (service *EngineRegistrationService) InstallInventory(
	ctx context.Context,
	inventory *Inventory,
	allowReplacement bool,
) ([]catalogdomain.Engine, error) {
	if service == nil {
		return nil, fmt.Errorf("Engine Package v2 registration service is required")
	}
	if ctx == nil {
		return nil, fmt.Errorf("Engine Package v2 registration context is required")
	}
	if inventory == nil || len(inventory.EnginePackages) == 0 {
		return nil, fmt.Errorf("Engine Package v2 registration inventory cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	registered := make([]catalogdomain.Engine, 0, len(inventory.EnginePackages))
	for index, item := range inventory.EnginePackages {
		record, err := service.Install(ctx, item.Candidates, allowReplacement)
		if err != nil {
			return nil, fmt.Errorf("register Engine Package v2 inventory item %d: %w", index, err)
		}
		registered = append(registered, cloneEngineRegistration(*record))
	}
	return registered, nil
}

func cloneEngineRegistration(record catalogdomain.Engine) catalogdomain.Engine {
	clone := record
	clone.Manifest = append(json.RawMessage(nil), record.Manifest...)
	return clone
}

func isNilRegistrationDependency(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
