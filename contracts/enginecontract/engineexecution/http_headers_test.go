package engineexecution

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestHTTPHeaderFormatSurvivesDecodeCloneAndRoundtrip(t *testing.T) {
	payload := []byte(`{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","defaultEnabled":true,"params":[{"key":"request-fields","type":"stringArray","format":"http-headers","default":[]},{"key":"labels","type":"stringArray","default":[]}]}]}`)
	definition, err := DecodeExecutionDefinition(payload, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	cloned := CloneExecutionDefinition(definition)
	if cloned.ConfigSections[0].Params[0].Format != ParamFormatHTTPHeaders || cloned.ConfigSections[0].Params[1].Format != "" {
		t.Fatalf("format metadata lost: %+v", cloned.ConfigSections[0].Params)
	}
	encoded, err := json.Marshal(cloned)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(encoded), `"format"`) != 1 {
		t.Fatalf("optional format omission lost: %s", encoded)
	}
	roundtrip, err := DecodeExecutionDefinition(encoded, "roundtrip")
	if err != nil || !reflect.DeepEqual(roundtrip, definition) {
		t.Fatalf("roundtrip = %+v, %v", roundtrip, err)
	}
}

func TestHTTPHeaderFormatRejectsInvalidDeclarations(t *testing.T) {
	for _, declaration := range []string{
		`{"key":"fields","type":"stringArray","format":"unknown","default":[]}`,
		`{"key":"fields","type":"stringArray","format":"","default":[]}`,
		`{"key":"fields","type":"string","format":"http-headers","default":"x"}`,
		`{"key":"fields","type":"integer","format":"http-headers","default":1}`,
		`{"key":"fields","type":"boolean","format":"http-headers","default":true}`,
		`{"key":"fields","type":"stringArray","format":"http-headers","enum":["X: v"],"default":[]}`,
		`{"key":"fields","type":"stringArray","format":null,"default":[]}`,
	} {
		t.Run(declaration, func(t *testing.T) {
			payload := `{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","params":[` + declaration + `]}]}`
			if _, err := DecodeExecutionDefinition([]byte(payload), "fixture"); err == nil {
				t.Fatal("invalid format declaration accepted")
			}
		})
	}
}

func TestHTTPHeaderDefaultsAndRuntimeValuesShareValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []string
		valid  bool
	}{
		{"empty", []string{}, true},
		{"punctuation and duplicates", []string{"Authorization: Bearer test-token", "Cookie: a=1; b=two", `X-Fields: a,b:c; "quoted"`, "X-Fields: other"}, true},
		{"missing colon", []string{"secret-token"}, false},
		{"invalid name", []string{"Bad Name: secret-token"}, false},
		{"empty value", []string{"Authorization: \t "}, false},
		{"CRLF", []string{"Authorization: secret-token\r\nX-Injected: true"}, false},
		{"control value", []string{"Authorization: secret-token\x00"}, false},
		{"DEL value", []string{"Authorization: secret-token\x7f"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition := validExecutionDefinitionForTest()
			definition.ConfigSections[0].Params = []ParamDefinition{{Key: "request-fields", Type: ParamTypeStringArray, Format: ParamFormatHTTPHeaders, Default: tc.values}}
			err := ValidateExecutionDefinition(definition)
			assertHeaderValidation(t, err, tc.valid)
			definition.ConfigSections[0].Params[0].Default = []string{}
			normalized, err := NormalizeAndValidateConfig(map[string]any{"bruteforce": map[string]any{"enabled": true, "request-fields": tc.values}}, definition)
			assertHeaderValidation(t, err, tc.valid)
			if tc.valid && !reflect.DeepEqual(normalized["bruteforce"].(map[string]any)["request-fields"], tc.values) {
				t.Fatalf("header bytes/order changed: %+v", normalized)
			}
		})
	}
}

func TestHTTPHeaderDefaultsAreEmptyAndDuplicateErrorsDoNotExposeValues(t *testing.T) {
	definition := validExecutionDefinitionForTest()
	definition.ConfigSections[0].Params = []ParamDefinition{{Key: "headers", Type: ParamTypeStringArray, Format: ParamFormatHTTPHeaders, Default: []string{}}}
	config, err := NormalizeAndValidateConfig(map[string]any{}, definition)
	if err != nil {
		t.Fatal(err)
	}
	if got := config["bruteforce"].(map[string]any)["headers"]; !reflect.DeepEqual(got, []string{}) {
		t.Fatalf("omitted headers default = %#v", got)
	}
	maximum := 3
	definition.ConfigSections[0].Params[0].MaxItems = &maximum
	definition.ConfigSections[0].Params[0].Default = []string{"Authorization: secret-token", "Authorization: secret-token"}
	assertHeaderValidation(t, ValidateExecutionDefinition(definition), false)
	definition.ConfigSections[0].Params[0].Default = []string{}
	_, err = NormalizeAndValidateConfig(map[string]any{"bruteforce": map[string]any{"headers": []string{"Authorization: secret-token", "Authorization: secret-token"}}}, definition)
	assertHeaderValidation(t, err, false)
}

func assertHeaderValidation(t *testing.T, err error, valid bool) {
	t.Helper()
	if valid && err != nil {
		t.Fatalf("valid headers rejected: %v", err)
	}
	if !valid && err == nil {
		t.Fatal("invalid headers accepted")
	}
	if err != nil && strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error exposed a header value: %v", err)
	}
}
