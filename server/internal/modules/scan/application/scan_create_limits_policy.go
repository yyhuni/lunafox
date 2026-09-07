package application

import (
	"fmt"
	"time"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	defaultPlanTaskMaxExecutionDuration    = 14 * 24 * time.Hour
	defaultPlanTaskProgressMessageMaxBytes = uint32(4096)
)

// PlanTaskLimitsProvider is a sealed Server composition policy. Scan requests,
// Engine config, and environment values cannot introduce another provider.
type PlanTaskLimitsProvider interface {
	Limits() PlanTaskLimits
	serverOwnedPlanTaskLimitsProvider()
}

type staticPlanTaskLimitsProvider struct {
	limits PlanTaskLimits
}

func (provider staticPlanTaskLimitsProvider) Limits() PlanTaskLimits {
	return provider.limits
}

func (staticPlanTaskLimitsProvider) serverOwnedPlanTaskLimitsProvider() {}

// DefaultPlanTaskLimitsProvider returns the production-owned execution limits.
func DefaultPlanTaskLimitsProvider() PlanTaskLimitsProvider {
	return staticPlanTaskLimitsProvider{limits: defaultPlanTaskLimits()}
}

// FixedPlanTaskLimitsProvider keeps protocol limits at their production values
// while allowing testsupport to select a short, positive execution budget.
func FixedPlanTaskLimitsProvider(maxExecutionDuration time.Duration) PlanTaskLimitsProvider {
	limits := defaultPlanTaskLimits()
	limits.MaxExecutionDuration = maxExecutionDuration
	return staticPlanTaskLimitsProvider{limits: limits}
}

func defaultPlanTaskLimits() PlanTaskLimits {
	resultLimits := contractresults.DefaultBatchLimits()
	return PlanTaskLimits{
		MaxExecutionDuration:    defaultPlanTaskMaxExecutionDuration,
		ProgressMessageMaxBytes: defaultPlanTaskProgressMessageMaxBytes,
		ResultBatchMaxItems:     uint32(resultLimits.MaxItems),
		ResultBatchMaxBytes:     uint32(resultLimits.MaxBytes),
	}
}

func validatePlanTaskLimits(limits PlanTaskLimits) error {
	if limits.MaxExecutionDuration <= 0 || limits.ProgressMessageMaxBytes == 0 || limits.ResultBatchMaxItems == 0 || limits.ResultBatchMaxBytes == 0 {
		return fmt.Errorf("all execution limits must be positive")
	}
	if !durationpb.New(limits.MaxExecutionDuration).IsValid() {
		return fmt.Errorf("max execution duration is not representable")
	}
	resultLimits := contractresults.DefaultBatchLimits()
	if limits.ResultBatchMaxItems > uint32(resultLimits.MaxItems) || limits.ResultBatchMaxBytes > uint32(resultLimits.MaxBytes) {
		return fmt.Errorf("result batch limits exceed the protocol maximum")
	}
	return nil
}
