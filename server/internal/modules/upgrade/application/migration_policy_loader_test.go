package application

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

func TestParseMigrationPolicyStrictly(t *testing.T) {
	raw, err := os.ReadFile(fixturePath("migration-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := ParseMigrationPolicy(raw)
	if err != nil {
		t.Fatalf("ParseMigrationPolicy() error = %v", err)
	}
	if policy.Phase != "disposable-development" || policy.SchemaVersion != 1 || policy.DataRetainingDeploymentAllowed {
		t.Fatalf("unexpected policy %#v", policy)
	}

	for _, test := range []struct {
		name string
		body string
	}{
		{name: "unknown field", body: strings.Replace(string(raw), `"phase":`, `"unknown": true, "phase":`, 1)},
		{name: "missing required field", body: strings.Replace(string(raw), `  "phase": "disposable-development",`, "", 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseMigrationPolicy([]byte(test.body))
			if !errors.Is(err, domain.ErrMigrationMetadataMissing) || domain.CodeOf(err) != domain.ErrorCodeMigrationMetadataMissing {
				t.Fatalf("expected migration metadata error, got %v (code=%q)", err, domain.CodeOf(err))
			}
		})
	}
}
