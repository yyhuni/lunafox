package application

import (
	"context"
	"errors"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

func TestCompositeVerifierFrontendOnlyRequiresPublicEdgeProbe(t *testing.T) {
	publicErr := errors.New("public edge has not converged")
	verifier, err := NewCompositeVerifier(CompositeVerifierConfig{
		Probes: map[string]HealthProbe{
			"frontend":       func(context.Context) error { return nil },
			"publicFrontend": func(context.Context) error { return publicErr },
		},
		RequiredProbes:  []string{"frontend", "publicFrontend"},
		RequiredDigests: []string{"frontend"},
	})
	if err != nil {
		t.Fatal(err)
	}
	operation := &domain.Operation{
		ExecutionMode:   domain.ExecutionModeFrontendOnly,
		WorkDisposition: domain.WorkDispositionNotRequired,
		MigrationType:   "none",
		MigrationStatus: domain.MigrationStatusNotStarted,
	}
	evidence := VerificationEvidence{
		ExpectedDigests: map[string]string{"frontend": "sha256:frontend"},
		ObservedDigests: map[string]string{"frontend": "sha256:frontend"},
	}
	result, err := verifier.Verify(context.Background(), operation, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed || result.Diagnostic == "" {
		t.Fatalf("frontend-only verification passed without public edge probe: %#v", result)
	}

	verifier, err = NewCompositeVerifier(CompositeVerifierConfig{
		Probes: map[string]HealthProbe{
			"frontend":       func(context.Context) error { return nil },
			"publicFrontend": func(context.Context) error { return nil },
		},
		RequiredProbes:  []string{"frontend", "publicFrontend"},
		RequiredDigests: []string{"frontend"},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err = verifier.Verify(context.Background(), operation, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed {
		t.Fatalf("frontend-only verification with public edge probe failed: %#v", result)
	}
}
