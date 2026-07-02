package slug

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)
	multiDash       = regexp.MustCompile(`-+`)
)

// Generate creates a URL-friendly slug from a string.
func Generate(s string) string {
	// Normalize unicode
	s = norm.NFKD.String(s)

	// Remove non-ASCII characters
	var b strings.Builder
	for _, r := range s {
		if r < unicode.MaxASCII {
			b.WriteRune(r)
		}
	}
	s = b.String()

	// Lowercase
	s = strings.ToLower(s)

	// Replace non-alphanumeric with dashes
	s = nonAlphanumeric.ReplaceAllString(s, "-")

	// Collapse multiple dashes
	s = multiDash.ReplaceAllString(s, "-")

	// Trim leading/trailing dashes
	s = strings.Trim(s, "-")

	return s
}
