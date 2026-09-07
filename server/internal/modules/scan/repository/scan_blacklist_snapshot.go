package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	// ErrScanBlacklistSnapshotNotFound distinguishes a missing immutable Scan
	// input from a valid empty snapshot.
	ErrScanBlacklistSnapshotNotFound = errors.New("scan blacklist snapshot not found")
	// ErrScanBlacklistSnapshotDataIntegrity rejects JSON or canonicality failures
	// instead of treating them as an empty policy.
	ErrScanBlacklistSnapshotDataIntegrity = errors.New("scan blacklist snapshot data integrity failure")
)

// LoadBlacklistSnapshot returns only the immutable patterns for one Scan. It
// intentionally does not join blacklist_policy, Scan, or Task state.
func (repository *ScanRepository) LoadBlacklistSnapshot(ctx context.Context, scanID int) ([]string, error) {
	if scanID <= 0 {
		return nil, fmt.Errorf("scan id is required")
	}
	db, err := repository.resolveScanDatabase(ctx)
	if err != nil {
		return nil, err
	}

	var snapshot model.ScanBlacklistSnapshot
	if err := db.WithContext(ctx).Where("scan_id = ?", scanID).Take(&snapshot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScanBlacklistSnapshotNotFound
		}
		return nil, err
	}
	patterns, err := decodeScanBlacklistSnapshotPatterns(snapshot.Patterns)
	if err != nil {
		return nil, err
	}
	return append([]string{}, patterns...), nil
}

func (repository *ScanRepository) createBlacklistSnapshot(tx *gorm.DB, scanID int, patterns []string) error {
	if tx == nil {
		return fmt.Errorf("scan blacklist snapshot transaction is required")
	}
	if scanID <= 0 {
		return fmt.Errorf("scan blacklist snapshot scan id is required")
	}
	canonical := append([]string{}, patterns...)
	if err := blacklistdomain.ValidateCanonicalEffectivePatterns(canonical); err != nil {
		return fmt.Errorf("%w: %v", ErrScanBlacklistSnapshotDataIntegrity, err)
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return fmt.Errorf("encode scan blacklist snapshot: %w", err)
	}
	return tx.Create(&model.ScanBlacklistSnapshot{
		ScanID:   scanID,
		Patterns: datatypes.JSON(append([]byte(nil), payload...)),
	}).Error
}

func decodeScanBlacklistSnapshotPatterns(payload datatypes.JSON) ([]string, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("%w: missing pattern array", ErrScanBlacklistSnapshotDataIntegrity)
	}
	var patterns []string
	if err := json.Unmarshal(payload, &patterns); err != nil || patterns == nil {
		return nil, fmt.Errorf("%w: invalid pattern array", ErrScanBlacklistSnapshotDataIntegrity)
	}
	if err := blacklistdomain.ValidateCanonicalEffectivePatterns(patterns); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrScanBlacklistSnapshotDataIntegrity, err)
	}
	return patterns, nil
}

func (repository *ScanRepository) resolveScanDatabase(ctx context.Context) (*gorm.DB, error) {
	if repository == nil || repository.db == nil {
		return nil, fmt.Errorf("scan repository is required")
	}
	if ctx == nil {
		return nil, fmt.Errorf("scan repository context is required")
	}
	return dbtx.Resolve(ctx, repository.db), nil
}
