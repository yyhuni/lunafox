package domain

import (
	"encoding/json"
	"time"
)

// Engine is the current successfully installed package for one engine ID.
// Cache paths are derived from PackageDigest and are not stored in the database.
type Engine struct {
	ID             int
	EngineID       string
	Publisher      string
	PackageVersion string
	ArtifactRef    string
	PackageDigest  string
	Manifest       json.RawMessage
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
