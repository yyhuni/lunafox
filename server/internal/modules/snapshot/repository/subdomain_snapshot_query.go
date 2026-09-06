package repository

import (
	"context"
	"strings"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// FindByScanID finds subdomain snapshots by scan ID with pagination and filter
func (r *SubdomainSnapshotRepository) ListByScanID(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.SubdomainSnapshot, int64, error) {
	var snapshots []model.SubdomainSnapshot
	var total int64

	baseQuery := r.db.Model(&model.SubdomainSnapshot{}).Where("scan_id = ?", scanID)
	baseQuery = baseQuery.Scopes(scope.WithFilterDefault(filter, subdomainSnapshotFilterMappingNormalized, "dnsName"))

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.Scopes(
		scope.WithPagination(page, pageSize),
		func(db *gorm.DB) *gorm.DB { return applySubdomainSnapshotOrder(db, orderBy) },
	).Find(&snapshots).Error
	if err != nil {
		return nil, 0, err
	}

	return subdomainSnapshotModelListToDomain(snapshots), total, nil
}

func applySubdomainSnapshotOrder(db *gorm.DB, orderBy string) *gorm.DB {
	switch strings.TrimSpace(orderBy) {
	case "dnsName", "dnsName asc":
		return db.Order("dns_name ASC").Order("id ASC")
	case "dnsName desc":
		return db.Order("dns_name DESC").Order("id DESC")
	case "createdAt", "createdAt asc":
		return db.Order("created_at ASC").Order("id ASC")
	case "createdAt desc", "":
		return db.Order("created_at DESC").Order("id DESC")
	default:
		return db.Where("1 = 0")
	}
}

// ForEachByScanID streams subdomain snapshots for a scan without exposing SQL cursor lifecycle to callers.
func (r *SubdomainSnapshotRepository) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.SubdomainSnapshot) error) error {
	rows, err := r.db.WithContext(ctx).Model(&model.SubdomainSnapshot{}).
		Where("scan_id = ?", scanID).
		Order("created_at DESC").
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var snapshot model.SubdomainSnapshot
		if err := r.db.ScanRows(rows, &snapshot); err != nil {
			return err
		}
		if err := visit(*subdomainSnapshotModelToDomain(&snapshot)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// ForEachDNSNameByScanID streams the finalized dns_name projection in stable
// canonical order without exposing SQL cursor lifecycle.
func (r *SubdomainSnapshotRepository) ForEachDNSNameByScanID(ctx context.Context, scanID int, visit func(string) error) error {
	rows, err := r.db.WithContext(ctx).Model(&model.SubdomainSnapshot{}).
		Select("dns_name").
		Where("scan_id = ?", scanID).
		Order("dns_name ASC").
		Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var dnsName string
		if err := rows.Scan(&dnsName); err != nil {
			return err
		}
		if err := visit(dnsName); err != nil {
			return err
		}
	}
	return rows.Err()
}

// CountByScanID returns the count of subdomain snapshots for a scan
func (r *SubdomainSnapshotRepository) CountByScanID(scanID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.SubdomainSnapshot{}).Where("scan_id = ?", scanID).Count(&count).Error
	return count, err
}
