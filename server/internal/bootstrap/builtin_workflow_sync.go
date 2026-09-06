package bootstrap

import (
	"fmt"

	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	workflowmanifest "github.com/yyhuni/lunafox/server/internal/scanworkflow/manifest"
)

// synchronizeBuiltinScanWorkflows consumes release files only during startup.
// Normal request paths read the persisted aggregate and must not reload files.
func synchronizeBuiltinScanWorkflows(repository *catalogrepo.ScanWorkflowRepository) error {
	manifests, err := workflowmanifest.ListManifests()
	if err != nil {
		return err
	}
	workflows := make([]catalogdomain.ManagedScanWorkflow, 0, len(manifests))
	for _, manifest := range manifests {
		digest, err := scanworkflow.CanonicalWorkflowDigest(manifest.ScanWorkflowID, manifest.DisplayName, manifest.Description, manifest.Stages)
		if err != nil {
			return fmt.Errorf("digest release workflow %q: %w", manifest.ScanWorkflowID, err)
		}
		workflows = append(workflows, catalogdomain.ManagedScanWorkflow{
			ScanWorkflowID:   manifest.ScanWorkflowID,
			DisplayName:      manifest.DisplayName,
			Description:      manifest.Description,
			Stages:           append([]scanworkflow.Stage(nil), manifest.Stages...),
			IsBuiltin:        true,
			DefinitionDigest: digest,
			Version:          1,
		})
	}
	return repository.SynchronizeBuiltinScanWorkflows(workflows)
}
