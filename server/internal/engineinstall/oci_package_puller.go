package engineinstall

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"hash"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/ocidistribution"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/errcode"
)

// ORASPackagePullerOptions defines bounded Server-side OCI transport limits
// for Engine Package v2 installation.
type ORASPackagePullerOptions struct {
	MaxManifestBytes     int64
	MaxPackageLayerBytes int64
	PerCandidateTimeout  time.Duration
	AllowPlainHTTP       bool
}

// PulledPackageLayer is the ephemeral signed package archive stream supplied
// to PackageLayerConsumer. The puller owns the stream lifetime; the consumer
// must read Reader to its terminal result before returning.
type PulledPackageLayer struct {
	Reference              ociartifact.EnginePackageArtifactReference
	ArtifactManifestDigest ociartifact.ArtifactManifestDigest
	PackageDigest          ociartifact.PackageDigest
	Size                   int64
	Reader                 io.Reader
}

// PackageLayerConsumer performs Server-owned streaming staging inside one
// candidate attempt. Keeping consumption inside the attempt lets transport
// failures during the layer body participate in the same closed failover rule.
// It may be called again for a later candidate, so each call must isolate and
// discard partial staging state until the stream and its own checks both pass.
type PackageLayerConsumer func(context.Context, PulledPackageLayer) error

// ConsumedPackageLayer records the immutable facts from the candidate whose
// complete layer stream the consumer accepted.
type ConsumedPackageLayer struct {
	Reference              ociartifact.EnginePackageArtifactReference
	ArtifactManifestDigest ociartifact.ArtifactManifestDigest
	PackageDigest          ociartifact.PackageDigest
	Size                   int64
}

type packageRepository interface {
	Resolve(context.Context, string) (ocispec.Descriptor, error)
	Fetch(context.Context, ocispec.Descriptor) (io.ReadCloser, error)
}

type packageRepositoryFactory func(ociartifact.EnginePackageArtifactReference, bool) (packageRepository, error)

// ORASPackagePuller owns only OCI transport, descriptor validation, and
// package artifact integrity verification.
type ORASPackagePuller struct {
	options       ORASPackagePullerOptions
	newRepository packageRepositoryFactory
}

func NewORASPackagePuller(options ORASPackagePullerOptions) (*ORASPackagePuller, error) {
	if options.MaxManifestBytes <= 0 || options.MaxManifestBytes == math.MaxInt64 {
		return nil, fmt.Errorf("maximum Engine Package v2 OCI manifest size must be positive and bounded")
	}
	if options.MaxPackageLayerBytes <= 0 || options.MaxPackageLayerBytes == math.MaxInt64 {
		return nil, fmt.Errorf("maximum Engine Package v2 layer size must be positive and bounded")
	}
	if options.PerCandidateTimeout <= 0 {
		return nil, fmt.Errorf("Engine Package v2 per-candidate timeout is required")
	}

	return &ORASPackagePuller{
		options:       options,
		newRepository: newRemotePackageRepository,
	}, nil
}

