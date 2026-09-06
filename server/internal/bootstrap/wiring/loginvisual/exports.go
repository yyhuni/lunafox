// Package loginvisualwiring assembles the login visual settings boundary.
package loginvisualwiring

import (
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/application"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/handler"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/infrastructure"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/repository"
	"gorm.io/gorm"
)

type Module struct{ Handler *handler.LoginVisualHandler }

func NewLoginVisualModule(db *gorm.DB, storageRoot string) (*Module, error) {
	if db == nil {
		panic("login visual wiring database is required")
	}
	media, err := infrastructure.NewLocalMediaStore(storageRoot)
	if err != nil {
		return nil, err
	}
	service := application.NewService(
		repository.NewSettingsRepository(db),
		repository.NewActiveSuperuserRepository(db),
		repository.NewDiscoverabilityRepository(db),
		media,
	)
	return &Module{Handler: handler.NewLoginVisualHandler(service)}, nil
}
