package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewWordlist(t *testing.T) {
	t.Run("empty file name", func(t *testing.T) {
		_, err := NewWordlist(" ", "desc")
		if !errors.Is(err, ErrWordlistFileNameEmpty) {
			t.Fatalf("expected ErrWordlistFileNameEmpty, got %v", err)
		}
	})

	t.Run("file name too long", func(t *testing.T) {
		fileName := strings.Repeat("a", MaxWordlistFileNameLength+1)
		_, err := NewWordlist(fileName, "desc")
		if !errors.Is(err, ErrWordlistFileNameTooLong) {
			t.Fatalf("expected ErrWordlistFileNameTooLong, got %v", err)
		}
	})

	t.Run("invalid control character", func(t *testing.T) {
		_, err := NewWordlist("a\tb", "desc")
		if !errors.Is(err, ErrWordlistFileNameInvalid) {
			t.Fatalf("expected ErrWordlistFileNameInvalid, got %v", err)
		}
	})

	t.Run("invalid path syntax", func(t *testing.T) {
		cases := []string{"nested/subs.txt", `nested\subs.txt`, ".", ".."}
		for _, fileName := range cases {
			_, err := NewWordlist(fileName, "desc")
			if !errors.Is(err, ErrWordlistFileNameInvalid) {
				t.Fatalf("expected ErrWordlistFileNameInvalid for %q, got %v", fileName, err)
			}
		}
	})

	t.Run("description normalization", func(t *testing.T) {
		description := "  hello\nworld\t!  "
		wordlist, err := NewWordlist(" dict ", description)
		if err != nil {
			t.Fatalf("new wordlist failed: %v", err)
		}
		if wordlist.FileName != "dict" {
			t.Fatalf("expected normalized fileName dict, got %q", wordlist.FileName)
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
