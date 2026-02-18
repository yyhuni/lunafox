package pkg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitLoggerSetsLogger(t *testing.T) {
	prev := Logger
	t.Cleanup(func() { Logger = prev })

	t.Setenv("ENV", "development")
	require.NoError(t, InitLogger("debug"))
	require.NotNil(t, Logger)
	Sync()
}

func TestInitLoggerInvalidLevel(t *testing.T) {
	prev := Logger
	t.Cleanup(func() { Logger = prev })

	require.NoError(t, InitLogger("not-a-level"))
	require.NotNil(t, Logger)
	Sync()
}
