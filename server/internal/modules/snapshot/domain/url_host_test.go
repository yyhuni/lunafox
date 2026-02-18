package domain

import "testing"

func TestExtractHostFromURL(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		expected string
	}{
		{name: "https", rawURL: "https://example.com/path", expected: "example.com"},
		{name: "http_with_port", rawURL: "http://example.com:8080/a", expected: "example.com:8080"},
		{name: "invalid", rawURL: "://bad-url", expected: ""},
		{name: "empty", rawURL: "", expected: ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			actual := ExtractHostFromURL(testCase.rawURL)
			if actual != testCase.expected {
				t.Fatalf("unexpected host want=%q got=%q", testCase.expected, actual)
			}
		})
	}
}
