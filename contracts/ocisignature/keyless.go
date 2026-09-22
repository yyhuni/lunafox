// Package ocisignature verifies first-party OCI manifest signatures.
package ocisignature

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	bundlepb "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	sigstorebundle "github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"google.golang.org/protobuf/encoding/protojson"
	"oras.land/oras-go/v2/registry/remote"
)

// Sigstore 0.3 moved the version into the media subtype. Registry referrer
// filtering must use the publisher's canonical media type, not the 0.1/0.2 form.
const (
	BundleArtifactType  = "application/vnd.dev.sigstore.bundle.v0.3+json"
	BundleLayerType     = "application/vnd.dev.sigstore.bundle.v0.3+json"
	GitHubActionsIssuer = "https://token.actions.githubusercontent.com"
)

// KeylessPolicy fixes the GitHub release identity that may sign first-party
// OCI content. It intentionally contains no signing key material.
type KeylessPolicy struct {
	Repository string
	Workflow   string
	// RefPattern is the only permitted GitHub Actions ref scope. Supported
	// values are refs/tags/* for private product releases and refs/heads/main
	// for the protected public Engine release.
	RefPattern string
}

// KeylessVerifier validates Cosign keyless bundle referrers for an immutable
// OCI manifest. It is shared by Server bootstrap and production installation.
type KeylessVerifier struct {
	policy      KeylessPolicy
	trustedRoot root.TrustedMaterial
}

// DigestReferenceSignatureTransportVerifier verifies an immutable identity
// reference while obtaining OCI signature content through a separate
// transport reference. The two references must share repository and digest.
type DigestReferenceSignatureTransportVerifier interface {
	VerifyReferenceWithTransport(context.Context, ociartifact.DigestReference, ociartifact.DigestReference) error
}

