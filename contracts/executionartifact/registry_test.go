package executionartifact

import "testing"

func TestRegistryOwnsExactPhaseOneDescriptors(t *testing.T) {
	want := []Descriptor{
		{
			Role:                RoleSubdomainsInput,
			FileKind:            "artifactFile",
			ContentType:         "application/vnd.lunafox.subdomains.v1",
			RecordCountPresence: RecordCountRequired,
			ContainerPath:       "/run/lunafox/inputs/subdomains.txt",
			RoleID:              "subdomains",
		},
		{
			Role:                RoleHostPortsInput,
			FileKind:            "artifactFile",
			ContentType:         "application/vnd.lunafox.host-ports.v1",
			RecordCountPresence: RecordCountRequired,
			ContainerPath:       "/run/lunafox/inputs/host-ports.jsonl",
			RoleID:              "hostPorts",
		},
		{Role: RoleWebsiteURLsInput, FileKind: FileKindArtifactFile, ContentType: ContentTypeWebsiteURLs, RecordCountPresence: RecordCountRequired, ContainerPath: "/run/lunafox/inputs/website-urls.txt", RoleID: "websiteURLs"},
		{Role: RoleEndpointURLsInput, FileKind: FileKindArtifactFile, ContentType: ContentTypeEndpointURLs, RecordCountPresence: RecordCountRequired, ContainerPath: "/run/lunafox/inputs/endpoint-urls.txt", RoleID: "endpointURLs"},
		{
			Role:                RoleWordlistConfigResource,
			FileKind:            "wordlist",
			ContentType:         "application/vnd.lunafox.wordlist.v1",
			RecordCountPresence: RecordCountRequired,
		},
		{
			Role:                  RoleSubfinderProviderConfigPlatformResource,
			FileKind:              "toolProviderConfig",
			ContentType:           "application/vnd.lunafox.subfinder-provider-config.v1+yaml",
			RecordCountPresence:   RecordCountAbsent,
			PlatformResourceID:    "subfinderProviderConfig",
			Filename:              "config.yaml",
			NativeHTTPContentType: "application/yaml",
			ContainerPath:         "/run/lunafox/resources/platform/subfinderProviderConfig/config.yaml",
		},
		{Role: RoleFingerprintLibraryFingerPrintHub, FileKind: FileKindFingerprintLibrary, ContentType: ContentTypeFingerprintLibraryFingerPrintHub, RecordCountPresence: RecordCountRequired, PlatformResourceID: "fingerprintLibraryFingerPrintHub", Filename: "fingerprinthub_web.json", Library: "fingerprinthub", NativeHTTPContentType: "application/json", ContainerPath: "/run/lunafox/resources/platform/fingerprintLibraryFingerPrintHub/fingerprinthub_web.json"},
		{Role: RoleRuntimeArtifact, FileKind: FileKindRuntimeArtifact, ContentType: ContentTypeRuntimeArtifact, RecordCountPresence: RecordCountAbsent, Filename: "templates", ContainerPath: "/run/lunafox/resources/runtime/nuclei/templates", ArtifactID: "nucleiTemplates"},
	}

	got := Descriptors()
	if len(got) != len(want) {
		t.Fatalf("Descriptors() returned %d entries, want exactly %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("Descriptors()[%d] = %#v, want %#v", index, got[index], want[index])
		}
		descriptor, ok := Lookup(want[index].Role)
		if !ok || descriptor != want[index] {
			t.Fatalf("Lookup(%d) = %#v, %v; want %#v, true", want[index].Role, descriptor, ok, want[index])
		}
		if err := ValidateBinding(want[index].Role, want[index].FileKind, want[index].ContentType); err != nil {
			t.Fatalf("ValidateBinding(%d) error = %v", want[index].Role, err)
		}
	}
}

func TestDescriptorsReturnsDetachedSlice(t *testing.T) {
	descriptors := Descriptors()
	descriptors[0].ContentType = "mutated"

	descriptor, ok := Lookup(RoleSubdomainsInput)
	if !ok {
		t.Fatal("Lookup(RoleSubdomainsInput) did not find canonical descriptor")
	}
	if descriptor.ContentType != ContentTypeSubdomains {
		t.Fatalf("registry mutated through Descriptors result: %q", descriptor.ContentType)
	}
}

func TestRuntimeArtifactRegistryRejectsRetiredTemplateBundleID(t *testing.T) {
	if _, ok := LookupRuntimeArtifactID("runtimeTemplateBundle"); ok {
		t.Fatal("retired runtimeTemplateBundle ID remains registered")
	}
	for _, retired := range []string{"nucleiTemplateBundle", "runtimeTemplateBundle", "nuclei-template-bundle"} {
		if descriptor, ok := LookupRuntimeArtifactID(retired); ok {
			t.Fatalf("retired runtime artifact %q remains registered: %#v", retired, descriptor)
		}
	}
	descriptor, ok := LookupRuntimeArtifactID("nucleiTemplates")
	if !ok || descriptor.ArtifactID != "nucleiTemplates" {
		t.Fatalf("nucleiTemplates lookup = %#v, %v", descriptor, ok)
	}
}

