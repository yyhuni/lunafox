package application

import (
	"io"
	"time"
)

type WordlistFileMetadata struct {
	FilePath  string
	FileSize  int64
	LineCount int
	FileHash  string
}

type WordlistFileStore interface {
	Save(basePath, filename string, content io.Reader) (*WordlistFileMetadata, error)
	Write(path, content string) (*WordlistFileMetadata, error)
	Read(path string) (string, error)
	Remove(path string) error
	Exists(path string) bool
	RefreshMetadata(path string, knownSize int64, knownUpdatedAt time.Time) (*WordlistFileMetadata, bool, error)
}
