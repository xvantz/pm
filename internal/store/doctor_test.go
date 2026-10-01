package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// layProject writes a project tree shaped like a real one straight to disk.
// Legacy and broken timestamps must be written raw: going through Save APIs
// would normalize them into canonical form on write.
func layProject(t *testing.T, root, dir, projectYAML string, files map[string]string) {
	t.Helper()
	pdir := filepath.Join(root, dir)
	for _, sub := range []string{"steps", "decisions"} {
		if err := os.MkdirAll(filepath.Join(pdir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(pdir, "project.yaml"), []byte(projectYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(pdir, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFileStore_CheckCountsLegacyAndBroken(t *testing.T) {
	dir := t.TempDir()
	layProject(t, dir, "p1",
		"id: p1\nnumber: 1\ntitle: T\nstatus: active\ncreated_at: \"2026-07-27\"\nupdated_at: \"2026-09-28\"\n",
		map[string]string{
			"steps/s1.yaml":     "id: s1\ntitle: S\nstatus: done\nproject_id: p1\ncreated_at: \"2026-09-01\"\nupdated_at: \"2026-09-28\"\n",
			"decisions/d1.yaml": "id: d1\ntitle: D\nproject_id: p1\ndate: \"2026-09-28\"\n",
		},
	)
	rep, err := NewFileStore(dir).Check()
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	// project created+updated = 2, step created+updated = 2, decision date = 1.
	if rep.LegacyTimestamps != 5 {
		t.Errorf("legacy = %d, want 5", rep.LegacyTimestamps)
	}
	if rep.BrokenTimestamps != 0 {
		t.Errorf("broken = %d, want 0", rep.BrokenTimestamps)
	}
	if len(rep.Projects) != 1 || rep.TotalSteps != 1 || rep.TotalDecisions != 1 {
		t.Errorf("counts off: %+v", rep)
	}
	if rep.HasIssues() {
		t.Errorf("clean store must have no issues: %+v", rep)
	}
}

func TestFileStore_CheckReportsBrokenNotParseError(t *testing.T) {
	dir := t.TempDir()
	layProject(t, dir, "p1",
		"id: p1\nnumber: 1\ntitle: T\nstatus: active\ncreated_at: \"2026-07-27\"\nupdated_at: \"nonsense\"\n",
		map[string]string{
			"steps/s1.yaml": "id: s1\ntitle: S\nstatus: done\nproject_id: p1\ncreated_at: \"2026-09-01\"\nupdated_at: \"2026-09-28T10:00:00Z\"\n",
		},
	)
	rep, err := NewFileStore(dir).Check()
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if rep.BrokenTimestamps != 1 {
		t.Errorf("broken = %d, want 1", rep.BrokenTimestamps)
	}
	// A broken stamp is a bad value, not a bad file: no parse-error issue.
	for _, is := range rep.Issues {
		if strings.Contains(is, "YAML parse error") {
			t.Errorf("broken stamp reported as parse error: %q", is)
		}
	}
}

func TestFileStore_CheckFindsOrphansAndMismatches(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lonely"), 0o755); err != nil {
		t.Fatal(err)
	}
	layProject(t, dir, "p1",
		"id: p1\nnumber: 1\ntitle: T\nstatus: active\ncreated_at: \"2026-09-28T10:00:00Z\"\nupdated_at: \"2026-09-28T10:00:00Z\"\n",
		map[string]string{
			"steps/s1.yaml": "id: s1\ntitle: S\nstatus: todo\nproject_id: WRONG\ncreated_at: \"2026-09-28T10:00:00Z\"\n",
		},
	)
	rep, err := NewFileStore(dir).Check()
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(rep.Orphans) != 1 || rep.Orphans[0] != "lonely" {
		t.Errorf("orphans = %v, want [lonely]", rep.Orphans)
	}
	found := false
	for _, is := range rep.Issues {
		if strings.Contains(is, "project_id mismatch") {
			found = true
		}
	}
	if !found {
		t.Errorf("mismatch not reported: %+v", rep.Issues)
	}
	if !rep.HasIssues() {
		t.Error("HasIssues must be true")
	}
}

func TestFileStore_CheckMissingStore(t *testing.T) {
	_, err := NewFileStore(filepath.Join(t.TempDir(), "nope")).Check()
	if err == nil {
		t.Fatal("missing store must be an error")
	}
	if !strings.Contains(err.Error(), "pm init") {
		t.Errorf("error must point at pm init, got: %v", err)
	}
}

func TestFileStore_CheckIsReadOnly(t *testing.T) {
	dir := t.TempDir()
	layProject(t, dir, "p1",
		"id: p1\nnumber: 1\ntitle: T\nstatus: active\ncreated_at: \"2026-07-27\"\nupdated_at: \"2026-09-28\"\n",
		map[string]string{
			"steps/s1.yaml": "id: s1\ntitle: S\nstatus: todo\nproject_id: p1\n",
		},
	)
	before := dirSnapshot(t, dir)
	if _, err := NewFileStore(dir).Check(); err != nil {
		t.Fatalf("Check: %v", err)
	}
	after := dirSnapshot(t, dir)
	if before != after {
		t.Error("Check modified the store")
	}
}

func dirSnapshot(t *testing.T, root string) string {
	t.Helper()
	var sb strings.Builder
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		sb.WriteString(rel + "\n" + string(data) + "\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return sb.String()
}
