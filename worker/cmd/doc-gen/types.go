package main

import (
	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
)

// TemplateFile represents the structure of templates.yaml
type TemplateFile struct {
	Metadata workflow.WorkflowMetadata           `yaml:"metadata"`
	Tools    map[string]activity.CommandTemplate `yaml:"tools"`
}

var requiredLabelKeys = []string{
	"workflow_includes",
	"properties",
	"required_stage",
	"optional_stage",
	"supports_parallel",
	"stage_tools",
	"tool_name",
	"workflow_name",
	"description",
	"warning",
	"base_command",
	"internal_params",
	"runtime_parameters",
	"cli_parameters",
	"parameter",
	"type",
	"default",
	"required",
	"yes",
	"no",
	"config_example",
	"global_notes",
	"config_notes",
	"stage_notes",
	"tool_notes",
	"stage_prefix",
	"parallel_execution",
}

var supportedSections = map[string]struct{}{
	"workflow": {},
	"tools":    {},
	"examples": {},
	"notes":    {},
}
