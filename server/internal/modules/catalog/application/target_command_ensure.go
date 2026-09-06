package application

import (
	"context"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

func (service *TargetCommandService) EnsureTargets(ctx context.Context, names []string) (*EnsureTargetsResult, error) {
	_ = ctx

	validTargets := make([]catalogdomain.Target, 0, len(names))
	validNames := make([]string, 0, len(names))
	errors := make([]EnsureTargetError, 0)
	seen := make(map[string]struct{})

	for _, rawName := range names {
		target, err := catalogdomain.BuildBatchTarget(rawName)
		if err != nil {
			normalized := catalogdomain.NormalizeBatchTargetName(rawName)
			errors = append(errors, EnsureTargetError{Input: normalized, Error: "unrecognized target format"})
			continue
		}
		if _, ok := seen[target.Name]; ok {
			continue
		}
		seen[target.Name] = struct{}{}
		validTargets = append(validTargets, *target)
		validNames = append(validNames, target.Name)
	}

	if len(validTargets) == 0 {
		return &EnsureTargetsResult{
			TargetStats: EnsureTargetStats{Failed: len(errors)},
			Errors:      errors,
		}, nil
	}

	existing, err := service.store.FindByNames(validNames)
	if err != nil {
		return nil, err
	}
	existingByName := make(map[string]catalogdomain.Target, len(existing))
	for _, target := range existing {
		existingByName[target.Name] = target
	}

	createdCount := 0
	for _, target := range validTargets {
		if _, ok := existingByName[target.Name]; ok {
			continue
		}
		targetToCreate := target
		if err := service.store.Create(&targetToCreate); err != nil {
			return nil, err
		}
		existingByName[targetToCreate.Name] = targetToCreate
		createdCount++
	}

	resolved := make([]catalogdomain.Target, 0, len(validNames))
	for _, name := range validNames {
		target, ok := existingByName[name]
		if !ok {
			errors = append(errors, EnsureTargetError{Input: name, Error: "target was not persisted"})
			continue
		}
		resolved = append(resolved, target)
	}

	return &EnsureTargetsResult{
		Targets: resolved,
		TargetStats: EnsureTargetStats{
			Created: createdCount,
			Skipped: len(validNames) - createdCount,
			Failed:  len(errors),
		},
		Errors: errors,
	}, nil
}
