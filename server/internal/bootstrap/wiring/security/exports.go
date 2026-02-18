package securitywiring

import (
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	securityapp "github.com/yyhuni/lunafox/server/internal/modules/security/application"
	securityrepo "github.com/yyhuni/lunafox/server/internal/modules/security/repository"
)

func NewSecurityTargetLookupAdapter(repo *catalogrepo.TargetRepository) securityapp.VulnerabilityTargetLookup {
	return newSecurityTargetLookupAdapter(repo)
}

func NewSecurityVulnerabilityStoreAdapter(repo securityrepo.VulnerabilityRepository) securityapp.VulnerabilityStore {
	return repo
}
