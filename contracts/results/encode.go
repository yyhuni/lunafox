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

// EncodeWebsiteV2 validates and serializes the versioned observed Website
// contract. Response evidence finalization is owned by generated Engine code.
func EncodeWebsiteV2(item WebsiteV2) (string, error) {
	if err := Validate(ResultKindAssetWebsiteV2, item); err != nil {
		return "", err
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("marshal website v2 result: %w", err)
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

// EncodeEndpointV2 validates and serializes the versioned observed Endpoint
// contract. Response evidence finalization is owned by generated Engine code.
func EncodeEndpointV2(item EndpointV2) (string, error) {
	if err := Validate(ResultKindAssetEndpointV2, item); err != nil {
		return "", err
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("marshal endpoint v2 result: %w", err)
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
