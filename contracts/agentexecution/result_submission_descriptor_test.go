package agentexecution

import (
	"testing"

	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestResultSubmissionHopResponsesAreStatusOnly(t *testing.T) {
	agentResponse := (&agentdatav1.BatchIngestTaskResultsResponse{}).ProtoReflect().Descriptor()
	assertDescriptorFields(t, agentResponse, nil)
	if !reservedFieldNumber(agentResponse, 1) || !reservedFieldName(agentResponse, "summary") {
		t.Fatalf("%s must reserve removed summary field 1/name", agentResponse.FullName())
	}
}

func TestAgentResultSubmissionRequestCarriesExplicitScopeAndEncodedItemsOnly(t *testing.T) {
	request := (&agentdatav1.BatchIngestTaskResultsRequest{}).ProtoReflect().Descriptor()
	assertDescriptorFields(t, request, []descriptorField{
		{name: "items_json", number: 5, kind: protoreflect.StringKind, cardinality: protoreflect.Repeated},
		{name: "task", number: 6, kind: protoreflect.StringKind},
		{name: "target", number: 7, kind: protoreflect.StringKind},
		{name: "result_type", number: 8, kind: protoreflect.StringKind},
	})
	for _, forbidden := range []protoreflect.Name{
		"summary", "accepted", "duplicate", "invalid", "filtered", "unsupported",
		"snapshot_count", "asset_count", "success", "request_id", "sequence",
	} {
		if request.Fields().ByName(forbidden) != nil {
			t.Fatalf("Agent result request contains forbidden field %q", forbidden)
		}
	}
}
