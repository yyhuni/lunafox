package packagemanifest

import (
	"reflect"
	"strings"
	"testing"
)

const packageManifestTestDigestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const packageManifestTestDigestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func TestDecodeAndValidatePackageManifestPreservesOrderedRuntimeImageRefs(t *testing.T) {
	payload := []byte(`{
  "packageFormatVersion": "lunafox.engine-package.v2",
  "engineId": "engine.lunafox.subdomain_discovery",
  "engineVersion": "1.2.3",
  "runtimeImage": {
    "refs": [
      "docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "ghcr.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    ]
  }
}`)

	manifest, err := DecodePackageManifest(payload, "package.json")
	if err != nil {
		t.Fatalf("DecodePackageManifest() error = %v", err)
	}
	if err := ValidatePackageManifest(manifest); err != nil {
		t.Fatalf("ValidatePackageManifest() error = %v", err)
	}
	wantRefs := []string{
		"docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + packageManifestTestDigestA,
		"ghcr.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + packageManifestTestDigestA,
	}
	if !reflect.DeepEqual(manifest.RuntimeImage.Refs, wantRefs) {
		t.Fatalf("runtimeImage.refs = %#v, want %#v", manifest.RuntimeImage.Refs, wantRefs)
	}
	if manifest.PackageFormatVersion != PackageFormatVersion {
		t.Fatalf("packageFormatVersion = %q, want %q", manifest.PackageFormatVersion, PackageFormatVersion)
	}
}

func TestValidatePackageManifestRejectsDuplicateRuntimeImageLocation(t *testing.T) {
	ref := "docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + packageManifestTestDigestA
	manifest := validPackageManifestForTest()
	manifest.RuntimeImage.Refs = []string{ref, ref}

	requirePackageManifestValidationError(t, manifest, "duplicate runtimeImage candidate location")
}

func TestValidatePackageManifestRejectsRuntimeImageDigestDrift(t *testing.T) {
	manifest := validPackageManifestForTest()
	manifest.RuntimeImage.Refs = []string{
		"docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + packageManifestTestDigestA,
		"ghcr.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + packageManifestTestDigestB,
	}

	requirePackageManifestValidationError(t, manifest, "same OCI image digest")
}

func TestValidatePackageManifestRejectsNonCanonicalRuntimeImageRef(t *testing.T) {
	manifest := validPackageManifestForTest()
	manifest.RuntimeImage.Refs = []string{
		"docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery @" + packageManifestTestDigestA,
	}

	requirePackageManifestValidationError(t, manifest, "canonical OCI digest reference")
}

func TestValidatePackageManifestAcceptsSingleRuntimeImageRef(t *testing.T) {
	if err := ValidatePackageManifest(validPackageManifestForTest()); err != nil {
		t.Fatalf("ValidatePackageManifest() error = %v", err)
	}
}

