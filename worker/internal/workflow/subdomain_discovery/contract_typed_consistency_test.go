package subdomain_discovery

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedTypedConfigMatchesContract(t *testing.T) {
	def := GetContractDefinition()
	root := reflect.TypeOf(WorkflowConfig{})

	for _, stage := range def.Stages {
		stageField, ok := findFieldByJSONTag(root, stage.ID)
		require.Truef(t, ok, "missing stage field for %s", stage.ID)

		stageType := stageField.Type
		_, ok = findFieldByJSONTag(stageType, "enabled")
		require.Truef(t, ok, "stage %s missing enabled field", stage.ID)

		stageToolsField, ok := findFieldByJSONTag(stageType, "tools")
		require.Truef(t, ok, "stage %s missing tools field", stage.ID)
		stageToolsType := stageToolsField.Type

		for _, tool := range stage.Tools {
			toolField, ok := findFieldByJSONTag(stageToolsType, tool.ID)
			require.Truef(t, ok, "stage %s missing tool %s", stage.ID, tool.ID)

			toolType := toolField.Type
			_, ok = findFieldByJSONTag(toolType, "enabled")
			require.Truef(t, ok, "tool %s missing enabled field", tool.ID)

			for _, param := range tool.Params {
				paramField, ok := findFieldByJSONTag(toolType, param.Key)
				require.Truef(t, ok, "tool %s missing param %s", tool.ID, param.Key)
				require.Equalf(t, expectedGoKind(param.Type), paramField.Type.Kind(), "param %s kind mismatch", param.Key)
			}
		}
	}
}

func findFieldByJSONTag(rt reflect.Type, tag string) (reflect.StructField, bool) {
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		rawTag := field.Tag.Get("json")
		jsonTag := strings.Split(rawTag, ",")[0]
		if jsonTag == tag {
			return field, true
		}
	}
	return reflect.StructField{}, false
}

func expectedGoKind(paramType string) reflect.Kind {
	switch strings.ToLower(strings.TrimSpace(paramType)) {
	case "integer":
		return reflect.Int
	case "boolean":
		return reflect.Bool
	default:
		return reflect.String
	}
}
