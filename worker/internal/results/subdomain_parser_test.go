package results

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
)

func TestParseSubdomains_DedupesAndSkipsEmpty(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() {
		pkg.Logger = prevLogger
	})

	dir := t.TempDir()
	file1 := filepath.Join(dir, "one.txt")
	file2 := filepath.Join(dir, "two.txt")

	require.NoError(t, os.WriteFile(file1, []byte("A.example.com\nb.example.com\n\n"), 0644))
	require.NoError(t, os.WriteFile(file2, []byte("a.example.com\nC.example.com\n"), 0644))

	ch, errCh := ParseSubdomains(context.Background(), []string{file1, file2})

	var got []string
	for sd := range ch {
		got = append(got, sd.Name)
	}

	err, ok := <-errCh
	require.False(t, ok)
	require.Nil(t, err)

	require.Equal(t, []string{"A.example.com", "b.example.com", "C.example.com"}, got)
}

func TestParseSubdomains_PropagatesScannerError(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() {
		pkg.Logger = prevLogger
	})

	dir := t.TempDir()
	file := filepath.Join(dir, "long.txt")
	longLine := strings.Repeat("a", 70*1024)
	require.NoError(t, os.WriteFile(file, []byte(longLine+"\n"), 0644))

	ch, errCh := ParseSubdomains(context.Background(), []string{file})
	for range ch {
	}

	err, ok := <-errCh
	require.True(t, ok)
	require.Error(t, err)
}

func TestParseSubdomains_MissingFileIsIgnored(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() {
		pkg.Logger = prevLogger
	})

	ch, errCh := ParseSubdomains(context.Background(), []string{filepath.Join(t.TempDir(), "missing.txt")})
	for range ch {
		t.Fatalf("expected no subdomains")
	}

	_, ok := <-errCh
	require.False(t, ok)
}

func TestParseSubdomains_CancelStopsProducer(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() {
		pkg.Logger = prevLogger
	})

	dir := t.TempDir()
	file := filepath.Join(dir, "many.txt")
	var b strings.Builder
	for i := 0; i < 10000; i++ {
		b.WriteString("sub")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(".example.com\n")
	}
	require.NoError(t, os.WriteFile(file, []byte(b.String()), 0644))

	ctx, cancel := context.WithCancel(context.Background())
	ch, errCh := ParseSubdomains(ctx, []string{file})

	time.Sleep(20 * time.Millisecond)
	cancel()

	timeout := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				goto done
			}
		case <-timeout:
			t.Fatal("timed out waiting for parser to stop after cancel")
		}
	}

done:
	if err, ok := <-errCh; ok {
		require.ErrorIs(t, err, context.Canceled)
	}
}
