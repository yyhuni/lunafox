package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewWordlist(t *testing.T) {
	t.Run("empty name", func(t *testing.T) {
		_, err := NewWordlist(" ", "desc")
		if !errors.Is(err, ErrWordlistNameEmpty) {
			t.Fatalf("expected ErrWordlistNameEmpty, got %v", err)
		}
	})

	t.Run("name too long", func(t *testing.T) {
		name := strings.Repeat("a", MaxWordlistNameLength+1)
		_, err := NewWordlist(name, "desc")
		if !errors.Is(err, ErrWordlistNameTooLong) {
			t.Fatalf("expected ErrWordlistNameTooLong, got %v", err)
		}
	})

	t.Run("invalid control character", func(t *testing.T) {
		_, err := NewWordlist("a\tb", "desc")
		if !errors.Is(err, ErrWordlistNameInvalid) {
			t.Fatalf("expected ErrWordlistNameInvalid, got %v", err)
		}
	})

	t.Run("description normalization", func(t *testing.T) {
		description := "  hello\nworld\t!  "
		wordlist, err := NewWordlist(" dict ", description)
		if err != nil {
			t.Fatalf("new wordlist failed: %v", err)
		}
		if wordlist.Name != "dict" {
			t.Fatalf("expected normalized name dict, got %q", wordlist.Name)
		}
		if strings.ContainsRune(wordlist.Description, '\n') || strings.ContainsRune(wordlist.Description, '\t') {
			t.Fatalf("description should remove control chars, got %q", wordlist.Description)
		}
	})
}

func TestCountWordlistContentLines(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		expected int
	}{
		{name: "empty", content: "", expected: 0},
		{name: "single_no_newline", content: "a", expected: 1},
		{name: "single_with_newline", content: "a\n", expected: 1},
		{name: "two_with_last_newline", content: "a\nb\n", expected: 2},
		{name: "two_without_last_newline", content: "a\nb", expected: 2},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := CountWordlistContentLines(testCase.content)
			if actual != testCase.expected {
				t.Fatalf("unexpected line count want=%d got=%d", testCase.expected, actual)
			}
		})
	}
}
