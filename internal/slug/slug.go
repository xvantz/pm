// Package slug converts titles to filesystem-safe identifiers.
//
// Both the CLI and MCP packages use this to derive step/blocker/decision IDs
// from user-provided titles. Keeping the algorithm in one place ensures they
// stay consistent.
package slug

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxSlugBytes caps IDs well under the filesystem NAME_MAX (255 bytes),
// leaving room for the `.yaml` suffix. Counted in bytes, not runes: the
// filesystem counts bytes and Cyrillic is multibyte. Longer titles are cut
// at a UTF-8 boundary, never mid-rune.
const maxSlugBytes = 200

// dropRunes are stripped rather than dashed. Kept for backward compatibility:
// IDs minted before the allowlist rule already treat these as absent, and
// flipping them to dashes would rename the mapping for existing titles.
var dropRunes = map[rune]bool{
	'\'': true, '"': true, '(': true, ')': true, '`': true,
}

// Of converts a title to a lowercased, dash-delimited identifier.
//
// Rules (see project-store "Slug contract"): Unicode letters and numbers are
// kept; spaces, punctuation and symbols become a single dash; the dropRunes
// above and everything else vanish. Dashes collapse, ends are trimmed, and
// the result is capped at maxSlugBytes on a rune boundary.
//
//	slug.Of("Hello World")     → "hello-world"
//	slug.Of("Настроить Caddy") → "настроить-caddy"
//	slug.Of("Fix it! #urgent") → "fix-it-urgent"
func Of(title string) string {
	var b strings.Builder
	b.Grow(len(title))
	prevDash := true // collapse leading separators without a post-pass
	for _, r := range strings.ToLower(title) {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
			prevDash = false
		case dropRunes[r]:
			// absent, as before
		case unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r):
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		default:
			// control, format and other non-printables: drop
		}
	}
	s := strings.Trim(b.String(), "-")
	return truncateBytes(s)
}

// truncateBytes cuts s to at most max bytes at a UTF-8 boundary, trimming a
// dash left hanging on the cut. A cut can only shorten: same prefix in, same
// prefix out, so two titles sharing the first max bytes collide honestly
// instead of producing a broken filename.
func truncateBytes(s string) string {
	if len(s) <= maxSlugBytes {
		return s
	}
	cut := maxSlugBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return strings.Trim(s[:cut], "-")
}

// Valid reports whether title produces a non-empty slug.
func Valid(title string) bool {
	return Of(title) != ""
}
