package subdomain_discovery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStageMetadataDerivedFromContract(t *testing.T) {
	w := New(t.TempDir())
	def := GetContractDefinition()

	require.Equal(t, len(def.Stages), len(w.stageMetadata))
	for _, stage := range def.Stages {
		meta, ok := w.stageMetadata[stage.ID]
		require.Truef(t, ok, "missing stage metadata for %s", stage.ID)
		require.Equal(t, stage.ID, meta.ID)
		require.Equal(t, stage.Required, meta.Required)
		require.Equal(t, stage.Parallel, meta.Parallel)
	}
}
