package application

import (
	"reflect"
	"strings"
	"testing"

	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
)

func TestResolvedEngineExecutionPlanDoesNotProjectResultAuthorization(t *testing.T) {
	typ := reflect.TypeOf(agentexecutionv1.ResolvedEngineExecutionPlan{})
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		name := strings.ToLower(field.Name)
		for _, forbidden := range []string{"result", "output", "schema", "allow"} {
			if strings.Contains(name, forbidden) {
				t.Fatalf("saved execution plan must not project result authorization field %q", field.Name)
			}
		}
	}
}
