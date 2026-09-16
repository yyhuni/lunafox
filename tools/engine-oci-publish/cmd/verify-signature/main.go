// Command verify-signature checks published OCI signatures through the same
// anonymous verifier used by Server installation, rather than Cosign alone.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/ocisignature"
)

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: verify-signature registry/repository@sha256:digest")
	}
	ref, err := ociartifact.ParseDigestReference(args[0])
	if err != nil {
		return err
	}
	verifier, err := ocisignature.NewProductionKeylessVerifier(ocisignature.KeylessPolicy{
		Repository: "yyhuni/lunafox", Workflow: ".github/workflows/public-validate.yml", RefPattern: "refs/heads/main",
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return verifyPublishedSignature(ctx, func() error { return verifier.VerifyReference(ctx, ref) }, func(ctx context.Context) error {
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	})
}

// Registry referrer indexes can lag a successful signature upload. Only an
// empty listing is retryable; malformed, untrusted or mismatched bundles stop.
func verifyPublishedSignature(ctx context.Context, verify func() error, wait func(context.Context) error) error {
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := verify()
		var missing *ocisignature.MissingBundleError
		if err == nil || !errors.As(err, &missing) || attempt >= 29 {
			return err
		}
		if err := wait(ctx); err != nil {
			return err
		}
	}
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
