package scanwiring

import (
	"context"
	"fmt"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
)

type hostPortCursorAdapter struct {
	repo *snapshotrepo.HostPortSnapshotRepository
}

func NewHostPortCursorAdapter(repo *snapshotrepo.HostPortSnapshotRepository) scanapp.HostPortCursor {
	return &hostPortCursorAdapter{repo: repo}
}

func (adapter *hostPortCursorAdapter) ForEachHostPortByScanID(ctx context.Context, scanID int, visit func(scanapp.HostPortEvidence) error) error {
	return adapter.repo.ForEachHostPortByScanID(ctx, scanID, func(evidence snapshotdomain.HostPortInputEvidence) error {
		return visit(scanapp.HostPortEvidence{Host: evidence.Host, IP: evidence.IP, Port: evidence.Port})
	})
}

type targetDNSNameCursorAdapter struct {
	repo *assetrepo.SubdomainRepository
}

// NewTargetInventoryDNSNameCursorAdapter binds the Scan execution boundary to
// the current Target subdomain repository without exposing asset persistence to
// the Agent data plane.
func NewTargetInventoryDNSNameCursorAdapter(repo *assetrepo.SubdomainRepository) scanapp.TargetDNSNameCursor {
	return &targetDNSNameCursorAdapter{repo: repo}
}

func (adapter *targetDNSNameCursorAdapter) ForEachDNSNameByTargetID(ctx context.Context, targetID int, visit func(string) error) error {
	if adapter == nil || adapter.repo == nil {
		return fmt.Errorf("Target inventory subdomain cursor is unavailable")
	}
	return adapter.repo.ForEachByTargetID(ctx, targetID, func(item assetdomain.Subdomain) error {
		return visit(item.DNSName)
	})
}

type targetHostPortCursorAdapter struct {
	repo *assetrepo.HostPortRepository
}

// NewTargetInventoryHostPortCursorAdapter exposes current Target HostPort
// evidence through the same narrow Scan application projection as snapshots.
func NewTargetInventoryHostPortCursorAdapter(repo *assetrepo.HostPortRepository) scanapp.TargetHostPortCursor {
	return &targetHostPortCursorAdapter{repo: repo}
}

func (adapter *targetHostPortCursorAdapter) ForEachHostPortByTargetID(ctx context.Context, targetID int, visit func(scanapp.HostPortEvidence) error) error {
	if adapter == nil || adapter.repo == nil {
		return fmt.Errorf("Target inventory HostPort cursor is unavailable")
	}
	return adapter.repo.ForEachByTargetID(ctx, targetID, func(item assetdomain.HostPort) error {
		return visit(scanapp.HostPortEvidence{Host: item.Host, IP: item.IP, Port: item.Port})
	})
}

type targetWebsiteURLCursorAdapter struct {
	repo *assetrepo.WebsiteRepository
}

type endpointURLCursorAdapter struct {
	repo *snapshotrepo.EndpointSnapshotRepository
}

func NewEndpointURLCursorAdapter(repo *snapshotrepo.EndpointSnapshotRepository) scanapp.EndpointURLCursor {
	return &endpointURLCursorAdapter{repo: repo}
}

func (adapter *endpointURLCursorAdapter) ForEachEndpointURLByScanID(ctx context.Context, scanID int, visit func(string) error) error {
	if adapter == nil || adapter.repo == nil {
		return fmt.Errorf("Endpoint snapshot cursor is unavailable")
	}
	return adapter.repo.ForEachByScanID(ctx, scanID, func(item snapshotdomain.EndpointSnapshot) error {
		return visit(item.URL)
	})
}

type targetEndpointURLCursorAdapter struct {
	repo *assetrepo.EndpointRepository
}

func NewTargetInventoryEndpointURLCursorAdapter(repo *assetrepo.EndpointRepository) scanapp.TargetEndpointURLCursor {
	return &targetEndpointURLCursorAdapter{repo: repo}
}

func (adapter *targetEndpointURLCursorAdapter) ForEachEndpointURLByTargetID(ctx context.Context, targetID int, visit func(string) error) error {
	if adapter == nil || adapter.repo == nil {
		return fmt.Errorf("Target inventory Endpoint cursor is unavailable")
	}
	return adapter.repo.ForEachByTargetID(ctx, targetID, func(item assetdomain.Endpoint) error {
		return visit(item.URL)
	})
}

// NewTargetInventoryWebsiteURLCursorAdapter exposes current Target website
// facts through the existing websiteURLs execution-input projection.
func NewTargetInventoryWebsiteURLCursorAdapter(repo *assetrepo.WebsiteRepository) scanapp.TargetWebsiteURLCursor {
	return &targetWebsiteURLCursorAdapter{repo: repo}
}

func (adapter *targetWebsiteURLCursorAdapter) ForEachWebsiteURLByTargetID(ctx context.Context, targetID int, visit func(string) error) error {
	if adapter == nil || adapter.repo == nil {
		return fmt.Errorf("Target inventory website cursor is unavailable")
	}
	return adapter.repo.ForEachByTargetID(ctx, targetID, func(item assetdomain.Website) error {
		return visit(item.URL)
	})
}