// ConsumePackageLayer tries immutable artifact locations in declared order. A
// different candidate may be used only for a positively classified availability
// failure; integrity, policy, caller-budget, and unknown failures stop at once.
func (puller *ORASPackagePuller) ConsumePackageLayer(
	ctx context.Context,
	candidates ociartifact.ArtifactCandidates,
	consume PackageLayerConsumer,
) (ConsumedPackageLayer, error) {
	if puller == nil {
		return ConsumedPackageLayer{}, fmt.Errorf("Engine Package v2 OCI puller is required")
	}
	if ctx == nil {
		return ConsumedPackageLayer{}, fmt.Errorf("Engine Package v2 pull context is required")
	}
	if consume == nil {
		return ConsumedPackageLayer{}, fmt.Errorf("Engine Package v2 layer consumer is required")
	}
	validatedCandidates, err := validatePackageCandidates(candidates)
	if err != nil {
		return ConsumedPackageLayer{}, err
	}
	var lastUnavailable error
	for _, reference := range validatedCandidates.References {
		if err := ctx.Err(); err != nil {
			return ConsumedPackageLayer{}, err
		}

		attemptCtx, cancel := context.WithTimeout(ctx, puller.options.PerCandidateTimeout)
		stream, verifiedReader, err := puller.pullCandidate(attemptCtx, ctx, reference)
		if err == nil {
			consumeErr := consume(attemptCtx, stream)
			closeErr := verifiedReader.Close()
			err = packageLayerConsumerAttemptError(consumeErr, closeErr, verifiedReader)
		}
		if parentErr := ctx.Err(); parentErr != nil {
			err = parentErr
		} else if err == nil {
			if attemptErr := attemptCtx.Err(); attemptErr != nil {
				// The callback may not turn an expired attempt into success by
				// ignoring its context. At this point no reader failure authorizes
				// Registry failover, so budget exhaustion fails this install attempt.
				err = fmt.Errorf("Engine Package v2 candidate execution budget exhausted: %w", attemptErr)
			}
		}
		cancel()
		if parentErr := ctx.Err(); parentErr != nil {
			return ConsumedPackageLayer{}, parentErr
		}
		if err == nil {
			return ConsumedPackageLayer{
				Reference:              stream.Reference,
				ArtifactManifestDigest: stream.ArtifactManifestDigest,
				PackageDigest:          stream.PackageDigest,
				Size:                   stream.Size,
			}, nil
		}
		if parentErr := ctx.Err(); parentErr != nil {
			return ConsumedPackageLayer{}, parentErr
		}
		if !ocidistribution.CanAdvanceAfterFailure(reference.DigestReference(), err) {
			return ConsumedPackageLayer{}, err
		}
		lastUnavailable = err
	}

	return ConsumedPackageLayer{}, fmt.Errorf("all Engine Package v2 OCI candidates are unavailable: %w", lastUnavailable)
}

func (puller *ORASPackagePuller) pullCandidate(
	attemptCtx context.Context,
	parentCtx context.Context,
	reference ociartifact.EnginePackageArtifactReference,
) (PulledPackageLayer, *verifiedPackageLayerReader, error) {
	repository, err := puller.newRepository(reference, puller.options.AllowPlainHTTP)
	if err != nil {
		return PulledPackageLayer{}, nil, fmt.Errorf("create Engine Package v2 OCI repository client: %w", err)
	}

	manifestDescriptor, err := repository.Resolve(attemptCtx, string(reference.ArtifactManifestDigest))
	if err != nil {
		return PulledPackageLayer{}, nil, fmt.Errorf(
			"resolve Engine Package v2 OCI manifest %q: %w",
			reference.String(),
			classifyPackageCandidateError(parentCtx, attemptCtx, err),
		)
	}
	if err := validatePackageManifestDescriptor(
		manifestDescriptor,
		reference.ArtifactManifestDigest,
		puller.options.MaxManifestBytes,
	); err != nil {
		return PulledPackageLayer{}, nil, fmt.Errorf("validate Engine Package v2 OCI manifest descriptor %q: %w", reference.String(), err)
	}

	manifestReader, err := repository.Fetch(attemptCtx, manifestDescriptor)
	if err != nil {
		return PulledPackageLayer{}, nil, fmt.Errorf(
			"fetch Engine Package v2 OCI manifest %q: %w",
			reference.String(),
			classifyPackageCandidateError(parentCtx, attemptCtx, err),
		)
	}
	manifestPayload, err := readPackageManifestPayload(
		manifestReader,
		manifestDescriptor,
		reference,
		puller.options.MaxManifestBytes,
		parentCtx,
		attemptCtx,
	)
	if err != nil {
		return PulledPackageLayer{}, nil, err
	}
	manifest, err := ociartifact.DecodeEnginePackageManifest(manifestPayload, reference.String())
	if err != nil {
		return PulledPackageLayer{}, nil, err
	}
	if manifest.ArtifactManifestDigest != reference.ArtifactManifestDigest {
		return PulledPackageLayer{}, nil, fmt.Errorf(
			"Engine Package v2 OCI manifest digest mismatch for %q: got %q, want %q",
			reference.String(),
			manifest.ArtifactManifestDigest,
			reference.ArtifactManifestDigest,
		)
	}
	if manifest.PackageLayer.Size > puller.options.MaxPackageLayerBytes {
		return PulledPackageLayer{}, nil, fmt.Errorf(
			"Engine Package v2 OCI layer size %d exceeds limit %d",
			manifest.PackageLayer.Size,
			puller.options.MaxPackageLayerBytes,
		)
	}

	layerDescriptor := ocispec.Descriptor{
		MediaType: manifest.PackageLayer.MediaType,
		Digest:    digest.Digest(manifest.PackageLayer.PackageDigest),
		Size:      manifest.PackageLayer.Size,
	}
	layerReader, err := repository.Fetch(attemptCtx, layerDescriptor)
	if err != nil {
		return PulledPackageLayer{}, nil, fmt.Errorf(
			"fetch Engine Package v2 OCI layer %q from %q: %w",
			manifest.PackageLayer.PackageDigest,
			reference.String(),
			classifyPackageCandidateError(parentCtx, attemptCtx, err),
		)
	}

	verifiedReader := newVerifiedPackageLayerReader(
		layerReader,
		manifest.PackageLayer.PackageDigest,
		manifest.PackageLayer.Size,
		parentCtx,
		attemptCtx,
	)
	return PulledPackageLayer{
		Reference:              reference,
		ArtifactManifestDigest: reference.ArtifactManifestDigest,
		PackageDigest:          manifest.PackageLayer.PackageDigest,
		Size:                   manifest.PackageLayer.Size,
		Reader:                 verifiedReader,
	}, verifiedReader, nil
}

