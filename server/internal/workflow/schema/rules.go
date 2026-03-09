package workflowschema

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	workflowIDPattern       = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	reservedWorkflowIDNames = map[string]struct{}{
		"all":     {},
		"default": {},
	}
)

func schemaCacheKey(workflowID string) string {
	return workflowID
}

func schemaFilename(workflowID string) string {
	return fmt.Sprintf("%s.schema.json", workflowID)
}

func workflowIDFromSchemaFilename(filename string) (string, error) {
	base := filepath.Base(strings.TrimSpace(filename))
	if base == "" || !strings.HasSuffix(base, ".schema.json") {
		return "", fmt.Errorf("invalid schema filename %q", filename)
	}

	workflowID := strings.TrimSuffix(base, ".schema.json")
	if err := validateWorkflowID(workflowID); err != nil {
		return "", err
	}

	return workflowID, nil
}

func validateWorkflowID(workflowID string) error {
	if !workflowIDPattern.MatchString(workflowID) {
		return fmt.Errorf(
			"invalid workflowId %q: must match %s",
			workflowID,
			workflowIDPattern.String(),
		)
	}
	if _, reserved := reservedWorkflowIDNames[workflowID]; reserved {
		return fmt.Errorf("reserved workflowId %q is not allowed", workflowID)
	}
	return nil
}
