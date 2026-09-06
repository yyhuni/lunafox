package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	enginepackagebuild "github.com/yyhuni/lunafox/contracts/enginemanifest/enginepackagebuild"
	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"oras.land/oras-go/v2/registry/remote/auth"
)

func TestEngineIDFromArchive(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "engine.lunafox.website_discovery-1.2.3.lfengine.tar.gz")
	entries := validPackageEntries("engine.lunafox.website_discovery")
	writePackageArchive(t, archivePath, entries)

	engineID, err := engineIDFromArchive(archivePath)
	if err != nil {
		t.Fatalf("engineIDFromArchive() error = %v", err)
	}
	if engineID != "engine.lunafox.website_discovery" {
		t.Fatalf("engineIDFromArchive() = %q", engineID)
	}
}

func TestEngineIDFromArchiveRejectsNonCanonicalWhitespace(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "engine.lunafox.website_discovery-1.2.3.lfengine.tar.gz")
	entries := validPackageEntries("engine.lunafox.website_discovery")
	entries[1].Payload = []byte(strings.Replace(string(entries[1].Payload), `"engineId":"engine.lunafox.website_discovery"`, `"engineId":" engine.lunafox.website_discovery"`, 1))
	writeUncheckedArchive(t, archivePath, entries)

	if _, err := engineIDFromArchive(archivePath); err == nil {
		t.Fatal("expected non-canonical engineId whitespace to be rejected")
	}
}

func TestEmptyOCIConfigDescriptorUsesCanonicalOCIEmptyDescriptor(t *testing.T) {
	config := emptyOCIConfigDescriptor()
	want := ocispec.DescriptorEmptyJSON
	if config.MediaType != want.MediaType || config.Digest != want.Digest || config.Size != want.Size {
		t.Fatalf("empty config descriptor = %#v, want OCI empty descriptor identity %#v", config, want)
	}
	if len(config.Data) != 0 {
		t.Fatalf("empty config descriptor must not inline optional data: %#v", config)
	}
}

func TestNewOCIArtifactManifestIncludesCanonicalEmptyConfig(t *testing.T) {
	config := emptyOCIConfigDescriptor()
	layer := ocispec.Descriptor{
		MediaType: ociartifact.EnginePackageLayerMediaType,
		Digest:    digest.FromBytes([]byte("package")),
		Size:      int64(len("package")),
	}

	manifest := newOCIArtifactManifest(ociartifact.EnginePackageArtifactType, config, []ocispec.Descriptor{layer}, nil)
	if manifest.Config.MediaType != config.MediaType || manifest.Config.Digest != config.Digest || manifest.Config.Size != config.Size {
		t.Fatalf("artifact manifest must retain the uploaded OCI config descriptor: got %#v want %#v", manifest.Config, config)
	}
	if manifest.MediaType != ocispec.MediaTypeImageManifest {
		t.Fatalf("artifact manifest media type = %q, want %q", manifest.MediaType, ocispec.MediaTypeImageManifest)
	}
	if manifest.ArtifactType != ociartifact.EnginePackageArtifactType || len(manifest.Layers) != 1 || manifest.Layers[0].MediaType != layer.MediaType || manifest.Layers[0].Digest != layer.Digest || manifest.Layers[0].Size != layer.Size {
		t.Fatalf("unexpected engine artifact manifest: %#v", manifest)
	}
}

func TestNewOCIArtifactManifestKeepsSignatureSubject(t *testing.T) {
	subject := ocispec.Descriptor{Digest: digest.FromBytes([]byte("subject"))}
	config := emptyOCIConfigDescriptor()
	manifest := newOCIArtifactManifest(devSignatureArtifactType, config, nil, &subject)
	if manifest.MediaType != ocispec.MediaTypeImageManifest {
		t.Fatalf("signature manifest media type = %q, want %q", manifest.MediaType, ocispec.MediaTypeImageManifest)
	}
	if manifest.Config.MediaType != config.MediaType || manifest.Config.Digest != config.Digest || manifest.Config.Size != config.Size {
		t.Fatalf("signature manifest config = %#v, want canonical empty config %#v", manifest.Config, config)
	}
	if manifest.Subject == nil || manifest.Subject.Digest != subject.Digest {
		t.Fatalf("signature manifest must retain its subject: %#v", manifest.Subject)
	}
}

