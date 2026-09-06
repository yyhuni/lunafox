package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/blacklist/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrPolicyNotFound      = errors.New("blacklist policy not found")
	ErrPolicyETagConflict  = errors.New("blacklist policy etag conflict")
	ErrPolicyDataIntegrity = errors.New("blacklist policy data integrity failure")
	ErrInvalidPolicyUpdate = errors.New("invalid blacklist policy update")
)

// PolicyRepository owns policy persistence. Its scan-effective read resolves a
// transaction handle from context so the lock belongs to Scan creation rather
// than an independently committed read.
type PolicyRepository struct {
	db *gorm.DB
}

func NewPolicyRepository(db *gorm.DB) *PolicyRepository {
	return &PolicyRepository{db: db}
}

func (repository *PolicyRepository) GetGlobal(ctx context.Context) (*blacklistdomain.Policy, error) {
	return repository.get(ctx, blacklistdomain.ScopeGlobal, nil, false)
}

func (repository *PolicyRepository) GetTarget(ctx context.Context, targetID int) (*blacklistdomain.Policy, error) {
	if targetID <= 0 {
		return nil, fmt.Errorf("%w: target id is required", ErrInvalidPolicyUpdate)
	}
	return repository.get(ctx, blacklistdomain.ScopeTarget, &targetID, false)
}

// ReadEffectivePatternsForScan reads global and Target-local rows through the
// caller transaction. The fixed query order prevents cross-scope lock-order
// drift while FOR SHARE keeps concurrent Scan creates compatible.
func (repository *PolicyRepository) ReadEffectivePatternsForScan(ctx context.Context, targetID int) ([]string, error) {
	if targetID <= 0 {
		return nil, fmt.Errorf("%w: target id is required", ErrInvalidPolicyUpdate)
	}
	db, err := repository.resolve(ctx)
	if err != nil {
		return nil, err
	}
	var rows []model.Policy
	err = db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "SHARE"}).
		Where(`(scope = ? AND target_id IS NULL) OR (scope = ? AND target_id = ?)`, blacklistdomain.ScopeGlobal, blacklistdomain.ScopeTarget, targetID).
		Order(`CASE WHEN scope = 'global' THEN 0 ELSE 1 END`).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) != 2 {
		return nil, ErrPolicyNotFound
	}
	var global, target *blacklistdomain.Policy
	for index := range rows {
		policy, err := policyModelToDomain(&rows[index])
		if err != nil {
			return nil, err
		}
		switch policy.Scope {
		case blacklistdomain.ScopeGlobal:
			if global != nil {
				return nil, fmt.Errorf("%w: duplicate global policy", ErrPolicyDataIntegrity)
			}
			global = policy
		case blacklistdomain.ScopeTarget:
			if policy.TargetID == nil || *policy.TargetID != targetID || target != nil {
				return nil, fmt.Errorf("%w: invalid target policy row", ErrPolicyDataIntegrity)
			}
			target = policy
		default:
			return nil, fmt.Errorf("%w: unknown policy scope", ErrPolicyDataIntegrity)
		}
	}
	if global == nil || target == nil {
		return nil, ErrPolicyNotFound
	}
	patterns, err := blacklistdomain.CanonicalizeEffectivePatterns(global.Patterns, target.Patterns)
	if err != nil {
		return nil, err
	}
	return append([]string(nil), patterns...), nil
}

// ReplacePatterns locks one singleton row, gives stale etag precedence over
// equality, and skips UPDATE entirely for canonically identical content.
func (repository *PolicyRepository) ReplacePatterns(
	ctx context.Context,
	scope blacklistdomain.Scope,
	targetID *int,
	expectedETag string,
	patterns []string,
) (*blacklistdomain.Policy, bool, error) {
	if strings.TrimSpace(expectedETag) == "" {
		return nil, false, fmt.Errorf("%w: etag is required", ErrInvalidPolicyUpdate)
	}
	if err := validatePolicyIdentity(scope, targetID); err != nil {
		return nil, false, err
	}
	if err := blacklistdomain.ValidateCanonicalPolicyPatterns(patterns); err != nil {
		return nil, false, err
	}
	db, err := repository.resolve(ctx)
	if err != nil {
		return nil, false, err
	}
	var result *blacklistdomain.Policy
	changed := false
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		persisted, err := repository.getFrom(tx, ctx, scope, targetID, true)
		if err != nil {
			return err
		}
		current, err := policyModelToDomain(persisted)
		if err != nil {
			return err
		}
		currentETag, err := blacklistdomain.ETag(current.Patterns)
		if err != nil {
			return fmt.Errorf("%w: derive current etag: %v", ErrPolicyDataIntegrity, err)
		}
		if expectedETag != currentETag {
			return ErrPolicyETagConflict
		}
		if slices.Equal(current.Patterns, patterns) {
			result = current
			return nil
		}
		payload, err := json.Marshal(patterns)
		if err != nil {
			return fmt.Errorf("encode canonical blacklist patterns: %w", err)
		}
		now := time.Now().UTC()
		if err := tx.Model(&model.Policy{}).
			Where("id = ?", persisted.ID).
			Updates(map[string]any{"patterns": modelJSON(payload), "updated_at": now}).Error; err != nil {
			return err
		}
		persisted.Patterns = modelJSON(payload)
		persisted.UpdatedAt = now
		result, err = policyModelToDomain(persisted)
		if err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return result, changed, nil
}