func validatePackageCandidates(candidates ociartifact.ArtifactCandidates) (ociartifact.ArtifactCandidates, error) {
	values := make([]string, len(candidates.References))
	for index, reference := range candidates.References {
		values[index] = reference.String()
	}
	validated, err := ociartifact.ParseArtifactCandidates(values)
	if err != nil {
		return ociartifact.ArtifactCandidates{}, fmt.Errorf("validate Engine Package v2 OCI candidates: %w", err)
	}
	if validated.ArtifactManifestDigest != candidates.ArtifactManifestDigest {
		return ociartifact.ArtifactCandidates{}, fmt.Errorf(
			"Engine Package v2 OCI candidate group digest mismatch: got %q, want %q",
			candidates.ArtifactManifestDigest,
			validated.ArtifactManifestDigest,
		)
	}
	return validated, nil
}

func newRemotePackageRepository(reference ociartifact.EnginePackageArtifactReference, plainHTTP bool) (packageRepository, error) {
	repository, err := remote.NewRepository(reference.Registry + "/" + reference.Repository)
	if err != nil {
		return nil, err
	}
	repository.PlainHTTP = plainHTTP
	repository.ManifestMediaTypes = []string{ocispec.MediaTypeImageManifest}
	return repository, nil
}

func validatePackageManifestDescriptor(
	descriptor ocispec.Descriptor,
	expectedDigest ociartifact.ArtifactManifestDigest,
	maxBytes int64,
) error {
	if descriptor.MediaType != ocispec.MediaTypeImageManifest {
		return fmt.Errorf("media type %q is not the required OCI image manifest media type", descriptor.MediaType)
	}
	if descriptor.Digest.String() != string(expectedDigest) {
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

func readPackageManifestPayload(
	reader io.ReadCloser,
	descriptor ocispec.Descriptor,
	reference ociartifact.EnginePackageArtifactReference,
	maxBytes int64,
	parentCtx context.Context,
	attemptCtx context.Context,
) ([]byte, error) {
	payload, readErr := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	closeErr := reader.Close()
	if readErr != nil {
		classifiedReadErr := classifyPackageCandidateError(parentCtx, attemptCtx, readErr)
		if closeErr != nil {
			return nil, fmt.Errorf(
				"read Engine Package v2 OCI manifest %q failed (%v) and close failed (%v)",
				reference.String(),
				classifiedReadErr,
				closeErr,
			)
		}
		return nil, fmt.Errorf(
			"read Engine Package v2 OCI manifest %q: %w",
			reference.String(),
			classifiedReadErr,
		)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close Engine Package v2 OCI manifest %q: %w", reference.String(), closeErr)
	}
	if int64(len(payload)) > maxBytes {
		return nil, fmt.Errorf("Engine Package v2 OCI manifest %q exceeds limit %d", reference.String(), maxBytes)
	}
	if int64(len(payload)) != descriptor.Size {
		return nil, fmt.Errorf(
			"Engine Package v2 OCI manifest payload size mismatch for %q: got %d, want %d",
			reference.String(),
			len(payload),
			descriptor.Size,
		)
	}
	if actualDigest := digest.FromBytes(payload).String(); actualDigest != string(reference.ArtifactManifestDigest) {
		return nil, fmt.Errorf(
			"Engine Package v2 OCI manifest payload digest mismatch for %q: got %q, want %q",
			reference.String(),
			actualDigest,
			reference.ArtifactManifestDigest,
		)
	}
	return payload, nil
}

type verifiedPackageLayerReader struct {
	reader         io.ReadCloser
	expectedDigest ociartifact.PackageDigest
	expectedSize   int64
	readSize       int64
	hash           hash.Hash
	parentCtx      context.Context
	attemptCtx     context.Context
	terminalErr    error
	candidateErr   *packageLayerCandidateFailure
	closeErr       error
	closed         bool
}

// packageLayerCandidateFailure proves that the verified reader, rather than
// its callback, observed the typed transport failure. The pointer instance is
// private to one stream and is required before callback errors can authorize
// Registry failover.
type packageLayerCandidateFailure struct{ cause error }

func (failure *packageLayerCandidateFailure) Error() string { return failure.cause.Error() }
func (failure *packageLayerCandidateFailure) Unwrap() error { return failure.cause }

func newVerifiedPackageLayerReader(
	reader io.ReadCloser,
	expectedDigest ociartifact.PackageDigest,
	expectedSize int64,
	parentCtx context.Context,
	attemptCtx context.Context,
) *verifiedPackageLayerReader {
	return &verifiedPackageLayerReader{
		reader:         reader,
		expectedDigest: expectedDigest,
		expectedSize:   expectedSize,
		hash:           sha256.New(),
		parentCtx:      parentCtx,
		attemptCtx:     attemptCtx,
	}
}

func (reader *verifiedPackageLayerReader) Read(payload []byte) (int, error) {
	if reader.terminalErr != nil {
		return 0, reader.terminalErr
	}
	if len(payload) == 0 {
		return 0, nil
	}
	remaining := reader.expectedSize - reader.readSize
	if remaining == 0 {
		return reader.finish()
	}
	if int64(len(payload)) > remaining {
		payload = payload[:remaining]
	}

	n, err := reader.reader.Read(payload)
	if n > 0 {
		_, _ = reader.hash.Write(payload[:n])
		reader.readSize += int64(n)
	}
	if reader.readSize > reader.expectedSize {
		reader.terminalErr = fmt.Errorf(
			"Engine Package v2 OCI layer size exceeds descriptor: got at least %d, want %d",
			reader.readSize,
			reader.expectedSize,
		)
		return n, reader.terminalErr
	}
	if err != nil {
		if errors.Is(err, io.EOF) {
			if reader.readSize != reader.expectedSize {
				reader.terminalErr = fmt.Errorf(
					"Engine Package v2 OCI layer size mismatch: got %d, want %d",
					reader.readSize,
					reader.expectedSize,
				)
				return n, reader.terminalErr
			}
			_, finishErr := reader.finish()
			return n, finishErr
		}
		reader.terminalErr = fmt.Errorf(
			"read Engine Package v2 OCI layer: %w",
			classifyPackageCandidateError(reader.parentCtx, reader.attemptCtx, err),
		)
		reader.markCandidateFailure()
		return n, reader.terminalErr
	}
	return n, nil
}

func (reader *verifiedPackageLayerReader) finish() (int, error) {
	var probe [1]byte
	n, err := reader.reader.Read(probe[:])
	if n > 0 {
		reader.terminalErr = fmt.Errorf(
			"Engine Package v2 OCI layer size exceeds descriptor: got at least %d, want %d",
			reader.expectedSize+int64(n),
			reader.expectedSize,
		)
		return 0, reader.terminalErr
	}
	if err != nil && !errors.Is(err, io.EOF) {
		reader.terminalErr = fmt.Errorf(
			"read Engine Package v2 OCI layer terminator: %w",
			classifyPackageCandidateError(reader.parentCtx, reader.attemptCtx, err),
		)
		reader.markCandidateFailure()
		return 0, reader.terminalErr
	}
	if err == nil {
		return 0, nil
	}
	if actualDigest := "sha256:" + fmt.Sprintf("%x", reader.hash.Sum(nil)); actualDigest != string(reader.expectedDigest) {
		reader.terminalErr = fmt.Errorf(
			"Engine Package v2 OCI layer digest mismatch: got %q, want %q",
			actualDigest,
			reader.expectedDigest,
		)
		return 0, reader.terminalErr
	}
	reader.terminalErr = io.EOF
	return 0, io.EOF
}

func (reader *verifiedPackageLayerReader) markCandidateFailure() {
	if !ociartifact.IsCandidateUnavailable(reader.terminalErr) {
		return
	}
	failure := &packageLayerCandidateFailure{cause: reader.terminalErr}
	reader.candidateErr = failure
	reader.terminalErr = failure
}

func (reader *verifiedPackageLayerReader) Close() error {
	if reader.closed {
		return nil
	}
	reader.closed = true
	reader.closeErr = reader.reader.Close()
	if reader.terminalErr == nil {
		if reader.closeErr != nil {
			return fmt.Errorf(
				"Engine Package v2 OCI layer closed before integrity verification: got %d bytes, want %d (close error: %v)",
				reader.readSize,
				reader.expectedSize,
				reader.closeErr,
			)
		}
		return fmt.Errorf(
			"Engine Package v2 OCI layer closed before integrity verification: got %d bytes, want %d",
			reader.readSize,
			reader.expectedSize,
		)
	}
	if !errors.Is(reader.terminalErr, io.EOF) {
		if reader.closeErr != nil {
			return fmt.Errorf(
				"close Engine Package v2 OCI layer after read failure %v: %v",
				reader.terminalErr,
				reader.closeErr,
			)
		}
		return reader.terminalErr
	}
	if reader.closeErr != nil {
		return fmt.Errorf("close verified Engine Package v2 OCI layer: %v", reader.closeErr)
	}
	return nil
}

func packageLayerConsumerAttemptError(
	consumerErr error,
	closeErr error,
	reader *verifiedPackageLayerReader,
) error {
	if reader.closeErr != nil {
		// A second unknown/local failure cannot be hidden by a typed read error.
		return closeErr
	}
	if consumerErr == nil {
		if closeErr != nil && ociartifact.IsCandidateUnavailable(closeErr) {
			return fmt.Errorf("Engine Package v2 layer consumer suppressed reader failure: %v", closeErr)
		}
		return closeErr
	}
	if reader.candidateErr != nil && linearlyWrapsPackageLayerCandidate(consumerErr, reader.candidateErr) {
		return consumerErr
	}
	if ociartifact.IsCandidateUnavailable(consumerErr) {
		return fmt.Errorf("Engine Package v2 layer consumer returned an unauthorized candidate failure: %v", consumerErr)
	}
	return consumerErr
}

func linearlyWrapsPackageLayerCandidate(err error, expected *packageLayerCandidateFailure) bool {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if _, multi := current.(interface{ Unwrap() []error }); multi {
			return false
		}
		if candidate, ok := current.(*packageLayerCandidateFailure); ok {
			return candidate == expected
		}
	}
	return false
}

func failClosedAggregateCandidateError(err error) error {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if _, aggregate := current.(interface{ Unwrap() []error }); aggregate {
			// Do not retain the aggregate unwrap graph: one availability branch
			// must never authorize fallback over an unknown/integrity sibling.
			return fmt.Errorf("aggregate OCI candidate error fails closed: %v", err)
		}
	}
	return nil
}

