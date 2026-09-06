package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (repository *EngineRepository) UpsertInstalledEngine(ctx context.Context, engine *catalogdomain.Engine, allowReplacement bool) error {
	if ctx == nil {
		return fmt.Errorf("installed engine persistence context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateEngineForPersistence(engine); err != nil {
		return err
	}
	record := engineDomainToModel(engine)
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.Engine
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("engine_id = ?", record.EngineID).
			First(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			insert := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "engine_id"}},
				DoNothing: true,
			}).Create(record)
			if insert.Error != nil {
				return insert.Error
			}
			if insert.RowsAffected == 1 {
				return nil
			}
			// A concurrent first install won the unique engine_id insert. Lock
			// that complete record and apply the same replay/replacement rules.
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("engine_id = ?", record.EngineID).
				First(&current).Error; err != nil {
				return fmt.Errorf("load concurrently installed engine %q: %w", record.EngineID, err)
			}
		} else if err != nil {
			return err
		}

		if current.PackageDigest == record.PackageDigest {
			if !samePackageRegistration(&current, record) {
				return fmt.Errorf(
					"installed engine %q package digest %q conflicts with existing package facts",
					record.EngineID,
					record.PackageDigest,
				)
			}
			if current.ArtifactRef == record.ArtifactRef {
				return nil
			}
			if !sameArtifactReferenceDigest(record.EngineID, current.ArtifactRef, record.ArtifactRef) {
				return fmt.Errorf(
					"installed engine %q package digest %q cannot change immutable OCI artifact reference digest",
					record.EngineID,
					record.PackageDigest,
				)
			}
		} else if !allowReplacement {
			return &catalogdomain.EngineReplacementConflictError{
				EngineID:              record.EngineID,
				CurrentPackageDigest:  current.PackageDigest,
				ProposedPackageDigest: record.PackageDigest,
			}
		}

		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "engine_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"publisher", "package_version", "artifact_ref", "package_digest", "manifest", "updated_at",
			}),
		}).Create(record).Error
	})
}

// The archive digest owns all package-derived registration facts. Replaying
// that digest may update only the successful Registry location; any other
// drift must fail instead of rewriting the current record under the same key.
func samePackageRegistration(current, next *model.Engine) bool {
	if current == nil || next == nil {
		return false
	}
	return current.EngineID == next.EngineID &&
		current.Publisher == next.Publisher &&
		current.PackageVersion == next.PackageVersion &&
		current.PackageDigest == next.PackageDigest &&
		jsonValuesEqual(current.Manifest, next.Manifest)
}

func jsonValuesEqual(left, right []byte) bool {
	var leftValue any
	var rightValue any
	if decodeJSONValue(left, &leftValue) != nil || decodeJSONValue(right, &rightValue) != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func decodeJSONValue(payload []byte, destination *any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("installed engine manifest must contain exactly one JSON value")
	}
	return nil
}

func sameArtifactReferenceDigest(engineID, left, right string) bool {
	leftReference, leftErr := parseInstalledEngineArtifactReference(engineID, left)
	rightReference, rightErr := parseInstalledEngineArtifactReference(engineID, right)
	return leftErr == nil &&
		rightErr == nil &&
		leftReference.Digest == rightReference.Digest
}

func parseInstalledEngineArtifactReference(_ string, value string) (ociartifact.DigestReference, error) {
	reference, err := ociartifact.ParseDigestReference(value)
	if err != nil {
		return ociartifact.DigestReference{}, fmt.Errorf("installed engine artifact_ref: %w", err)
	}
	if reference.String() != value {
		return ociartifact.DigestReference{}, fmt.Errorf("installed engine artifact_ref must be canonical")
	}
	return reference, nil
}

func validateEngineForPersistence(engine *catalogdomain.Engine) error {
	if engine == nil {
		return fmt.Errorf("installed engine is required")
	}
	for _, field := range []struct{ name, value string }{
		{"engine_id", engine.EngineID}, {"publisher", engine.Publisher}, {"package_version", engine.PackageVersion},
		{"artifact_ref", engine.ArtifactRef}, {"package_digest", engine.PackageDigest},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("installed engine %s is required", field.name)
		}
	}
	if _, err := parseInstalledEngineArtifactReference(engine.EngineID, engine.ArtifactRef); err != nil {
		return err
	}
	if len(engine.Manifest) == 0 {
		return fmt.Errorf("installed engine manifest is required")
	}
	var manifest any
	if err := decodeJSONValue(engine.Manifest, &manifest); err != nil {
		return fmt.Errorf("installed engine manifest must be valid JSON: %w", err)
	}
	return nil
}
