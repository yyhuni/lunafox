package results

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func DecodeSubdomainItems(itemsJSON []string) ([]Subdomain, error) {
	items := make([]Subdomain, 0, len(itemsJSON))
	for index := range itemsJSON {
		var item Subdomain
		if err := decodeCanonicalResultObject(itemsJSON[index], subdomainResultFields, &item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d] is invalid JSON: %w", index, err)
		}
		if err := Validate(ResultKindAssetSubdomain, item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d]: %w", index, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func DecodeHostPortItems(itemsJSON []string) ([]HostPort, error) {
	items := make([]HostPort, 0, len(itemsJSON))
	for index := range itemsJSON {
		var item HostPort
		if err := decodeCanonicalResultObject(itemsJSON[index], hostPortResultFields, &item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d] is invalid JSON: %w", index, err)
		}
		if err := Validate(ResultKindAssetHostPort, item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d]: %w", index, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func DecodeWebsiteItems(itemsJSON []string) ([]Website, error) {
	items := make([]Website, 0, len(itemsJSON))
	for index := range itemsJSON {
		var item Website
		if err := decodeCanonicalResultObject(itemsJSON[index], websiteResultFields, &item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d] is invalid JSON: %w", index, err)
		}
		if err := Validate(ResultKindAssetWebsite, item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d]: %w", index, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func DecodeEndpointItems(itemsJSON []string) ([]Endpoint, error) {
	items := make([]Endpoint, 0, len(itemsJSON))
	for index := range itemsJSON {
		var item Endpoint
		if err := decodeCanonicalResultObject(itemsJSON[index], endpointResultFields, &item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d] is invalid JSON: %w", index, err)
		}
		if err := Validate(ResultKindAssetEndpoint, item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d]: %w", index, err)
		}
		items = append(items, item)
	}
	return items, nil
}

// DecodeDirectoryItems decodes complete Directory observations without
// changing any accepted value. Explicit presence checks keep JSON null and a
// missing field distinct from legitimate zero and empty-string observations.
func DecodeDirectoryItems(itemsJSON []string) ([]Directory, error) {
	items := make([]Directory, 0, len(itemsJSON))
	for index, payload := range itemsJSON {
		var item Directory
		if err := decodeCanonicalResultObject(payload, directoryResultFields, &item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d] is invalid JSON: %w", index, err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(payload), &fields); err != nil {
			return nil, fmt.Errorf("itemsJson[%d] is invalid JSON: %w", index, err)
		}
		for _, field := range []string{"url", "status", "contentLength", "contentType", "duration"} {
			raw, ok := fields[field]
			if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
				return nil, fmt.Errorf("itemsJson[%d]: %s is required", index, field)
			}
		}
		if err := validateDirectory(item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d]: %w", index, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func DecodeVulnerabilityItems(itemsJSON []string) ([]Vulnerability, error) {
	items := make([]Vulnerability, 0, len(itemsJSON))
	for index, payload := range itemsJSON {
		var item Vulnerability
		if err := decodeCanonicalResultObject(payload, vulnerabilityResultFields, &item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d] is invalid JSON: %w", index, err)
		}
		if err := validateVulnerability(item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d]: %w", index, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func DecodeScreenshotItems(itemsJSON []string) ([]Screenshot, error) {
	items := make([]Screenshot, 0, len(itemsJSON))
	for index := range itemsJSON {
		var item Screenshot
		if err := decodeCanonicalResultObject(itemsJSON[index], screenshotResultFields, &item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d] is invalid JSON: %w", index, err)
		}
		if err := validateScreenshotWireImage(itemsJSON[index], item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d]: %w", index, err)
		}
		// Screenshot items are already on the strict result wire. Ingestion
		// validates the accepted observed URL without rewriting its bytes.
		if err := ValidateScreenshot(item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d]: %w", index, err)
		}
		items = append(items, item)
	}
	return items, nil
}

// DecodeWebsiteTechnologyItems decodes the closed, complete technology
// observation contract. Missing/null fields are rejected explicitly because
// encoding/json would otherwise collapse them into the same zero values as a
// trusted empty observation.
func DecodeWebsiteTechnologyItems(itemsJSON []string) ([]WebsiteTechnology, error) {
	items := make([]WebsiteTechnology, 0, len(itemsJSON))
	for index, payload := range itemsJSON {
		var item WebsiteTechnology
		if err := decodeCanonicalResultObject(payload, websiteTechnologyResultFields, &item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d] is invalid JSON: %w", index, err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(payload), &fields); err != nil {
			return nil, fmt.Errorf("itemsJson[%d] is invalid JSON: %w", index, err)
		}
		for _, field := range []string{"url", "tech"} {
			raw, ok := fields[field]
			if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
				return nil, fmt.Errorf("itemsJson[%d]: %s is required", index, field)
			}
		}
		if err := validateWebsiteTechnology(item); err != nil {
			return nil, fmt.Errorf("itemsJson[%d]: %w", index, err)
		}
		items = append(items, item)
	}
	return items, nil
}
