package cli

import (
	"regexp"
	"strings"
)

var slugMultiDash = regexp.MustCompile(`-{2,}`)

// slug converts a title to a filesystem-safe ID.
// "Настроить Caddy reverse proxy" → "настроить-caddy-reverse-proxy"
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
