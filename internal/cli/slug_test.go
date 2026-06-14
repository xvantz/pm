package cli

import "testing"

func TestSlug_Simple(t *testing.T) {
	if slug("Hello World") != "hello-world" {
		t.Errorf("slug(Hello World) = %q, want %q", slug("Hello World"), "hello-world")
	}
}

func TestSlug_Cyrillic(t *testing.T) {
	s := slug("Настроить Caddy reverse proxy")
	if s != "настроить-caddy-reverse-proxy" {
		t.Errorf("slug() = %q, want %q", s, "настроить-caddy-reverse-proxy")
	}
}

func TestSlug_SpecialChars(t *testing.T) {
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
	}
	for _, c := range cases {
		got := slug(c.input)
		if got != c.want {
			t.Errorf("slug(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestSlug_TrimDashes(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"-leading", "leading"},
		{"trailing-", "trailing"},
		{"-both-", "both"},
	}
	for _, c := range cases {
		got := slug(c.input)
		if got != c.want {
			t.Errorf("slug(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestSlug_CollapseMultipleDashes(t *testing.T) {
	s := slug("foo   bar___baz")
	if s != "foo-bar-baz" {
		t.Errorf("slug() = %q, want %q", s, "foo-bar-baz")
	}
}

func TestSlug_Empty(t *testing.T) {
	cases := []string{"", "'", `"`, "`", "'\"`"}
	for _, c := range cases {
		if slug(c) != "" {
			t.Errorf("slug(%q) should be empty, got %q", c, slug(c))
		}
	}
}

func TestSlug_TruncateAt50(t *testing.T) {
	long := "abcdefghij-abcdefghij-abcdefghij-abcdefghij-abcdefghij-xxx"
	s := slug(long)
	if len(s) > 50 {
		t.Errorf("slug length = %d, want <= 50", len(s))
	}
	if s != "abcdefghij-abcdefghij-abcdefghij-abcdefghij-abcdef" {
		t.Errorf("slug() = %q, want %q", s, "abcdefghij-abcdefghij-abcdefghij-abcdefghij-abcdef")
	}
}

func TestSlug_TwoDifferentAfter50(t *testing.T) {
	// Known limitation: titles differing after ~50 chars collide on truncation
	// This test documents the current behavior.
	a := "abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-AAAA"
	b := "abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-BBBB"
	sa := slug(a)
	sb := slug(b)
	if sa != sb {
		t.Errorf("expected collision due to 50-char truncation, but slugs differ: %q vs %q", sa, sb)
	}
}

func TestSlug_Lowercase(t *testing.T) {
	if slug("HELLO WORLD") != "hello-world" {
		t.Errorf("slug() = %q, want %q", slug("HELLO WORLD"), "hello-world")
	}
}