func TestDevelopmentRuntimeImageTransportReferencePreservesImmutableIdentity(t *testing.T) {
	identityValue := "localhost:55433/lunafox/lunafox-engine-runtime-port-scan@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	identity, transport, err := developmentRuntimeImageTransportReference(identityValue, "registry:5000")
	if err != nil {
		t.Fatalf("developmentRuntimeImageTransportReference() error = %v", err)
	}
	if identity.String() != identityValue {
		t.Fatalf("identity = %q, want %q", identity.String(), identityValue)
	}
	if transport.String() != "registry:5000/lunafox/lunafox-engine-runtime-port-scan@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("transport = %q", transport.String())
	}
}

func TestDevelopmentRuntimeImageTransportReferenceRejectsIdentityOrTransportDrift(t *testing.T) {
	validIdentity := "localhost:55433/lunafox/lunafox-engine-runtime-port-scan@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tests := []struct {
		name      string
		identity  string
		transport string
	}{
		{name: "tag only", identity: "localhost:55433/lunafox/lunafox-engine-runtime-port-scan:dev", transport: "registry:5000"},
		{name: "identity whitespace", identity: " " + validIdentity, transport: "registry:5000"},
		{name: "missing transport", identity: validIdentity},
		{name: "transport scheme", identity: validIdentity, transport: "http://registry:5000"},
		{name: "transport path", identity: validIdentity, transport: "registry:5000/lunafox"},
		{name: "same Registry", identity: validIdentity, transport: "localhost:55433"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := developmentRuntimeImageTransportReference(test.identity, test.transport); err == nil {
				t.Fatal("expected invalid identity/transport pair to fail")
			}
		})
	}
}

