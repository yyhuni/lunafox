package domain

import (
	"strings"
	"time"
)

const (
	MaxWordlistNameLength        = 200
	MaxWordlistDescriptionLength = 200
)

// Wordlist represents a dictionary file for scanning in domain layer.
type Wordlist struct {
	ID          int
	Name        string
	Description string
	FilePath    string
	FileSize    int64
	LineCount   int
	FileHash    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NormalizeWordlistName(name string) string {
	return strings.TrimSpace(name)
}

func NormalizeWordlistDescription(description string) string {
	normalized := strings.TrimSpace(description)
	normalized = RemoveWordlistControlChars(normalized)
	if len(normalized) > MaxWordlistDescriptionLength {
		normalized = normalized[:MaxWordlistDescriptionLength]
	}
	return normalized
}

func ValidateWordlistName(name string) (string, error) {
	normalized := NormalizeWordlistName(name)
	if normalized == "" {
		return "", ErrWordlistNameEmpty
	}
	if len(normalized) > MaxWordlistNameLength {
		return "", ErrWordlistNameTooLong
	}
	if ContainsWordlistControlChars(normalized) {
		return "", ErrWordlistNameInvalid
	}
	return normalized, nil
}

func NewWordlist(name, description string) (*Wordlist, error) {
	normalizedName, err := ValidateWordlistName(name)
	if err != nil {
		return nil, err
	}

	return &Wordlist{
		Name:        normalizedName,
		Description: NormalizeWordlistDescription(description),
	}, nil
}

func (wordlist *Wordlist) AttachFile(path string, fileSize int64, lineCount int, fileHash string) {
	wordlist.FilePath = path
	wordlist.UpdateFileStats(fileSize, lineCount, fileHash)
}

func (wordlist *Wordlist) UpdateFileStats(fileSize int64, lineCount int, fileHash string) {
	wordlist.FileSize = fileSize
	wordlist.LineCount = lineCount
	wordlist.FileHash = fileHash
}

func CountWordlistContentLines(content string) int {
	if content == "" {
		return 0
	}

	lineCount := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		lineCount++
	}
	return lineCount
}

func ContainsWordlistControlChars(value string) bool {
	for _, character := range value {
		if character < 32 && character != ' ' {
			return true
		}
	}
	return false
}

func RemoveWordlistControlChars(value string) string {
	return strings.Map(func(character rune) rune {
		if character < 32 && character != ' ' {
			return -1
		}
		return character
	}, value)
}
