package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/contracts/scanworkflow"
)

// ManagedScanWorkflow is the persisted workflow aggregate. Engine definitions
// own configuration; this aggregate owns only ordered orchestration.
type ManagedScanWorkflow struct {
	ScanWorkflowID   string
	DisplayName      string
	Description      string
	Stages           []scanworkflow.Stage
	IsBuiltin        bool
	DefinitionDigest string
	RequestID        string
	Version          int64
	CreateTime       time.Time
	UpdateTime       time.Time
}

func (workflow ManagedScanWorkflow) Validate() error {
	if workflow.IsBuiltin {
		if err := scanworkflow.ValidateBuiltinScanWorkflowID(workflow.ScanWorkflowID); err != nil {
			return err
		}
		if len(workflow.DefinitionDigest) != 64 {
			return fmt.Errorf("built-in workflow definition digest is required")
		}
		if workflow.RequestID != "" {
			return fmt.Errorf("built-in workflow must not have requestId")
		}
	} else {
		if err := scanworkflow.ValidateUserScanWorkflowID(workflow.ScanWorkflowID); err != nil {
			return err
		}
		if workflow.DefinitionDigest != "" {
			return fmt.Errorf("user workflow must not have definition digest")
		}
		if workflow.RequestID != "" {
			parsed, err := uuid.Parse(workflow.RequestID)
			if err != nil || parsed.String() != workflow.RequestID {
				return fmt.Errorf("requestId must be a canonical UUID")
			}
		}
	}
	if err := scanworkflow.ValidateWorkflowMetadata(workflow.DisplayName, workflow.Description); err != nil {
		return err
	}
	if err := scanworkflow.ValidateTopology(workflow.Stages); err != nil {
		return err
	}
	if workflow.Version < 0 {
		return fmt.Errorf("workflow version must not be negative")
	}
	return nil
}

type ScanWorkflowListFilter struct {
	Page     int
	PageSize int
	Filter   string
}

func (filter ScanWorkflowListFilter) NormalizedFilter() string {
	return strings.TrimSpace(filter.Filter)
}

type ScanWorkflowCommandRepository interface {
	CreateScanWorkflow(workflow *ManagedScanWorkflow) error
	FindScanWorkflowByRequestID(requestID string) (*ManagedScanWorkflow, error)
	UpdateUserScanWorkflow(workflow *ManagedScanWorkflow, expectedVersion int64) (bool, error)
}

type ScanWorkflowQueryRepository interface {
	GetScanWorkflowByID(scanWorkflowID string) (*ManagedScanWorkflow, error)
	ListScanWorkflows(filter ScanWorkflowListFilter) ([]ManagedScanWorkflow, int64, error)
}
