package cli

import (
	"testing"
)

func TestAdd_Dispatch(t *testing.T) {
	cases := []struct {
		args []string
		want string // expected error fragment
	}{
		{[]string{}, "usage: pm add"},
		{[]string{"invalid"}, "unknown: pm add invalid"},
		{[]string{"project"}, "usage: pm add project"},
		{[]string{"step"}, "usage: pm add step"},
		{[]string{"blocker"}, "usage: pm blocker add"},
		{[]string{"decision"}, "usage: pm decision add"},
	}

	for _, c := range cases {
		err := cmdAdd(c.args)
		if err == nil {
			t.Errorf("cmdAdd(%v) expected error containing %q, got nil", c.args, c.want)
			continue
		}
		if !contains(err.Error(), c.want) {
			t.Errorf("cmdAdd(%v) error = %q, want %q", c.args, err.Error(), c.want)
		}
	}
}

func TestDel_Dispatch(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{}, "usage: pm del"},
		{[]string{"invalid"}, "unknown: pm del invalid"},
		{[]string{"project"}, "usage: pm del project"},
		{[]string{"step", "1"}, "usage: pm del step"},
		{[]string{"blocker", "1"}, "usage: pm del blocker"},
		{[]string{"decision", "1"}, "usage: pm del decision"},
	}

	for _, c := range cases {
		err := cmdDel(c.args)
		if err == nil {
			t.Errorf("cmdDel(%v) expected error containing %q, got nil", c.args, c.want)
			continue
		}
		if !contains(err.Error(), c.want) {
			t.Errorf("cmdDel(%v) error = %q, want %q", c.args, err.Error(), c.want)
		}
	}
}

func TestProject_Dispatch(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{}, "usage: pm project"},
		{[]string{"invalid"}, "unknown project subcommand"},
	}

	for _, c := range cases {
		err := cmdProject(c.args)
		if err == nil {
			t.Errorf("cmdProject(%v) expected error, got nil", c.args)
			continue
		}
		if !contains(err.Error(), c.want) {
			t.Errorf("cmdProject(%v) error = %q, want %q", c.args, err.Error(), c.want)
		}
	}
}

func TestStep_Dispatch(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{}, "usage: pm step"},
		{[]string{"invalid"}, "unknown step subcommand"},
	}

	for _, c := range cases {
		err := cmdStep(c.args)
		if err == nil {
			t.Errorf("cmdStep(%v) expected error, got nil", c.args)
			continue
		}
		if !contains(err.Error(), c.want) {
			t.Errorf("cmdStep(%v) error = %q, want %q", c.args, err.Error(), c.want)
		}
	}
}

func TestBlocker_Dispatch(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{}, "usage: pm blocker"},
		{[]string{"invalid"}, "unknown blocker subcommand"},
	}

	for _, c := range cases {
		err := cmdBlocker(c.args)
		if err == nil {
			t.Errorf("cmdBlocker(%v) expected error, got nil", c.args)
			continue
		}
		if !contains(err.Error(), c.want) {
			t.Errorf("cmdBlocker(%v) error = %q, want %q", c.args, err.Error(), c.want)
		}
	}
}

func TestDecision_Dispatch(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{}, "usage: pm decision"},
		{[]string{"invalid"}, "unknown decision subcommand"},
	}

	for _, c := range cases {
		err := cmdDecision(c.args)
		if err == nil {
			t.Errorf("cmdDecision(%v) expected error, got nil", c.args)
			continue
		}
		if !contains(err.Error(), c.want) {
			t.Errorf("cmdDecision(%v) error = %q, want %q", c.args, err.Error(), c.want)
		}
	}
}

func TestRun_Help(t *testing.T) {
	if err := Run([]string{"--help"}); err != nil {
		t.Errorf("Run --help error = %v", err)
	}
	if err := Run([]string{"-h"}); err != nil {
		t.Errorf("Run -h error = %v", err)
	}
	if err := Run([]string{"help"}); err != nil {
		t.Errorf("Run help error = %v", err)
	}
}

func TestRun_Unknown(t *testing.T) {
	err := Run([]string{"nonexistent"})
	if err == nil {
		t.Fatal("Run(nonexistent) expected error, got nil")
	}
	if !contains(err.Error(), "unknown command") {
		t.Errorf("err = %q, want %q", err.Error(), "unknown command")
	}
}

// TestBriefing_Mock runs a full briefing generation via Run (no store needed).
func TestBriefing_Mock(t *testing.T) {
	err := Run([]string{"briefing", "--mock", "--date", "2026-06-14"})
	if err != nil {
		t.Fatalf("briefing --mock error = %v", err)
	}
}

// Helper: string contains
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
