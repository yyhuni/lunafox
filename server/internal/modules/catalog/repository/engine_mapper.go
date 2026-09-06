package repository

import (
	"encoding/json"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
	"gorm.io/datatypes"
)

func engineModelToDomain(engine *model.Engine) *catalogdomain.Engine {
	if engine == nil {
		return nil
	}
	return &catalogdomain.Engine{
		ID: engine.ID, EngineID: engine.EngineID, Publisher: engine.Publisher,
		PackageVersion: engine.PackageVersion, ArtifactRef: engine.ArtifactRef,
		PackageDigest: engine.PackageDigest,
		Manifest:      append(json.RawMessage(nil), engine.Manifest...),
		CreatedAt:     timeutil.ToUTC(engine.CreatedAt), UpdatedAt: timeutil.ToUTC(engine.UpdatedAt),
	}
}

func engineDomainToModel(engine *catalogdomain.Engine) *model.Engine {
	if engine == nil {
		return nil
	}
	return &model.Engine{
		ID: engine.ID, EngineID: engine.EngineID, Publisher: engine.Publisher,
		PackageVersion: engine.PackageVersion, ArtifactRef: engine.ArtifactRef,
		PackageDigest: engine.PackageDigest,
		Manifest:      datatypes.JSON(append(json.RawMessage(nil), engine.Manifest...)),
		CreatedAt:     timeutil.ToUTC(engine.CreatedAt), UpdatedAt: timeutil.ToUTC(engine.UpdatedAt),
	}
}
