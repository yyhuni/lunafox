package protocol

import "fmt"

// Limits is the transport-only portion of an Engine Context.
type Limits struct {
	ProgressMessageMaxBytes uint32
	ResultBatchMaxItems     uint32
	ResultBatchMaxBytes     uint32
}

// ValidateLimits rejects non-positive or over-ceiling transport limits.
func ValidateLimits(limits *ExecutionLimits) error {
	if limits == nil || limits.GetProgressMessageMaxBytes() == 0 || limits.GetResultBatchMaxItems() == 0 || limits.GetResultBatchMaxBytes() == 0 {
		return fmt.Errorf("positive Engine reporting limits are required")
	}
	if limits.GetProgressMessageMaxBytes() > ProgressMessageMaxBytesCeiling ||
		limits.GetResultBatchMaxItems() > ResultBatchMaxItemsCeiling ||
		limits.GetResultBatchMaxBytes() > ResultBatchMaxBytesCeiling {
		return fmt.Errorf("Engine reporting limits exceed protocol hard ceiling")
	}
	return nil
}
