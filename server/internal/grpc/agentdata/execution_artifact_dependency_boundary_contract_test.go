package agentdata

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestExecutionArtifactInputBoundaryDoesNotEvaluatePredecessorState(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	agentDataDir := filepath.Dir(thisFile)
	testCases := []struct {
		name      string
		path      string
		forbidden []string
	}{
		{
			name: "producer",
			path: filepath.Join(agentDataDir, "..", "..", "modules", "scan", "application", "execution_input_producers.go"),
			forbidden: []string{
				"portScanSucceeded",
				"PriorWorkflowStagesSatisfied",
				"PriorEngineTasksSucceeded",
			},
		},
		{
			name: "artifact resolver and task reader port",
			path: filepath.Join(agentDataDir, "execution_artifact_resolver.go"),
			forbidden: []string{
				"portScanEngineID",
				"portScanSucceeded",
				"PriorWorkflowStagesSatisfied",
				"PriorEngineTasksSucceeded",
				"requirePriorStagesReady",
			},
		},
		{
			name: "frozen blacklist snapshot source",
			path: filepath.Join(agentDataDir, "execution_input_blacklist_snapshot.go"),
			forbidden: []string{
				"ReadEffectivePatternsForScan",
				"modules/blacklist/application",
				"blacklistapp.",
			},
		},
		{
			name: "scan task repository",
			path: filepath.Join(agentDataDir, "..", "..", "modules", "scan", "repository", "scan_task.go"),
			forbidden: []string{
				"PriorWorkflowStagesSatisfied",
				"PriorEngineTasksSucceeded",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			content, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatalf("read %s: %v", test.path, err)
			}
			source := string(content)
			for _, forbidden := range test.forbidden {
				if strings.Contains(source, forbidden) {
					t.Fatalf("%s must not evaluate predecessor state through %q", test.name, forbidden)
				}
			}
		})
	}
}

func TestHostPortsProducerAcceptsOnlyTargetAndEvidenceSources(t *testing.T) {
	producerPath := executionArtifactBoundarySourcePath(t, "..", "..", "modules", "scan", "application", "execution_input_producers.go")
	file := parseExecutionArtifactBoundarySource(t, producerPath)

	var parameters []string
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "NewHostPortsProducer" {
			continue
		}
		for _, field := range function.Type.Params.List {
			for _, name := range field.Names {
				parameters = append(parameters, name.Name)
			}
		}
		break
	}
	if want := []string{"ctx", "scanID", "cursor", "blacklist"}; !reflect.DeepEqual(parameters, want) {
		t.Fatalf("NewHostPortsProducer parameters = %v, want %v; only the private frozen-blacklist filter may join finalized evidence", parameters, want)
	}
}

func TestExecutionArtifactResolverDependencyPortIsPlanLeaseAndURLSeedInputIntegrityOnly(t *testing.T) {
	resolverPath := executionArtifactBoundarySourcePath(t, "execution_artifact_resolver.go")
	file := parseExecutionArtifactBoundarySource(t, resolverPath)
	snapshotPath := executionArtifactBoundarySourcePath(t, "execution_input_blacklist_snapshot.go")
	snapshotFile := parseExecutionArtifactBoundarySource(t, snapshotPath)

	if got, want := namedInterfaceMethodNames(t, file, "ExecutionArtifactTaskReader"), []string{"GetSavedExecutionPlanLease"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExecutionArtifactTaskReader methods = %v, want %v; Workflow owns predecessor gating", got, want)
	}
	if got, want := namedStructFieldNames(t, file, "ServerExecutionArtifactResolverDependencies"), []string{"Tasks", "Sessions", "DNSNames", "HostPorts", "WebsiteURLs", "EndpointURLs", "InventoryDNSNames", "InventoryHostPorts", "InventoryWebsiteURLs", "InventoryEndpointURLs", "BlacklistSnapshots", "Wordlists", "ProviderConfig", "FingerprintArtifacts", "NucleiTemplates"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ServerExecutionArtifactResolverDependencies fields = %v, want %v", got, want)
	}
	if got, want := namedInterfaceMethodNames(t, snapshotFile, "ExecutionInputBlacklistSnapshotSource"), []string{"ResolveExecutionInputBlacklistFilter"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExecutionInputBlacklistSnapshotSource methods = %v, want %v; current Policy must not be an execution-input fallback", got, want)
	}
}

func executionArtifactBoundarySourcePath(t *testing.T, elements ...string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Join(append([]string{filepath.Dir(thisFile)}, elements...)...)
}

func parseExecutionArtifactBoundarySource(t *testing.T, path string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return file
}

func namedInterfaceMethodNames(t *testing.T, file *ast.File, typeName string) []string {
	t.Helper()
	typeExpression := namedTypeExpression(t, file, typeName)
	interfaceType, ok := typeExpression.(*ast.InterfaceType)
	if !ok {
		t.Fatalf("%s is %T, want interface", typeName, typeExpression)
	}
	var names []string
	for _, method := range interfaceType.Methods.List {
		if len(method.Names) != 1 {
			t.Fatalf("%s has an embedded or unnamed method", typeName)
		}
		if _, ok := method.Type.(*ast.FuncType); !ok {
			t.Fatalf("%s.%s is not a method", typeName, method.Names[0].Name)
		}
		names = append(names, method.Names[0].Name)
	}
	return names
}

func namedStructFieldNames(t *testing.T, file *ast.File, typeName string) []string {
	t.Helper()
	typeExpression := namedTypeExpression(t, file, typeName)
	structType, ok := typeExpression.(*ast.StructType)
	if !ok {
		t.Fatalf("%s is %T, want struct", typeName, typeExpression)
	}
	var names []string
	for _, field := range structType.Fields.List {
		if len(field.Names) != 1 {
			t.Fatalf("%s has an embedded or unnamed field", typeName)
		}
		names = append(names, field.Names[0].Name)
	}
	return names
}

func namedTypeExpression(t *testing.T, file *ast.File, typeName string) ast.Expr {
	t.Helper()
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if ok && typeSpec.Name.Name == typeName {
				return typeSpec.Type
			}
		}
	}
	t.Fatalf("type %s not found", typeName)
	return nil
}
