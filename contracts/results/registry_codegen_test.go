package results

import "testing"

func TestCodegenDescriptorsExposeCanonicalOrderAndFieldMetadata(t *testing.T) {
	descriptors, err := CodegenDescriptors()
	if err != nil {
		t.Fatalf("CodegenDescriptors() error = %v", err)
	}
	if len(descriptors) != 8 {
		t.Fatalf("CodegenDescriptors() returned %d descriptors, want 8", len(descriptors))
	}
	wantTypes := []string{
		ResultKindAssetSubdomain,
		ResultKindAssetEndpoint,
		ResultKindAssetHostPort,
		ResultKindAssetWebsite,
		ResultKindAssetWebsiteTechnology,
		ResultKindAssetScreenshot,
		ResultKindAssetDirectory,
		ResultKindAssetVulnerability,
	}
	for index, want := range wantTypes {
		if descriptors[index].ResultType != want {
			t.Fatalf("descriptor[%d].ResultType = %q, want %q", index, descriptors[index].ResultType, want)
		}
		if !descriptors[index].ClosedObject {
			t.Fatalf("descriptor[%d] must be a closed object", index)
		}
		if len(descriptors[index].Fields) == 0 {
			t.Fatalf("descriptor[%d] has no fields", index)
		}
	}

	website := descriptors[3]
	if website.Fields[0].GoName != "URL" || website.Fields[0].JSONName != "url" || website.Fields[0].GoType != "string" || website.Fields[0].Presence != FieldPresenceRequired {
		t.Fatalf("website URL metadata = %#v", website.Fields[0])
	}
	if website.Fields[3].GoName != "StatusCode" || website.Fields[3].GoType != "*int" || website.Fields[3].Presence != FieldPresenceOptional || website.Fields[3].SchemaBasics.MinInt == nil || *website.Fields[3].SchemaBasics.MinInt != 100 {
		t.Fatalf("website statusCode metadata = %#v", website.Fields[3])
	}
	vulnerability := descriptors[7]
	if vulnerability.Fields[len(vulnerability.Fields)-1].GoType != "map[string]any" || !vulnerability.Fields[len(vulnerability.Fields)-1].SchemaBasics.ExplicitObject {
		t.Fatalf("vulnerability rawOutput metadata = %#v", vulnerability.Fields[len(vulnerability.Fields)-1])
	}
}

func TestCodegenDescriptorsAreDeeplyDetached(t *testing.T) {
	descriptors, err := CodegenDescriptors()
	if err != nil {
		t.Fatalf("CodegenDescriptors() error = %v", err)
	}
	descriptors[0].Fields[0].JSONName = "changed"
	descriptors[0].Fields[0].SchemaBasics.EnumValues = append(descriptors[0].Fields[0].SchemaBasics.EnumValues, "changed")
	if fresh, err := CodegenDescriptors(); err != nil {
		t.Fatalf("second CodegenDescriptors() error = %v", err)
	} else if fresh[0].Fields[0].JSONName != "dnsName" || len(fresh[0].Fields[0].SchemaBasics.EnumValues) != 0 {
		t.Fatalf("descriptor registry was mutated through returned metadata: %#v", fresh[0].Fields[0])
	}
}

func TestCodegenDescriptorMappingsAreUniqueAndSupported(t *testing.T) {
	descriptors, err := CodegenDescriptors()
	if err != nil {
		t.Fatalf("CodegenDescriptors() error = %v", err)
	}
	seen := map[string]struct{}{}
	for _, descriptor := range descriptors {
		for _, field := range descriptor.Fields {
			if _, exists := seen[descriptor.ResultType+":"+field.JSONName]; exists {
				t.Fatalf("duplicate field mapping for %s.%s", descriptor.ResultType, field.JSONName)
			}
			seen[descriptor.ResultType+":"+field.JSONName] = struct{}{}
			if field.Presence != FieldPresenceRequired && field.Presence != FieldPresenceOptional {
				t.Fatalf("unsupported presence %q for %s.%s", field.Presence, descriptor.ResultType, field.JSONName)
			}
		}
	}
}
