package packagemanifest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

// ComputePackageDigest hashes the exact compressed archive byte stream. It
// does not inspect or normalize expanded package contents because serialization
// differences define different package identities.
func ComputePackageDigest(reader io.Reader) (ociartifact.PackageDigest, int64, error) {
	if reader == nil {
		return "", 0, fmt.Errorf("engine package archive reader is required")
	}
	hash := sha256.New()
	size, err := io.Copy(hash, reader)
	if err != nil {
		return "", size, fmt.Errorf("hash engine package archive: %w", err)
	}
	digest := ociartifact.PackageDigest("sha256:" + hex.EncodeToString(hash.Sum(nil)))
	return digest, size, nil
}
