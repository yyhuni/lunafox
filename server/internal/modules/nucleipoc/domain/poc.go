package domain

import (
	"time"

	"github.com/google/uuid"
)

// POC is the canonical, read-only content projection of one imported Nuclei
// template. IsEnabled is intentionally an independent runtime overlay.
type POC struct {
	ID            uuid.UUID
	SourceID      uuid.UUID
	TemplateID    string
	DisplayName   string
	Severity      string
	Tags          []string
	Author        string
	Description   string
	CVE           []string
	CWE           []string
	References    []string
	Remediation   string
	RelativePath  string
	ContentSHA256 string
	Content       string
	IsEnabled     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CandidatePOC is staged under a sync task and is promoted only as a complete
// set. It deliberately mirrors POC content without runtime enablement.
type CandidatePOC struct {
	ID            uuid.UUID
	TaskID        uuid.UUID
	SourceID      uuid.UUID
	TemplateID    string
	DisplayName   string
	Severity      string
	Tags          []string
	Author        string
	Description   string
	CVE           []string
	CWE           []string
	References    []string
	Remediation   string
	RelativePath  string
	ContentSHA256 string
	Content       string
	CreatedAt     time.Time
}
