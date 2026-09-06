package blacklistwiring

import (
	"errors"
	"testing"
	"time"

	blacklistapp "github.com/yyhuni/lunafox/server/internal/modules/blacklist/application"
	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	blacklistrepo "github.com/yyhuni/lunafox/server/internal/modules/blacklist/repository"
)

func TestMapBlacklistPolicyRepositoryErrorPreservesApplicationSemantics(t *testing.T) {
	tests := []struct {
		name  string
		input error
		want  error
	}{
		{name: "missing", input: blacklistrepo.ErrPolicyNotFound, want: blacklistapp.ErrBlacklistPolicyNotFound},
		{name: "stale etag", input: blacklistrepo.ErrPolicyETagConflict, want: blacklistapp.ErrBlacklistPolicyConflict},
		{name: "corrupt row", input: blacklistrepo.ErrPolicyDataIntegrity, want: blacklistapp.ErrBlacklistPolicyDataIntegrity},
		{name: "invalid replacement", input: blacklistrepo.ErrInvalidPolicyUpdate, want: blacklistapp.ErrBlacklistPolicyInvalidArgument},
		{name: "invalid syntax", input: blacklistdomain.ErrInvalidPattern, want: blacklistapp.ErrBlacklistPolicyInvalidArgument},
		{name: "effective overflow", input: blacklistdomain.ErrEffectiveLimitExceeded, want: blacklistapp.ErrBlacklistPolicyDataIntegrity},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := mapBlacklistPolicyRepositoryError(test.input); !errors.Is(got, test.want) {
				t.Fatalf("mapBlacklistPolicyRepositoryError(%v) = %v, want %v", test.input, got, test.want)
			}
		})
	}
}

func TestToBlacklistPolicyRecordDoesNotExposePersistenceIdentity(t *testing.T) {
	targetID := 17
	updatedAt := time.Date(2026, 8, 5, 13, 0, 0, 0, time.UTC)
	policy := &blacklistdomain.Policy{
		ID:        999,
		Scope:     blacklistdomain.ScopeTarget,
		TargetID:  &targetID,
		Patterns:  []string{"example.com"},
		UpdatedAt: updatedAt,
	}
	record, err := toBlacklistPolicyRecord(policy)
	if err != nil {
		t.Fatal(err)
	}
	if record.Scope != blacklistapp.BlacklistPolicyScopeTarget || record.TargetID == nil || *record.TargetID != targetID || record.UpdatedAt != updatedAt {
		t.Fatalf("unexpected mapped record: %#v", record)
	}
	record.Patterns[0] = "changed.example"
	if policy.Patterns[0] != "example.com" {
		t.Fatal("mapped record must own a detached pattern slice")
	}
}
