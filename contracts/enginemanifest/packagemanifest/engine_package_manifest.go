package packagemanifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/yyhuni/lunafox/contracts/enginemanifest/runtimeimage"
)

const (
	// PackageFormatVersion identifies the active package.json contract.
	PackageFormatVersion = "lunafox.engine-package.v2"
	// EnginePackageArchiveSuffix is the canonical published package archive suffix.
	EnginePackageArchiveSuffix = ".lfengine.tar.gz"
)

// PackageManifest binds one Engine release to one ordered Runtime Image
// candidate list.
type PackageManifest struct {
	PackageFormatVersion string       `json:"packageFormatVersion"`
	EngineID             string       `json:"engineId"`
	EngineVersion        string       `json:"engineVersion"`
	RuntimeImage         RuntimeImage `json:"runtimeImage"`
}

// RuntimeImage contains ordered Registry locations for one immutable image
// digest. Candidate order is preserved as package-authored availability order.
type RuntimeImage struct {
	Refs []string `json:"refs"`
}

// DecodePackageManifest decodes only package v2 fields and rejects unknown
// fields or trailing JSON. It has no package v1 compatibility fallback.
func DecodePackageManifest(payload []byte, source string) (PackageManifest, error) {
	if err := rejectDuplicatePackageManifestJSONFields(payload); err != nil {
		return PackageManifest{}, fmt.Errorf("decode package manifest v2 %q: %w", source, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var manifest PackageManifest
	if err := decoder.Decode(&manifest); err != nil {
		return PackageManifest{}, fmt.Errorf("decode package manifest v2 %q: %w", source, err)
	}
	if err := consumePackageManifestJSONEOF(decoder); err != nil {
		return PackageManifest{}, fmt.Errorf("decode package manifest v2 %q: %w", source, err)
	}
	return manifest, nil
}

func consumePackageManifestJSONEOF(decoder *json.Decoder) error {
	if _, err := decoder.Token(); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return fmt.Errorf("unexpected trailing JSON content")
}

func rejectDuplicatePackageManifestJSONFields(payload []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	return walkPackageManifestJSONValue(decoder)
}

func walkPackageManifestJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}

	switch delimiter {
	case '{':
		seenFields := map[string]struct{}{}
		for decoder.More() {
			fieldToken, err := decoder.Token()
			if err != nil {
				return err
			}
			field, ok := fieldToken.(string)
			if !ok {
				return fmt.Errorf("json object field name must be a string")
			}
			if _, exists := seenFields[field]; exists {
				return fmt.Errorf("duplicate JSON field %q", field)
			}
			seenFields[field] = struct{}{}
			if err := walkPackageManifestJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return fmt.Errorf("unexpected JSON delimiter %q", closing)
		}
	case '[':
		for decoder.More() {
			if err := walkPackageManifestJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return fmt.Errorf("unexpected JSON delimiter %q", closing)
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

// ValidatePackageManifest validates package-local identity relationships.
// Remote descriptor, signature, media-type, and platform verification belongs
// to OCI-aware publisher and installer boundaries.
func ValidatePackageManifest(manifest PackageManifest) error {
	if strings.TrimSpace(manifest.PackageFormatVersion) == "" {
		return fmt.Errorf("packageFormatVersion is required")
	}
	if manifest.PackageFormatVersion != PackageFormatVersion {
		return fmt.Errorf("unsupported packageFormatVersion %q", manifest.PackageFormatVersion)
	}
	engineID := strings.TrimSpace(manifest.EngineID)
	if engineID == "" {
		return fmt.Errorf("engineId is required")
	}
	if engineID != manifest.EngineID {
		return fmt.Errorf("invalid engineId %q", manifest.EngineID)
	}
	if strings.TrimSpace(manifest.EngineVersion) == "" {
		return fmt.Errorf("engineVersion is required")
	}
	if manifest.EngineVersion != strings.TrimSpace(manifest.EngineVersion) {
		return fmt.Errorf("engineVersion must be canonical")
	}
	if len(manifest.RuntimeImage.Refs) == 0 {
		return fmt.Errorf("runtimeImage.refs is required")
	}
	if _, err := runtimeimage.ParseCandidates(manifest.RuntimeImage.Refs); err != nil {
		return fmt.Errorf("runtimeImage.refs: %w", err)
	}
	return nil
}
