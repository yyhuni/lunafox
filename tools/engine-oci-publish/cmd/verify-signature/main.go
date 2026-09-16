// Command verify-signature checks published OCI signatures through the same
// anonymous verifier used by Server installation, rather than Cosign alone.
package main

import (
	"context"
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
	return verifier.VerifyReference(ctx, ref)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
