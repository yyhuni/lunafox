package application

import (
	"fmt"
	"math"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
)

// ExecutionInputBlacklistStats is the aggregate counter value for one
// Server-owned finalized-fact materialization. It never becomes a Scan, Task,
// or protocol projection; the gRPC boundary consumes only a copied value for
// aggregate observations.
type ExecutionInputBlacklistStats struct {
	Examined uint64
	Excluded uint64
	Emitted  uint64
}

// ExecutionInputBlacklistFilter holds the one matcher compiled from the
// immutable Scan snapshot for one materialization attempt. It intentionally
// receives stored facts verbatim: finalization, not this boundary, owns repair
// and canonicalization of those facts.
type ExecutionInputBlacklistFilter struct {
	matcher *blacklistdomain.Matcher
	stats   ExecutionInputBlacklistStats
}

// NewExecutionInputBlacklistFilter requires an already compiled immutable
// matcher. The execution artifact resolver creates one before its stream
// header, so a producer cannot silently continue without a snapshot.
func NewExecutionInputBlacklistFilter(matcher *blacklistdomain.Matcher) (*ExecutionInputBlacklistFilter, error) {
	if matcher == nil {
		return nil, fmt.Errorf("execution input blacklist matcher is required")
	}
	return &ExecutionInputBlacklistFilter{matcher: matcher}, nil
}

// ShouldExcludeHostname records one finalized hostname inspection without
// normalizing or repairing its stored value.
func (filter *ExecutionInputBlacklistFilter) ShouldExcludeHostname(value string) (bool, error) {
	if err := filter.recordExamined(); err != nil {
		return false, err
	}
	if !filter.matcher.MatchesHostname(value) {
		return false, nil
	}
	if err := filter.recordExcluded(); err != nil {
		return false, err
	}
	return true, nil
}

// ShouldExcludeHostPort matches the two persisted identities independently;
// either match excludes the complete fact before any downstream derivation.
func (filter *ExecutionInputBlacklistFilter) ShouldExcludeHostPort(host, ip string) (bool, error) {
	if err := filter.recordExamined(); err != nil {
		return false, err
	}
	if !filter.matcher.MatchesHostPort(host, ip) {
		return false, nil
	}
	if err := filter.recordExcluded(); err != nil {
		return false, err
	}
	return true, nil
}

// ShouldExcludeURL matches only a Host derived from raw URL authority. Path,
// query, fragment, and percent syntax stay outside the blacklist language and
// are never parsed or rewritten here.
func (filter *ExecutionInputBlacklistFilter) ShouldExcludeURL(value string) (bool, error) {
	if err := filter.recordExamined(); err != nil {
		return false, err
	}
	host, err := contractresults.DeriveObservedAssetURLHost(value)
	if err != nil {
		return false, err
	}
	if !filter.matcher.MatchesHostname(host) {
		return false, nil
	}
	if err := filter.recordExcluded(); err != nil {
		return false, err
	}
	return true, nil
}

// RecordEmitted advances the counter only after the producer's callback has
// accepted a record, preserving partial-work counts on stream failure.
func (filter *ExecutionInputBlacklistFilter) RecordEmitted() error {
	if filter == nil {
		return fmt.Errorf("execution input blacklist filter is required")
	}
	if filter.stats.Emitted == math.MaxUint64 {
		return fmt.Errorf("execution input blacklist emitted counter overflow")
	}
	filter.stats.Emitted++
	return nil
}

// Stats returns a copy so callers cannot alter materialization accounting.
func (filter *ExecutionInputBlacklistFilter) Stats() ExecutionInputBlacklistStats {
	if filter == nil {
		return ExecutionInputBlacklistStats{}
	}
	return filter.stats
}

func (filter *ExecutionInputBlacklistFilter) recordExamined() error {
	if filter == nil || filter.matcher == nil {
		return fmt.Errorf("execution input blacklist filter is required")
	}
	if filter.stats.Examined == math.MaxUint64 {
		return fmt.Errorf("execution input blacklist examined counter overflow")
	}
	filter.stats.Examined++
	return nil
}

func (filter *ExecutionInputBlacklistFilter) recordExcluded() error {
	if filter == nil {
		return fmt.Errorf("execution input blacklist filter is required")
	}
	if filter.stats.Excluded == math.MaxUint64 {
		return fmt.Errorf("execution input blacklist excluded counter overflow")
	}
	filter.stats.Excluded++
	return nil
}
