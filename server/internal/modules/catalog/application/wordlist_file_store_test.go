package application

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type wordlistFileStoreTestStub struct{}

func newWordlistFileStoreTestStub() *wordlistFileStoreTestStub {
	return &wordlistFileStoreTestStub{}
}

func (stub *wordlistFileStoreTestStub) Save(basePath, filename string, content io.Reader) (*WordlistFileMetadata, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, err
	}
	if strings.TrimSpace(filename) == "" {
		filename = "wordlist.txt"
	}
	path := filepath.Join(basePath, filename)

	body, err := io.ReadAll(content)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return nil, err
	}

	hasher := sha256.New()
	_, _ = hasher.Write(body)
	return &WordlistFileMetadata{
		FilePath:  path,
		FileSize:  int64(len(body)),
		LineCount: catalogdomain.CountWordlistContentLines(string(body)),
		FileHash:  hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func (stub *wordlistFileStoreTestStub) Write(path, content string) (*WordlistFileMetadata, error) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(content))
	return &WordlistFileMetadata{
		FilePath:  path,
		FileSize:  info.Size(),
		LineCount: catalogdomain.CountWordlistContentLines(content),
		FileHash:  hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func (stub *wordlistFileStoreTestStub) Read(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (stub *wordlistFileStoreTestStub) Remove(path string) error {
	if path == "" {
		return nil
	}
	return os.Remove(path)
}

func (stub *wordlistFileStoreTestStub) Exists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func (stub *wordlistFileStoreTestStub) RefreshMetadata(path string, knownSize int64, knownUpdatedAt time.Time) (*WordlistFileMetadata, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false, err
	}
	if info.Size() == knownSize && !info.ModTime().After(knownUpdatedAt) {
		return nil, false, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	hasher := sha256.New()
	_, _ = hasher.Write(body)
	return &WordlistFileMetadata{
		FilePath:  path,
		FileSize:  info.Size(),
		LineCount: catalogdomain.CountWordlistContentLines(string(body)),
		FileHash:  hex.EncodeToString(hasher.Sum(nil)),
	}, true, nil
}
