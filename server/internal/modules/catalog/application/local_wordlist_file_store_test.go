package application

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type failingWordlistReader struct {
	payload []byte
	sent    bool
}

func (reader *failingWordlistReader) Read(p []byte) (int, error) {
	if reader.sent {
		return 0, errors.New("forced read error")
	}
	reader.sent = true
	n := copy(p, reader.payload)
	return n, nil
}

func TestLocalWordlistFileStoreSaveRejectsLongLine(t *testing.T) {
	store := NewLocalWordlistFileStore()
	basePath := t.TempDir()
	filename := "long.txt"
	longLine := strings.Repeat("a", 70*1024)

	_, err := store.Save(basePath, filename, strings.NewReader(longLine))
	if !errors.Is(err, catalogdomain.ErrWordlistLineTooLong) {
		t.Fatalf("expected ErrWordlistLineTooLong, got %v", err)
	}
	filePath := filepath.Join(basePath, filename)
	if _, statErr := os.Stat(filePath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected rejected file to be removed, stat error: %v", statErr)
	}
}

func TestLocalWordlistFileStoreSaveAcceptsCRLFLineAtLimit(t *testing.T) {
	store := NewLocalWordlistFileStore()
	basePath := t.TempDir()
	maxLine := strings.Repeat("m", 64*1024)
	content := maxLine + "\r\nsecond\r\n"

	metadata, err := store.Save(basePath, "crlf-limit.txt", strings.NewReader(content))
	if err != nil {
		t.Fatalf("save CRLF wordlist at limit failed: %v", err)
	}
	if metadata.LineCount != 2 {
		t.Fatalf("expected line count 2, got %d", metadata.LineCount)
	}
}

func TestLocalWordlistFileStoreSaveRejectsCRLFLineOverLimit(t *testing.T) {
	store := NewLocalWordlistFileStore()
	basePath := t.TempDir()
	overLimitLine := strings.Repeat("o", 64*1024+1)

	_, err := store.Save(basePath, "crlf-over-limit.txt", strings.NewReader(overLimitLine+"\r\n"))
	if !errors.Is(err, catalogdomain.ErrWordlistLineTooLong) {
		t.Fatalf("expected ErrWordlistLineTooLong, got %v", err)
	}
}

func TestLocalWordlistFileStoreRefreshMetadataRejectsLongLine(t *testing.T) {
	store := NewLocalWordlistFileStore()
	basePath := t.TempDir()
	path := filepath.Join(basePath, "long.txt")
	longLine := strings.Repeat("b", 70*1024)
	content := longLine + "\nsecond-line"

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write long wordlist fixture failed: %v", err)
	}

	metadata, changed, err := store.RefreshMetadata(path, 0, time.Time{})
	if !errors.Is(err, catalogdomain.ErrWordlistLineTooLong) {
		t.Fatalf("expected ErrWordlistLineTooLong, got %v", err)
	}
	if changed {
		t.Fatalf("expected changed=false when refresh fails")
	}
	if metadata != nil {
		t.Fatalf("expected metadata=nil when refresh fails")
	}
}

func TestLocalWordlistFileStoreWriteRejectsLongLine(t *testing.T) {
	store := NewLocalWordlistFileStore()
	basePath := t.TempDir()
	path := filepath.Join(basePath, "editable.txt")
	original := "short\ncontent\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture file failed: %v", err)
	}

	_, err := store.Write(path, strings.Repeat("x", 70*1024))
	if !errors.Is(err, catalogdomain.ErrWordlistLineTooLong) {
		t.Fatalf("expected ErrWordlistLineTooLong, got %v", err)
	}

	current, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read file after rejected write failed: %v", readErr)
	}
	if string(current) != original {
		t.Fatalf("expected file content to remain unchanged after rejected write")
	}
}

func TestLocalWordlistFileStoreSaveReadErrorCleansUpFile(t *testing.T) {
	store := NewLocalWordlistFileStore()
	basePath := t.TempDir()
	filename := "broken.txt"

	_, err := store.Save(basePath, filename, &failingWordlistReader{payload: []byte("partial-content")})
	if err == nil {
		t.Fatalf("expected save to fail when reader errors")
	}

	filePath := filepath.Join(basePath, filename)
	if _, statErr := os.Stat(filePath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected temporary file to be removed, stat error: %v", statErr)
	}
}
