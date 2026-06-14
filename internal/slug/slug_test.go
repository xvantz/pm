package slug

import "testing"

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

func TestSlug_UnlimitedLength(t *testing.T) {
	t.Parallel()
	long := "abcdefghij-abcdefghij-abcdefghij-abcdefghij-abcdefghij-xxx"
	s := Of(long)
	if len(s) <= 50 {
		t.Errorf("Of length = %d, want > 50 (no truncation)", len(s))
	}
	if s != long {
		t.Errorf("Of() = %q, want full string", s)
	}
}

func TestSlug_NoCollisionOnLongTitles(t *testing.T) {
	t.Parallel()
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
