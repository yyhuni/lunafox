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

const wordlistBinaryCheckSize = 8192

// Max bytes per logical line (excluding newline separators).
const wordlistMaxLineBytes = 64 * 1024

var _ WordlistFileStore = (*LocalWordlistFileStore)(nil)

type LocalWordlistFileStore struct{}

func NewLocalWordlistFileStore() *LocalWordlistFileStore {
	return &LocalWordlistFileStore{}
}

func (store *LocalWordlistFileStore) Save(basePath, filename string, content io.Reader) (*WordlistFileMetadata, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, err
	}

	safeFilename := sanitizeWordlistFilename(filename)
	fullPath := filepath.Join(basePath, safeFilename)

	file, err := os.Create(fullPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)

	written, err := io.Copy(writer, content)
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, err
	}

	if isWordlistBinaryFile(fullPath) {
		_ = os.Remove(fullPath)
		return nil, catalogdomain.ErrWordlistInvalidFileType
	}

	lineCount, err := countWordlistLines(fullPath)
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, err
	}

	return &WordlistFileMetadata{
		FilePath:  fullPath,
		FileSize:  written,
		LineCount: lineCount,
		FileHash:  hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func (store *LocalWordlistFileStore) Write(path, content string) (*WordlistFileMetadata, error) {
	lineCount, err := countWordlistLinesFromReader(strings.NewReader(content))
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	hasher := sha256.New()
	_, _ = hasher.Write([]byte(content))

	return &WordlistFileMetadata{
		FilePath:  path,
		FileSize:  fileInfo.Size(),
		LineCount: lineCount,
		FileHash:  hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func (store *LocalWordlistFileStore) Read(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (store *LocalWordlistFileStore) Remove(path string) error {
	if path == "" {
		return nil
	}
	return os.Remove(path)
}

func (store *LocalWordlistFileStore) Exists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func (store *LocalWordlistFileStore) RefreshMetadata(path string, knownSize int64, knownUpdatedAt time.Time) (*WordlistFileMetadata, bool, error) {
	if path == "" {
		return nil, false, nil
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, false, err
	}

	if fileInfo.Size() == knownSize && !fileInfo.ModTime().After(knownUpdatedAt) {
		return nil, false, nil
	}

	hashValue, err := hashWordlistFile(path)
	if err != nil {
		return nil, false, err
	}

	lineCount, err := countWordlistLines(path)
	if err != nil {
		return nil, false, err
	}

	return &WordlistFileMetadata{
		FilePath:  path,
		FileSize:  fileInfo.Size(),
		LineCount: lineCount,
		FileHash:  hashValue,
	}, true, nil
}

func sanitizeWordlistFilename(filename string) string {
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	if filename == "" {
		filename = "wordlist.txt"
	}
	if filepath.Ext(filename) == "" {
		filename += ".txt"
	}
	return filename
}

func countWordlistLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = file.Close() }()

	return countWordlistLinesFromReader(file)
}

func countWordlistLinesFromReader(reader io.Reader) (int, error) {
	buffer := make([]byte, 32*1024)
	lineCount := 0
	hasContent := false
	lastByteWasNewline := false
	currentLineBytes := 0
	pendingCR := false

	for {
		n, readErr := reader.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			hasContent = true
			for _, b := range chunk {
				if pendingCR {
					// Treat CRLF as a single newline separator and don't count CR as line content.
					if b == '\n' {
						lineCount++
						currentLineBytes = 0
						lastByteWasNewline = true
						pendingCR = false
						continue
					}

					// Standalone CR counts as line content when not followed by LF.
					currentLineBytes++
					if currentLineBytes > wordlistMaxLineBytes {
						return 0, catalogdomain.ErrWordlistLineTooLong
					}
					pendingCR = false
				}

				if b == '\r' {
					pendingCR = true
					lastByteWasNewline = false
					continue
				}

				if b == '\n' {
					lineCount++
					currentLineBytes = 0
					lastByteWasNewline = true
					continue
				}
				currentLineBytes++
				lastByteWasNewline = false
				if currentLineBytes > wordlistMaxLineBytes {
					return 0, catalogdomain.ErrWordlistLineTooLong
				}
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return 0, readErr
		}
	}

	// Trailing standalone CR should be treated as content.
	if pendingCR {
		currentLineBytes++
		if currentLineBytes > wordlistMaxLineBytes {
			return 0, catalogdomain.ErrWordlistLineTooLong
		}
		lastByteWasNewline = false
	}

	if hasContent && !lastByteWasNewline {
		lineCount++
	}

	return lineCount, nil
}

func isWordlistBinaryFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	buffer := make([]byte, wordlistBinaryCheckSize)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	for index := 0; index < n; index++ {
		if buffer[index] == 0 {
			return true
		}
	}
	return false
}

func hashWordlistFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
