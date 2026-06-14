package cli

import (
	"testing"

	"github.com/google/uuid"
	"github.com/xvantz/pm/internal/types"
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

func TestDoctor_EmptyStore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PM_DIR", dir)

	// Create a minimal project
	st, err := openStore()
	if err != nil {
		t.Fatalf("openStore() error = %v", err)
	}
	uid, _ := uuid.NewV7()
	now := types.NowISO()
	p := types.Project{ID: uid.String(), Number: 1, Title: "Test", Status: types.StatusActive, CreatedAt: now, UpdatedAt: now}
	if err := st.SaveProject(p); err != nil {
		t.Fatalf("SaveProject() error = %v", err)
	}

	// Run doctor
	err = cmdDoctor(nil)
	if err != nil {
		t.Fatalf("cmdDoctor error = %v", err)
	}
}

func TestTrash_ListEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PM_DIR", dir)

	err := cmdTrashList(nil)
	if err != nil {
		t.Fatalf("cmdTrashList error = %v", err)
	}
}

func TestTrash_RestoreClean(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PM_DIR", dir)

	st, err := openStore()
	if err != nil {
		t.Fatalf("openStore() error = %v", err)
	}

	uid, _ := uuid.NewV7()
	now := types.NowISO()
	pid := uid.String()
	p := types.Project{ID: pid, Number: 1, Title: "Trash Test", Status: types.StatusActive, CreatedAt: now, UpdatedAt: now}
	if err := st.SaveProject(p); err != nil {
		t.Fatalf("SaveProject() error = %v", err)
	}

	// Delete to trash
	if err := st.DeleteProject(pid); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}

	// List trash
	items, err := st.TrashList()
	if err != nil {
		t.Fatalf("TrashList() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("TrashList returned %d items, want 1", len(items))
	}

	// Restore
	if err := st.TrashRestore(items[0]); err != nil {
		t.Fatalf("TrashRestore() error = %v", err)
	}

	// Verify project is back
	_, err = st.GetProject(pid)
	if err != nil {
		t.Errorf("Project not found after restore: %v", err)
	}

	// Clean trash (should be empty after restore)
	if err := st.TrashClean(); err != nil {
		t.Fatalf("TrashClean() error = %v", err)
	}
}
