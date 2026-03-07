package cache

import (
	"reflect"
	"testing"
)

func TestHeartbeatDataUpdatedAtJSONTagUsesCamelCase(t *testing.T) {
	field, ok := reflect.TypeOf(HeartbeatData{}).FieldByName("UpdatedAt")
	if !ok {
		t.Fatalf("heartbeat data must define UpdatedAt")
	}
	if got := field.Tag.Get("json"); got != "updatedAt" {
		t.Fatalf("UpdatedAt must use updatedAt json tag, got %q", got)
	}
}
