package repository

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func engineModelToDomain(engine *model.ScanEngine) *catalogdomain.ScanEngine {
	if engine == nil {
		return nil
	}
	return &catalogdomain.ScanEngine{
		ID:            engine.ID,
		Name:          engine.Name,
		Configuration: engine.Configuration,
		CreatedAt:     timeutil.ToUTC(engine.CreatedAt),
		UpdatedAt:     timeutil.ToUTC(engine.UpdatedAt),
	}
}

func engineDomainToModel(engine *catalogdomain.ScanEngine) *model.ScanEngine {
	if engine == nil {
		return nil
	}
	return &model.ScanEngine{
		ID:            engine.ID,
		Name:          engine.Name,
		Configuration: engine.Configuration,
		CreatedAt:     timeutil.ToUTC(engine.CreatedAt),
		UpdatedAt:     timeutil.ToUTC(engine.UpdatedAt),
	}
}

func engineModelListToDomain(engines []model.ScanEngine) []catalogdomain.ScanEngine {
	results := make([]catalogdomain.ScanEngine, 0, len(engines))
	for index := range engines {
		results = append(results, *engineModelToDomain(&engines[index]))
	}
	return results
}
