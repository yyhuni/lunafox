package engineinstall

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/runtimeimage"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/ocidistribution"
	"github.com/yyhuni/lunafox/contracts/ocisignature"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/errcode"
)

// RuntimeImageIndexVerifierOptions is the Server-side policy for
// resolving and verifying package-bound Engine Runtime Image candidates.
type RuntimeImageIndexVerifierOptions struct {
	MaxIndexBytes                  int64
	PerCandidateTimeout            time.Duration
	AllowPlainHTTP                 bool
	DevelopmentRegistryTransport   *RuntimeImageRegistryTransport
	AllowDevelopmentSinglePlatform bool
	CloudflareAcceleration         bool
	SignatureVerifier              DigestReferenceSignatureVerifier
}

// RuntimeImageRegistryTransport maps one package-declared Registry identity to
// the network authority reachable by an isolated development bootstrap.
type RuntimeImageRegistryTransport struct {
	identityRegistry  string
	transportRegistry string
}

// VerifiedRuntimeImageIndex contains transient registration-ready facts.
// It must not become a second persisted Runtime Image identity beside package refs.
type VerifiedRuntimeImageIndex struct {
	Reference          ociartifact.DigestReference
	RuntimeImageDigest runtimeimage.RuntimeImageDigest
	Descriptor         ocispec.Descriptor
	Index              ocispec.Index
}

// RuntimeImageIndexVerifier verifies the actual OCI index behind a package's
// Runtime Image refs.
type RuntimeImageIndexVerifier struct {
	options       RuntimeImageIndexVerifierOptions
	newRepository runtimeImageOCIRepositoryFactory
}

type runtimeImageOCIRepository interface {
	Resolve(context.Context, string) (ocispec.Descriptor, error)
	Fetch(context.Context, ocispec.Descriptor) (io.ReadCloser, error)
}

type runtimeImageOCIRepositoryFactory func(ociartifact.DigestReference, string, bool) (runtimeImageOCIRepository, error)

// NewDevelopmentRuntimeImageRegistryTransport validates one exact bootstrap-only
// Runtime Image Registry identity-to-transport mapping.
func NewDevelopmentRuntimeImageRegistryTransport(identityRegistry, transportRegistry string) (*RuntimeImageRegistryTransport, error) {
	if err := validateRuntimeImageRegistryAuthority(identityRegistry); err != nil {
		return nil, fmt.Errorf("development Runtime Image identity Registry: %w", err)
	}
	if err := validateRuntimeImageRegistryAuthority(transportRegistry); err != nil {
		return nil, fmt.Errorf("development Runtime Image transport Registry: %w", err)
	}
	return &RuntimeImageRegistryTransport{
		identityRegistry:  identityRegistry,
		transportRegistry: transportRegistry,
	}, nil
}

func NewRuntimeImageIndexVerifier(options RuntimeImageIndexVerifierOptions) (*RuntimeImageIndexVerifier, error) {
	if options.MaxIndexBytes <= 0 || options.MaxIndexBytes == math.MaxInt64 {
		return nil, fmt.Errorf("maximum Runtime Image OCI index size must be positive and bounded")
	}
	if options.PerCandidateTimeout <= 0 {
		return nil, fmt.Errorf("Runtime Image per-candidate timeout is required")
	}
	usesDevelopmentTransport := options.AllowPlainHTTP ||
		options.DevelopmentRegistryTransport != nil ||
		options.AllowDevelopmentSinglePlatform
	if usesDevelopmentTransport && (!options.AllowPlainHTTP ||
		options.DevelopmentRegistryTransport == nil ||
		!options.AllowDevelopmentSinglePlatform) {
		return nil, fmt.Errorf("Runtime Image development transport requires plain HTTP, an identity-to-transport Registry mapping, and single-platform policy")
	}
	if options.CloudflareAcceleration {
		if usesDevelopmentTransport {
			return nil, fmt.Errorf("Cloudflare Runtime Image acceleration cannot be combined with development transport")
		}
		if options.SignatureVerifier == nil {
			return nil, fmt.Errorf("Cloudflare Runtime Image acceleration requires a GHCR signature verifier")
		}
	}

	return &RuntimeImageIndexVerifier{options: options, newRepository: newRuntimeImageRepository}, nil
}

