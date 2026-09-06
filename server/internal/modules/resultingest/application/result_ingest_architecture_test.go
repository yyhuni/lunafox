package application

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"reflect"
	"strings"
	"testing"
)

func TestResultIngestFacadeIsTheOnlyExportedPersistenceInterface(t *testing.T) {
	typeOfFacade := reflect.TypeOf((*ResultIngestFacade)(nil))
	if typeOfFacade.NumMethod() != 1 || typeOfFacade.Method(0).Name != "Ingest" {
		t.Fatalf("ResultIngestFacade exported methods = %v; want only Ingest", exportedMethodNames(typeOfFacade))
	}

	packages, err := parser.ParseDir(token.NewFileSet(), ".", func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse resultingest package: %v", err)
	}
	pkg := packages["application"]
	if pkg == nil {
		t.Fatal("resultingest application package is missing")
	}
	for _, file := range pkg.Files {
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, spec := range general.Specs {
				typeSpec := spec.(*ast.TypeSpec)
				if ast.IsExported(typeSpec.Name.Name) && strings.HasSuffix(typeSpec.Name.Name, "ResultMaterializer") {
					t.Fatalf("exported per-result materializer port %s is forbidden", typeSpec.Name.Name)
				}
			}
		}
	}
}

func TestResultIngestOutcomeHasExactlyServerLocalDiagnosticFields(t *testing.T) {
	typeOfOutcome := reflect.TypeOf(ResultIngestOutcome{})
	want := []string{"ReceivedItems", "RejectedItems", "DuplicateItems", "ScopeFilteredItems", "UnsupportedItems", "SnapshotCount", "AssetCount"}
	if typeOfOutcome.NumField() != len(want) {
		t.Fatalf("ResultIngestOutcome field count = %d, want %d", typeOfOutcome.NumField(), len(want))
	}
	for index, name := range want {
		if typeOfOutcome.Field(index).Name != name {
			t.Fatalf("ResultIngestOutcome field %d = %s, want %s", index, typeOfOutcome.Field(index).Name, name)
		}
	}
}

func exportedMethodNames(value reflect.Type) []string {
	names := make([]string, value.NumMethod())
	for index := 0; index < value.NumMethod(); index++ {
		names[index] = value.Method(index).Name
	}
	return names
}
