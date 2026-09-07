package engineinstall

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/ocidistribution"
	"github.com/yyhuni/lunafox/contracts/ocisignature"
)

type packageLayerConsumerPuller interface {
	ConsumePackageLayer(context.Context, ociartifact.ArtifactCandidates, PackageLayerConsumer) (ConsumedPackageLayer, error)
}

type packageCacheStager interface {
	StageAndPromotePackage(
		context.Context,
		io.Reader,
		ociartifact.PackageDigest,
		int64,
		PackagePromotionValidator,
	) (CachedEnginePackage, error)
}

type runtimeImageIndexVerifier interface {
	Verify(context.Context, string, []string) (VerifiedRuntimeImageIndex, error)
}

// VerifiedEngineInstallation is the repository-neutral output of the v2
// installer. It intentionally omits derivable artifact-manifest
// digest, Runtime Image digest, selected Runtime Image location, and cache path
// sibling fields; registration may commit only these package facts.
type VerifiedEngineInstallation struct {
	EngineID                 string
	Publisher                string
	PackageVersion           string
	ArtifactRef              string
	PackageDigest            ociartifact.PackageDigest
	EngineManifestProjection json.RawMessage
}

// EnginePackageInstaller verifies one complete package-to-runtime chain but
// does not write the engine repository.
type EnginePackageInstaller struct {
	puller               packageLayerConsumerPuller
	cache                packageCacheStager
	runtimeImageVerifier runtimeImageIndexVerifier
	signatureVerifier    DigestReferenceSignatureVerifier
}

func NewEnginePackageInstaller(
	puller packageLayerConsumerPuller,
	cache packageCacheStager,
	runtimeImageVerifier runtimeImageIndexVerifier,
	signatureVerifiers ...DigestReferenceSignatureVerifier,
) (*EnginePackageInstaller, error) {
	if puller == nil || cache == nil || runtimeImageVerifier == nil {
		return nil, fmt.Errorf("Engine Package v2 puller, cache, and Runtime Image verifier are required")
	}
	if len(signatureVerifiers) > 1 {
		return nil, fmt.Errorf("at most one Engine Package signature verifier is supported")
	}
	var signatureVerifier DigestReferenceSignatureVerifier
	if len(signatureVerifiers) == 1 {
		signatureVerifier = signatureVerifiers[0]
	}
	return &EnginePackageInstaller{
		puller:               puller,
		cache:                cache,
		runtimeImageVerifier: runtimeImageVerifier,
		signatureVerifier:    signatureVerifier,
	}, nil
}

// InstallInventory verifies required packages in declared order. A later
// package failure may leave already verified content-addressed cache entries,
// but this repository-neutral lane cannot make any package catalog-visible.
func (installer *EnginePackageInstaller) InstallInventory(
	ctx context.Context,
	inventory *Inventory,
) ([]VerifiedEngineInstallation, error) {
	if installer == nil {
		return nil, fmt.Errorf("Engine Package v2 installer is required")
	}
	if ctx == nil {
		return nil, fmt.Errorf("Engine Package v2 installation context is required")
	}
	if inventory == nil || len(inventory.EnginePackages) == 0 {
		return nil, fmt.Errorf("Engine Package v2 inventory cannot be empty")
	}

	verified := make([]VerifiedEngineInstallation, 0, len(inventory.EnginePackages))
	for index, item := range inventory.EnginePackages {
		installation, err := installer.Install(ctx, item.Candidates)
		if err != nil {
			return nil, fmt.Errorf("install Engine Package v2 inventory item %d: %w", index, err)
		}
		verified = append(verified, installation)
	}
	return verified, nil
}

