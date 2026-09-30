package types

import (
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Timestamp is the only sanctioned way to read, compare or write an event
// time in PM. It exists because a bare `time.Time` cannot read the legacy
// date-only form PM files already carry, and because it has no working
// `omitempty` (a zero time marshals as 0001-01-01T00:00:00Z noise).
//
// Being a named type over time.Time also makes string comparison of event
// times a compile error rather than a silent bug: `step.UpdatedAt == "…"`
// does not typecheck, so ordering must go through Before/After/Equal.
//
// An unparseable value does NOT fail the surrounding YAML document. It is kept
// as a zero time plus its raw text, and Invalid() reports it, so a single bad
// stamp in one step cannot make the whole project unreadable — it makes that
// one value loud instead (see the "Unreadable timestamps are loud" spec).
type Timestamp struct {
	time.Time

	// raw holds the original text when it could not be parsed. Empty means
	// "parsed fine or genuinely absent" — either way, not a problem.
	raw string

	// legacy marks a value that parsed, but only via the date-only form. It
	// is still valid and still orders correctly, yet it carries no
	// intraday information: two such stamps on the same day are
	// indistinguishable. `doctor` counts these so the migration is
	// observable, and the next write rewrites them in canonical form.
	legacy bool
}

// IsLegacy reports that the value came in as a date-only stamp. It is not an
// error: the value is usable, it is just less precise than the storage format.
func (t Timestamp) IsLegacy() bool { return t.legacy }

// Layouts accepted when reading. Writes always use writeLayout.
const (
	// writeLayout is the single on-disk format: RFC3339, UTC, second
	// precision. A fixed width and a constant-zone suffix keep the textual
	// order equal to the chronological order, which matters for diffs and for
	// any future string-based tooling.
	writeLayout = "2006-01-02T15:04:05Z"

	// legacyDateLayout is the pre-RFC3339 form written by the old NowISO.
	legacyDateLayout = "2006-01-02"
)

// ParseTimestamp reads either form PM has ever written. A bare legacy date is
// interpreted as start-of-day UTC.
func ParseTimestamp(s string) (Timestamp, error) {
	if s == "" {
		return Timestamp{}, nil
	}
	if t, err := time.Parse(writeLayout, s); err == nil {
		return Timestamp{Time: t.UTC()}, nil
	}
	if t, err := time.Parse(legacyDateLayout, s); err == nil {
		return Timestamp{Time: t.UTC(), legacy: true}, nil
	}
	// Tolerate RFC3339 with an offset or fractional seconds on read: some
	// values may be hand-edited or produced by other tooling. Normalised to
	// UTC on the way in, so storage stays single-format.
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return Timestamp{Time: t.UTC()}, nil
	}
	return newUnparseable(s), fmt.Errorf("unparseable timestamp %q: want RFC3339 UTC (%q) or legacy %q",
		s, "2026-09-29T19:57:55Z", "2026-09-29")
}

// NewInvalidTimestamp is how a caller (tests, migration tooling, a doctor
// report) constructs a timestamp that is known-unreadable, so the
// loud-failure path can be exercised without hand-rolling the struct.
func NewInvalidTimestamp(raw string) Timestamp { return newUnparseable(raw) }

// NewTimestamp wraps a time.Time, normalised to UTC.
func NewTimestamp(t time.Time) Timestamp { return Timestamp{Time: t.UTC()} }

// NowTimestamp returns the current time in the canonical storage form.
func NowTimestamp() Timestamp {
	return Timestamp{Time: time.Now().UTC().Truncate(time.Second)}
}

// String renders the canonical storage form, or "" for a zero value.
func (t Timestamp) String() string {
	if t.Time.IsZero() {
		return ""
	}
	return t.Time.UTC().Format(writeLayout)
}

// Before/After/Equal are time.Time's, named explicitly so call sites read as
// intent rather than as an accident of method promotion.
func (t Timestamp) Before(u Timestamp) bool { return t.Time.Before(u.Time) }
func (t Timestamp) After(u Timestamp) bool  { return t.Time.After(u.Time) }
func (t Timestamp) Equal(u Timestamp) bool  { return t.Time.Equal(u.Time) }

// newUnparseable keeps the offending text so a caller can warn with the exact
// value instead of a generic "bad value".
func newUnparseable(raw string) Timestamp {
	return Timestamp{raw: raw}
}

// Invalid reports that this timestamp was present in the store but could not be
// parsed, and carries the raw text for the warning. Consumers MUST surface it
// (log + response field) instead of silently treating the value as absent.
func (t Timestamp) Invalid() (string, bool) { return t.raw, t.raw != "" }

// IsZero reports an absent timestamp. It returns true for an invalid one too —
// callers that care about the difference check Invalid first.
func (t Timestamp) IsZero() bool { return t.Time.IsZero() }

// DayKey is the calendar day this timestamp belongs to, in loc. Comparing
// DayKeys is how the briefing buckets "today": instant comparison decides
// order, but the day a user means is the local one, not the UTC one.
func (t Timestamp) DayKey(loc *time.Location) string {
	if t.Time.IsZero() {
		return ""
	}
	return t.Time.In(loc).Format(legacyDateLayout)
}

// UnmarshalYAML accepts the quoted legacy form as well as RFC3339. yaml.v3
// hands both to the node as a string, but the legacy form is quoted in real
// files, which the native time.Time decoder rejects.
//
// An unparseable scalar is stored as invalid and does NOT fail the decode: a
// single bad stamp must not make a whole project unreadable, which would be a
// worse failure than the one we are fixing. The value stays loud through
// Invalid() instead of loud through a parse error.
func (t *Timestamp) UnmarshalYAML(node *yaml.Node) error {
	var raw string
	if err := node.Decode(&raw); err != nil {
		// Non-scalar (mapping, sequence): record it and keep the document.
		return fmt.Errorf("timestamp: expected scalar, got %s: %w", node.Tag, err)
	}
	if raw == "" {
		*t = Timestamp{}
		return nil
	}
	parsed, _ := ParseTimestamp(raw) // error is carried by Invalid(), not by decode
	*t = parsed
	return nil
}

// MarshalYAML writes the canonical form, and omits a zero value entirely so
// absent fields stay absent instead of becoming zero-time noise. An invalid
// value is written back verbatim: rewriting it would silently destroy evidence
// of whatever went wrong upstream.
func (t Timestamp) MarshalYAML() (any, error) {
	if raw, invalid := t.Invalid(); invalid {
		return raw, nil
	}
	if t.IsZero() {
		return nil, nil
	}
	return t.String(), nil
}

// MarshalJSON emits the canonical form; a zero value becomes null so MCP and
// HTTP clients see an explicit absence.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.String())
}

// UnmarshalJSON accepts both forms symmetrically with the YAML path, and keeps
// an unparseable value invalid rather than failing the whole payload.
func (t *Timestamp) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("timestamp: expected string, got %s: %w", string(b), err)
	}
	if raw == "" {
		*t = Timestamp{}
		return nil
	}
	parsed, _ := ParseTimestamp(raw) // error is carried by Invalid(), not by decode
	*t = parsed
	return nil
}
