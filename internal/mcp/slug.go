package mcp

import (
	"regexp"
	"strings"
)

var slugMultiDash = regexp.MustCompile(`-{2,}`)

// slug converts a title to a filesystem-safe identifier.
// Must match the behavior of internal/cli/slug.go.
func slug(title string) string {
	s := strings.ToLower(title)
	s = strings.NewReplacer(
		" ", "-", "_", "-", "/", "-", "\\", "-",
		".", "-", ":", "-", ",", "-",
		"'", "", "\"", "", "(", "", ")", "", "`", "",
	).Replace(s)
	s = slugMultiDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
