package utils

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"My Project", "my-project"},
		{"Jasper's Classroom", "jaspers-classroom"},
		{"roamar", "roamar"},
		{"  spaces  ", "spaces"},
		{"UPPER_CASE", "upper-case"},
		{"dots.and.more", "dots-and-more"},
		{"special!@#chars", "specialchars"},
		{"multiple---dashes", "multiple-dashes"},
		{"", ""},
		{"123-numbers", "123-numbers"},
		{"a/b/c", "a-b-c"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidSlug(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"hello-world", true},
		{"roamar", true},
		{"my-project-123", true},
		{"", false},
		{"Hello", false},
		{"-leading", false},
		{"trailing-", false},
		{"double--dash", false},
		{"has space", false},
		{"has_underscore", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsValidSlug(tt.input)
			if got != tt.want {
				t.Errorf("IsValidSlug(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
