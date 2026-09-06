package adapters

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type scanOperationAdapter struct{ facade *scanapp.ScanFacade }

// NewScanStarter projects the canonical Scan-create service onto start_scan.
func NewScanStarter(facade *scanapp.ScanFacade) tools.ScanStarter {
	return &scanOperationAdapter{facade: facade}
}

// NewOperationReader projects Scan-backed operation polling onto MCP.
func NewOperationReader(facade *scanapp.ScanFacade) tools.OperationReader {
	return &scanOperationAdapter{facade: facade}
}

func (adapter *scanOperationAdapter) Start(ctx context.Context, input tools.StartScanInput) (tools.StartScanOutput, error) {
	if adapter == nil || adapter.facade == nil {
		return tools.StartScanOutput{}, mcpErrors.ErrInternal
	}
	targetID, err := resourcenames.ParseTarget(input.Target)
	if err != nil || resourcenames.Target(targetID) != strings.TrimSpace(input.Target) {
		return tools.StartScanOutput{}, mcpErrors.ErrInvalidInput
	}
	workflow, err := resourcenames.ParseScanWorkflow(input.ScanWorkflow)
	if err != nil || resourcenames.ScanWorkflow(workflow) != strings.TrimSpace(input.ScanWorkflow) {
		return tools.StartScanOutput{}, mcpErrors.ErrInvalidInput
	}
	agentID, err := parseOptionalAgent(input.Agent)
	if err != nil {
		return tools.StartScanOutput{}, mcpErrors.ErrInvalidInput
	}
	result, err := adapter.facade.StartMCPScan(ctx, scanapp.MCPScanStartRequest{
		TargetID: targetID, ScanWorkflow: resourcenames.ScanWorkflow(workflow), Configuration: input.Configuration, AgentID: agentID, RequestID: input.RequestID,
	})
	if err != nil {
		return tools.StartScanOutput{}, mapMCPScanOperationError(err)
	}
	return tools.StartScanOutput{Operation: result.Operation, Scan: result.Scan}, nil
}

func (adapter *scanOperationAdapter) GetOperation(ctx context.Context, name string) (tools.OperationRecord, error) {
	if adapter == nil || adapter.facade == nil {
		return tools.OperationRecord{}, mcpErrors.ErrInternal
	}
	operationID, err := parseOperationName(name)
	if err != nil {
		return tools.OperationRecord{}, mcpErrors.ErrInvalidInput
	}
	operation, err := adapter.facade.GetMCPOperation(ctx, operationID)
	if err != nil {
		return tools.OperationRecord{}, mapMCPScanOperationError(err)
	}
	if operation == nil {
		return tools.OperationRecord{}, mcpErrors.ErrNotFound
	}
	if _, err := canonicalOperationStatus(operation.Status); err != nil {
		// A persisted value outside the Scan status contract must never be
		// presented as a fabricated PENDING operation.
		return tools.OperationRecord{}, mcpErrors.ErrInternal
	}
	return operationRecord(*operation), nil
}

func parseOptionalAgent(name string) (*int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	value, err := resourcenames.ParseAgent(name)
	if err != nil || resourcenames.Agent(value) != name {
		return nil, fmt.Errorf("agent must be canonical")
	}
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("agent ID must be positive")
	}
	return &id, nil
}

func parseOperationName(name string) (string, error) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 2 || parts[0] != "operations" {
		return "", fmt.Errorf("operation must use operations/{uuid}")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil || id.String() != parts[1] {
		return "", fmt.Errorf("operation must use a canonical UUID")
	}
	return id.String(), nil
}

func operationRecord(operation scanapp.MCPScanOperation) tools.OperationRecord {
	status, _ := canonicalOperationStatus(operation.Status)
	progress := operation.Progress
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	if status == "SUCCEEDED" {
		// A successful Scan is a complete operation even if a stale read model
		// has not yet observed its final numeric progress write.
		progress = 100
	}
	record := tools.OperationRecord{
		Name:        "operations/" + operation.ID,
		Scan:        resourcenames.Scan(operation.ScanID),
		Target:      resourcenames.Target(operation.TargetID),
		Status:      status,
		Phase:       operation.Phase,
		Progress:    progress,
		CurrentTask: operation.CurrentTask,
		CreateTime:  operation.CreatedAt.UTC(),
		UpdateTime:  operation.UpdatedAt.UTC(),
	}
	switch status {
	case "SUCCEEDED":
		record.Response = map[string]any{"scan": record.Scan}
	case "FAILED":
		record.Error = map[string]any{"code": "SCAN_FAILED", "message": "The scan failed."}
	case "CANCELLED":
		record.Error = map[string]any{"code": "SCAN_CANCELLED", "message": "The scan was cancelled."}
	}
	return record
}

func canonicalOperationStatus(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pending":
		return "PENDING", nil
	case "running":
		return "RUNNING", nil
	case "succeeded":
		return "SUCCEEDED", nil
	case "failed":
		return "FAILED", nil
	case "cancelled":
		return "CANCELLED", nil
	}
	return "", fmt.Errorf("unsupported Scan operation status")
}

func mapMCPScanOperationError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, scanapp.ErrMCPRequestIDConflict),
		errors.Is(err, scanapp.ErrCreateInvalidConfig),
		errors.Is(err, scanapp.ErrCreateInvalidScanWorkflow),
		errors.Is(err, scanapp.ErrCreateNoScanWorkflows),
		errors.Is(err, scanapp.ErrScanEngineConfigInvalid),
		errors.Is(err, scanapp.ErrCreateInvalidInputSource),
		errors.Is(err, scanapp.ErrCreateInvalidTriggerType):
		return mcpErrors.ErrInvalidInput
	case errors.Is(err, scanapp.ErrMCPOperationNotFound),
		errors.Is(err, scanapp.ErrCreateTargetNotFound),
		errors.Is(err, scanapp.ErrTargetNotFound),
		errors.Is(err, scanapp.ErrCreateAgentNotFound),
		errors.Is(err, scanapp.ErrScanAgentNotFound):
		return mcpErrors.ErrNotFound
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	default:
		return mcpErrors.ErrCommandFailed
	}
}
