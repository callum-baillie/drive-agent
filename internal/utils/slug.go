package utils

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	nonAlphaNum = regexp.MustCompile(`[^a-z0-9-]+`)
	multiDash   = regexp.MustCompile(`-{2,}`)
)

// Slugify converts a string into a URL/filesystem-safe slug.
func Slugify(s string) string {
	// Normalize unicode
	s = norm.NFC.String(s)

	// Lowercase
	s = strings.ToLower(s)

	// Replace common separators with dashes
	s = strings.NewReplacer(
		" ", "-",
		"_", "-",
		".", "-",
		"/", "-",
		"'", "",
		"\"", "",
	).Replace(s)

	// Remove non-alphanumeric characters (except dashes)
	s = nonAlphaNum.ReplaceAllString(s, "")

	// Collapse multiple dashes
	s = multiDash.ReplaceAllString(s, "-")

	// Trim leading/trailing dashes
	s = strings.Trim(s, "-")

	return s
}

// IsValidSlug checks if a string is already a valid slug.
func IsValidSlug(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLower(r) && !unicode.IsDigit(r) && r != '-' {
			return false
		}
	}
	if strings.HasPrefix(s, "-") || strings.HasSuffix(s, "-") {
		return false
	}
	if strings.Contains(s, "--") {
		return false
	}
	return true
}
