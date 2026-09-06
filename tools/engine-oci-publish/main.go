package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/repositoryname"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func main() {
	archive := flag.String("archive", "", "verified .lfengine.tar.gz archive")
	repositoryRef := flag.String("repository", "", "registry/repository without tag")
	repositoryPrefix := flag.String("repository-prefix", "", "registry/namespace used with the archive engineId to derive the repository")
	printEngineID := flag.Bool("print-engine-id", false, "print the archive engineId without publishing")
	signRuntimeImage := flag.String("sign-runtime-image", "", "verified development Runtime Image digest ref to sign")
	transportRegistry := flag.String("transport-registry", "", "development-only Registry transport host for the signed Runtime Image identity")
	tag := flag.String("tag", "dev", "reference tag used only to publish the immutable manifest")
	devPrivateKey := flag.String("dev-private-key", "", "PKCS#8 Ed25519 private key used only by local development publisher")
	plainHTTP := flag.Bool("plain-http", false, "allow HTTP; development registry only")
	flag.Parse()
	if strings.TrimSpace(*signRuntimeImage) != "" {
		if strings.TrimSpace(*archive) != "" || strings.TrimSpace(*repositoryRef) != "" || strings.TrimSpace(*repositoryPrefix) != "" || *printEngineID {
			fmt.Fprintln(os.Stderr, "sign-runtime-image is mutually exclusive with Engine Package operations")
			os.Exit(1)
		}
		if err := publishDevelopmentRuntimeImageSignature(
			context.Background(),
			*signRuntimeImage,
			*transportRegistry,
			*devPrivateKey,
			*plainHTTP,
			os.Stdout,
		); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *printEngineID {
		engineID, err := engineIDFromArchive(*archive)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, engineID)
		return
	}
	resolvedRepository, err := resolveRepository(*archive, *repositoryRef, *repositoryPrefix)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := publish(context.Background(), *archive, resolvedRepository, *tag, *devPrivateKey, *plainHTTP, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resolveRepository(archivePath, repositoryRef, repositoryPrefix string) (string, error) {
	repositoryRef = strings.TrimSpace(repositoryRef)
	repositoryPrefix = strings.Trim(strings.TrimSpace(repositoryPrefix), "/")
	if repositoryRef != "" && repositoryPrefix != "" {
		return "", fmt.Errorf("repository and repository-prefix are mutually exclusive")
	}
	if repositoryRef != "" {
		return repositoryRef, nil
	}
	if repositoryPrefix == "" {
		return "", fmt.Errorf("repository or repository-prefix is required")
	}
	engineID, err := engineIDFromArchive(archivePath)
	if err != nil {
		return "", err
	}
	repositoryName, err := repositoryname.FirstPartyOCIRepositoryName(engineID)
	if err != nil {
		return "", err
	}
	return repositoryPrefix + "/" + repositoryName, nil
}

func engineIDFromArchive(archivePath string) (string, error) {
	layout, err := loadPackageArchive(archivePath)
	if err != nil {
		return "", err
	}
	return layout.Definition.EngineDefinition.EngineID, nil
}

func publish(ctx context.Context, archivePath, repositoryRef, tag, devPrivateKeyPath string, plainHTTP bool, output io.Writer) error {
	if strings.TrimSpace(archivePath) == "" || strings.TrimSpace(repositoryRef) == "" || strings.TrimSpace(tag) == "" {
		return fmt.Errorf("archive, repository and tag are required")
	}
	payload, err := os.ReadFile(archivePath)
	if err != nil {
		return err
	}
	artifactType, layerMediaType, err := packageArtifactMediaTypes(archivePath)
	if err != nil {
		return err
	}
	repository, err := newRemoteRepository(strings.TrimSpace(repositoryRef), plainHTTP)
	if err != nil {
		return err
	}
	layer := ocispec.Descriptor{MediaType: layerMediaType, Digest: digest.FromBytes(payload), Size: int64(len(payload))}
	if err := repository.Push(ctx, layer, strings.NewReader(string(payload))); err != nil {
		return fmt.Errorf("push engine package layer: %w", err)
	}
	config, err := pushEmptyOCIConfig(ctx, repository)
	if err != nil {
		return err
	}
	manifest := newOCIArtifactManifest(artifactType, config, []ocispec.Descriptor{layer}, nil)
	manifestPayload, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	descriptor := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromBytes(manifestPayload), Size: int64(len(manifestPayload))}
	if err := repository.PushReference(ctx, descriptor, strings.NewReader(string(manifestPayload)), tag); err != nil {
		return fmt.Errorf("push engine package manifest: %w", err)
	}
	if strings.TrimSpace(devPrivateKeyPath) != "" {
		if err := publishDevelopmentSignature(ctx, repository, descriptor, devPrivateKeyPath); err != nil {
			return err
		}
	}
	fmt.Fprintf(output, "%s@%s\n", strings.TrimSpace(repositoryRef), descriptor.Digest)
	return nil
}

