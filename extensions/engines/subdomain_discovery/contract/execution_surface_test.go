package contract

import (
	"context"
	"reflect"
	"testing"
)

func TestGeneratedExecutionKeepsSubdomainNamespacesTypedAndSeparate(t *testing.T) {
	executionType := reflect.TypeOf(Execution{})
	wantExecutionFields := []struct {
		name   string
		typeOf reflect.Type
	}{
		{name: "Target", typeOf: reflect.TypeOf(Target{})},
		{name: "Input", typeOf: reflect.TypeOf(Input{})},
		{name: "Config", typeOf: reflect.TypeOf(Config{})},
		{name: "PlatformResources", typeOf: reflect.TypeOf(PlatformResources{})},
		{name: "Workspace", typeOf: reflect.TypeOf(DirectoryPath(""))},
		{name: "Progress", typeOf: reflect.TypeOf((*Progress)(nil)).Elem()},
		{name: "Results", typeOf: reflect.TypeOf(Results{})},
	}
	if executionType.NumField() != len(wantExecutionFields) {
		t.Fatalf("Execution field count = %d, want %d", executionType.NumField(), len(wantExecutionFields))
	}
	for index, expected := range wantExecutionFields {
		field := executionType.Field(index)
		if field.Name != expected.name || field.Type != expected.typeOf {
			t.Fatalf("Execution field %d = %s %v, want %s %v", index, field.Name, field.Type, expected.name, expected.typeOf)
		}
	}
	assertTargetShape(t)

	assertTypedInputHandles(t, reflect.TypeOf(Input{}))
	platformType := reflect.TypeOf(PlatformResources{})
	if platformType.NumField() != 1 || platformType.Field(0).Name != "SubfinderProviderConfig" || platformType.Field(0).Type.Kind() != reflect.String {
		t.Fatalf("PlatformResources = %v, want direct provider path only", platformType)
	}
	configType := reflect.TypeOf(Config{})
	wantConfigSections := []struct {
		name   string
		typeOf reflect.Type
	}{
		{name: "Recon", typeOf: reflect.TypeOf(ReconConfig{})},
		{name: "Bruteforce", typeOf: reflect.TypeOf(BruteforceConfig{})},
		{name: "Resolve", typeOf: reflect.TypeOf(ResolveConfig{})},
	}
	if configType.NumField() != len(wantConfigSections) {
		t.Fatalf("Config field count = %d, want %d", configType.NumField(), len(wantConfigSections))
	}
	for index, expected := range wantConfigSections {
		field := configType.Field(index)
		if field.Name != expected.name || field.Type != expected.typeOf {
			t.Fatalf("Config field %d = %s %v, want %s %v", index, field.Name, field.Type, expected.name, expected.typeOf)
		}
	}
	if field, ok := reflect.TypeOf(BruteforceConfig{}).FieldByName("Wordlist"); !ok || field.Type.Kind() != reflect.String {
		t.Fatalf("Config.Bruteforce.Wordlist must remain a direct config-resource path")
	}
	if field, ok := reflect.TypeOf(BruteforceConfig{}).FieldByName("Resolvers"); !ok || field.Type.Kind() != reflect.String {
		t.Fatalf("Config.Bruteforce.Resolvers must remain a direct config-resource path")
	}
	if field, ok := reflect.TypeOf(ResolveConfig{}).FieldByName("Resolvers"); !ok || field.Type.Kind() != reflect.String {
		t.Fatalf("Config.Resolve.Resolvers must remain a direct config-resource path")
	}
	if field, ok := reflect.TypeOf(BruteforceConfig{}).FieldByName("WildcardFilter"); !ok || field.Type.Kind() != reflect.Bool {
		t.Fatalf("Config.Bruteforce.WildcardFilter must be a boolean")
	}
	if field, ok := reflect.TypeOf(ResolveConfig{}).FieldByName("WildcardFilter"); !ok || field.Type.Kind() != reflect.Bool {
		t.Fatalf("Config.Resolve.WildcardFilter must be a boolean")
	}
	assertTypedAuthorPorts(t)
}

