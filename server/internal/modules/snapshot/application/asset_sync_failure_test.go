package application

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

func TestSnapshotCommandServicesReturnAssetSyncErrors(t *testing.T) {
	assetErr := errors.New("asset projection failed")
	domainTarget := &snapshotdomain.ScanTargetRef{ID: 6, Name: "example.com", Type: "domain"}

	t.Run("subdomain", func(t *testing.T) {
		service := NewSubdomainSnapshotCommandService(
			&subdomainSnapshotCommandStoreStub{},
			&snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 4, TargetID: 6}, target: domainTarget},
			&subdomainAssetSyncStub{err: assetErr},
		)

		summary, err := service.SaveAndSync(context.Background(), 4, 6, []SubdomainSnapshotItem{{DNSName: "api.example.com"}})
		if !errors.Is(err, assetErr) {
			t.Fatalf("expected asset sync error, got %v", err)
		}
		if summary.SnapshotCount != 1 || summary.AssetCount != 0 {
			t.Fatalf("unexpected summary: %+v", summary)
		}
	})

	t.Run("website", func(t *testing.T) {
		service := NewWebsiteSnapshotCommandService(
			&websiteSnapshotCommandStoreStub{},
			&snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 4, TargetID: 6}, target: domainTarget},
			&websiteAssetSyncStub{err: assetErr},
		)

		summary, err := service.SaveAndSync(context.Background(), 4, 6, []WebsiteSnapshotItem{{URL: "https://example.com", Host: "example.com"}})
		if !errors.Is(err, assetErr) {
			t.Fatalf("expected asset sync error, got %v", err)
		}
		if summary.SnapshotCount != 1 || summary.AssetCount != 0 {
			t.Fatalf("unexpected summary: %+v", summary)
		}
	})

	t.Run("endpoint", func(t *testing.T) {
		store := &endpointSnapshotCommandStoreStub{}
		service := NewEndpointSnapshotCommandService(
			store,
			&snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 4, TargetID: 6}, target: domainTarget},
			&endpointAssetSyncStub{err: assetErr},
		)

		summary, err := service.SaveAndSync(context.Background(), 4, 6, []EndpointSnapshotItem{{URL: "https://example.com/api", Host: "example.com"}})
		if !errors.Is(err, assetErr) {
			t.Fatalf("expected asset sync error, got %v", err)
		}
		if summary.SnapshotCount != 1 || summary.AssetCount != 0 {
			t.Fatalf("unexpected summary: %+v", summary)
		}
		if len(store.snapshots) != 1 {
			t.Fatalf("snapshot write must remain after endpoint asset failure, got %d rows", len(store.snapshots))
		}
	})

	t.Run("directory", func(t *testing.T) {
		service := NewDirectorySnapshotCommandService(
			&directorySnapshotCommandStoreStub{},
			&snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 4, TargetID: 6}, target: domainTarget},
			&directoryAssetSyncStub{err: assetErr},
		)

		summary, err := service.SaveAndSync(context.Background(), 4, 6, []DirectorySnapshotItem{{URL: "https://example.com/admin"}})
		if !errors.Is(err, assetErr) {
			t.Fatalf("expected asset sync error, got %v", err)
		}
		if summary.SnapshotCount != 1 || summary.AssetCount != 0 {
			t.Fatalf("unexpected summary: %+v", summary)
		}
	})

	t.Run("host port", func(t *testing.T) {
		service := NewHostPortSnapshotCommandService(
			&hostPortSnapshotCommandStoreStub{},
			&snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 4, TargetID: 6}, target: domainTarget},
			&hostPortAssetSyncStub{err: assetErr},
		)

		summary, err := service.SaveAndSync(context.Background(), 4, 6, []HostPortSnapshotItem{{Host: "api.example.com", IP: "1.1.1.1", Port: 443}})
		if !errors.Is(err, assetErr) {
			t.Fatalf("expected asset sync error, got %v", err)
		}
		if summary.SnapshotCount != 1 || summary.AssetCount != 0 {
			t.Fatalf("unexpected summary: %+v", summary)
		}
	})

	t.Run("screenshot", func(t *testing.T) {
		service := NewScreenshotSnapshotCommandService(
			&screenshotSnapshotCommandStoreStub{},
			&snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 4, TargetID: 6}, target: domainTarget},
			&screenshotAssetSyncStub{err: assetErr},
		)

		summary, err := service.SaveAndSync(context.Background(), 4, 6, []ScreenshotSnapshotItem{{URL: "https://example.com", Image: validScreenshotImage()}})
		if !errors.Is(err, assetErr) {
			t.Fatalf("expected asset sync error, got %v", err)
		}
		if summary.SnapshotCount != 1 || summary.AssetCount != 0 {
			t.Fatalf("unexpected summary: %+v", summary)
		}
	})

	t.Run("vulnerability", func(t *testing.T) {
		score := decimal.NewFromFloat(7.2)
		service := NewVulnerabilitySnapshotCommandService(
			&vulnerabilitySnapshotCommandStoreStub{},
			&snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 4, TargetID: 6}, target: domainTarget},
			&vulnerabilityAssetSyncStub{err: assetErr},
			newVulnerabilityRawOutputCodecStub(),
		)

		summary, err := service.SaveAndSync(context.Background(), 4, []VulnerabilitySnapshotItem{{URL: "https://example.com", VulnType: "xss", CVSSScore: &score}})
		if !errors.Is(err, assetErr) {
			t.Fatalf("expected asset sync error, got %v", err)
		}
		if summary.SnapshotCount != 1 || summary.AssetCount != 0 {
			t.Fatalf("unexpected summary: %+v", summary)
		}
	})
}
