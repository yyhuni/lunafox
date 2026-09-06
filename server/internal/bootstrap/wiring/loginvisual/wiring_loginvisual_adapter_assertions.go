package loginvisualwiring

import (
	loginvisualapp "github.com/yyhuni/lunafox/server/internal/modules/loginvisual/application"
	loginvisualinfra "github.com/yyhuni/lunafox/server/internal/modules/loginvisual/infrastructure"
	loginvisualrepo "github.com/yyhuni/lunafox/server/internal/modules/loginvisual/repository"
)

var _ loginvisualapp.SettingsStore = (*loginvisualrepo.SettingsRepository)(nil)
var _ loginvisualapp.ActiveSuperuserAuthorizer = (*loginvisualrepo.ActiveSuperuserRepository)(nil)
var _ loginvisualapp.DiscoverabilityStore = (*loginvisualrepo.DiscoverabilityRepository)(nil)
var _ loginvisualapp.MediaStore = (*loginvisualinfra.LocalMediaStore)(nil)