func TestDecodeAndValidatePackageManifestRejectsMissingOrNullRuntimeImage(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "missing",
			payload: `{"packageFormatVersion":"lunafox.engine-package.v2","engineId":"engine.lunafox.subdomain_discovery","engineVersion":"1.2.3"}`,
		},
		{
			name:    "null",
			payload: `{"packageFormatVersion":"lunafox.engine-package.v2","engineId":"engine.lunafox.subdomain_discovery","engineVersion":"1.2.3","runtimeImage":null}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest, err := DecodePackageManifest([]byte(test.payload), "package.json")
			if err != nil {
				t.Fatalf("DecodePackageManifest() error = %v", err)
			}
			requirePackageManifestValidationError(t, manifest, "runtimeImage.refs is required")
		})
	}
}

func TestDecodePackageManifestRejectsLegacyAndDuplicateIdentityFields(t *testing.T) {
	tests := []struct {
		name  string
		field string
		value string
	}{
		{name: "v1 files", field: "files", value: `[{"path":"engine.json","sha256":"abc","sizeBytes":1}]`},
		{name: "runtime bundles", field: "runtimeBundles", value: `[]`},
		{name: "runtime images", field: "runtimeImages", value: `[]`},
		{name: "runtime ref", field: "runtimeRef", value: `"runtime.subdomain_discovery"`},
		{name: "runtime source", field: "runtimeSource", value: `"runtime.subdomain_discovery"`},
		{name: "host binary", field: "binary", value: `"bin/subdomain-discovery"`},
		{name: "host install", field: "install", value: `{"command":"bin/subdomain-discovery"}`},
		{name: "per-file sha256", field: "sha256", value: `"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`},
		{name: "per-file size", field: "sizeBytes", value: `1`},
		{name: "checksums projection", field: "checksums", value: `"checksums.txt"`},
		{name: "config schema ref", field: "configSchema", value: `{"ref":"schema.engine.subdomain_discovery","path":"artifacts/schemas/subdomain_discovery.schema.json"}`},
		{name: "config schema reference sibling", field: "configSchemaRef", value: `"schema.engine.subdomain_discovery"`},
		{name: "config schema path sibling", field: "configSchemaPath", value: `"artifacts/schemas/subdomain_discovery.schema.json"`},
		{name: "config schema id sibling", field: "configSchemaId", value: `"schema.engine.subdomain_discovery"`},
		{name: "json schema id sibling", field: "$id", value: `"schema.engine.subdomain_discovery"`},
		{name: "runtime image digest sibling", field: "runtimeImageDigest", value: `"` + packageManifestTestDigestA + `"`},
		{name: "package digest sibling", field: "packageDigest", value: `"` + packageManifestTestDigestA + `"`},
		{name: "artifact manifest digest sibling", field: "artifactManifestDigest", value: `"` + packageManifestTestDigestA + `"`},
		{name: "engine manifest version", field: "manifestVersion", value: `"engine.v5"`},
		{name: "engine API major", field: "engineApiMajor", value: `2`},
		{name: "top-level entrypoint", field: "entrypoint", value: `["/engine"]`},
		{name: "top-level command", field: "command", value: `["scan"]`},
		{name: "top-level cmd", field: "cmd", value: `["scan"]`},
		{name: "top-level args", field: "args", value: `["--target","example.com"]`},
		{name: "publisher", field: "publisher", value: `"lunafox"`},
		{name: "trust metadata", field: "trust", value: `{"class":"builtin"}`},
		{name: "supported target types", field: "supportedTargetTypes", value: `["domain"]`},
		{name: "execution inputs", field: "inputs", value: `["subdomains"]`},
		{name: "config sections", field: "configSections", value: `[]`},
		{name: "platform resources", field: "platformResources", value: `["subfinderProviderConfig"]`},
		{name: "result types", field: "resultTypes", value: `[]`},
		{name: "result schema reference", field: "resultSchemaRef", value: `"schema.result.demo.v1"`},
		{name: "result schema list", field: "resultSchemas", value: `[]`},
		{name: "result schema", field: "resultSchema", value: `{"ref":"schema.result.demo.v1"}`},
		{name: "allowed result types", field: "allowedResultTypes", value: `[]`},
		{name: "output authorization", field: "outputAuthorization", value: `{}`},
		{name: "output allowlist", field: "outputAllowlist", value: `[]`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := validPackageManifestJSONWithExtraField(test.field, test.value)
			_, err := DecodePackageManifest(payload, "package.json")
			if err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("DecodePackageManifest() error = %v, want unknown field rejection", err)
			}
		})
	}
}

func TestDecodePackageManifestRejectsRuntimeImageSiblingFields(t *testing.T) {
	tests := []struct {
		name  string
		field string
		value string
	}{
		{name: "digest", field: "digest", value: `"` + packageManifestTestDigestA + `"`},
		{name: "repository", field: "repository", value: `"lunafox-engine-runtime-subdomain-discovery"`},
		{name: "tag", field: "tag", value: `"latest"`},
		{name: "entrypoint", field: "entrypoint", value: `["/engine"]`},
		{name: "command", field: "command", value: `["scan"]`},
		{name: "cmd", field: "cmd", value: `["scan"]`},
		{name: "args", field: "args", value: `["--target","example.com"]`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := []byte(`{
  "packageFormatVersion":"lunafox.engine-package.v2",
  "engineId":"engine.lunafox.subdomain_discovery",
  "engineVersion":"1.2.3",
  "runtimeImage":{
    "refs":["docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"],
    "` + test.field + `":` + test.value + `
  }
}`)
			_, err := DecodePackageManifest(payload, "package.json")
			if err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("DecodePackageManifest() error = %v, want unknown field rejection", err)
			}
		})
	}
}

func TestDecodePackageManifestRejectsTrailingJSON(t *testing.T) {
	payload := append(validPackageManifestJSON(), []byte(`{"extra":true}`)...)

	_, err := DecodePackageManifest(payload, "package.json")
	if err == nil || !strings.Contains(err.Error(), "unexpected trailing JSON content") {
		t.Fatalf("DecodePackageManifest() error = %v, want trailing JSON rejection", err)
	}
}

func TestDecodePackageManifestRejectsDuplicateRuntimeImageField(t *testing.T) {
	payload := []byte(`{
  "packageFormatVersion":"lunafox.engine-package.v2",
  "engineId":"engine.lunafox.subdomain_discovery",
  "engineVersion":"1.2.3",
  "runtimeImage":{"refs":["docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"]},
  "runtimeImage":{"refs":["ghcr.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"]}
}`)

	_, err := DecodePackageManifest(payload, "package.json")
	if err == nil || !strings.Contains(err.Error(), `duplicate JSON field "runtimeImage"`) {
		t.Fatalf("DecodePackageManifest() error = %v, want duplicate runtimeImage rejection", err)
	}
}