// Verify resolves candidates in package order and advances only after an
// explicitly classified availability failure. Integrity and policy failures
// stop immediately so another Registry cannot hide a different or invalid DAG.
func (verifier *RuntimeImageIndexVerifier) Verify(ctx context.Context, _ string, refs []string) (VerifiedRuntimeImageIndex, error) {
	if verifier == nil {
		return VerifiedRuntimeImageIndex{}, fmt.Errorf("Runtime Image OCI index verifier is required")
	}
	if ctx == nil {
		return VerifiedRuntimeImageIndex{}, fmt.Errorf("Runtime Image verification context is required")
	}
	if err := ctx.Err(); err != nil {
		return VerifiedRuntimeImageIndex{}, err
	}
	candidates, err := runtimeimage.ParseCandidates(refs)
	if err != nil {
		return VerifiedRuntimeImageIndex{}, fmt.Errorf("parse Runtime Image candidates: %w", err)
	}
	if verifier.options.CloudflareAcceleration {
		acceleration, err := ocidistribution.BuildCloudflareAcceleration(refs)
		if err != nil {
			return VerifiedRuntimeImageIndex{}, fmt.Errorf("map Runtime Image Cloudflare acceleration: %w", err)
		}
		if err := verifier.options.SignatureVerifier.VerifyReference(ctx, acceleration.SignatureReference); err != nil {
			return VerifiedRuntimeImageIndex{}, fmt.Errorf("verify Runtime Image GHCR signature %q: %w", acceleration.SignatureReference.String(), err)
		}
		candidates, err = runtimeimage.ParseCandidates(acceleration.DownloadReferenceStrings())
		if err != nil {
			return VerifiedRuntimeImageIndex{}, fmt.Errorf("parse accelerated Runtime Image candidates: %w", err)
		}
	}

	var lastUnavailable error
	for _, reference := range candidates.References {
		if err := ctx.Err(); err != nil {
			return VerifiedRuntimeImageIndex{}, err
		}
		if verifier.options.SignatureVerifier != nil && !verifier.options.CloudflareAcceleration {
			if err := verifier.options.SignatureVerifier.VerifyReference(ctx, reference); err != nil {
				if ocisignature.IsTransportError(err) {
					lastUnavailable = err
					continue
				}
				return VerifiedRuntimeImageIndex{}, fmt.Errorf("verify Runtime Image signature %q: %w", reference.String(), err)
			}
		}
		verified, err := verifier.verifyCandidate(ctx, candidates.RuntimeImageDigest, reference)
		if err == nil {
			return verified, nil
		}
		if !ocidistribution.CanAdvanceAfterFailure(reference, err) {
			return VerifiedRuntimeImageIndex{}, err
		}
		lastUnavailable = err
	}
	return VerifiedRuntimeImageIndex{}, fmt.Errorf("all Runtime Image OCI candidates are unavailable: %w", lastUnavailable)
}

func (verifier *RuntimeImageIndexVerifier) verifyCandidate(
	ctx context.Context,
	runtimeImageDigest runtimeimage.RuntimeImageDigest,
	reference ociartifact.DigestReference,
) (VerifiedRuntimeImageIndex, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, verifier.options.PerCandidateTimeout)
	defer cancel()

	transportRegistry, err := verifier.transportRegistry(reference)
	if err != nil {
		return VerifiedRuntimeImageIndex{}, err
	}
	repository, err := verifier.newRepository(reference, transportRegistry, verifier.options.AllowPlainHTTP)
	if err != nil {
		return VerifiedRuntimeImageIndex{}, fmt.Errorf("create Runtime Image OCI repository client: %w", err)
	}
	descriptor, err := repository.Resolve(attemptCtx, reference.Digest)
	if err != nil {
		return VerifiedRuntimeImageIndex{}, fmt.Errorf(
			"resolve Runtime Image OCI index %q: %w",
			reference.String(),
			classifyRuntimeImageCandidateError(ctx, attemptCtx, err),
		)
	}
	if err := validateRuntimeImageIndexDescriptor(descriptor, reference.Digest, verifier.options.MaxIndexBytes); err != nil {
		return VerifiedRuntimeImageIndex{}, fmt.Errorf("validate Runtime Image OCI index descriptor %q: %w", reference.String(), err)
	}

	reader, err := repository.Fetch(attemptCtx, descriptor)
	if err != nil {
		return VerifiedRuntimeImageIndex{}, fmt.Errorf(
			"fetch Runtime Image OCI index %q: %w",
			reference.String(),
			classifyRuntimeImageCandidateError(ctx, attemptCtx, err),
		)
	}
	payload, err := readRuntimeImageIndexPayload(
		reader,
		verifier.options.MaxIndexBytes,
		reference,
		ctx,
		attemptCtx,
	)
	if err != nil {
		return VerifiedRuntimeImageIndex{}, err
	}
	if int64(len(payload)) > verifier.options.MaxIndexBytes {
		return VerifiedRuntimeImageIndex{}, fmt.Errorf(
			"Runtime Image OCI index %q exceeds limit %d",
			reference.String(),
			verifier.options.MaxIndexBytes,
		)
	}
	if int64(len(payload)) != descriptor.Size {
		return VerifiedRuntimeImageIndex{}, fmt.Errorf(
			"Runtime Image OCI index payload size mismatch for %q: got %d, want %d",
			reference.String(),
			len(payload),
			descriptor.Size,
		)
	}
	if actualDigest := digest.FromBytes(payload).String(); actualDigest != reference.Digest {
		return VerifiedRuntimeImageIndex{}, fmt.Errorf(
			"Runtime Image OCI index payload digest mismatch for %q: got %q, want %q",
			reference.String(),
			actualDigest,
			reference.Digest,
		)
	}

	index, err := decodeRuntimeImageIndex(payload, verifier.options.AllowDevelopmentSinglePlatform)
	if err != nil {
		return VerifiedRuntimeImageIndex{}, fmt.Errorf("validate Runtime Image OCI index %q: %w", reference.String(), err)
	}
	if err := runtimeImageAttemptContextError(ctx, attemptCtx); err != nil {
		return VerifiedRuntimeImageIndex{}, fmt.Errorf("verify Runtime Image OCI index %q: %w", reference.String(), err)
	}
	if err := runtimeImageAttemptContextError(ctx, attemptCtx); err != nil {
		return VerifiedRuntimeImageIndex{}, fmt.Errorf("complete Runtime Image OCI index verification %q: %w", reference.String(), err)
	}

	return VerifiedRuntimeImageIndex{
		Reference:          reference,
		RuntimeImageDigest: runtimeImageDigest,
		Descriptor:         descriptor,
		Index:              index,
	}, nil
}

