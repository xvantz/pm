package cli

import (
	"regexp"
	"strings"
)

// slug converts a title to a filesystem-safe ID.
// "Настроить Caddy reverse proxy" → "настроить-caddy-reverse-proxy"
func slug(title string) string {
	s := strings.ToLower(title)
	s = strings.NewReplacer(
		" ", "-", "_", "-", "/", "-", "\\", "-",
		".", "-", ":", "-", ",", "-",
		"'", "", "\"", "", "(", "", ")", "", "`", "",
	).Replace(s)
	re := regexp.MustCompile(`-{2,}`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 50 {
		s = s[:50]
		s = strings.TrimRight(s, "-")
	}
	return s
}
