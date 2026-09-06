package blacklistwiring

import blacklistapp "github.com/yyhuni/lunafox/server/internal/modules/blacklist/application"

var _ blacklistapp.BlacklistPolicyStore = (*blacklistPolicyStoreAdapter)(nil)
