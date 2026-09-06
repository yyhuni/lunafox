package engineinstall

import (
	"context"

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

func NewSigstoreKeylessVerifier(policy SigstoreKeylessPolicy, trustedRoot root.TrustedMaterial) (*SigstoreKeylessVerifier, error) {
	return ocisignature.NewKeylessVerifier(policy, trustedRoot)
}

func NewProductionSigstoreKeylessVerifier(policy SigstoreKeylessPolicy) (*SigstoreKeylessVerifier, error) {
	return ocisignature.NewProductionKeylessVerifier(policy)
}
