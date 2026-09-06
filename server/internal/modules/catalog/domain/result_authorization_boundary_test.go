package domain

import (
	"reflect"
	"strings"
	"testing"
)

func TestInstalledEngineRecordDoesNotProjectResultAuthorization(t *testing.T) {
	typ := reflect.TypeOf(Engine{})
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		name := strings.ToLower(field.Name)
		if strings.Contains(name, "result") || strings.Contains(name, "output") || strings.Contains(name, "schema") {
			t.Fatalf("installed engine record must not project result authorization field %q", field.Name)
		}
	}
}
