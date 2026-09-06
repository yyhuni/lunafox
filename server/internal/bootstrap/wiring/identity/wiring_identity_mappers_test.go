package identitywiring

import (
	"testing"
	"time"

	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
	identitymodel "github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
)

func TestIdentityModelUsersToIdentityDomainUsers(t *testing.T) {
	users := []identitymodel.User{{ID: 1, Username: "alice"}, {ID: 2, Username: "bob"}}
	result := identityModelUsersToIdentityDomainUsers(users)
	if len(result) != 2 {
		t.Fatalf("expected 2 users, got %d", len(result))
	}
	if result[0].Username != "alice" || result[1].Username != "bob" {
		t.Fatalf("unexpected users: %+v", result)
	}
}

func TestIdentityRepositoryOrganizationWithCountsToIdentityDomainOrganizationWithTargetCounts(t *testing.T) {
	now := time.Now().UTC()
	items := []identityrepo.OrganizationWithCount{{Organization: identitymodel.Organization{ID: 10, Name: "acme", CreatedAt: now}, TargetCount: 7}}
	result := identityRepositoryOrganizationWithCountsToIdentityDomainOrganizationWithTargetCounts(items)
	if len(result) != 1 {
		t.Fatalf("expected 1 org, got %d", len(result))
	}
	if result[0].Name != "acme" || result[0].TargetCount != 7 {
		t.Fatalf("unexpected orgs: %+v", result)
	}
}

func TestIdentityModelTargetRefsToIdentityDomainTargetRefs(t *testing.T) {
	now := time.Now().UTC()
	items := []identitymodel.OrganizationTargetRef{{ID: 21, Name: "example.com", Type: "domain", CreatedAt: now}}
	result := identityModelTargetRefsToIdentityDomainTargetRefs(items)
	if len(result) != 1 {
		t.Fatalf("expected 1 target, got %d", len(result))
	}
	if result[0].Name != "example.com" || result[0].Type != "domain" {
		t.Fatalf("unexpected targets: %+v", result)
	}
}