func TestDecodePackageManifestRejectsDuplicateRefsField(t *testing.T) {
	payload := []byte(`{
  "packageFormatVersion":"lunafox.engine-package.v2",
  "engineId":"engine.lunafox.subdomain_discovery",
  "engineVersion":"1.2.3",
  "runtimeImage":{
    "refs":["docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"],
    "refs":["ghcr.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"]
  }
}`)

	_, err := DecodePackageManifest(payload, "package.json")
	if err == nil || !strings.Contains(err.Error(), `duplicate JSON field "refs"`) {
		t.Fatalf("DecodePackageManifest() error = %v, want duplicate refs rejection", err)
	}
}

func TestValidatePackageManifestRejectsUnsupportedOrMalformedIdentity(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*PackageManifest)
		want   string
	}{
		{name: "v1 format", mutate: func(manifest *PackageManifest) {
			manifest.PackageFormatVersion = "lunafox.engine-package.v1"
		}, want: "unsupported packageFormatVersion"},
		{name: "missing format", mutate: func(manifest *PackageManifest) {
			manifest.PackageFormatVersion = ""
		}, want: "packageFormatVersion is required"},
		{name: "missing engine id", mutate: func(manifest *PackageManifest) {
			manifest.EngineID = ""
		}, want: "engineId is required"},
		{name: "non canonical engine id is package-local", mutate: func(manifest *PackageManifest) {
			manifest.EngineID = "engine.lunafox.subdomain-discovery"
		}, want: ""},
		{name: "missing engine version", mutate: func(manifest *PackageManifest) {
			manifest.EngineVersion = ""
		}, want: "engineVersion is required"},
		{name: "blank engine version", mutate: func(manifest *PackageManifest) {
			manifest.EngineVersion = "   "
		}, want: "engineVersion is required"},
		{name: "non canonical engine version", mutate: func(manifest *PackageManifest) {
			manifest.EngineVersion = " 1.2.3 "
		}, want: "engineVersion must be canonical"},
		{name: "missing refs", mutate: func(manifest *PackageManifest) {
			manifest.RuntimeImage.Refs = nil
		}, want: "runtimeImage.refs is required"},
		{name: "empty refs", mutate: func(manifest *PackageManifest) {
			manifest.RuntimeImage.Refs = []string{}
		}, want: "runtimeImage.refs is required"},
		{name: "empty ref", mutate: func(manifest *PackageManifest) {
			manifest.RuntimeImage.Refs = []string{""}
		}, want: "non-empty canonical OCI digest reference"},
		{name: "whitespace ref", mutate: func(manifest *PackageManifest) {
			manifest.RuntimeImage.Refs = []string{" docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + packageManifestTestDigestA}
		}, want: "non-empty canonical OCI digest reference"},
		{name: "tag only", mutate: func(manifest *PackageManifest) {
			manifest.RuntimeImage.Refs = []string{"docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery:latest"}
		}, want: "digest separator"},
		{name: "short digest", mutate: func(manifest *PackageManifest) {
			manifest.RuntimeImage.Refs = []string{"docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@sha256:abc"}
		}, want: "lowercase sha256 digest"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := validPackageManifestForTest()
			test.mutate(&manifest)
			if test.want == "" {
				if err := ValidatePackageManifest(manifest); err != nil {
					t.Fatalf("ValidatePackageManifest() error = %v, want acceptance", err)
				}
				return
			}
			requirePackageManifestValidationError(t, manifest, test.want)
		})
	}
}

func validPackageManifestJSON() []byte {
	return []byte(`{
  "packageFormatVersion":"lunafox.engine-package.v2",
  "engineId":"engine.lunafox.subdomain_discovery",
  "engineVersion":"1.2.3",
  "runtimeImage":{"refs":["docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"]}
}`)
}

func validPackageManifestJSONWithExtraField(field, value string) []byte {
	return []byte(`{
  "packageFormatVersion":"lunafox.engine-package.v2",
  "engineId":"engine.lunafox.subdomain_discovery",
  "engineVersion":"1.2.3",
  "runtimeImage":{"refs":["docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"]},
  "` + field + `":` + value + `
}`)
}

func validPackageManifestForTest() PackageManifest {
	return PackageManifest{
		PackageFormatVersion: PackageFormatVersion,
		EngineID:             "engine.lunafox.subdomain_discovery",
		EngineVersion:        "1.2.3",
		RuntimeImage: RuntimeImage{Refs: []string{
			"docker.io/yyhuni/lunafox-engine-runtime-subdomain-discovery@" + packageManifestTestDigestA,
		}},
	}
}

func requirePackageManifestValidationError(t *testing.T, manifest PackageManifest, want string) {
	t.Helper()
	err := ValidatePackageManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("ValidatePackageManifest() error = %v, want error containing %q", err, want)
	}
}