func (verifier *RuntimeImageIndexVerifier) transportRegistry(reference ociartifact.DigestReference) (string, error) {
	transport := verifier.options.DevelopmentRegistryTransport
	if transport == nil {
		return reference.Registry, nil
	}
	if reference.Registry != transport.identityRegistry {
		return "", fmt.Errorf(
			"Runtime Image Registry %q does not match configured development identity Registry %q",
			reference.Registry,
			transport.identityRegistry,
		)
	}
	return transport.transportRegistry, nil
}

func newRuntimeImageRepository(reference ociartifact.DigestReference, transportRegistry string, plainHTTP bool) (runtimeImageOCIRepository, error) {
	repository, err := remote.NewRepository(transportRegistry + "/" + reference.Repository)
	if err != nil {
		return nil, err
	}
	repository.PlainHTTP = plainHTTP
	repository.ManifestMediaTypes = []string{ocispec.MediaTypeImageIndex}
	return repository, nil
}

func validateRuntimeImageRegistryAuthority(value string) error {
	if value == "" || value != strings.TrimSpace(value) {
		return fmt.Errorf("Registry authority must be a non-empty canonical value")
	}
	const validationSuffix = "/lunafox/runtime-image-transport@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	reference, err := ociartifact.ParseDigestReference(value + validationSuffix)
	if err != nil || reference.Registry != value {
		if err == nil {
			err = fmt.Errorf("parsed Registry %q does not match", reference.Registry)
		}
		return fmt.Errorf("invalid Registry authority %q: %w", value, err)
	}
	return nil
}

func runtimeImageAttemptContextError(parentCtx, attemptCtx context.Context) error {
	if parentCtx != nil {
		if err := parentCtx.Err(); err != nil {
			return err
		}
	}
	if attemptCtx != nil {
		if err := attemptCtx.Err(); err != nil {
			return classifyRuntimeImageCandidateError(parentCtx, attemptCtx, err)
		}
	}
	return nil
}

