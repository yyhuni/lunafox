package enginemanifest

import (
	"reflect"
	"strings"
	"testing"
)

func TestDecodeLocaleResourceRejectsDuplicateObjectFields(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		field   string
	}{
		{
			name:    "root field",
			payload: `{"engine":{"displayName":"First"},"engine":{"displayName":"Second"}}`,
			field:   "engine",
		},
		{
			name:    "nested field",
			payload: `{"engine":{"displayName":"First","displayName":"Second"}}`,
			field:   "displayName",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeLocaleResource([]byte(test.payload), "locales/en.json")
			if err == nil || !strings.Contains(err.Error(), `duplicate JSON field "`+test.field+`"`) {
				t.Fatalf("DecodeLocaleResource() error = %v, want duplicate field rejection", err)
			}
		})
	}
}

func TestManifestLocalizationKeysAreDerivedFromNormalizedDefinition(t *testing.T) {
	definition, err := DecodeEngineDefinition(validRootManifestPayload(), "engine.json")
	if err != nil {
		t.Fatalf("DecodeEngineDefinition() error = %v", err)
	}

	got, err := ManifestLocalizationKeys(definition)
	if err != nil {
		t.Fatalf("ManifestLocalizationKeys() error = %v", err)
	}
	want := []string{
		"engine.description",
		"engine.displayName",
		"sections.scan.description",
		"sections.scan.name",
		"sections.scan.params.wordlist.description",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ManifestLocalizationKeys() = %#v, want %#v", got, want)
	}
}
