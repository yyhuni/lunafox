package enginemanifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
)

const (
	BuiltinLocaleEnglish = "en"
	BuiltinLocaleChinese = "zh"
	CanonicalDocsLocale  = BuiltinLocaleEnglish
)

var requiredLocales = [...]string{BuiltinLocaleEnglish, BuiltinLocaleChinese}

type LocaleResources map[string]map[string]any

const (
	EngineDisplayNameLocalizationKey = "engine.displayName"
	EngineDescriptionLocalizationKey = "engine.description"
)

func executionLocalizationKeys(sections []engineexecution.ConfigSectionDefinition) []string {
	keys := make([]string, 0)
	for _, section := range sections {
		sectionID := strings.TrimSpace(section.ID)
		if sectionID == "" {
			continue
		}
		keys = append(keys, SectionNameLocalizationKey(sectionID), SectionDescriptionLocalizationKey(sectionID))
		for _, param := range section.Params {
			paramKey := strings.TrimSpace(param.Key)
			if paramKey == "" {
				continue
			}
			keys = append(keys, ParamDescriptionLocalizationKey(sectionID, paramKey))
		}
	}
	return keys
}

func SectionNameLocalizationKey(sectionID string) string {
	return fmt.Sprintf("sections.%s.name", strings.TrimSpace(sectionID))
}

func SectionDescriptionLocalizationKey(sectionID string) string {
	return fmt.Sprintf("sections.%s.description", strings.TrimSpace(sectionID))
}

func ParamDescriptionLocalizationKey(sectionID string, paramKey string) string {
	return fmt.Sprintf("sections.%s.params.%s.description", strings.TrimSpace(sectionID), strings.TrimSpace(paramKey))
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func LocaleResourcePath(locale string) (string, error) {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		return "", fmt.Errorf("locale is required")
	}
	if strings.Contains(locale, "/") || strings.Contains(locale, "\\") || strings.Contains(locale, "..") {
		return "", fmt.Errorf("locale %q is invalid", locale)
	}
	return filepath.ToSlash(filepath.Join("locales", locale+".json")), nil
}

// RequiredLocales returns the closed package-v2 locale set in canonical
// archive order.
func RequiredLocales() []string {
	return append([]string(nil), requiredLocales[:]...)
}

// RequiredLocaleResourcePaths derives the closed package-v2 locale paths
// from RequiredLocales and the canonical locale path helper. The returned
// slice is detached from shared state.
func RequiredLocaleResourcePaths() []string {
	locales := RequiredLocales()
	paths := make([]string, 0, len(locales))
	for _, locale := range locales {
		path, err := LocaleResourcePath(locale)
		if err != nil {
			panic(err)
		}
		paths = append(paths, path)
	}
	return paths
}

// DecodeLocaleResource rejects every non-object JSON root explicitly.
func DecodeLocaleResource(payload []byte, source string) (map[string]any, error) {
	if err := rejectDuplicateLocaleResourceFields(payload); err != nil {
		return nil, fmt.Errorf("decode locale resource %q: %w", source, err)
	}
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, fmt.Errorf("decode locale resource %q: %w", source, err)
	}
	resource, ok := value.(map[string]any)
	if !ok || resource == nil {
		return nil, fmt.Errorf("locale resource %q must be an object", source)
	}
	return resource, nil
}

func rejectDuplicateLocaleResourceFields(payload []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	return walkLocaleResourceJSONValue(decoder)
}

func walkLocaleResourceJSONValue(decoder *json.Decoder) error {
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
		seen := map[string]struct{}{}
		for decoder.More() {
			fieldToken, err := decoder.Token()
			if err != nil {
				return err
			}
			field, ok := fieldToken.(string)
			if !ok {
				return fmt.Errorf("json object field name must be a string")
			}
			if _, duplicate := seen[field]; duplicate {
				return fmt.Errorf("duplicate JSON field %q", field)
			}
			seen[field] = struct{}{}
			if err := walkLocaleResourceJSONValue(decoder); err != nil {
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
			if err := walkLocaleResourceJSONValue(decoder); err != nil {
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

func ResolveLocaleString(resource map[string]any, key string) (string, bool) {
	key = strings.TrimSpace(key)
	if key == "" || resource == nil {
		return "", false
	}
	var current any = resource
	for _, part := range strings.Split(key, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			return "", false
		}
		object, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = object[part]
		if !ok {
			return "", false
		}
	}
	value, ok := current.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}
	return value, true
}

// ManifestLocalizationKeys derives locale requirements only from the
// contracts-owned normalized engine.v5 Definition.
func ManifestLocalizationKeys(definition EngineDefinition) ([]string, error) {
	normalized, err := NormalizeEngineDefinition(definition)
	if err != nil {
		return nil, fmt.Errorf("normalize engine definition v2 for locale keys: %w", err)
	}
	keys := []string{
		EngineDisplayNameLocalizationKey,
		EngineDescriptionLocalizationKey,
	}
	keys = append(keys, executionLocalizationKeys(normalized.Execution.ConfigSections)...)
	return uniqueNonEmptyStrings(keys), nil
}

// ValidateLocaleResourceKeys requires every key derived from the exact
// engine.v5 Definition; package.json cannot enumerate or override locale keys.
func ValidateLocaleResourceKeys(definition EngineDefinition, locale string, resource map[string]any) error {
	keys, err := ManifestLocalizationKeys(definition)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if _, ok := ResolveLocaleString(resource, key); !ok {
			return fmt.Errorf("locale %q missing localization key %q for engine %q", locale, key, definition.EngineID)
		}
	}
	return nil
}
