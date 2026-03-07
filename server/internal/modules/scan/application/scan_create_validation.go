package application

import (
	"fmt"
	"io/fs"
	"strings"

	workflowmanifest "github.com/yyhuni/lunafox/server/internal/workflow/manifest"
)

func validateWorkflowIDsStrict(workflowIds []string) error {
	if len(workflowIds) == 0 {
		return invalidWorkflowIDsf("workflowIds is required")
	}
	seen := make(map[string]struct{}, len(workflowIds))
	for index, workflowID := range workflowIds {
		if workflowID == "" {
			return invalidWorkflowIDsf("workflowIds[%d] must not be empty", index)
		}
		if strings.TrimSpace(workflowID) != workflowID {
			return invalidWorkflowIDsf("workflowIds[%d] must not contain leading or trailing spaces", index)
		}
		if _, exists := seen[workflowID]; exists {
			return invalidWorkflowIDsf("workflowIds[%d]=%q is duplicated", index, workflowID)
		}
		seen[workflowID] = struct{}{}
	}
	return nil
}

func (service *ScanCreateService) validateRequestedWorkflows(workflowIds []string) error {
	available, err := workflowmanifest.ListWorkflows()
	if err != nil {
		return err
	}
	set := make(map[string]struct{}, len(available))
	for _, item := range available {
		workflowID := strings.TrimSpace(item)
		if workflowID == "" {
			continue
		}
		set[workflowID] = struct{}{}
	}
	for _, workflowID := range workflowIds {
		workflowID = strings.TrimSpace(workflowID)
		if workflowID == "" {
			continue
		}
		if _, ok := set[workflowID]; ok {
			continue
		}
		return WrapSchemaInvalid(workflowID, "workflow "+workflowID+" not found in available workflow manifests", fs.ErrNotExist)
	}
	return nil
}

func invalidWorkflowIDsf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrCreateInvalidWorkflowIDs, fmt.Sprintf(format, args...))
}
