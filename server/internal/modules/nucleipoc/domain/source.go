package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// SourceType is the operator-selected public Git source kind. The kind is
// descriptive; only the URL validator decides whether a host is acceptable.
type SourceType string

const (
	SourceTypeGit    SourceType = "git"
	SourceTypeGitee  SourceType = "gitee"
	SourceTypeCustom SourceType = "custom"
)

func (sourceType SourceType) Valid() bool {
	switch sourceType {
	case SourceTypeGit, SourceTypeGitee, SourceTypeCustom:
		return true
	default:
		return false
	}
}

// Source is the global source projection. An inactive source may be a staged
// candidate and must never be returned as the current source.
type Source struct {
	ID         uuid.UUID
	SourceType SourceType
	RepoURL    string
	IsActive   bool
	CommitSHA  string
	SyncedAt   *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (source Source) NormalizedURL() string { return strings.TrimSpace(source.RepoURL) }