func TestRegistryRolesAndContentTypesAreUnique(t *testing.T) {
	roles := make(map[Role]struct{})
	contentTypes := make(map[string]struct{})
	for _, descriptor := range Descriptors() {
		if _, duplicate := roles[descriptor.Role]; duplicate {
			t.Fatalf("duplicate role in registry: %d", descriptor.Role)
		}
		roles[descriptor.Role] = struct{}{}
		if _, duplicate := contentTypes[descriptor.ContentType]; duplicate {
			t.Fatalf("duplicate content type in registry: %q", descriptor.ContentType)
		}
		contentTypes[descriptor.ContentType] = struct{}{}
	}
}

func TestExecutionInputRegistryIsClosedAndExcludesEagerResources(t *testing.T) {
	descriptors := ExecutionInputRegistry()
	wantRoles := []string{RoleSubdomains, RoleHostPorts, RoleWebsiteURLs, RoleEndpointURLs}
	if len(descriptors) != len(wantRoles) {
		t.Fatalf("ExecutionInputRegistry() returned %d entries, want %d", len(descriptors), len(wantRoles))
	}
	for index, wantRole := range wantRoles {
		if descriptors[index].RoleID != wantRole || descriptors[index].PlatformResourceID != "" || descriptors[index].FileKind != FileKindArtifactFile {
			t.Fatalf("input registry entry %d = %#v, contains non-input metadata", index, descriptors[index])
		}
		if got, ok := LookupRoleID(wantRole); !ok || got.Role != descriptors[index].Role || got.ContentType != descriptors[index].ContentType || got.ContainerPath != descriptors[index].ContainerPath {
			t.Fatalf("LookupRoleID(%q) = %#v/%v, want canonical input descriptor", wantRole, got, ok)
		}
	}
	for _, role := range []string{"wordlist", "subfinderProviderConfig", "fingerprintLibraryFingerPrintHub", "", " Subdomains"} {
		if _, ok := LookupRoleID(role); ok {
			t.Fatalf("LookupRoleID(%q) authorized a non-input or unknown role", role)
		}
	}
}

func TestRegistryRejectsUnknownLegacyAndWrongRoleBindings(t *testing.T) {
	descriptors := Descriptors()
	tests := []struct {
		name        string
		role        Role
		fileKind    string
		contentType string
	}{
		{name: "zero role", role: 0, fileKind: FileKindArtifactFile, contentType: ContentTypeSubdomains},
		{name: "unknown role", role: Role(255), fileKind: FileKindArtifactFile, contentType: ContentTypeSubdomains},
		{name: "missing file kind", role: RoleSubdomainsInput, contentType: ContentTypeSubdomains},
		{name: "unknown file kind", role: RoleSubdomainsInput, fileKind: "file", contentType: ContentTypeSubdomains},
		{name: "missing content type", role: RoleSubdomainsInput, fileKind: FileKindArtifactFile},
		{name: "legacy host content type", role: RoleSubdomainsInput, fileKind: FileKindArtifactFile, contentType: "text/x-lunafox-subdomains"},
		{name: "legacy URL content type", role: RoleHostPortsInput, fileKind: FileKindArtifactFile, contentType: "text/x-lunafox-url-list"},
		{name: "unknown version", role: RoleSubdomainsInput, fileKind: FileKindArtifactFile, contentType: "application/vnd.lunafox.subdomains.v2"},
		{name: "case folded content type", role: RoleSubdomainsInput, fileKind: FileKindArtifactFile, contentType: "application/vnd.lunafox.subdomains."},
		{name: "content type with whitespace", role: RoleSubdomainsInput, fileKind: FileKindArtifactFile, contentType: " application/vnd.lunafox.subdomains.v1"},
	}

	for _, descriptor := range descriptors {
		for _, other := range descriptors {
			if descriptor.Role == other.Role {
				continue
			}
			tests = append(tests, struct {
				name        string
				role        Role
				fileKind    string
				contentType string
			}{
				name:        "content type copied from another role",
				role:        descriptor.Role,
				fileKind:    descriptor.FileKind,
				contentType: other.ContentType,
			})
		}
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateBinding(test.role, test.fileKind, test.contentType); err == nil {
				t.Fatal("ValidateBinding() accepted contract drift")
			}
		})
	}
}

func TestRegistryRejectsFileKindCopiedFromAnotherRole(t *testing.T) {
	for _, test := range []struct {
		name     string
		role     Role
		fileKind string
	}{
		{name: "subdomains as wordlist", role: RoleSubdomainsInput, fileKind: FileKindWordlist},
		{name: "hostPorts as provider config", role: RoleHostPortsInput, fileKind: FileKindToolProviderConfig},
		{name: "wordlist as artifact file", role: RoleWordlistConfigResource, fileKind: FileKindArtifactFile},
		{name: "provider config as wordlist", role: RoleSubfinderProviderConfigPlatformResource, fileKind: FileKindWordlist},
	} {
		t.Run(test.name, func(t *testing.T) {
			descriptor, ok := Lookup(test.role)
			if !ok {
				t.Fatalf("Lookup(%d) did not find canonical descriptor", test.role)
			}
			if err := ValidateBinding(test.role, test.fileKind, descriptor.ContentType); err == nil {
				t.Fatal("ValidateBinding() accepted file kind from another role")
			}
		})
	}
}

func TestLookupRejectsRolesOutsideClosedRegistry(t *testing.T) {
	for _, role := range []Role{0, 12, 255} {
		if descriptor, ok := Lookup(role); ok || descriptor != (Descriptor{}) {
			t.Fatalf("Lookup(%d) = %#v, %v; want zero descriptor, false", role, descriptor, ok)
		}
	}
}
