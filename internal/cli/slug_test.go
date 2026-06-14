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

func TestSlug_UnlimitedLength(t *testing.T) {
	long := "abcdefghij-abcdefghij-abcdefghij-abcdefghij-abcdefghij-xxx"
	s := slug(long)
	// Should be full length, no truncation
	if len(s) <= 50 {
		t.Errorf("slug length = %d, want > 50 (no truncation)", len(s))
	}
	if s != "abcdefghij-abcdefghij-abcdefghij-abcdefghij-abcdefghij-xxx" {
		t.Errorf("slug() = %q, want full string", s)
	}
}

func TestSlug_NoCollisionOnLongTitles(t *testing.T) {
	a := "abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-AAAA"
	b := "abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-BBBB"
	sa := slug(a)
	sb := slug(b)
	if sa == sb {
		t.Errorf("slugs should differ: both are %q", sa)
	}
	if sa != "abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-aaaa" {
		t.Errorf("slug(A) = %q, want lowercase-aaaa-suffix", sa)
	}
	if sb != "abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-abcde-bbbb" {
		t.Errorf("slug(B) = %q, want lowercase-bbbb-suffix", sb)
	}
}

func TestSlug_Lowercase(t *testing.T) {
	if slug("HELLO WORLD") != "hello-world" {
		t.Errorf("slug() = %q, want %q", slug("HELLO WORLD"), "hello-world")
	}
}
