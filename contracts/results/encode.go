package results

import (
	"encoding/json"
	"fmt"
)

// EncodeSubdomain validates and serializes one result item without changing
// the Engine-selected observation.
func EncodeSubdomain(item Subdomain) (string, error) {
	if err := Validate(ResultKindAssetSubdomain, item); err != nil {
		return "", err
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("marshal subdomain result: %w", err)
	}
	return string(payload), nil
}

// EncodeHostPort validates and serializes one result item without changing
// the Engine-selected observation.
func EncodeHostPort(item HostPort) (string, error) {
	if err := Validate(ResultKindAssetHostPort, item); err != nil {
		return "", err
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("marshal host-port result: %w", err)
	}
	return string(payload), nil
}

// EncodeWebsite validates and serializes one website item. It preserves the
// accepted URL bytes while preparing only non-URL evidence and Host assertion.
func EncodeWebsite(item Website) (string, error) {
	if err := Validate(ResultKindAssetWebsite, item); err != nil {
		return "", err
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("marshal website result: %w", err)
	}
	return string(payload), nil
}

// EncodeWebsiteTechnology emits the strict current-only technology result
// without routing through the complete Website observation normalizer.
func EncodeWebsiteTechnology(item WebsiteTechnology) (string, error) {
	if err := validateWebsiteTechnology(item); err != nil {
		return "", err
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("marshal website technology result: %w", err)
	}
	return string(payload), nil
}

func EncodeEndpoint(item Endpoint) (string, error) {
	if err := Validate(ResultKindAssetEndpoint, item); err != nil {
		return "", err
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("marshal endpoint result: %w", err)
	}
	return string(payload), nil
}

func EncodeScreenshot(item Screenshot) (string, error) {
	if err := Validate(ResultKindAssetScreenshot, item); err != nil {
		return "", err
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("marshal screenshot result: %w", err)
	}
	return string(payload), nil
}

// EncodeDirectory preserves the complete observation exactly, including its
// accepted raw URL identity.
func EncodeDirectory(item Directory) (string, error) {
	if err := validateDirectory(item); err != nil {
		return "", err
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("marshal directory result: %w", err)
	}
	return string(payload), nil
}