func (repository *PolicyRepository) get(ctx context.Context, scope blacklistdomain.Scope, targetID *int, lock bool) (*blacklistdomain.Policy, error) {
	if err := validatePolicyIdentity(scope, targetID); err != nil {
		return nil, err
	}
	db, err := repository.resolve(ctx)
	if err != nil {
		return nil, err
	}
	persisted, err := repository.getFrom(db, ctx, scope, targetID, lock)
	if err != nil {
		return nil, err
	}
	return policyModelToDomain(persisted)
}

func (repository *PolicyRepository) getFrom(db *gorm.DB, ctx context.Context, scope blacklistdomain.Scope, targetID *int, lock bool) (*model.Policy, error) {
	query := db.WithContext(ctx).Where("scope = ?", scope)
	if targetID == nil {
		query = query.Where("target_id IS NULL")
	} else {
		query = query.Where("target_id = ?", *targetID)
	}
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var persisted model.Policy
	if err := query.First(&persisted).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPolicyNotFound
		}
		return nil, err
	}
	return &persisted, nil
}

func (repository *PolicyRepository) resolve(ctx context.Context) (*gorm.DB, error) {
	if repository == nil || repository.db == nil {
		return nil, fmt.Errorf("blacklist policy repository is required")
	}
	if ctx == nil {
		return nil, fmt.Errorf("blacklist policy context is required")
	}
	return dbtx.Resolve(ctx, repository.db), nil
}

func validatePolicyIdentity(scope blacklistdomain.Scope, targetID *int) error {
	switch scope {
	case blacklistdomain.ScopeGlobal:
		if targetID != nil {
			return fmt.Errorf("%w: global policy must not have target id", ErrInvalidPolicyUpdate)
		}
	case blacklistdomain.ScopeTarget:
		if targetID == nil || *targetID <= 0 {
			return fmt.Errorf("%w: target policy requires target id", ErrInvalidPolicyUpdate)
		}
	default:
		return fmt.Errorf("%w: unsupported policy scope", ErrInvalidPolicyUpdate)
	}
	return nil
}

func policyModelToDomain(persisted *model.Policy) (*blacklistdomain.Policy, error) {
	if persisted == nil {
		return nil, ErrPolicyNotFound
	}
	if persisted.ID <= 0 {
		return nil, fmt.Errorf("%w: missing policy id", ErrPolicyDataIntegrity)
	}
	scope := blacklistdomain.Scope(persisted.Scope)
	if err := validatePolicyIdentity(scope, persisted.TargetID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPolicyDataIntegrity, err)
	}
	if len(persisted.Patterns) == 0 || strings.TrimSpace(string(persisted.Patterns)) == "null" {
		return nil, fmt.Errorf("%w: missing pattern array", ErrPolicyDataIntegrity)
	}
	var patterns []string
	if err := json.Unmarshal(persisted.Patterns, &patterns); err != nil || patterns == nil {
		return nil, fmt.Errorf("%w: invalid pattern array", ErrPolicyDataIntegrity)
	}
	if err := blacklistdomain.ValidateCanonicalPolicyPatterns(patterns); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPolicyDataIntegrity, err)
	}
	return &blacklistdomain.Policy{
		ID:        persisted.ID,
		Scope:     scope,
		TargetID:  cloneTargetID(persisted.TargetID),
		Patterns:  append([]string(nil), patterns...),
		UpdatedAt: persisted.UpdatedAt.UTC(),
	}, nil
}

func cloneTargetID(value *int) *int {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
