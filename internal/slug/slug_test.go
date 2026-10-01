package slug

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSlug_Simple(t *testing.T) {
	t.Parallel()
	if Of("Hello World") != "hello-world" {
		t.Errorf("Of(Hello World) = %q, want %q", Of("Hello World"), "hello-world")
	}
}

func TestSlug_Cyrillic(t *testing.T) {
	t.Parallel()
	s := Of("Настроить Caddy reverse proxy")
	want := "настроить-caddy-reverse-proxy"
	if s != want {
		t.Errorf("Of() = %q, want %q", s, want)
	}
}

func TestSlug_SpecialChars(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input, want string
	}{
		{"Test/Path", "test-path"},
		{"test_file.yaml", "test-file-yaml"},
		{"key:value", "key-value"},
		{"a,b,c", "a-b-c"},
		{"it's ok", "its-ok"},
		{`"quoted"`, "quoted"},
		{"(parens)", "parens"},
		{"back`tick`", "backtick"},
		{"Fix it! #urgent?", "fix-it-urgent"},
		{"100% & $money$", "100-money"},
		{"a+b=c@d", "a-b-c-d"},
	}
	for _, c := range cases {
		got := Of(c.input)
		if got != c.want {
			t.Errorf("Of(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestSlug_TrimDashes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input, want string
	}{
		{"-leading", "leading"},
		{"trailing-", "trailing"},
		{"-both-", "both"},
	}
	for _, c := range cases {
		got := Of(c.input)
		if got != c.want {
			t.Errorf("Of(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestSlug_CollapseMultipleDashes(t *testing.T) {
	t.Parallel()
	s := Of("foo   bar___baz")
	if s != "foo-bar-baz" {
		t.Errorf("Of() = %q, want %q", s, "foo-bar-baz")
	}
}

func TestSlug_Empty(t *testing.T) {
	t.Parallel()
	cases := []string{"", "'", `"`, "`", "'\"`"}
	for _, c := range cases {
		if Of(c) != "" {
			t.Errorf("Of(%q) should be empty, got %q", c, Of(c))
		}
	}
}

func TestSlug_CappedAt200Bytes(t *testing.T) {
	t.Parallel()
	// 300 ASCII chars: the old code returned all of them, long enough to
	// break the filesystem. Now the ID is capped, valid, and dash-clean.
	long := strings.Repeat("abcdefghij-", 30)
	s := Of(long)
	if len(s) > maxSlugBytes {
		t.Errorf("Of length = %d bytes, want <= %d", len(s), maxSlugBytes)
	}
	if !utf8.ValidString(s) {
		t.Errorf("Of(%q...) is not valid UTF-8", s[:50])
	}
	if strings.HasSuffix(s, "-") {
		t.Errorf("Of() ends with a hanging dash: %q", s[len(s)-10:])
	}
	if !Valid(long) {
		t.Error("a long title must still be valid, just capped")
	}
}

func TestSlug_CapRespectsRuneBoundary(t *testing.T) {
	t.Parallel()
	// Cyrillic is 2 bytes per rune: a byte cut can land mid-rune. Fill past
	// the cap with multibyte text and require intact output.
	long := strings.Repeat("настроить-", 30)
	s := Of(long)
	if len(s) > maxSlugBytes {
		t.Errorf("Of length = %d bytes, want <= %d", len(s), maxSlugBytes)
	}
	if !utf8.ValidString(s) {
		t.Error("cap split a multibyte rune")
	}
}

func TestSlug_SharedPrefixCollides(t *testing.T) {
	t.Parallel()
	// Two titles differing only past byte 200 collapse to one slug. That is
	// an honest conflict at the caller (409), not a silent merge.
	a := strings.Repeat("abcde-", 40) + "AAAA"
	b := strings.Repeat("abcde-", 40) + "BBBB"
	if Of(a) != Of(b) {
		t.Errorf("Of(A) = %q, Of(B) = %q, want equal (shared 200-byte prefix)", Of(a), Of(b))
	}
}

func TestSlug_NoCollisionOnLongTitles(t *testing.T) {
	t.Parallel()
	// Titles differing WITHIN the cap stay distinct: the cap must not eat
	// differences it can keep.
	a := "abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-AAAA"
	b := "abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-BBBB"
	sa := Of(a)
	sb := Of(b)
	if sa == sb {
		t.Errorf("slugs should differ: both are %q", sa)
	}
	if sa != "abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-aaaa" {
		t.Errorf("Of(A) = %q, want lowercase-aaaa-suffix", sa)
	}
	if sb != "abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-bbbb" {
		t.Errorf("Of(B) = %q, want lowercase-bbbb-suffix", sb)
	}
}

func TestSlug_Lowercase(t *testing.T) {
	t.Parallel()
	if Of("HELLO WORLD") != "hello-world" {
		t.Errorf("Of() = %q, want %q", Of("HELLO WORLD"), "hello-world")
	}
}

func TestSlug_Valid(t *testing.T) {
	t.Parallel()
	if !Valid("Hello World") {
		t.Error("Valid(Hello World) should be true")
	}
	if Valid("") {
		t.Error("Valid empty should be false")
	}
	if Valid("'") {
		t.Error("Valid single quote should be false")
	}
}
