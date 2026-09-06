package engineexecution

import (
	"reflect"
	"strings"
	"testing"
)

func TestExecutionDefinitionDoesNotExposeRetiredInputOrResultAuthorization(t *testing.T) {
	definitionType := reflect.TypeOf(ExecutionDefinition{})
	for _, field := range []string{"Inputs", "ResultTypes", "ResultSchemas", "Results", "ResultAuthorization"} {
		if _, exists := definitionType.FieldByName(field); exists {
			t.Fatalf("ExecutionDefinition must not expose result authorization field %q", field)
		}
	}
}

func TestDecodeExecutionDefinitionRejectsResultAuthorizationFields(t *testing.T) {
	base := `{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","params":[{"key":"mode","type":"string"}]}]}`
	for _, field := range []string{"resultTypes", "resultSchemas", "results", "resultAuthorization"} {
		t.Run(field, func(t *testing.T) {
			payload := strings.TrimSuffix(base, "}") + `,"` + field + `":[]}`
			if _, err := DecodeExecutionDefinition([]byte(payload), "execution.json"); err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("DecodeExecutionDefinition() error = %v, want rejection of %q", err, field)
			}
		})
	}
}
