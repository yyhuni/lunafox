package domain

import (
	"reflect"
	"testing"
)

func TestAgentHeartbeatUpdateDoesNotDefineRuntimeStatus(t *testing.T) {
	if _, ok := reflect.TypeOf(AgentHeartbeatUpdate{}).FieldByName("RuntimeStatus"); ok {
		t.Fatalf("AgentHeartbeatUpdate must not define RuntimeStatus")
	}
}