func TestNewRemoteRepositoryUsesDockerCredentialStore(t *testing.T) {
	const (
		username = "release-publisher"
		password = "registry-token"
	)
	manifest := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.empty.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},"layers":[]}`)
	manifestDigest := digest.FromBytes(manifest)
	authenticated := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotUsername, gotPassword, ok := request.BasicAuth()
		if !ok || gotUsername != username || gotPassword != password {
			writer.Header().Set("WWW-Authenticate", `Basic realm="engine-package-test"`)
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		authenticated = true
		writer.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
		writer.Header().Set("Docker-Content-Digest", manifestDigest.String())
		writer.Header().Set("Content-Length", fmt.Sprint(len(manifest)))
		if request.Method != http.MethodHead {
			_, _ = writer.Write(manifest)
		}
	}))
	defer server.Close()

	registryHost := strings.TrimPrefix(server.URL, "http://")
	dockerConfig := t.TempDir()
	authValue := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	configPayload := []byte(fmt.Sprintf(`{"auths":{%q:{"auth":%q}}}`, registryHost, authValue))
	if err := os.WriteFile(filepath.Join(dockerConfig, "config.json"), configPayload, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOCKER_CONFIG", dockerConfig)

	repository, err := newRemoteRepository(registryHost+"/lunafox/engine-package", true)
	if err != nil {
		t.Fatalf("newRemoteRepository() error = %v", err)
	}
	descriptor, err := repository.Resolve(t.Context(), "package-v2")
	if err != nil {
		t.Fatalf("Resolve() through authenticated Registry error = %v", err)
	}
	if descriptor.Digest != manifestDigest || !authenticated {
		t.Fatalf("authenticated descriptor = %#v, authenticated = %t", descriptor, authenticated)
	}
}

func TestNewRemoteRepositoryAllowsAnonymousDevelopmentRegistry(t *testing.T) {
	t.Setenv("DOCKER_CONFIG", t.TempDir())
	repository, err := newRemoteRepository("localhost:5000/lunafox/engine-package", true)
	if err != nil {
		t.Fatalf("newRemoteRepository() error = %v", err)
	}
	client, ok := repository.Client.(*auth.Client)
	if !ok || client.Credential == nil {
		t.Fatalf("repository client = %#v, want credential-aware auth client", repository.Client)
	}
	credential, err := client.Credential(t.Context(), "localhost:5000")
	if err != nil {
		t.Fatalf("anonymous credential lookup error = %v", err)
	}
	if credential != auth.EmptyCredential {
		t.Fatalf("anonymous credential = %#v, want empty credential", credential)
	}
}

func TestPackageArtifactMediaTypesSelectsOnlyAfterClosedArchiveValidation(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "engine.lunafox.port_scan-1.2.3.lfengine.tar.gz")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := enginepackagebuild.WriteEnginePackageArchive(file, validPackageEntries("engine.lunafox.port_scan")); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	artifactType, layerType, err := packageArtifactMediaTypes(archivePath)
	if err != nil {
		t.Fatalf("packageArtifactMediaTypes() error = %v", err)
	}
	if artifactType != ociartifact.EnginePackageArtifactType || layerType != ociartifact.EnginePackageLayerMediaType {
		t.Fatalf("unexpected v2 media types: %q %q", artifactType, layerType)
	}
}

func TestPackageArtifactMediaTypesRejectsPackageArchive(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "legacy-v1.lfengine.tar.gz")
	writeArchiveEntries(t, archivePath, map[string]string{
		"package.json": `{"packageFormatVersion":"lunafox.engine-package.v1","engineId":"engine.lunafox.port_scan","engineVersion":"1.2.3","runtimeBundles":[]}`,
		"engine.json":  `{"manifestVersion":"engine.v4","engineId":"engine.lunafox.port_scan"}`,
	})
	if _, _, err := packageArtifactMediaTypes(archivePath); err == nil {
		t.Fatal("expected package v1 archive to be rejected")
	}
}

func TestPackageArtifactMediaTypesRejectsIncompleteArchive(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "invalid-v2.lfengine.tar.gz")
	writeArchiveEntries(t, archivePath, map[string]string{
		"package.json": `{"packageFormatVersion":"lunafox.engine-package.v2"}`,
		"engine.json":  `{"engineId":"engine.lunafox.port_scan"}`,
	})
	if _, _, err := packageArtifactMediaTypes(archivePath); err == nil {
		t.Fatal("expected incomplete package v2 archive to be rejected")
	}
}

func validPackageEntries(engineID string) []enginepackagecatalog.PackageLayoutEntry {
	localID := strings.TrimPrefix(engineID, "engine.lunafox.")
	repository := strings.ReplaceAll(localID, "_", "-")
	locale := []byte(`{"engine":{"displayName":"Engine","description":"Engine"},"sections":{"scan":{"name":"Scan","description":"Scan","params":{"timeout":{"description":"Timeout"}}}}}`)
	return []enginepackagecatalog.PackageLayoutEntry{
		{Path: "package.json", Mode: 0o644, Payload: []byte(`{"packageFormatVersion":"lunafox.engine-package.v2","engineId":"` + engineID + `","engineVersion":"1.2.3","runtimeImage":{"refs":["docker.io/lunafox/lunafox-engine-runtime-` + repository + `@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"]}}`)},
		{Path: "engine.json", Mode: 0o644, Payload: []byte(`{"manifestVersion":"engine.v5","engineId":"` + engineID + `","publisher":"lunafox","execution":{"engineApiMajor":2,"supportedTargetTypes":["domain","ip","cidr"],"configSections":[{"id":"scan","defaultEnabled":true,"params":[{"key":"timeout","type":"integer","default":30,"minimum":1}]}]}}`)},
		{Path: "locales/en.json", Mode: 0o644, Payload: append([]byte(nil), locale...)},
		{Path: "locales/zh.json", Mode: 0o644, Payload: append([]byte(nil), locale...)},
	}
}

func writePackageArchive(t *testing.T, path string, entries []enginepackagecatalog.PackageLayoutEntry) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := enginepackagebuild.WriteEnginePackageArchive(file, entries); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeUncheckedArchive(t *testing.T, path string, entries []enginepackagecatalog.PackageLayoutEntry) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		header := &tar.Header{Name: entry.Path, Mode: int64(entry.Mode.Perm()), Size: int64(len(entry.Payload)), Typeflag: tar.TypeReg}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(entry.Payload); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeArchiveEntries(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, payload := range entries {
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(payload))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write([]byte(payload)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