func packageArtifactMediaTypes(archivePath string) (string, string, error) {
	if _, err := loadPackageArchive(archivePath); err != nil {
		return "", "", err
	}
	return ociartifact.EnginePackageArtifactType, ociartifact.EnginePackageLayerMediaType, nil
}

func loadPackageArchive(archivePath string) (enginepackagecatalog.EnginePackageLayout, error) {
	file, err := os.Open(strings.TrimSpace(archivePath))
	if err != nil {
		return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("open package v2 archive: %w", err)
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("open package v2 archive gzip: %w", err)
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	entries := make([]enginepackagecatalog.PackageLayoutEntry, 0, 4)
	var total int64
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("read package v2 archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("package v2 archive file %q must be a regular file", header.Name)
		}
		if header.Size < 0 || header.Size > 4*1024*1024 || total > 16*1024*1024-header.Size {
			return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("package v2 archive payload exceeds release verifier limit")
		}
		payload, err := io.ReadAll(io.LimitReader(tarReader, header.Size+1))
		if err != nil {
			return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("read package v2 archive file %q: %w", header.Name, err)
		}
		if int64(len(payload)) != header.Size {
			return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("package v2 archive file %q size mismatch", header.Name)
		}
		total += int64(len(payload))
		entries = append(entries, enginepackagecatalog.PackageLayoutEntry{
			Path:    header.Name,
			Mode:    header.FileInfo().Mode(),
			Payload: payload,
		})
	}
	layout, err := enginepackagecatalog.DecodeEnginePackageLayout(entries, archivePath)
	if err != nil {
		return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("validate package v2 archive: %w", err)
	}
	return layout, nil
}

const (
	devSignatureArtifactType = "application/vnd.lunafox.oci.dev-signature.v1"
	devSignatureLayerType    = "application/vnd.lunafox.oci.dev-signature.v1+json"
)

func publishDevelopmentRuntimeImageSignature(
	ctx context.Context,
	identityValue string,
	transportRegistry string,
	privateKeyPath string,
	plainHTTP bool,
	output io.Writer,
) error {
	if strings.TrimSpace(privateKeyPath) == "" {
		return fmt.Errorf("development private key is required to sign a Runtime Image")
	}
	identityReference, transportReference, err := developmentRuntimeImageTransportReference(identityValue, transportRegistry)
	if err != nil {
		return err
	}
	repository, err := newRemoteRepository(transportReference.Registry+"/"+transportReference.Repository, plainHTTP)
	if err != nil {
		return err
	}
	repository.ManifestMediaTypes = []string{ocispec.MediaTypeImageIndex}
	descriptor, err := repository.Resolve(ctx, transportReference.Digest)
	if err != nil {
		return fmt.Errorf("resolve development Runtime Image %q through %q: %w", identityReference.String(), transportReference.Registry, err)
	}
	if descriptor.Digest.String() != identityReference.Digest {
		return fmt.Errorf("development Runtime Image descriptor digest mismatch: got %q, want %q", descriptor.Digest, identityReference.Digest)
	}
	if descriptor.MediaType != ocispec.MediaTypeImageIndex || descriptor.Size <= 0 {
		return fmt.Errorf("development Runtime Image %q is not a non-empty OCI image index", identityReference.String())
	}
	if err := publishDevelopmentSignature(ctx, repository, descriptor, privateKeyPath); err != nil {
		return fmt.Errorf("sign development Runtime Image %q: %w", identityReference.String(), err)
	}
	_, err = fmt.Fprintln(output, identityReference.String())
	return err
}

func newRemoteRepository(repositoryRef string, plainHTTP bool) (*remote.Repository, error) {
	repository, err := remote.NewRepository(strings.TrimSpace(repositoryRef))
	if err != nil {
		return nil, err
	}
	credentialStore, err := credentials.NewStoreFromDocker(credentials.StoreOptions{})
	if err != nil {
		return nil, fmt.Errorf("load Docker credential store: %w", err)
	}
	// Release CI authenticates with docker/login-action. ORAS does not consult
	// that Docker config unless its credential callback is wired explicitly.
	repository.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credentialStore),
	}
	repository.PlainHTTP = plainHTTP
	return repository, nil
}

