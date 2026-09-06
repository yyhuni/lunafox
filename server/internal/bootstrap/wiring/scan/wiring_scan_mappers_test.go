package scanwiring

import (
	"testing"
	"time"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

func TestCatalogDomainTargetToScanAppTargetRef(t *testing.T) {
	now := time.Now().UTC()
	target := &catalogdomain.Target{ID: 1, Name: "example.com", Type: "domain", CreatedAt: now}
	result := catalogDomainTargetToScanAppTargetRef(target)
	if result == nil {
		t.Fatalf("expected target ref")
	}
	if result.ID != 1 || result.Name != "example.com" || result.Type != "domain" {
		t.Fatalf("unexpected target ref: %+v", result)
	}
}
