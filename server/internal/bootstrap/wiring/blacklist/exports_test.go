package blacklistwiring

import (
	"errors"
	"testing"

	blacklistapp "github.com/yyhuni/lunafox/server/internal/modules/blacklist/application"
)

func TestBlacklistPolicyWiringFailsFastForMissingRepository(t *testing.T) {
	if store := NewBlacklistPolicyStoreAdapter(nil); store != nil {
		t.Fatalf("nil repository store = %#v, want nil", store)
	}
	if _, err := NewBlacklistPolicyApplicationService(nil); !errors.Is(err, blacklistapp.ErrBlacklistPolicyDependency) {
		t.Fatalf("missing store error = %v", err)
	}
}
