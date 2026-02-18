package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCalcFileHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0644))

	hash, err := calcFileHash(path)
	require.NoError(t, err)

	expected := sha256.Sum256([]byte("hello"))
	assert.Equal(t, hex.EncodeToString(expected[:]), hash)
}

func TestDownloadWordlistSuccess(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() { pkg.Logger = prevLogger })

	client := &Client{
		downloadClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return newResponse(http.StatusOK, "wordlist-data"), nil
			}),
		},
	}

	dir := t.TempDir()
	dest := filepath.Join(dir, "lists", "wordlist.txt")
	err := client.DownloadWordlist(context.Background(), "wordlist.txt", dest)
	require.NoError(t, err)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "wordlist-data", string(data))

	if _, err := os.Stat(dest + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("expected temp file to be removed")
	}
}

func TestDownloadWordlistHTTPError(t *testing.T) {
	client := &Client{
		downloadClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return newResponse(http.StatusNotFound, "missing"), nil
			}),
		},
	}

	err := client.DownloadWordlist(context.Background(), "wordlist.txt", filepath.Join(t.TempDir(), "wordlist.txt"))
	var httpErr *HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusNotFound, httpErr.StatusCode)
}

func TestEnsureWordlistLocalCacheHit(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() { pkg.Logger = prevLogger })

	dir := t.TempDir()
	localPath := filepath.Join(dir, "list.txt")
	require.NoError(t, os.WriteFile(localPath, []byte("data"), 0644))

	hash := sha256.Sum256([]byte("data"))
	info := WordlistInfo{
		Name:     "list.txt",
		FileHash: hex.EncodeToString(hash[:]),
	}
	infoJSON, err := json.Marshal(info)
	require.NoError(t, err)

	downloadCalls := 0
	client := &Client{
		baseURL: "http://example",
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return newResponse(http.StatusOK, string(infoJSON)), nil
			}),
		},
		downloadClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				downloadCalls++
				return newResponse(http.StatusInternalServerError, "should not download"), nil
			}),
		},
	}

	path, err := client.EnsureWordlistLocal(context.Background(), "list.txt", dir)
	require.NoError(t, err)
	assert.Equal(t, localPath, path)
	assert.Equal(t, 0, downloadCalls)
}

func TestEnsureWordlistLocalDownloadsAndVerifies(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() { pkg.Logger = prevLogger })

	content := "downloaded-data"
	hash := sha256.Sum256([]byte(content))
	info := WordlistInfo{
		Name:     "list.txt",
		FileHash: hex.EncodeToString(hash[:]),
	}
	infoJSON, err := json.Marshal(info)
	require.NoError(t, err)

	client := &Client{
		baseURL: "http://example",
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return newResponse(http.StatusOK, string(infoJSON)), nil
			}),
		},
		downloadClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return newResponse(http.StatusOK, content), nil
			}),
		},
	}

	dir := t.TempDir()
	path, err := client.EnsureWordlistLocal(context.Background(), "list.txt", dir)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestEnsureWordlistLocalHashMismatch(t *testing.T) {
	prevLogger := pkg.Logger
	pkg.Logger = zap.NewNop()
	t.Cleanup(func() { pkg.Logger = prevLogger })

	info := WordlistInfo{
		Name:     "list.txt",
		FileHash: strings.Repeat("a", 64),
	}
	infoJSON, err := json.Marshal(info)
	require.NoError(t, err)

	client := &Client{
		baseURL: "http://example",
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return newResponse(http.StatusOK, string(infoJSON)), nil
			}),
		},
		downloadClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return newResponse(http.StatusOK, "bad"), nil
			}),
		},
	}

	dir := t.TempDir()
	expectedPath := filepath.Join(dir, "list.txt")
	_, err = client.EnsureWordlistLocal(context.Background(), "list.txt", dir)
	require.Error(t, err)

	if _, statErr := os.Stat(expectedPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected file to be removed after hash mismatch")
	}
}

func TestEnsureWordlistLocalEmptyName(t *testing.T) {
	client := &Client{}
	_, err := client.EnsureWordlistLocal(context.Background(), "", t.TempDir())
	assert.Error(t, err)
}
