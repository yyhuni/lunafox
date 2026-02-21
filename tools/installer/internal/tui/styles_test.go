package tui

import "testing"

func TestShouldUseColorDisabledByNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("CLICOLOR", "")
	t.Setenv("TERM", "xterm-256color")
	if shouldUseColor() {
		t.Fatalf("expected color disabled when NO_COLOR is set")
	}
}

func TestShouldUseColorDisabledByCliColorZero(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "0")
	t.Setenv("TERM", "xterm-256color")
	if shouldUseColor() {
		t.Fatalf("expected color disabled when CLICOLOR=0")
	}
}

func TestShouldUseColorDisabledForDumbTerm(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "")
	t.Setenv("TERM", "dumb")
	if shouldUseColor() {
		t.Fatalf("expected color disabled for TERM=dumb")
	}
}
