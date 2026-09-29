package types

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestParseTimestamp_AcceptedForms(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"canonical Z", "2026-09-29T19:57:55Z", "2026-09-29T19:57:55Z"},
		{"legacy date is start-of-day UTC", "2026-09-29", "2026-09-29T00:00:00Z"},
		{"offset normalised to UTC", "2026-09-29T22:57:55+03:00", "2026-09-29T19:57:55Z"},
		{"fractional seconds tolerated", "2026-09-29T19:57:55.123Z", "2026-09-29T19:57:55Z"},
		{"empty is zero, not an error", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseTimestamp(c.input)
			if err != nil {
				t.Fatalf("ParseTimestamp(%q) error = %v", c.input, err)
			}
			if got.String() != c.want {
				t.Errorf("ParseTimestamp(%q) = %q, want %q", c.input, got.String(), c.want)
			}
		})
	}
}

func TestParseTimestamp_RejectsGarbage(t *testing.T) {
	t.Parallel()
	// ParseTimestamp is the strict entry point: garbage is an error, and the
	// returned value is invalid (not silently zero-and-ok). Consumers that
	// decode a whole document use the Unmarshal* paths, which keep the raw
	// text and report through Invalid() instead of failing the document.
	for _, bad := range []string{"2026-13-45", "yesterday", "29/09/2026", "2026-09-29T19:57:55"} {
		got, err := ParseTimestamp(bad)
		if err == nil {
			t.Errorf("ParseTimestamp(%q) = %q, want error", bad, got.String())
		}
		if !got.IsZero() {
			t.Errorf("ParseTimestamp(%q) returned non-zero %q on error", bad, got.String())
		}
		if raw, invalid := got.Invalid(); !invalid || raw != bad {
			t.Errorf("ParseTimestamp(%q).Invalid() = %q,%v; want the raw value carried", bad, raw, invalid)
		}
	}
}

func TestTimestamp_UnparseableDoesNotBreakTheDocument(t *testing.T) {
	t.Parallel()
	// The regression this guards: one bad stamp must not make a whole project
	// unreadable. A hard error here would take out every field of the step.
	doc := "id: s1\nupdated_at: \"2026-13-45\"\nstatus: done\ntitle: \"Still readable\"\n"
	var step struct {
		ID        string    `yaml:"id"`
		UpdatedAt Timestamp `yaml:"updated_at"`
		Status    string    `yaml:"status"`
		Title     string    `yaml:"title"`
	}
	if err := yaml.Unmarshal([]byte(doc), &step); err != nil {
		t.Fatalf("a bad timestamp must not fail the document, got: %v", err)
	}
	raw, invalid := step.UpdatedAt.Invalid()
	if !invalid || raw != "2026-13-45" {
		t.Errorf("Invalid() = %q,%v; want the raw value surfaced", raw, invalid)
	}
	if step.Title != "Still readable" || step.Status != "done" {
		t.Errorf("sibling fields lost: %+v", step)
	}
}

func TestTimestamp_InvalidDoesNotRewriteStorage(t *testing.T) {
	t.Parallel()
	// An invalid stamp must round-trip as the original text, not be silently
	// rewritten to a zero time that hides the problem on disk.
	var step struct {
		UpdatedAt Timestamp `yaml:"updated_at"`
	}
	if err := yaml.Unmarshal([]byte("updated_at: \"2026-13-45\"\n"), &step); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := yaml.Marshal(step)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), "2026-13-45") {
		t.Errorf("marshal = %q, want the original raw value preserved", out)
	}
}

func TestTimestamp_LegacyQuotedRoundTrip(t *testing.T) {
	t.Parallel()
	// The real regression this type exists for: yaml.v3's native time.Time
	// decoder rejects a QUOTED legacy date, and PM files store it quoted.
	var doc struct {
		UpdatedAt Timestamp `yaml:"updated_at"`
	}
	if err := yaml.Unmarshal([]byte("updated_at: \"2026-09-27\"\n"), &doc); err != nil {
		t.Fatalf("quoted legacy date must decode, got error: %v", err)
	}
	if want := "2026-09-27T00:00:00Z"; doc.UpdatedAt.String() != want {
		t.Errorf("legacy date = %q, want %q", doc.UpdatedAt.String(), want)
	}

	// Unquoted forms must work too.
	for _, line := range []string{"updated_at: 2026-09-27", "updated_at: \"2026-09-27T19:57:55Z\"", "updated_at: 2026-09-27T19:57:55Z"} {
		var d struct {
			UpdatedAt Timestamp `yaml:"updated_at"`
		}
		if err := yaml.Unmarshal([]byte(line+"\n"), &d); err != nil {
			t.Errorf("decode %q: unexpected error %v", line, err)
		}
	}
}

