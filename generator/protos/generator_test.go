package protos

import (
	"path/filepath"
	"testing"
)

func TestExtractRelativePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single level",
			input:    filepath.Join("capabilities", "internal", "foo", "v1", "foo.proto"),
			expected: "foo.proto",
		},
		{
			name:     "nested path",
			input:    filepath.Join("capabilities", "internal", "foo", "bar", "v1", "foo.proto"),
			expected: filepath.Join("bar", "foo.proto"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractRelativePath(tc.input)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
