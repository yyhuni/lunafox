package workflowmanifest

import "github.com/yyhuni/lunafox/contracts/scanworkflow"

func decodeManifest(payload []byte, source string) (Manifest, error) {
	return scanworkflow.DecodeDefinition(payload, source)
}
