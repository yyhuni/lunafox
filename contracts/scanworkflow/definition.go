package scanworkflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var (
	scanWorkflowIDPattern       = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	engineIDPattern             = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}$`)
	componentIDPattern          = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,127}$`)
	reservedScanWorkflowIDNames = map[string]struct{}{"all": {}}
)

type Definition struct {
	ScanWorkflowID string  `json:"scanWorkflowId"`
	DisplayName    string  `json:"displayName"`
	Description    string  `json:"description"`
	Stages         []Stage `json:"stages"`
}

type Stage struct {
	StageID string `json:"stageId"`
	Steps   []Step `json:"steps"`
}

type Step struct {
	StepID                string `json:"stepId"`
	EngineID              string `json:"engineId"`
	ProfileDefaultEnabled bool   `json:"profileDefaultEnabled"`
}

func (step *Step) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for key := range fields {
		switch key {
		case "stepId", "engineId", "profileDefaultEnabled":
		default:
			return fmt.Errorf("unknown step field %q", key)
		}
	}

	var stepID string
	if err := requireStringField(fields, "stepId", &stepID); err != nil {
		return err
	}
	var engineID string
	if err := requireStringField(fields, "engineId", &engineID); err != nil {
		return err
	}
	var profileDefaultEnabled bool
	if err := requireBoolField(fields, "profileDefaultEnabled", &profileDefaultEnabled); err != nil {
		return err
	}
	step.StepID = stepID
	step.EngineID = engineID
	step.ProfileDefaultEnabled = profileDefaultEnabled
	return nil
}

func requireStringField(fields map[string]json.RawMessage, key string, target *string) error {
	raw, ok := fields[key]
	if !ok {
		return fmt.Errorf("%s is required", key)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("%s must be string: %w", key, err)
	}
	return nil
}

func requireBoolField(fields map[string]json.RawMessage, key string, target *bool) error {
	raw, ok := fields[key]
	if !ok {
		return fmt.Errorf("%s is required", key)
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("%s must be boolean", key)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("%s must be boolean: %w", key, err)
	}
	return nil
}

func (definition *Definition) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for key := range fields {
		switch key {
		case "scanWorkflowId", "displayName", "description", "stages":
		default:
			return fmt.Errorf("unknown scan workflow field %q", key)
		}
	}

	var scanWorkflowID string
	if err := requireStringField(fields, "scanWorkflowId", &scanWorkflowID); err != nil {
		return err
	}
	var displayName string
	if err := requireStringField(fields, "displayName", &displayName); err != nil {
		return err
	}
	var description string
	if err := requireStringField(fields, "description", &description); err != nil {
		return err
	}
	rawStages, ok := fields["stages"]
	if !ok {
		return fmt.Errorf("stages is required")
	}
	var stages []Stage
	if err := json.Unmarshal(rawStages, &stages); err != nil {
		return fmt.Errorf("stages must be array: %w", err)
	}
	definition.ScanWorkflowID = scanWorkflowID
	definition.DisplayName = displayName
	definition.Description = description
	definition.Stages = stages
	return nil
}

func DecodeDefinition(payload []byte, source string) (Definition, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var definition Definition
	if err := decoder.Decode(&definition); err != nil {
		return Definition{}, fmt.Errorf("decode scan workflow definition %q: %w", source, err)
	}
	if err := consumeJSONEOF(decoder); err != nil {
		return Definition{}, fmt.Errorf("decode scan workflow definition %q: %w", source, err)
	}
	return definition, nil
}

func consumeJSONEOF(decoder *json.Decoder) error {
	if decoder == nil {
		return nil
	}
	if _, err := decoder.Token(); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return fmt.Errorf("unexpected trailing JSON content")
}

func ValidateDefinition(definition Definition) error {
	scanWorkflowID := strings.TrimSpace(definition.ScanWorkflowID)
	if scanWorkflowID == "" {
		return fmt.Errorf("scanWorkflowId is required")
	}
	if err := ValidateBuiltinScanWorkflowID(scanWorkflowID); err != nil {
		return err
	}
	if err := ValidateWorkflowMetadata(definition.DisplayName, definition.Description); err != nil {
		return err
	}
	return ValidateTopology(definition.Stages)
}

// ValidateComponentID validates the canonical workflow stage/step identifier
// grammar shared by definitions and compiled execution plans.
func ValidateComponentID(value string) error {
	if value != strings.TrimSpace(value) || !componentIDPattern.MatchString(value) {
		return fmt.Errorf("workflow component ID must use lower-case letters, digits, underscores, or hyphens")
	}
	return nil
}

func ValidateDefinitionList(items []Definition) error {
	seen := map[string]struct{}{}
	for _, item := range items {
		scanWorkflowID := strings.TrimSpace(item.ScanWorkflowID)
		if scanWorkflowID == "" {
			return fmt.Errorf("scanWorkflowId must not be empty")
		}
		if _, exists := seen[scanWorkflowID]; exists {
			return fmt.Errorf("duplicate manifest for scan workflow %q", scanWorkflowID)
		}
		seen[scanWorkflowID] = struct{}{}
	}
	return nil
}
