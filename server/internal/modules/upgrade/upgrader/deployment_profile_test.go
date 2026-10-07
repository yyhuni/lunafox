package upgrader

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveComposeProfileAcceptsDocumentedInterpolation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		mode      string
		profiles  string
		want      string
		wantError string
	}{
		{name: "template embedded", mode: "embedded", profiles: "${DATABASE_MODE:-embedded}", want: "embedded"},
		{name: "template follows external mode", mode: "external", profiles: "${DATABASE_MODE:-embedded}", want: "external"},
		{name: "set mode ignores unused default", mode: "embedded", profiles: "${DATABASE_MODE:-external}", want: "embedded"},
		{name: "empty mode uses embedded default", profiles: "${DATABASE_MODE:-embedded}", want: "embedded"},
		{name: "direct reference", mode: "external", profiles: "${DATABASE_MODE}", want: "external"},
		{name: "literal match", mode: "embedded", profiles: "embedded", want: "embedded"},
		{name: "empty profiles follow mode", mode: "external", want: "external"},
		{name: "real mismatch", mode: "embedded", profiles: "external", wantError: "COMPOSE_PROFILES must match DATABASE_MODE"},
		{name: "empty mode with external default", profiles: "${DATABASE_MODE:-external}", wantError: "COMPOSE_PROFILES must match DATABASE_MODE"},
		{name: "invalid mode", mode: "sidecar", profiles: "${DATABASE_MODE:-embedded}", wantError: "DATABASE_MODE must be embedded or external"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveComposeProfile(test.mode, test.profiles)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("resolveComposeProfile() = %q, %v", got, err)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("resolveComposeProfile() = %q, %v; want %s", got, err, test.want)
			}
		})
	}
}

func TestDeploymentPreheatProfileReadsDocumentedEnvFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	contents := "DATABASE_MODE=embedded\nCOMPOSE_PROFILES=${DATABASE_MODE:-embedded}\n"
	if err := os.WriteFile(filepath.Join(root, publicEnvFile), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	profile, err := deploymentPreheatProfile(root)
	if err != nil || profile != "embedded" {
		t.Fatalf("deploymentPreheatProfile() = %q, %v", profile, err)
	}
	parsed, err := deploymentPreheatProfileFromEnvironment([]byte(contents))
	if err != nil || parsed != "embedded" {
		t.Fatalf("deploymentPreheatProfileFromEnvironment() = %q, %v", parsed, err)
	}
}

func TestPreheatValidationDiagnosticKeepsSafeReason(t *testing.T) {
	t.Parallel()
	got := preheatValidationDiagnostic(errors.New("COMPOSE_PROFILES must match DATABASE_MODE"))
	if got != "preheat manifest validation failed: COMPOSE_PROFILES must match DATABASE_MODE" {
		t.Fatalf("diagnostic = %q", got)
	}
	if got := preheatValidationDiagnostic(errors.New("read /deployment/.env")); got != "preheat manifest validation failed" {
		t.Fatalf("path diagnostic = %q", got)
	}
}