func developmentRuntimeImageTransportReference(identityValue, transportRegistry string) (ociartifact.DigestReference, ociartifact.DigestReference, error) {
	if identityValue == "" || identityValue != strings.TrimSpace(identityValue) {
		return ociartifact.DigestReference{}, ociartifact.DigestReference{}, fmt.Errorf("development Runtime Image identity must be a canonical digest reference")
	}
	identityReference, err := ociartifact.ParseDigestReference(identityValue)
	if err != nil || identityReference.String() != identityValue {
		return ociartifact.DigestReference{}, ociartifact.DigestReference{}, fmt.Errorf("development Runtime Image identity must be a canonical digest reference")
	}
	if transportRegistry == "" || transportRegistry != strings.TrimSpace(transportRegistry) {
		return ociartifact.DigestReference{}, ociartifact.DigestReference{}, fmt.Errorf("development Runtime Image transport Registry is required and must be canonical")
	}
	transportReference, err := ociartifact.ParseDigestReference(transportRegistry + "/" + identityReference.Repository + "@" + identityReference.Digest)
	if err != nil || transportReference.Registry != transportRegistry {
		return ociartifact.DigestReference{}, ociartifact.DigestReference{}, fmt.Errorf("development Runtime Image transport Registry must be host[:port] without scheme, path, or whitespace")
	}
	if identityReference.Registry == transportReference.Registry {
		return ociartifact.DigestReference{}, ociartifact.DigestReference{}, fmt.Errorf("development Runtime Image identity and transport Registries must differ")
	}
	return identityReference, transportReference, nil
}

func publishDevelopmentSignature(ctx context.Context, repository *remote.Repository, subject ocispec.Descriptor, privateKeyPath string) error {
	pemBytes, err := os.ReadFile(strings.TrimSpace(privateKeyPath))
	if err != nil {
		return err
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return fmt.Errorf("decode development private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return err
	}
	privateKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return fmt.Errorf("development private key must be Ed25519")
	}
	payload, err := json.Marshal(struct {
		SubjectDigest string `json:"subjectDigest"`
		Signature     string `json:"signature"`
	}{subject.Digest.String(), base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(subject.Digest.String())))})
	if err != nil {
		return err
	}
	layer := ocispec.Descriptor{MediaType: devSignatureLayerType, Digest: digest.FromBytes(payload), Size: int64(len(payload))}
	if err := repository.Push(ctx, layer, strings.NewReader(string(payload))); err != nil {
		return err
	}
	config, err := pushEmptyOCIConfig(ctx, repository)
	if err != nil {
		return err
	}
	manifest := newOCIArtifactManifest(devSignatureArtifactType, config, []ocispec.Descriptor{layer}, &subject)
	manifestPayload, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	descriptor := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromBytes(manifestPayload), Size: int64(len(manifestPayload))}
	return repository.PushReference(ctx, descriptor, strings.NewReader(string(manifestPayload)), strings.ReplaceAll(subject.Digest.String(), ":", "-")+".devsig")
}

func pushEmptyOCIConfig(ctx context.Context, repository *remote.Repository) (ocispec.Descriptor, error) {
	// OCI registries require the manifest's config descriptor to reference an
	// existing blob even though LunaFox stores all package metadata in its layer.
	payload := []byte("{}")
	descriptor := emptyOCIConfigDescriptor()
	if err := repository.Push(ctx, descriptor, strings.NewReader(string(payload))); err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("push OCI empty config: %w", err)
	}
	return descriptor, nil
}

func emptyOCIConfigDescriptor() ocispec.Descriptor {
	empty := ocispec.DescriptorEmptyJSON
	// DescriptorEmptyJSON includes optional inline data. It is useful to push
	// the blob but must not be serialized into the manifest, or it changes the
	// artifact's signed bytes without changing the empty config identity.
	return ocispec.Descriptor{
		MediaType: empty.MediaType,
		Digest:    empty.Digest,
		Size:      empty.Size,
	}
}

func newOCIArtifactManifest(artifactType string, config ocispec.Descriptor, layers []ocispec.Descriptor, subject *ocispec.Descriptor) ocispec.Manifest {
	return ocispec.Manifest{
		Versioned:    specs.Versioned{SchemaVersion: 2},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: artifactType,
		Config:       config,
		Layers:       layers,
		Subject:      subject,
	}
}