func NewKeylessVerifier(policy KeylessPolicy, trustedRoot root.TrustedMaterial) (*KeylessVerifier, error) {
	policy.Repository = strings.TrimSpace(policy.Repository)
	policy.Workflow = strings.TrimSpace(policy.Workflow)
	policy.RefPattern = strings.TrimSpace(policy.RefPattern)
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`).MatchString(policy.Repository) {
		return nil, fmt.Errorf("sigstore repository policy is invalid")
	}
	if !strings.HasPrefix(policy.Workflow, ".github/workflows/") || !strings.HasSuffix(policy.Workflow, ".yml") {
		return nil, fmt.Errorf("sigstore workflow policy must name a .github/workflows/*.yml file")
	}
	if policy.RefPattern == "" {
		policy.RefPattern = "refs/tags/*"
	}
	if policy.RefPattern != "refs/tags/*" && policy.RefPattern != "refs/heads/main" {
		return nil, fmt.Errorf("sigstore ref policy must be refs/tags/* or refs/heads/main")
	}
	if trustedRoot == nil {
		return nil, fmt.Errorf("sigstore trusted root is required")
	}
	return &KeylessVerifier{policy: policy, trustedRoot: trustedRoot}, nil
}

// NewProductionKeylessVerifier obtains Sigstore's current trusted root before
// installation. The pinned GitHub issuer/repository/workflow policy remains
// the first-party trust boundary; no long-lived signing key is loaded.
func NewProductionKeylessVerifier(policy KeylessPolicy) (*KeylessVerifier, error) {
	trustedRoot, err := root.FetchTrustedRoot()
	if err != nil {
		return nil, fmt.Errorf("load Sigstore trusted root: %w", err)
	}
	return NewKeylessVerifier(policy, trustedRoot)
}

// MissingBundleError reports an empty signature referrer listing. Publication
// clients may poll this after a successful signature upload while a Registry's
// index converges; installation still fails closed when no bundle is present.
type MissingBundleError struct{ Reference string }

func (err *MissingBundleError) Error() string {
	return fmt.Sprintf("no Sigstore signature bundle referrer exists for %q", err.Reference)
}

// TransportError means a Registry request could not complete. It is distinct
// from a received-but-invalid signature so candidate fallback cannot mask an
// integrity failure.
type TransportError struct{ err error }

func (err *TransportError) Error() string { return err.err.Error() }
func (err *TransportError) Unwrap() error { return err.err }

func NewTransportError(err error) error { return transportError(err) }

func IsTransportError(err error) bool {
	var transport *TransportError
	return errors.As(err, &transport)
}

func transportError(err error) error {
	if err == nil {
		return nil
	}
	return &TransportError{err: err}
}

func (verifier *KeylessVerifier) Verify(ctx context.Context, reference ociartifact.DigestReference, descriptor ocispec.Descriptor) error {
	if verifier == nil || verifier.trustedRoot == nil {
		return fmt.Errorf("sigstore verifier is required")
	}
	if descriptor.Digest.String() != reference.Digest {
		return fmt.Errorf("signed OCI descriptor digest mismatch: got %q, want %q", descriptor.Digest, reference.Digest)
	}
	repository, err := remote.NewRepository(reference.Registry + "/" + reference.Repository)
	if err != nil {
		return fmt.Errorf("create signature OCI repository client: %w", err)
	}
	return verifier.verifyWithRepository(ctx, reference, descriptor, repository)
}

func (verifier *KeylessVerifier) verifyWithRepository(
	ctx context.Context,
	identityReference ociartifact.DigestReference,
	descriptor ocispec.Descriptor,
	repository *remote.Repository,
) error {
	var referrers []ocispec.Descriptor
	if err := repository.Referrers(ctx, descriptor, BundleArtifactType, func(found []ocispec.Descriptor) error {
		referrers = append(referrers, found...)
		return nil
	}); err != nil {
		return transportError(fmt.Errorf("discover Sigstore signature referrers: %w", err))
	}
	if len(referrers) == 0 {
		return &MissingBundleError{Reference: identityReference.String()}
	}
	var firstFailure error
	for _, signatureManifest := range referrers {
		if err := verifier.verifyBundleReferrer(ctx, repository, descriptor, signatureManifest); err == nil {
			return nil
		} else if IsTransportError(err) {
			return err
		} else if firstFailure == nil {
			firstFailure = err
		}
	}
	return fmt.Errorf("no Sigstore signature bundle passed verification: %w", firstFailure)
}

// VerifyReference resolves and verifies one immutable OCI reference at its
// identity Registry. Callers that use an alternate download transport must use
// this method before switching transport so a reachable proxy cannot bypass
// GHCR's signed release identity.
func (verifier *KeylessVerifier) VerifyReference(ctx context.Context, reference ociartifact.DigestReference) error {
	return verifier.verifyReferenceWithTransport(ctx, reference, reference, false)
}

// VerifyReferenceWithTransport verifies the identity reference while resolving
// and fetching all OCI signature material from the transport reference. It is
// intentionally separate from VerifyReference so an accelerated caller cannot
// accidentally verify through one Registry and download through another with
// a different repository or digest.
func (verifier *KeylessVerifier) VerifyReferenceWithTransport(
	ctx context.Context,
	identityReference ociartifact.DigestReference,
	transportReference ociartifact.DigestReference,
) error {
	if err := validateSignatureReferencePair(identityReference, transportReference); err != nil {
		return err
	}
	return verifier.verifyReferenceWithTransport(ctx, identityReference, transportReference, true)
}

func (verifier *KeylessVerifier) verifyReferenceWithTransport(
	ctx context.Context,
	identityReference ociartifact.DigestReference,
	transportReference ociartifact.DigestReference,
	forceReferrersAPI bool,
) error {
	if verifier == nil || verifier.trustedRoot == nil {
		return fmt.Errorf("sigstore verifier is required")
	}
	repository, err := newSignatureRepository(transportReference, forceReferrersAPI)
	if err != nil {
		return fmt.Errorf("create signature OCI repository client: %w", err)
	}
	descriptor, err := repository.Resolve(ctx, transportReference.Digest)
	if err != nil {
		return transportError(fmt.Errorf("resolve signed OCI reference %q: %w", identityReference.String(), err))
	}
	if descriptor.Digest.String() != identityReference.Digest {
		return fmt.Errorf("signed OCI descriptor digest mismatch: got %q, want %q", descriptor.Digest, identityReference.Digest)
	}
	return verifier.verifyWithRepository(ctx, identityReference, descriptor, repository)
}

func newSignatureRepository(reference ociartifact.DigestReference, forceReferrersAPI bool) (*remote.Repository, error) {
	repository, err := remote.NewRepository(reference.Registry + "/" + reference.Repository)
	if err != nil {
		return nil, err
	}
	if forceReferrersAPI {
		if err := repository.SetReferrersCapability(true); err != nil {
			return nil, err
		}
	}
	return repository, nil
}

func validateSignatureReferencePair(identityReference, transportReference ociartifact.DigestReference) error {
	if _, err := canonicalSignatureReference(identityReference); err != nil {
		return fmt.Errorf("signature identity reference is invalid: %w", err)
	}
	if _, err := canonicalSignatureReference(transportReference); err != nil {
		return fmt.Errorf("signature transport reference is invalid: %w", err)
	}
	if identityReference.Repository != transportReference.Repository {
		return fmt.Errorf("signature identity and transport references must use the same repository: %q != %q", identityReference.Repository, transportReference.Repository)
	}
	if identityReference.Digest != transportReference.Digest {
		return fmt.Errorf("signature identity and transport references must use the same digest: %q != %q", identityReference.Digest, transportReference.Digest)
	}
	return nil
}

func canonicalSignatureReference(reference ociartifact.DigestReference) (ociartifact.DigestReference, error) {
	parsed, err := ociartifact.ParseDigestReference(reference.String())
	if err != nil {
		return ociartifact.DigestReference{}, err
	}
	if parsed != reference {
		return ociartifact.DigestReference{}, fmt.Errorf("reference is not canonical")
	}
	return parsed, nil
}

func (verifier *KeylessVerifier) verifyBundleReferrer(ctx context.Context, repository *remote.Repository, subject, signatureManifest ocispec.Descriptor) error {
	reader, err := repository.Fetch(ctx, signatureManifest)
	if err != nil {
		return transportError(err)
	}
	payload, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil {
		return transportError(readErr)
	}
	if closeErr != nil {
		return transportError(closeErr)
	}
	var manifest ocispec.Manifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return fmt.Errorf("decode signature referrer manifest: %w", err)
	}
	if manifest.ArtifactType != BundleArtifactType || manifest.Subject == nil || manifest.Subject.Digest != subject.Digest || len(manifest.Layers) != 1 {
		return fmt.Errorf("invalid Sigstore signature referrer manifest")
	}
	layer := manifest.Layers[0]
	if layer.MediaType != BundleLayerType || layer.Size <= 0 || layer.Size > 4<<20 {
		return fmt.Errorf("invalid Sigstore bundle layer descriptor")
	}
	bundleReader, err := repository.Fetch(ctx, layer)
	if err != nil {
		return transportError(err)
	}
	bundlePayload, readErr := io.ReadAll(io.LimitReader(bundleReader, 4<<20+1))
	closeErr = bundleReader.Close()
	if readErr != nil {
		return transportError(readErr)
	}
	if closeErr != nil {
		return transportError(closeErr)
	}
	if len(bundlePayload) > 4<<20 {
		return fmt.Errorf("sigstore bundle exceeds maximum size")
	}
	protobufBundle := &bundlepb.Bundle{}
	if err := protojson.Unmarshal(bundlePayload, protobufBundle); err != nil {
		return fmt.Errorf("decode Sigstore bundle: %w", err)
	}
	verifiedBundle, err := sigstorebundle.NewBundle(protobufBundle)
	if err != nil {
		return fmt.Errorf("validate Sigstore bundle: %w", err)
	}
	identity, err := verifier.identity()
	if err != nil {
		return err
	}
	signedEntityVerifier, err := verify.NewVerifier(verifier.trustedRoot, verify.WithTransparencyLog(1), verify.WithObserverTimestamps(1), verify.WithSignedCertificateTimestamps(1))
	if err != nil {
		return err
	}
	if err := verifyBundleSubject(verifiedBundle, subject.Digest.String()); err != nil {
		return err
	}
	_, err = signedEntityVerifier.Verify(verifiedBundle, verify.NewPolicy(verify.WithoutArtifactUnsafe(), verify.WithCertificateIdentity(identity)))
	return err
}

func verifyBundleSubject(bundle *sigstorebundle.Bundle, expectedDigest string) error {
	content, err := bundle.SignatureContent()
	if err != nil {
		return err
	}
	statement, err := content.EnvelopeContent().Statement()
	if err != nil {
		return err
	}
	for _, subject := range statement.Subject {
		if subject != nil && "sha256:"+subject.Digest["sha256"] == expectedDigest {
			return nil
		}
	}
	return fmt.Errorf("sigstore bundle subject does not match OCI manifest digest %q", expectedDigest)
}

func (verifier *KeylessVerifier) identity() (verify.CertificateIdentity, error) {
	refPattern := regexp.QuoteMeta(verifier.policy.RefPattern)
	refPattern = strings.ReplaceAll(refPattern, `\*`, `.+`)
	subject := "^https://github\\.com/" + regexp.QuoteMeta(verifier.policy.Repository) + "/" + regexp.QuoteMeta(verifier.policy.Workflow) + "@" + refPattern + "$"
	sanMatcher, err := verify.NewSANMatcher("", subject)
	if err != nil {
		return verify.CertificateIdentity{}, err
	}
	issuerMatcher, err := verify.NewIssuerMatcher(GitHubActionsIssuer, "")
	if err != nil {
		return verify.CertificateIdentity{}, err
	}
	return verify.NewCertificateIdentity(sanMatcher, issuerMatcher, certificate.Extensions{GithubWorkflowRepository: verifier.policy.Repository})
}
