package application

import "context"

// BlacklistPolicyStore provides local singleton reads and etag-guarded
// replacements. ReadEffectivePatternsForScan uses the caller transaction when
// one is present so Scan creation can freeze a locked policy view.
type BlacklistPolicyStore interface {
	GetGlobal(context.Context) (*BlacklistPolicyRecord, error)
	GetTarget(context.Context, int) (*BlacklistPolicyRecord, error)
	ReplaceGlobal(context.Context, string, []string) (*BlacklistPolicyRecord, error)
	ReplaceTarget(context.Context, int, string, []string) (*BlacklistPolicyRecord, error)
	ReadEffectivePatternsForScan(context.Context, int) ([]string, error)
}

// BlacklistPolicyApplicationService is the narrow public application boundary
// consumed by HTTP and later by Scan creation wiring.
type BlacklistPolicyApplicationService interface {
	GetGlobal(context.Context) (*BlacklistPolicy, error)
	GetTarget(context.Context, int) (*BlacklistPolicy, error)
	ReplaceGlobal(context.Context, ReplaceBlacklistPolicyInput) (*BlacklistPolicy, error)
	ReplaceTarget(context.Context, int, ReplaceBlacklistPolicyInput) (*BlacklistPolicy, error)
	ResolveEffectivePatternsForScan(context.Context, int) ([]string, error)
}