func readRuntimeImageIndexPayload(
	reader io.ReadCloser,
	maxBytes int64,
	reference ociartifact.DigestReference,
	parentCtx context.Context,
	attemptCtx context.Context,
) ([]byte, error) {
	payload, readErr := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	closeErr := reader.Close()
	if readErr != nil {
		classifiedReadErr := classifyRuntimeImageCandidateError(parentCtx, attemptCtx, readErr)
		if closeErr != nil {
			return nil, fmt.Errorf(
				"read Runtime Image OCI index %q failed (%v) and close failed (%v)",
				reference.String(),
				classifiedReadErr,
				closeErr,
			)
		}
		return nil, fmt.Errorf("read Runtime Image OCI index %q: %w", reference.String(), classifiedReadErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf(
			"close Runtime Image OCI index %q: %w",
			reference.String(),
			classifyRuntimeImageCandidateError(parentCtx, attemptCtx, closeErr),
		)
	}
	return payload, nil
}

func validateRuntimeImageIndexDescriptor(descriptor ocispec.Descriptor, expectedDigest string, maxBytes int64) error {
	if descriptor.MediaType != ocispec.MediaTypeImageIndex {
		return fmt.Errorf("media type %q is not the required OCI image index media type", descriptor.MediaType)
	}
	if descriptor.Digest.String() != expectedDigest {
		return fmt.Errorf("digest mismatch: got %q, want %q", descriptor.Digest, expectedDigest)
	}
	if err := descriptor.Digest.Validate(); err != nil || descriptor.Digest.Algorithm() != digest.SHA256 {
		return fmt.Errorf("digest %q is not a canonical SHA-256 digest", descriptor.Digest)
	}
	if descriptor.Size <= 0 {
		return fmt.Errorf("size must be positive")
	}
	if descriptor.Size > maxBytes {
		return fmt.Errorf("size %d exceeds limit %d", descriptor.Size, maxBytes)
	}
	return nil
}

func decodeRuntimeImageIndex(payload []byte, allowDevelopmentSinglePlatform bool) (ocispec.Index, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var index ocispec.Index
	if err := decoder.Decode(&index); err != nil {
		return ocispec.Index{}, fmt.Errorf("decode strict OCI image index: %w", err)
	}
	if err := consumeRuntimeImageIndexJSONEOF(decoder); err != nil {
		return ocispec.Index{}, fmt.Errorf("decode strict OCI image index: %w", err)
	}
	if index.SchemaVersion != 2 {
		return ocispec.Index{}, fmt.Errorf("schemaVersion must be 2, got %d", index.SchemaVersion)
	}
	if index.MediaType != ocispec.MediaTypeImageIndex {
		return ocispec.Index{}, fmt.Errorf("mediaType %q is not the required OCI image index media type", index.MediaType)
	}
	if index.ArtifactType != "" {
		return ocispec.Index{}, fmt.Errorf("artifactType is not permitted on an Engine Runtime Image index")
	}
	if index.Subject != nil {
		return ocispec.Index{}, fmt.Errorf("subject is not permitted on an Engine Runtime Image index")
	}

	requiredPlatforms := map[string]int{
		"linux/amd64": 0,
		"linux/arm64": 0,
	}
	seenDigests := make(map[digest.Digest]struct{}, len(index.Manifests))
	for position, descriptor := range index.Manifests {
		if descriptor.MediaType != ocispec.MediaTypeImageManifest {
			return ocispec.Index{}, fmt.Errorf("manifests[%d] media type %q is not an OCI image manifest", position, descriptor.MediaType)
		}
		if err := descriptor.Digest.Validate(); err != nil || descriptor.Digest.Algorithm() != digest.SHA256 {
			return ocispec.Index{}, fmt.Errorf("manifests[%d] digest %q is not a canonical SHA-256 digest", position, descriptor.Digest)
		}
		if descriptor.Size <= 0 {
			return ocispec.Index{}, fmt.Errorf("manifests[%d] size must be positive", position)
		}
		if descriptor.Platform == nil || descriptor.Platform.OS == "" || descriptor.Platform.Architecture == "" {
			return ocispec.Index{}, fmt.Errorf("manifests[%d] must declare a non-empty platform", position)
		}
		if descriptor.ArtifactType != "" {
			return ocispec.Index{}, fmt.Errorf("manifests[%d] artifactType is not permitted for a Runtime Image manifest", position)
		}
		if _, duplicate := seenDigests[descriptor.Digest]; duplicate {
			return ocispec.Index{}, fmt.Errorf("manifests[%d] repeats child digest %q", position, descriptor.Digest)
		}
		seenDigests[descriptor.Digest] = struct{}{}

		platformKey := descriptor.Platform.OS + "/" + descriptor.Platform.Architecture
		if _, required := requiredPlatforms[platformKey]; required {
			requiredPlatforms[platformKey]++
		}
	}
	if allowDevelopmentSinglePlatform {
		platformCount := requiredPlatforms["linux/amd64"] + requiredPlatforms["linux/arm64"]
		if platformCount != 1 || len(index.Manifests) != 1 {
			return ocispec.Index{}, fmt.Errorf("development Runtime Image index must contain exactly one linux/amd64 or linux/arm64 manifest")
		}
		for _, platform := range []string{"linux/amd64", "linux/arm64"} {
			if requiredPlatforms[platform] > 1 {
				return ocispec.Index{}, fmt.Errorf("development Runtime Image index must not repeat platform %s", platform)
			}
		}
		return index, nil
	}
	for _, platform := range []string{"linux/amd64", "linux/arm64"} {
		if requiredPlatforms[platform] != 1 {
			return ocispec.Index{}, fmt.Errorf("required platform %s must appear exactly once, got %d", platform, requiredPlatforms[platform])
		}
	}
	return index, nil
}

func consumeRuntimeImageIndexJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != nil {
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("unexpected trailing JSON content: %w", err)
	}
	return fmt.Errorf("unexpected trailing JSON content")
}

