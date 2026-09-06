package nucleipocwiring

import (
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/application"
	nucleipocrepo "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/repository"
)

var _ application.Store = (*nucleipocrepo.NucleiPOCRepository)(nil)