func assertTypedInputHandles(t *testing.T, inputType reflect.Type) {
	t.Helper()
	want := []string{"Subdomains", "HostPorts", "WebsiteURLs", "EndpointURLs"}
	if inputType.NumField() != len(want) {
		t.Fatalf("Input field count = %d, want %d", inputType.NumField(), len(want))
	}
	for index, name := range want {
		field := inputType.Field(index)
		_, ok := field.Type.MethodByName("Path")
		if field.Name != name || field.Type.Kind() != reflect.Interface || field.Type.NumMethod() != 1 || !ok {
			t.Fatalf("Input field %d = %s %v, want typed Path handle %s", index, field.Name, field.Type, name)
		}
	}
}

func assertTargetShape(t *testing.T) {
	t.Helper()
	targetType := reflect.TypeOf(Target{})
	want := []struct {
		name   string
		typeOf reflect.Type
	}{
		{name: "Type", typeOf: reflect.TypeOf(TargetType(""))},
		{name: "Value", typeOf: reflect.TypeOf("")},
	}
	if targetType.NumField() != len(want) {
		t.Fatalf("Target field count = %d, want %d", targetType.NumField(), len(want))
	}
	for index, expected := range want {
		field := targetType.Field(index)
		if field.Name != expected.name || field.Type != expected.typeOf {
			t.Fatalf("Target field %d = %s %v, want %s %v", index, field.Name, field.Type, expected.name, expected.typeOf)
		}
	}
}

func assertTypedAuthorPorts(t *testing.T) {
	t.Helper()
	progressType := reflect.TypeOf((*Progress)(nil)).Elem()
	if progressType.NumMethod() != 1 {
		t.Fatalf("Progress method count = %d, want message-only Report", progressType.NumMethod())
	}
	report, ok := progressType.MethodByName("Report")
	if !ok || report.Type.NumIn() != 2 || report.Type.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() || report.Type.In(1).Kind() != reflect.String || report.Type.NumOut() != 1 || report.Type.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
		t.Fatalf("Progress.Report signature = %v", report.Type)
	}

	resultsType := reflect.TypeOf(Results{})
	wantResults := []struct {
		field string
		item  reflect.Type
	}{
		{field: "Subdomains", item: reflect.TypeOf(Subdomain{})},
		{field: "Endpoints", item: reflect.TypeOf(Endpoint{})},
		{field: "HostPorts", item: reflect.TypeOf(HostPort{})},
		{field: "Websites", item: reflect.TypeOf(Website{})},
		{field: "WebsiteTechnologies", item: reflect.TypeOf(WebsiteTechnology{})},
		{field: "Screenshots", item: reflect.TypeOf(Screenshot{})},
		{field: "Directories", item: reflect.TypeOf(Directory{})},
		{field: "Vulnerabilities", item: reflect.TypeOf(Vulnerability{})},
	}
	if resultsType.NumField() != len(wantResults) || resultsType.NumMethod() != 0 {
		t.Fatalf("Results surface = %d fields/%d methods, want %d typed fields/no generic methods", resultsType.NumField(), resultsType.NumMethod(), len(wantResults))
	}
	for index, expected := range wantResults {
		field := resultsType.Field(index)
		expectedMethods := 1
		if field.Name != expected.field || field.Type.Kind() != reflect.Interface || field.Type.NumMethod() != expectedMethods {
			t.Fatalf("Results field %d = %s %v, want %d-method %s port", index, field.Name, field.Type, expectedMethods, expected.field)
		}
		submit, ok := field.Type.MethodByName("Submit")
		if !ok || submit.Type.NumIn() != 2 || submit.Type.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() || submit.Type.In(1).Kind() != reflect.Chan || submit.Type.In(1).ChanDir() != reflect.RecvDir || submit.Type.In(1).Elem() != expected.item || submit.Type.NumOut() != 1 || submit.Type.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			t.Fatalf("Results.%s.Submit signature = %v", expected.field, submit.Type)
		}
	}
}