func classifyPackageCandidateError(parentCtx, attemptCtx context.Context, err error) error {
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
	if reason, ok := packageCandidateFailureReason(err); ok {
		return ociartifact.NewCandidateFailure(reason, err)
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		(attemptCtx != nil && errors.Is(attemptCtx.Err(), context.DeadlineExceeded) && errors.Is(err, context.Canceled)) {
		return ociartifact.NewCandidateFailure(ociartifact.CandidateFailureReasonAttemptTimeout, err)
	}
	return err
}

func packageCandidateFailureReason(err error) (ociartifact.CandidateFailureReason, bool) {
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
	if isPackageTLSError(err) {
		return ociartifact.CandidateFailureReasonTLS, true
	}
	var networkError *net.OpError
	if errors.As(err, &networkError) {
		return ociartifact.CandidateFailureReasonTCP, true
	}
	var syscallError *os.SyscallError
	if errors.As(err, &syscallError) && isPackageTCPSystemError(syscallError.Err) {
		return ociartifact.CandidateFailureReasonTCP, true
	}
	if isPackageTCPSystemError(err) {
		return ociartifact.CandidateFailureReasonTCP, true
	}
	return ociartifact.CandidateFailureReasonUnknown, false
}

func isPackageTLSError(err error) bool {
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

func isPackageTCPSystemError(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.ENETUNREACH) ||
		errors.Is(err, syscall.ETIMEDOUT)
}
