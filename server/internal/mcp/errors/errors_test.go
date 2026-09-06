package errors

import (
	"errors"
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

func TestMapDomainErrorUsesSafeCreationVocabulary(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want error
	}{
		{name: "organization exists", err: identitydomain.ErrOrganizationExists, want: ErrAlreadyExists},
		{name: "organization absent", err: identitydomain.ErrOrganizationNotFound, want: ErrNotFound},
		{name: "target organization absent", err: catalogdomain.ErrTargetOrgNotFound, want: ErrNotFound},
		{name: "invalid target", err: catalogdomain.ErrInvalidTarget, want: ErrInvalidInput},
		{name: "association failure", err: catalogdomain.ErrTargetOrgBindingFail, want: ErrCommandFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := MapDomainError(tc.err); !errors.Is(got, tc.want) {
				t.Fatalf("MapDomainError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestAlreadyExistsFailureIsBounded(t *testing.T) {
	failure := From(ErrAlreadyExists)
	if failure.Category != CategoryAlreadyExists || failure.Message == "" || failure.Recovery == "" {
		t.Fatalf("already-exists failure = %+v", failure)
	}
	if !IsExpected(ErrAlreadyExists) || !IsExpected(ErrCommandFailed) {
		t.Fatal("expected command failures must remain tools/call errors")
	}
}