func classifyRuntimeImageCandidateError(parentCtx, attemptCtx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if parentCtx != nil {
		if parentErr := parentCtx.Err(); parentErr != nil {
			return parentErr
		}
	}
	if aggregateErr := failClosedAggregateCandidateError(err); aggregateErr != nil {
		return aggregateErr
	}
	if ociartifact.IsCandidateUnavailable(err) {
		return err
	}
	if reason, ok := runtimeImageCandidateFailureReason(err); ok {
		return ociartifact.NewCandidateFailure(reason, err)
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		(attemptCtx != nil && errors.Is(attemptCtx.Err(), context.DeadlineExceeded) && errors.Is(err, context.Canceled)) {
		return ociartifact.NewCandidateFailure(ociartifact.CandidateFailureReasonAttemptTimeout, err)
	}
	return err
}

func runtimeImageCandidateFailureReason(err error) (ociartifact.CandidateFailureReason, bool) {
	var response *errcode.ErrorResponse
	if errors.As(err, &response) {
		var responseReason ociartifact.CandidateFailureReason
		for index, registryError := range response.Errors {
			var reason ociartifact.CandidateFailureReason
			switch registryError.Code {
			case errcode.ErrorCodeManifestUnknown:
				reason = ociartifact.CandidateFailureReasonManifestUnknown
			case errcode.ErrorCodeBlobUnknown:
				reason = ociartifact.CandidateFailureReasonBlobUnknown
			case errcode.ErrorCodeUnauthorized:
				reason = ociartifact.CandidateFailureReasonUnauthorized
			case errcode.ErrorCodeDenied:
				reason = ociartifact.CandidateFailureReasonForbidden
			default:
				return ociartifact.CandidateFailureReasonUnknown, false
			}
			if index == 0 {
				responseReason = reason
			}
		}
		if len(response.Errors) > 0 {
			return responseReason, true
		}
		switch response.StatusCode {
		case http.StatusUnauthorized:
			return ociartifact.CandidateFailureReasonUnauthorized, true
		case http.StatusForbidden:
			return ociartifact.CandidateFailureReasonForbidden, true
		case http.StatusNotFound:
			return ociartifact.CandidateFailureReasonNotFound, true
		case http.StatusTooManyRequests:
			return ociartifact.CandidateFailureReasonRateLimited, true
		case http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return ociartifact.CandidateFailureReasonTransientServer, true
		}
	}
	if errors.Is(err, errdef.ErrNotFound) {
		return ociartifact.CandidateFailureReasonNotFound, true
	}

	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return ociartifact.CandidateFailureReasonDNS, true
	}
	if isRuntimeImageTLSError(err) {
		return ociartifact.CandidateFailureReasonTLS, true
	}
	var networkError *net.OpError
	if errors.As(err, &networkError) {
		return ociartifact.CandidateFailureReasonTCP, true
	}
	var syscallError *os.SyscallError
	if errors.As(err, &syscallError) && isRuntimeImageTCPSystemError(syscallError.Err) {
		return ociartifact.CandidateFailureReasonTCP, true
	}
	if isRuntimeImageTCPSystemError(err) {
		return ociartifact.CandidateFailureReasonTCP, true
	}
	return ociartifact.CandidateFailureReasonUnknown, false
}

func isRuntimeImageTLSError(err error) bool {
	var certificateVerificationError *tls.CertificateVerificationError
	var recordHeaderError tls.RecordHeaderError
	var alertError tls.AlertError
	var unknownAuthorityError x509.UnknownAuthorityError
	var hostnameError x509.HostnameError
	var certificateInvalidError x509.CertificateInvalidError
	var systemRootsError x509.SystemRootsError
	return errors.As(err, &certificateVerificationError) ||
		errors.As(err, &recordHeaderError) ||
		errors.As(err, &alertError) ||
		errors.As(err, &unknownAuthorityError) ||
		errors.As(err, &hostnameError) ||
		errors.As(err, &certificateInvalidError) ||
		errors.As(err, &systemRootsError)
}

func isRuntimeImageTCPSystemError(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.ENETUNREACH) ||
		errors.Is(err, syscall.ETIMEDOUT)
}