// Install verifies a package whose Engine identity is discovered only after its
// immutable bytes have passed the package contract. Callers must not supply an
// out-of-band identity that can override the package declaration.
func (installer *EnginePackageInstaller) Install(
	ctx context.Context,
	candidates ociartifact.ArtifactCandidates,
) (VerifiedEngineInstallation, error) {
	if installer == nil {
		return VerifiedEngineInstallation{}, fmt.Errorf("Engine Package v2 installer is required")
	}
	if ctx == nil {
		return VerifiedEngineInstallation{}, fmt.Errorf("Engine Package v2 installation context is required")
	}
	validatedCandidates, err := validateArtifactCandidates(candidates)
	if err != nil {
		return VerifiedEngineInstallation{}, err
	}
	if installer.signatureVerifier != nil && len(validatedCandidates.References) != 3 {
		verified := make([]ociartifact.EnginePackageArtifactReference, 0, len(validatedCandidates.References))
		var lastTransport error
		for _, reference := range validatedCandidates.References {
			if err := installer.signatureVerifier.VerifyReference(ctx, reference.DigestReference()); err != nil {
				if ocisignature.IsTransportError(err) {
					lastTransport = err
					continue
				}
				return VerifiedEngineInstallation{}, fmt.Errorf("verify Engine Package signature %q: %w", reference.String(), err)
			}
			verified = append(verified, reference)
		}
		if len(verified) == 0 {
			return VerifiedEngineInstallation{}, fmt.Errorf("verify Engine Package signatures: %w", lastTransport)
		}
		validatedCandidates.References = verified
	}
	if len(validatedCandidates.References) == 3 {
		references := make([]string, len(validatedCandidates.References))
		for index, reference := range validatedCandidates.References {
			references[index] = reference.String()
		}
		acceleration, err := ocidistribution.ParseCloudflareAcceleration(references)
		if err != nil {
			return VerifiedEngineInstallation{}, fmt.Errorf("parse Engine Package Cloudflare acceleration: %w", err)
		}
		if installer.signatureVerifier == nil {
			return VerifiedEngineInstallation{}, fmt.Errorf("Cloudflare Engine Package acceleration requires a GHCR signature verifier")
		}
		if err := installer.signatureVerifier.VerifyReference(ctx, acceleration.SignatureReference); err != nil {
			return VerifiedEngineInstallation{}, fmt.Errorf("verify Engine Package GHCR signature %q: %w", acceleration.SignatureReference.String(), err)
		}
	}

	var cached CachedEnginePackage
	var projection json.RawMessage
	consumed, err := installer.puller.ConsumePackageLayer(ctx, validatedCandidates, func(
		attemptCtx context.Context,
		layer PulledPackageLayer,
	) error {
		staged, err := installer.cache.StageAndPromotePackage(
			attemptCtx,
			layer.Reader,
			layer.PackageDigest,
			layer.Size,
			func(validationCtx context.Context, layout enginepackagecatalog.EnginePackageLayout) error {
				definition := layout.Definition
				if strings.TrimSpace(definition.EngineDefinition.Publisher) == "" {
					return fmt.Errorf("verified Engine Package v2 publisher is required")
				}
				if _, err := installer.runtimeImageVerifier.Verify(
					validationCtx,
					definition.EngineDefinition.EngineID,
					definition.PackageManifest.RuntimeImage.Refs,
				); err != nil {
					if contextErr := validationCtx.Err(); contextErr != nil {
						return contextErr
					}
					// Runtime Image candidate fallback is already complete inside its
					// verifier. Do not let that typed error advance to another package
					// artifact location and obscure package-dependent validation.
					return nonCandidateValidationError("verify package-bound Runtime Image", err)
				}
				encoded, err := json.Marshal(definition.EngineDefinition)
				if err != nil {
					return fmt.Errorf("encode verified engine.v5 manifest projection: %w", err)
				}
				projection = append(projection[:0], encoded...)
				return nil
			},
		)
		if err != nil {
			return err
		}
		cached = staged
		return nil
	})
	if err != nil {
		return VerifiedEngineInstallation{}, err
	}
	if cached.PackageDigest == "" || cached.PackageDigest != consumed.PackageDigest {
		return VerifiedEngineInstallation{}, fmt.Errorf(
			"Engine Package v2 cache identity mismatch: cache %q, consumed layer %q",
			cached.PackageDigest,
			consumed.PackageDigest,
		)
	}
	definition := cached.Layout.Definition
	if len(projection) == 0 {
		return VerifiedEngineInstallation{}, fmt.Errorf("verified engine.v5 manifest projection is missing")
	}
	return VerifiedEngineInstallation{
		EngineID:                 definition.EngineDefinition.EngineID,
		Publisher:                definition.EngineDefinition.Publisher,
		PackageVersion:           definition.PackageManifest.EngineVersion,
		ArtifactRef:              consumed.Reference.String(),
		PackageDigest:            consumed.PackageDigest,
		EngineManifestProjection: append(json.RawMessage(nil), projection...),
	}, nil
}

func validateArtifactCandidates(candidates ociartifact.ArtifactCandidates) (ociartifact.ArtifactCandidates, error) {
	values := make([]string, len(candidates.References))
	for index, reference := range candidates.References {
		values[index] = reference.String()
	}
	validated, err := ociartifact.ParseArtifactCandidates(values)
	if err != nil {
		return ociartifact.ArtifactCandidates{}, fmt.Errorf("validate Engine Package v2 artifact candidates: %w", err)
	}
	if validated.ArtifactManifestDigest != candidates.ArtifactManifestDigest {
		return ociartifact.ArtifactCandidates{}, fmt.Errorf(
			"Engine Package v2 artifact candidate digest mismatch: got %q want %q",
			candidates.ArtifactManifestDigest,
			validated.ArtifactManifestDigest,
		)
	}
	return validated, nil
}

type nonCandidateValidationFailure struct {
	operation string
	cause     error
}

func nonCandidateValidationError(operation string, cause error) error {
	return &nonCandidateValidationFailure{operation: operation, cause: cause}
}

func (failure *nonCandidateValidationFailure) Error() string {
	if failure == nil || failure.cause == nil {
		return "Engine Package v2 validation failed"
	}
	return failure.operation + ": " + failure.cause.Error()
}
