package subdomain_discovery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestTimeoutFromSeconds(t *testing.T) {
	timeout, err := timeoutFromSeconds(5)
	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, timeout)

	_, err = timeoutFromSeconds(0)
	require.Error(t, err)

	_, err = timeoutFromSeconds(-1)
	require.Error(t, err)
}
