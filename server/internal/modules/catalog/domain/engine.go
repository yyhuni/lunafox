package domain

import (
	"strings"
	"time"
)

// ScanEngine represents a scan engine catalog entry in domain layer.
type ScanEngine struct {
	ID            int
	Name          string
	Configuration string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NormalizeEngineName(name string) string {
	return strings.TrimSpace(name)
}

func NewScanEngine(name, configuration string) (*ScanEngine, error) {
	normalized := NormalizeEngineName(name)
	if normalized == "" {
		return nil, ErrInvalidEngine
	}

	return &ScanEngine{
		Name:          normalized,
		Configuration: configuration,
	}, nil
}

func (engine *ScanEngine) Rename(name string) error {
	normalized := NormalizeEngineName(name)
	if normalized == "" {
		return ErrInvalidEngine
	}
	engine.Name = normalized
	return nil
}

func (engine *ScanEngine) Reconfigure(configuration string) {
	engine.Configuration = configuration
}
