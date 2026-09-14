package application

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

// HealthProbe is a narrow, side-effect-free check owned by the deployment
// wiring. Keeping probes behind this interface lets the upgrade state machine
// remain independent from HTTP, Redis, Loki and ORM clients.
type HealthProbe func(context.Context) error

// CompositeVerifierConfig declares the required service/dependency probes. A
// missing probe is treated as a failed check; callers must explicitly provide
// every production signal instead of silently degrading the success contract.
type CompositeVerifierConfig struct {
	Probes          map[string]HealthProbe
	RequiredProbes  []string
	RequiredDigests []string
}

// CompositeVerifier evaluates all evidence needed to mark an Upgrade
// Operation succeeded. It never mutates the Operation and never treats a host
// completion receipt as sufficient evidence.
type CompositeVerifier struct {
	probes          map[string]HealthProbe
	requiredProbes  []string
	requiredDigests []string
}

func NewCompositeVerifier(config CompositeVerifierConfig) (*CompositeVerifier, error) {
	if len(config.Probes) == 0 {
		return nil, fmt.Errorf("upgrade verifier probes are required")
	}
	probes := make(map[string]HealthProbe, len(config.Probes))
	for name, probe := range config.Probes {
		name = strings.TrimSpace(name)
		if name == "" || probe == nil {
			return nil, fmt.Errorf("upgrade verifier probe %q is invalid", name)
		}
		probes[name] = probe
	}
	requiredProbes := uniqueSorted(config.RequiredProbes)
	if len(requiredProbes) == 0 {
		for name := range probes {
			requiredProbes = append(requiredProbes, name)
		}
		sort.Strings(requiredProbes)
	}
	for _, name := range requiredProbes {
		if _, ok := probes[name]; !ok {
			return nil, fmt.Errorf("required upgrade verifier probe %q is not configured", name)
		}
	}
	requiredDigests := uniqueSorted(config.RequiredDigests)
	if len(requiredDigests) == 0 {
		requiredDigests = []string{"server", "frontend", "nginx"}
	}
	return &CompositeVerifier{probes: probes, requiredProbes: requiredProbes, requiredDigests: requiredDigests}, nil
}

// NewUpgradeVerifier is a descriptive alias for deployment wiring.
func NewUpgradeVerifier(config CompositeVerifierConfig) (*CompositeVerifier, error) {
	return NewCompositeVerifier(config)
}

func (verifier *CompositeVerifier) Verify(ctx context.Context, operation *domain.Operation, evidence VerificationEvidence) (VerificationResult, error) {
	if verifier == nil {
		return VerificationResult{}, fmt.Errorf("upgrade verifier is not configured")
	}
	if operation == nil {
		return VerificationResult{}, fmt.Errorf("upgrade operation is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return VerificationResult{}, err
	}
	if operation.MigrationType != "none" && operation.MigrationStatus != domain.MigrationStatusSucceeded {
		return VerificationResult{Passed: false, Diagnostic: "database migration has not been proved successful"}, nil
	}
	if operation.AgentSummary.Ready != operation.AgentSummary.Expected || operation.AgentSummary.Missing != 0 || operation.AgentSummary.Unhealthy != 0 {
		return VerificationResult{Passed: false, Diagnostic: "one or more Agents are not ready"}, nil
	}

	failed := make([]string, 0)
	for _, name := range verifier.requiredProbes {
		if err := verifier.probes[name](ctx); err != nil {
			failed = append(failed, name)
		}
	}
	if len(failed) > 0 {
		return VerificationResult{Passed: false, ObservedDigests: cloneStringMap(evidence.ObservedDigests), Diagnostic: "health checks incomplete: " + strings.Join(failed, ", ")}, nil
	}
	for _, name := range verifier.requiredDigests {
		want := strings.TrimSpace(evidence.ExpectedDigests[name])
		got := strings.TrimSpace(evidence.ObservedDigests[name])
		if want == "" || got == "" || want != got {
			return VerificationResult{Passed: false, ObservedDigests: cloneStringMap(evidence.ObservedDigests), Diagnostic: fmt.Sprintf("observed digest for %s does not match the release target", name)}, nil
		}
	}
	return VerificationResult{Passed: true, ObservedDigests: cloneStringMap(evidence.ObservedDigests)}, nil
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

var _ UpgradeVerifier = (*CompositeVerifier)(nil)
