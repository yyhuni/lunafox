package subdomain_discovery

import (
	"os"
	"path/filepath"
	"regexp"
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

func TestSanitizeFilename(t *testing.T) {
	got := sanitizeFilename(`a/b c:d`)
	re := regexp.MustCompile(`[<>:"/\\|?*\s]`)
	assert.False(t, re.MatchString(got))
	assert.NotEmpty(t, got)
}
