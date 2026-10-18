package engineinstall

import (
	"context"
	"fmt"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/ocisignature"
)

// SigstoreKeylessPolicy and verifier aliases preserve the Server installation
// boundary while the cryptographic implementation is shared with installer.
type SigstoreKeylessPolicy = ocisignature.KeylessPolicy
type SigstoreKeylessVerifier = ocisignature.KeylessVerifier

type DigestReferenceSignatureVerifier interface {
	VerifyReference(context.Context, ociartifact.DigestReference) error
}

type DigestReferenceSignatureTransportVerifier = ocisignature.DigestReferenceSignatureTransportVerifier

func NewSigstoreKeylessVerifier(policy SigstoreKeylessPolicy, trustedRoot root.TrustedMaterial) (*SigstoreKeylessVerifier, error) {
	return ocisignature.NewKeylessVerifier(policy, trustedRoot)
}

func NewProductionSigstoreKeylessVerifier(policy SigstoreKeylessPolicy) (*SigstoreKeylessVerifier, error) {
	return ocisignature.NewProductionKeylessVerifier(policy)
}

// WithSignatureRegistry fetches signature material from the configured mirror
// while verifying the original signed repository and digest identity.
func WithSignatureRegistry(verifier DigestReferenceSignatureVerifier, registry string) (DigestReferenceSignatureVerifier, error) {
	if registry == "" {
		return verifier, nil
	}
	if err := ociartifact.ValidateRegistryAuthority(registry); err != nil {
		return nil, err
	}
	transportVerifier, ok := verifier.(DigestReferenceSignatureTransportVerifier)
	if !ok {
		return nil, fmt.Errorf("signature verifier does not support Registry transport")
	}
	return registrySignatureVerifier{direct: verifier, verifier: transportVerifier, registry: registry}, nil
}

type registrySignatureVerifier struct {
	direct   DigestReferenceSignatureVerifier
	verifier DigestReferenceSignatureTransportVerifier
	registry string
}

func (verifier registrySignatureVerifier) VerifyReference(ctx context.Context, identity ociartifact.DigestReference) error {
	if identity.Registry == verifier.registry {
		return verifier.direct.VerifyReference(ctx, identity)
	}
	transport := identity
	transport.Registry = verifier.registry
	return verifier.verifier.VerifyReferenceWithTransport(ctx, identity, transport)
}
