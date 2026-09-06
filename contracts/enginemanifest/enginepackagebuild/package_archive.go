package enginepackagebuild

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

// WriteEnginePackageArchive validates the closed four-file package and writes
// its deterministic tar+gzip serialization in canonical path order. It returns
// the identity of the exact compressed bytes written, never a file-level or
// expanded-content digest.
func WriteEnginePackageArchive(writer io.Writer, entries []enginepackagecatalog.PackageLayoutEntry) (ociartifact.PackageDigest, int64, error) {
	if writer == nil {
		return "", 0, fmt.Errorf("engine package v2 archive writer is required")
	}
	if _, err := enginepackagecatalog.DecodeEnginePackageLayout(entries, "engine package v2 archive input"); err != nil {
		return "", 0, err
	}

	payloads := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		payloads[entry.Path] = append([]byte(nil), entry.Payload...)
	}
	hash := sha256.New()
	counter := &archiveByteCounter{}
	output := io.MultiWriter(writer, hash, counter)
	if err := writeDeterministicTarGzipInOrder(output, payloads, enginepackagecatalog.RequiredEnginePackagePaths()); err != nil {
		return "", counter.size, err
	}
	digest := ociartifact.PackageDigest("sha256:" + hex.EncodeToString(hash.Sum(nil)))
	return digest, counter.size, nil
}

type archiveByteCounter struct {
	size int64
}

func (counter *archiveByteCounter) Write(payload []byte) (int, error) {
	counter.size += int64(len(payload))
	return len(payload), nil
}
