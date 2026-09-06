package workflowmanifest

import "github.com/yyhuni/lunafox/contracts/scanworkflow"

func validateManifest(manifest Manifest) error {
	return scanworkflow.ValidateDefinition(manifest)
}

func validateManifestList(items []Manifest) error {
	return scanworkflow.ValidateDefinitionList(items)
}
