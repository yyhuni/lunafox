package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
)

// WordlistInfo contains wordlist metadata from server
type WordlistInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	FilePath    string `json:"filePath"`
	FileHash    string `json:"fileHash"`
	FileSize    int64  `json:"fileSize"`
	LineCount   int    `json:"lineCount"`
	Description string `json:"description"`
}

// GetWordlistInfo fetches wordlist metadata from server
func (c *Client) GetWordlistInfo(ctx context.Context, wordlistName string) (*WordlistInfo, error) {
	url := fmt.Sprintf("%s/api/worker/wordlists/%s", c.baseURL, wordlistName)
	return fetchJSON[*WordlistInfo](ctx, c, url)
}

// DownloadWordlist downloads a wordlist file from server with atomic write
func (c *Client) DownloadWordlist(ctx context.Context, wordlistName, destPath string) error {
	url := fmt.Sprintf("%s/api/worker/wordlists/%s/download", c.baseURL, wordlistName)

	pkg.Logger.Info("Downloading wordlist",
		zap.String("name", wordlistName),
		zap.String("dest", destPath))

	resp, err := c.getDownload(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to download wordlist: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       fmt.Sprintf("downloading wordlist %s: %s", wordlistName, string(body)),
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create wordlist directory: %w", err)
	}

	// Write to temporary file first (atomic write)
	tempPath := destPath + ".tmp"
	out, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary wordlist file: %w", err)
	}
	defer func() {
		_ = out.Close()
		_ = os.Remove(tempPath) // Clean up temp file on error
	}()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write wordlist file: %w", err)
	}

	// Close file before rename
	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, destPath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	pkg.Logger.Info("Wordlist downloaded successfully",
		zap.String("name", wordlistName),
		zap.String("path", destPath))

	return nil
}

// EnsureWordlistLocal ensures a wordlist file exists locally, downloading if needed
// If local file exists but hash doesn't match, re-download and verify
func (c *Client) EnsureWordlistLocal(ctx context.Context, wordlistName, basePath string) (string, error) {
	if wordlistName == "" {
		return "", fmt.Errorf("wordlist name is empty")
	}

	// Get wordlist info from server (includes expected hash)
	info, err := c.GetWordlistInfo(ctx, wordlistName)
	if err != nil {
		return "", fmt.Errorf("failed to get wordlist info: %w", err)
	}

	// Local path: basePath/wordlistName
	localPath := filepath.Join(basePath, wordlistName)

	// Check if file already exists and hash matches
	if _, err := os.Stat(localPath); err == nil {
		localHash, hashErr := calcFileHash(localPath)
		if hashErr == nil && localHash == info.FileHash {
			pkg.Logger.Debug("Wordlist hash matches, using local file",
				zap.String("path", localPath))
			return localPath, nil
		}
		pkg.Logger.Info("Wordlist hash mismatch, re-downloading",
			zap.String("name", wordlistName),
			zap.String("expected", info.FileHash),
			zap.String("local", localHash))
	}

	// Download from server
	if err := c.DownloadWordlist(ctx, wordlistName, localPath); err != nil {
		return "", err
	}

	// Verify downloaded file hash
	downloadedHash, err := calcFileHash(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to calculate hash of downloaded file: %w", err)
	}

	if downloadedHash != info.FileHash {
		// Remove corrupted file
		_ = os.Remove(localPath)
		return "", fmt.Errorf("downloaded file hash mismatch: expected=%s, got=%s", info.FileHash, downloadedHash)
	}

	pkg.Logger.Info("Wordlist verified and ready",
		zap.String("name", wordlistName),
		zap.String("path", localPath),
		zap.String("hash", downloadedHash))

	return localPath, nil
}

// calcFileHash calculates SHA-256 hash of a file
func calcFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
