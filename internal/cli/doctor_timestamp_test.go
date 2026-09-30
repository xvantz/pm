package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeStore lays out a store directory shaped like a real one: one project
// with a step and a decision, so doctor's scan has something to walk.
func writeStore(t *testing.T, projectYAML, stepYAML, decisionYAML string) string {
	t.Helper()
	// doctor reads defaultProjectsDir() = $PM_DIR/projects, so the fixture
	// lives one level deeper and captureDoctorOutput returns the $PM_DIR root.
	root := t.TempDir()
	projects := filepath.Join(root, "projects")
	pid := "0196f1aa-0000-7000-8000-000000000001"
	pdir := filepath.Join(projects, pid)
	for _, sub := range []string{"steps", "decisions"} {
		if err := os.MkdirAll(filepath.Join(pdir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(pdir, "project.yaml"):         projectYAML,
		filepath.Join(pdir, "steps", "s1.yaml"):     stepYAML,
		filepath.Join(pdir, "decisions", "d1.yaml"): decisionYAML,
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func captureDoctorOutput(t *testing.T, root string) string {
	t.Helper()
	t.Setenv("PM_DIR", root)

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := cmdDoctor(nil)
	w.Close()
	os.Stdout = orig

	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	if runErr != nil {
		t.Fatalf("cmdDoctor error = %v\n%s", runErr, sb.String())
	}
	return sb.String()
}

func TestDoctor_CountsLegacyTimestamps(t *testing.T) {
	root := writeStore(t,
		"id: p1\nnumber: 1\ntitle: T\nstatus: active\ncreated_at: \"2026-07-27\"\nupdated_at: \"2026-09-28\"\n",
		"id: s1\ntitle: S\nstatus: done\nproject_id: p1\ncreated_at: \"2026-09-01\"\nupdated_at: \"2026-09-28\"\n",
		"id: d1\ntitle: D\nproject_id: p1\ndate: \"2026-09-28\"\n",
	)
	out := captureDoctorOutput(t, root)
	if !strings.Contains(out, "устаревших") {
		t.Fatalf("doctor did not report legacy timestamps:\n%s", out)
	}
	// project: created_at + updated_at = 2, step: created_at + updated_at = 2,
	// decision: date = 1. All five are date-only in this fixture.
	if !strings.Contains(out, "5 устаревших") {
		t.Errorf("want 5 legacy stamps, got:\n%s", out)
	}
	if !strings.Contains(out, "0 битых") {
		t.Errorf("want 0 broken stamps, got:\n%s", out)
	}
	// The hint must tell the user this is not an error needing repair.
	if !strings.Contains(out, "при следующем изменении") {
		t.Errorf("doctor should explain that legacy stamps self-heal:\n%s", out)
	}
}

func TestDoctor_ReportsBrokenTimestamps(t *testing.T) {
	root := writeStore(t,
		"id: p1\nnumber: 1\ntitle: T\nstatus: active\ncreated_at: \"2026-07-27\"\nupdated_at: \"nonsense\"\n",
		"id: s1\ntitle: S\nstatus: done\nproject_id: p1\ncreated_at: \"2026-09-01\"\nupdated_at: \"2026-09-28\"\n",
		"id: d1\ntitle: D\nproject_id: p1\ndate: \"2026-09-28\"\n",
	)
	out := captureDoctorOutput(t, root)
	if !strings.Contains(out, "1 битых") {
		t.Errorf("want 1 broken stamp reported, got:\n%s", out)
	}
	// A broken stamp must never be reported as a parse failure of the file:
	// the file IS readable, the value is not.
	if strings.Contains(out, "YAML parse error") {
		t.Errorf("broken timestamp must not be reported as a YAML parse error:\n%s", out)
	}
	if !strings.Contains(out, "исключает их из подсчётов") {
		t.Errorf("doctor should say broken stamps are excluded from counts:\n%s", out)
	}
}

func TestDoctor_CleanStoreReportsZeroes(t *testing.T) {
	root := writeStore(t,
		"id: p1\nnumber: 1\ntitle: T\nstatus: active\ncreated_at: \"2026-07-27T10:00:00Z\"\nupdated_at: \"2026-09-28T10:00:00Z\"\n",
		"id: s1\ntitle: S\nstatus: done\nproject_id: p1\ncreated_at: \"2026-09-01T10:00:00Z\"\nupdated_at: \"2026-09-28T10:00:00Z\"\n",
		"id: d1\ntitle: D\nproject_id: p1\ndate: \"2026-09-28T10:00:00Z\"\n",
	)
	out := captureDoctorOutput(t, root)
	if !strings.Contains(out, "0 устаревших") || !strings.Contains(out, "0 битых") {
		t.Errorf("a migrated store must report zero legacy and zero broken:\n%s", out)
	}
}

func TestDoctor_DoesNotRewriteTheStore(t *testing.T) {
	root := writeStore(t,
		"id: p1\nnumber: 1\ntitle: T\nstatus: active\ncreated_at: \"2026-07-27\"\nupdated_at: \"2026-09-28\"\n",
		"id: s1\ntitle: S\nstatus: done\nproject_id: p1\ncreated_at: \"2026-09-01\"\nupdated_at: \"2026-09-28\"\n",
		"id: d1\ntitle: D\nproject_id: p1\ndate: \"2026-09-28\"\n",
	)
	pfile := filepath.Join(root, "projects", "0196f1aa-0000-7000-8000-000000000001", "project.yaml")
	before, err := os.ReadFile(pfile)
	if err != nil {
		t.Fatal(err)
	}
	_ = captureDoctorOutput(t, root)
	after, err := os.ReadFile(pfile)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("doctor must not modify the store:\nbefore: %s\nafter:  %s", before, after)
	}
}
