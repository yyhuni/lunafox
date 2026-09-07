package application

import (
	"context"
	"io"

	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/domain"
)

type ActiveSuperuserAuthorizer interface {
	IsActiveSuperuser(context.Context, int) (bool, error)
}

type DiscoverabilityStore interface {
	IsUnlocked(context.Context, int) (bool, error)
	Unlock(context.Context, int) error
}

type SettingsStore interface {
	Get(context.Context) (domain.Settings, error)
	SaveDraft(context.Context, domain.Media) (domain.Settings, error)
	PublishDraft(context.Context) (domain.Settings, error)
	RestoreDefault(context.Context) (domain.Settings, error)
}

type MediaStore interface {
	Save(context.Context, []byte) (domain.Media, error)
	Open(context.Context, domain.Media, bool) (io.ReadCloser, error)
	Delete(context.Context, domain.Media) error
}
