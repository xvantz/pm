package mcp

import "strings"

// slug generates a filesystem-safe identifier from a title.
// It keeps the full string — no length truncation.
func slug(title string) string {
	var b strings.Builder
	prevDash := true
	for _, r := range strings.TrimSpace(title) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r >= 'а' && r <= 'я' {
			b.WriteRune(r)
			prevDash = false
		} else if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + 32) // to lowercase
			prevDash = false
		} else if r >= 'А' && r <= 'Я' {
			b.WriteRune(r + 32) // Cyrillic uppercase → lowercase
			prevDash = false
		} else {
			if !prevDash {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return ""
	}
	return result
}
