package securitywiring

import securityapp "github.com/yyhuni/lunafox/server/internal/modules/security/application"

var _ securityapp.VulnerabilityTargetLookup = (*securityTargetLookupAdapter)(nil)
