package domain

import (
	"net"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
)

const (
	TargetTypeDomain = "domain"
	TargetTypeIP     = "ip"
	TargetTypeCIDR   = "cidr"
)

// TargetOrganizationRef is a local projection for target-organization relation.
type TargetOrganizationRef struct {
	ID          int
	Name        string
	Description string
	CreatedAt   time.Time
	DeletedAt   *time.Time
}

// Target represents scan target aggregate in domain layer.
type Target struct {
	ID            int
	Name          string
	Type          string
	CreatedAt     time.Time
	LastScannedAt *time.Time
	DeletedAt     *time.Time
	Organizations []TargetOrganizationRef
}

func NormalizeTargetName(name string) string {
	return strings.TrimSpace(name)
}

func NormalizeBatchTargetName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func DetectTargetType(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	if _, _, err := net.ParseCIDR(name); err == nil {
		return TargetTypeCIDR
	}

	if net.ParseIP(name) != nil {
		return TargetTypeIP
	}

	if looksLikeIP(name) {
		return ""
	}

	if govalidator.IsDNSName(name) {
		return TargetTypeDomain
	}

	return ""
}

func BuildTarget(rawName string) (*Target, error) {
	normalizedName := NormalizeTargetName(rawName)
	if normalizedName == "" {
		return nil, ErrInvalidTarget
	}

	targetType := DetectTargetType(normalizedName)
	if targetType == "" {
		return nil, ErrInvalidTarget
	}

	return &Target{Name: normalizedName, Type: targetType}, nil
}

func BuildBatchTarget(rawName string) (*Target, error) {
	normalizedName := NormalizeBatchTargetName(rawName)
	if normalizedName == "" {
		return nil, ErrInvalidTarget
	}

	targetType := DetectTargetType(normalizedName)
	if targetType == "" {
		return nil, ErrInvalidTarget
	}

	return &Target{Name: normalizedName, Type: targetType}, nil
}

func (target *Target) Rename(rawName string) error {
	next, err := BuildTarget(rawName)
	if err != nil {
		return err
	}
	target.Name = next.Name
	target.Type = next.Type
	return nil
}

func looksLikeIP(value string) bool {
	if strings.Count(value, ".") == 3 {
		parts := strings.Split(value, ".")
		allNumeric := true
		for _, part := range parts {
			if part == "" {
				allNumeric = false
				break
			}
			for _, character := range part {
				if character < '0' || character > '9' {
					allNumeric = false
					break
				}
			}
			if !allNumeric {
				break
			}
		}
		if allNumeric {
			return true
		}
	}

	if strings.Contains(value, ":") && !strings.Contains(value, "://") {
		return true
	}

	return false
}
