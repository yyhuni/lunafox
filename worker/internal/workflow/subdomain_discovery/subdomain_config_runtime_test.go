package subdomain_discovery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsStageEnabled(t *testing.T) {
	config := map[string]any{
		stageRecon: map[string]any{
			"enabled": true,
		},
	}
	assert.True(t, isStageEnabled(config, stageRecon))
	assert.False(t, isStageEnabled(config, stageBruteforce))
	assert.False(t, isStageEnabled(map[string]any{"bad": "value"}, stageRecon))
}

func TestIsToolEnabled(t *testing.T) {
	stageConfig := map[string]any{
		"tools": map[string]any{
			toolSubfinder: map[string]any{
				"enabled": true,
			},
		},
	}
	assert.True(t, isToolEnabled(stageConfig, toolSubfinder))
	assert.False(t, isToolEnabled(stageConfig, "missing_tool"))
	assert.False(t, isToolEnabled(map[string]any{"tools": "bad"}, toolSubfinder))
}

func TestGetConfigPath(t *testing.T) {
	config := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": map[string]any{
					"enabled": true,
				},
			},
		},
	}
	got := getConfigPath(config, "a.b.c")
	require.NotNil(t, got)
	assert.Equal(t, true, got["enabled"])

	assert.Nil(t, getConfigPath(config, "a.b.missing"))
	assert.Nil(t, getConfigPath(nil, "a.b"))
}

func TestGetIntValue(t *testing.T) {
	cfg := map[string]any{
		"a":       1,
		"b":       int64(2),
		"c":       float64(3),
		"d":       uint(4),
		"badtype": "oops",
	}

	val, err := getIntValue(cfg, "a")
	require.NoError(t, err)
	assert.Equal(t, 1, val)

	val, err = getIntValue(cfg, "b")
	require.NoError(t, err)
	assert.Equal(t, 2, val)

	val, err = getIntValue(cfg, "c")
	require.NoError(t, err)
	assert.Equal(t, 3, val)

	val, err = getIntValue(cfg, "d")
	require.NoError(t, err)
	assert.Equal(t, 4, val)

	_, err = getIntValue(cfg, "missing")
	require.Error(t, err)

	_, err = getIntValue(cfg, "badtype")
	require.Error(t, err)
}

func TestGetTimeout(t *testing.T) {
	cfg := map[string]any{
		"timeout-runtime": 5,
	}
	timeout, err := getTimeout(cfg)
	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, timeout)

	_, err = getTimeout(map[string]any{"timeout-runtime": 0})
	require.Error(t, err)

	_, err = getTimeout(nil)
	require.Error(t, err)
}

func TestGetStringValue(t *testing.T) {
	assert.Equal(t, "default", getStringValue(nil, "key", "default"))
	assert.Equal(t, "default", getStringValue(map[string]any{"key": ""}, "key", "default"))
	assert.Equal(t, "value", getStringValue(map[string]any{"key": "value"}, "key", "default"))
}
