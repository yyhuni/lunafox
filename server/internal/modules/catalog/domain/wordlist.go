// Package domain models catalog business rules and entities.
package domain

import (
	"slices"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
)

const (
	MaxWordlistFileNameLength    = 200
	MaxWordlistDescriptionLength = 200
)

// Wordlist represents a dictionary file for scanning in domain layer.
type Wordlist struct {
	ID          int
	FileName    string
	Description string
	Tags        []string
	FilePath    string
	FileSize    int64
	LineCount   int
	FileHash    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WordlistTagSummary is a derived read model for tags attached to wordlists.
type WordlistTagSummary struct {
	DisplayName   string
	WordlistCount int64
}

type WordlistListQuery struct {
	Page     int
	PageSize int
	Filter   string
	OrderBy  string
}

// ResourceName returns the immutable canonical resource identity for this row.
func (wordlist Wordlist) ResourceName() string {
	return resourcenames.Wordlist(wordlist.ID)
}

func NormalizeWordlistFileName(fileName string) string {
	return strings.TrimSpace(fileName)
}

func NormalizeWordlistDescription(description string) string {
	normalized := strings.TrimSpace(description)
	normalized = RemoveWordlistControlChars(normalized)
	if len(normalized) > MaxWordlistDescriptionLength {
		normalized = normalized[:MaxWordlistDescriptionLength]
	}
	return normalized
}

func ValidateWordlistFileName(fileName string) (string, error) {
	normalized := NormalizeWordlistFileName(fileName)
	if normalized == "" {
		return "", ErrWordlistFileNameEmpty
	}
	if len(normalized) > MaxWordlistFileNameLength {
		return "", ErrWordlistFileNameTooLong
	}
	if ContainsWordlistControlChars(normalized) || ContainsWordlistFileNamePathSyntax(normalized) {
		return "", ErrWordlistFileNameInvalid
	}
	return normalized, nil
}

func NewWordlist(fileName, description string) (*Wordlist, error) {
	normalizedFileName, err := ValidateWordlistFileName(fileName)
	if err != nil {
		return nil, err
	}

	return &Wordlist{
		FileName:    normalizedFileName,
		Description: NormalizeWordlistDescription(description),
		Tags:        []string{},
	}, nil
}

func NormalizeWordlistTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		value := RemoveWordlistControlChars(strings.TrimSpace(tag))
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	slices.Sort(normalized)
	return normalized
}

func (wordlist *Wordlist) UpdateTags(tags []string) {
	wordlist.Tags = NormalizeWordlistTags(tags)
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

// ContainsWordlistFileNamePathSyntax reports whether an uploaded file name contains path syntax.
// Wordlist file names later become leaf filenames on the
// Agent, so path-like tokens must be rejected at the source instead of relying
// on the execution artifact materializer to fail them later.
func ContainsWordlistFileNamePathSyntax(value string) bool {
	if value == "." || value == ".." {
		return true
	}
	return strings.ContainsAny(value, `/\`)
}

func RemoveWordlistControlChars(value string) string {
	return strings.Map(func(character rune) rune {
		if character < 32 && character != ' ' {
			return -1
		}
		return character
	}, value)
}