func TestTimestamp_EmptyStaysAbsent(t *testing.T) {
	t.Parallel()
	// time.Time has no working omitempty; a zero value must vanish instead of
	// becoming 0001-01-01T00:00:00Z noise in project.yaml.
	out, err := yaml.Marshal(struct {
		UpdatedAt Timestamp `yaml:"updated_at,omitempty"`
	}{})
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "{}" {
		t.Errorf("zero timestamp marshalled as %q, want empty", got)
	}
	if strings.Contains(string(out), "0001-01-01") {
		t.Errorf("zero-time noise leaked into YAML: %q", out)
	}
}

func TestTimestamp_MarshalIsCanonical(t *testing.T) {
	t.Parallel()
	ts, err := ParseTimestamp("2026-09-29T22:57:55+03:00")
	if err != nil {
		t.Fatalf("ParseTimestamp error = %v", err)
	}
	out, err := yaml.Marshal(struct {
		UpdatedAt Timestamp `yaml:"updated_at"`
	}{UpdatedAt: ts})
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	want := "updated_at: \"2026-09-29T19:57:55Z\"\n"
	if string(out) != want {
		t.Errorf("Marshal = %q, want %q", out, want)
	}
}

func TestTimestamp_JSONSymmetric(t *testing.T) {
	t.Parallel()
	type payload struct {
		CompletedAt Timestamp `json:"completed_at,omitempty"`
	}
	// Zero → explicit null over the wire, not a fake year-1 instant.
	out, err := json.Marshal(payload{})
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != `{"completed_at":null}` {
		t.Errorf("zero JSON = %q, want completed_at:null", got)
	}

	var back payload
	if err := json.Unmarshal([]byte(`{"completed_at":"2026-09-27"}`), &back); err != nil {
		t.Fatalf("legacy JSON decode error = %v", err)
	}
	if want := "2026-09-27T00:00:00Z"; back.CompletedAt.String() != want {
		t.Errorf("legacy JSON = %q, want %q", back.CompletedAt.String(), want)
	}
}

func TestTimestamp_OrderingIsByInstant(t *testing.T) {
	t.Parallel()
	// A legacy midnight must sort before a timed event on the same day, and
	// string comparison would get this backwards ("2026-09-29" < "2026-09-29T…").
	midnight, err := ParseTimestamp("2026-09-29")
	if err != nil {
		t.Fatalf("parse midnight: %v", err)
	}
	evening, err := ParseTimestamp("2026-09-29T19:57:55Z")
	if err != nil {
		t.Fatalf("parse evening: %v", err)
	}
	if !midnight.Before(evening) {
		t.Error("start-of-day must order before a timed event on the same day")
	}
	if !evening.After(midnight) {
		t.Error("timed event must order after start-of-day")
	}
	if midnight.Equal(evening) {
		t.Error("different instants must not compare equal")
	}
}

func TestTimestamp_DayKeyUsesLocalZone(t *testing.T) {
	t.Parallel()
	// The briefing bug this prevents: 01:00 MSK is 22:00 UTC the PREVIOUS
	// day, so a UTC-day reader would file it under yesterday.
	ts, err := ParseTimestamp("2026-09-29T01:00:00+03:00")
	if err != nil {
		t.Fatalf("ParseTimestamp error = %v", err)
	}
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	if got, want := ts.DayKey(moscow), "2026-09-29"; got != want {
		t.Errorf("DayKey(Moscow) = %q, want %q (the user's day, not the UTC one)", got, want)
	}
	if got, want := ts.DayKey(time.UTC), "2026-09-28"; got != want {
		t.Errorf("DayKey(UTC) = %q, want %q", got, want)
	}
	if got := ts.DayKey(moscow); got != ts.DayKey(moscow) {
		t.Error("DayKey must be stable")
	}
}

func TestTimestamp_ZeroDayKeyIsEmpty(t *testing.T) {
	t.Parallel()
	var ts Timestamp
	if got := ts.DayKey(time.UTC); got != "" {
		t.Errorf("zero DayKey = %q, want empty", got)
	}
	if !ts.IsZero() {
		t.Error("zero Timestamp must report IsZero")
	}
}

func TestNowTimestamp_IsCanonicalUTC(t *testing.T) {
	t.Parallel()
	now := NowTimestamp()
	if !strings.HasSuffix(now.String(), "Z") {
		t.Errorf("NowTimestamp = %q, want a Z suffix", now.String())
	}
	if _, err := time.Parse(writeLayout, now.String()); err != nil {
		t.Errorf("NowTimestamp not parseable by the write layout: %v", err)
	}
	if strings.Contains(now.String(), ".") {
		t.Errorf("NowTimestamp = %q, want second precision without fraction", now.String())
	}
}

func TestNewTimestamp_NormalisesToUTC(t *testing.T) {
	t.Parallel()
	zone := time.FixedZone("MSK", 3*60*60)
	ts := NewTimestamp(time.Date(2026, 9, 29, 1, 0, 0, 0, zone))
	if want := "2026-09-28T22:00:00Z"; ts.String() != want {
		t.Errorf("NewTimestamp = %q, want %q", ts.String(), want)
	}
}
