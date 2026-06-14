// Package slug converts titles to filesystem-safe identifiers.
//
// Both the CLI and MCP packages use this to derive step/blocker/decision IDs
// from user-provided titles. Keeping the algorithm in one place ensures they
// stay consistent.
package slug

import (
	"regexp"
	"strings"
)

var multiDash = regexp.MustCompile(`-{2,}`)

// Of converts a title to a lowercased, dash-delimited identifier.
//
//	slug.Of("Hello World")     → "hello-world"
//	slug.Of("Настроить Caddy") → "настроить-caddy"
//	slug.Of("it's ok")         → "its-ok"
func Of(title string) string {
	s := strings.ToLower(title)
	s = strings.NewReplacer(
		" ", "-", "_", "-", "/", "-", "\\", "-",
		".", "-", ":", "-", ",", "-",
		"'", "", "\"", "", "(", "", ")", "", "`", "",
	).Replace(s)
	s = multiDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// Valid reports whether title produces a non-empty slug.
func Valid(title string) bool {
	return Of(title) != ""
}
