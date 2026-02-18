package subdomain_discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeToolConfigUnknownTool(t *testing.T) {
	_, err := normalizeToolConfig("missing_tool", map[string]any{})
	assert.Error(t, err)
}

func TestBuildCommandUnknownTool(t *testing.T) {
	_, err := buildCommand("missing_tool", map[string]any{}, map[string]any{})
	assert.Error(t, err)
}
