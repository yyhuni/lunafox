package snapshotwiring

import (
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type snapshotEndpointCommandStoreAdapter struct {
	repo *snapshotrepo.EndpointSnapshotRepository
}

func newSnapshotEndpointCommandStoreAdapter(repo *snapshotrepo.EndpointSnapshotRepository) *snapshotEndpointCommandStoreAdapter {
	return &snapshotEndpointCommandStoreAdapter{repo: repo}
}

func (adapter *snapshotEndpointCommandStoreAdapter) BulkCreate(snapshots []snapshotdomain.EndpointSnapshot) (int64, error) {
	return adapter.repo.BulkCreate(snapshots)
}
