package repository

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func targetModelToDomain(target *model.Target) *catalogdomain.Target {
	if target == nil {
		return nil
	}

	organizations := make([]catalogdomain.TargetOrganizationRef, 0, len(target.Organizations))
	for _, organization := range target.Organizations {
		organizations = append(organizations, catalogdomain.TargetOrganizationRef{
			ID:          organization.ID,
			Name:        organization.Name,
			Description: organization.Description,
			CreatedAt:   timeutil.ToUTC(organization.CreatedAt),
			DeletedAt:   timeutil.ToUTCPtr(organization.DeletedAt),
		})
	}

	return &catalogdomain.Target{
		ID:            target.ID,
		Name:          target.Name,
		Type:          target.Type,
		CreatedAt:     timeutil.ToUTC(target.CreatedAt),
		LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt),
		DeletedAt:     timeutil.ToUTCPtr(target.DeletedAt),
		Organizations: organizations,
	}
}

func targetDomainToModel(target *catalogdomain.Target) *model.Target {
	if target == nil {
		return nil
	}

	organizations := make([]model.TargetOrganizationRef, 0, len(target.Organizations))
	for _, organization := range target.Organizations {
		organizations = append(organizations, model.TargetOrganizationRef{
			ID:          organization.ID,
			Name:        organization.Name,
			Description: organization.Description,
			CreatedAt:   timeutil.ToUTC(organization.CreatedAt),
			DeletedAt:   timeutil.ToUTCPtr(organization.DeletedAt),
		})
	}

	return &model.Target{
		ID:            target.ID,
		Name:          target.Name,
		Type:          target.Type,
		CreatedAt:     timeutil.ToUTC(target.CreatedAt),
		LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt),
		DeletedAt:     timeutil.ToUTCPtr(target.DeletedAt),
		Organizations: organizations,
	}
}

func targetModelListToDomain(targets []model.Target) []catalogdomain.Target {
	results := make([]catalogdomain.Target, 0, len(targets))
	for index := range targets {
		results = append(results, *targetModelToDomain(&targets[index]))
	}
	return results
}

func targetDomainListToModel(targets []catalogdomain.Target) []model.Target {
	results := make([]model.Target, 0, len(targets))
	for index := range targets {
		results = append(results, *targetDomainToModel(&targets[index]))
	}
	return results
}
