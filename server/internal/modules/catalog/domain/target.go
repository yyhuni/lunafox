package domain

import (
	"net"
	"strconv"
	"strings"
	"time"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
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
	canonical, _, err := canonicalTarget(strings.TrimSpace(name))
	if err == nil {
		return canonical
	}
	return strings.TrimSpace(name)
}

func NormalizeBatchTargetName(name string) string {
	canonical, _, err := canonicalTarget(strings.TrimSpace(name))
	if err == nil {
		return canonical
	}
	return strings.ToLower(strings.TrimSpace(name))
}

func DetectTargetType(name string) string {
	_, targetType, err := canonicalTarget(name)
	if err != nil {
		return ""
	}
	return targetType
}

func BuildTarget(rawName string) (*Target, error) {
	normalizedName, targetType, err := canonicalTarget(rawName)
	if err != nil {
		return nil, ErrInvalidTarget
	}

	return &Target{Name: normalizedName, Type: targetType}, nil
}

func BuildBatchTarget(rawName string) (*Target, error) {
	normalizedName, targetType, err := canonicalTarget(rawName)
	if err != nil {
		return nil, ErrInvalidTarget
	}

	return &Target{Name: normalizedName, Type: targetType}, nil
}

// canonicalTarget is the single target write-boundary normalizer. IPv6 is
// rejected here so it cannot reach planning, while IPv4 CIDRs are persisted
// from their masked network address rather than the caller's host address.
func canonicalTarget(rawName string) (string, string, error) {
	value := strings.TrimSpace(rawName)
	if value == "" || strings.ContainsAny(value, "\r\n\x00") {
		return "", "", ErrInvalidTarget
	}

	if strings.Contains(value, ":") {
		return "", "", ErrInvalidTarget
	}
	if _, network, err := net.ParseCIDR(value); err == nil {
		ones, bits := network.Mask.Size()
		if bits != 32 || network.IP.To4() == nil {
			return "", "", ErrInvalidTarget
		}
		return network.IP.To4().String() + "/" + strconv.Itoa(ones), TargetTypeCIDR, nil
	}
	if ip := net.ParseIP(value); ip != nil {
		if ip.To4() == nil {
			return "", "", ErrInvalidTarget
		}
		return ip.To4().String(), TargetTypeIP, nil
	}
	if looksLikeIP(value) {
		return "", "", ErrInvalidTarget
	}
	if domain, ok := contractresults.NormalizeSubdomainDNSName(value); ok {
		return domain, TargetTypeDomain, nil
	}
	return "", "", ErrInvalidTarget
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
