package agentdata

import (
	"reflect"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// This boundary is deliberately structural: a future Server execution quota
// must arrive through a separately reviewed change, not by smuggling a
// record/byte/candidate field into an existing Engine plan or Artifact wire
// contract.
func TestExecutionArtifactBoundariesHaveNoServerBusinessQuotaFields(t *testing.T) {
	for _, descriptor := range []protoreflect.FileDescriptor{
		agentdatav1.File_lunafox_agent_data_v1_execution_artifact_proto,
		agentexecutionv1.File_lunafox_agent_execution_v1_resolved_engine_execution_plan_proto,
	} {
		walkQuotaFields(t, descriptor.Messages())
	}

	for _, descriptor := range executionartifact.Descriptors() {
		value := reflect.ValueOf(descriptor)
		for index := 0; index < value.NumField(); index++ {
			fieldName := value.Type().Field(index).Name
			if isBusinessQuotaField(fieldName) {
				t.Fatalf("execution artifact registry field %q looks like a business quota", fieldName)
			}
		}
	}
}

func walkQuotaFields(t *testing.T, messages protoreflect.MessageDescriptors) {
	t.Helper()
	for index := 0; index < messages.Len(); index++ {
		message := messages.Get(index)
		fields := message.Fields()
		for fieldIndex := 0; fieldIndex < fields.Len(); fieldIndex++ {
			field := fields.Get(fieldIndex)
			if isBusinessQuotaField(string(field.Name())) {
				t.Fatalf("protobuf field %s.%s introduces a Server business quota", message.FullName(), field.Name())
			}
		}
		walkQuotaFields(t, message.Messages())
	}
}

func isBusinessQuotaField(name string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(name, "_", ""))
	for _, token := range []string{"quota", "candidatecountlimit", "recordcountlimit", "bytecountlimit", "inputrecordlimit", "inputbytelimit", "websiteurllimit", "subdomainlimit", "hostportlimit", "cidrexpansionlimit"} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}
