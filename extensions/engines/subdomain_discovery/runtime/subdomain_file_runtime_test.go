package subdomaindiscoveryruntime

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountFileLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lines.txt")
	require.NoError(t, os.WriteFile(path, []byte("a\n\nb\n"), 0644))
	assert.Equal(t, 2, countFileLines(path))

	assert.Equal(t, 0, countFileLines(filepath.Join(dir, "missing.txt")))
}

func TestCountFileLinesReturnsZeroOnScannerError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "too-long.txt")
	require.NoError(t, os.WriteFile(path, []byte(strings.Repeat("a", bufio.MaxScanTokenSize+1)), 0644))

	assert.Equal(t, 0, countFileLines(path))
}
