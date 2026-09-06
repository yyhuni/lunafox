package blacklistwiring

import (
	blacklistapp "github.com/yyhuni/lunafox/server/internal/modules/blacklist/application"
	blacklistrepo "github.com/yyhuni/lunafox/server/internal/modules/blacklist/repository"
)

// NewBlacklistPolicyStoreAdapter exposes policy persistence through the
// application-owned store port.
func NewBlacklistPolicyStoreAdapter(repo *blacklistrepo.PolicyRepository) blacklistapp.BlacklistPolicyStore {
	if repo == nil {
		return nil
	}
	return newBlacklistPolicyStoreAdapter(repo)
}

// NewBlacklistPolicyApplicationService assembles the query and update paths
// behind one narrow application boundary.
func NewBlacklistPolicyApplicationService(store blacklistapp.BlacklistPolicyStore) (blacklistapp.BlacklistPolicyApplicationService, error) {
	query, err := blacklistapp.NewBlacklistPolicyQueryService(store)
	if err != nil {
		return nil, err
	}
	update, err := blacklistapp.NewBlacklistPolicyUpdateService(store)
	if err != nil {
		return nil, err
	}
	return blacklistapp.NewBlacklistPolicyFacade(query, update)
}
