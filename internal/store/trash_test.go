package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/types"
)

func seedProject(t *testing.T, s *FileStore, id string, number int, title string) {
	t.Helper()
	now := types.NowTimestamp()
	p := types.Project{ID: id, Number: number, Title: title, Status: types.StatusActive, CreatedAt: now, UpdatedAt: now}
	if err := s.SaveProject(p); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}
}

func backupRuns(t *testing.T, s *FileStore) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(s.root, "_meta", "backups"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read backups: %v", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

func TestFileStore_StepDeleteWritesBackup(t *testing.T) {
	s := NewFileStore(t.TempDir())
	seedProject(t, s, "p1", 1, "Backup Me")
	now := types.NowTimestamp()
	if err := s.SaveStep(types.Step{ID: "s1", Title: "Work", Status: types.StepTodo, ProjectID: "p1", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	orig, err := os.ReadFile(filepath.Join(s.root, "p1", "steps", "s1.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteStep("p1", "s1"); err != nil {
		t.Fatalf("DeleteStep: %v", err)
	}
	runs := backupRuns(t, s)
	if len(runs) != 1 {
		t.Fatalf("want 1 backup run, got %d", len(runs))
	}
	backed, err := os.ReadFile(filepath.Join(s.root, "_meta", "backups", runs[0], "p1", "steps", "s1.yaml"))
	if err != nil {
		t.Fatalf("backup file missing: %v", err)
	}
	if string(backed) != string(orig) {
		t.Error("backup bytes differ from what was deleted")
	}
	if _, err := os.Stat(filepath.Join(s.root, "p1", "steps", "s1.yaml")); !os.IsNotExist(err) {
		t.Error("step file still live after delete")
	}
}

func TestFileStore_BackupRotationKeepsTwenty(t *testing.T) {
	s := NewFileStore(t.TempDir())
	seedProject(t, s, "p1", 1, "Rotation")
	now := types.NowTimestamp()
	for i := 0; i < 21; i++ {
		sid := fmt.Sprintf("step-%02d", i)
		if err := s.SaveStep(types.Step{ID: sid, Title: "T", Status: types.StepTodo, ProjectID: "p1", CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
		if err := s.DeleteStep("p1", sid); err != nil {
			t.Fatalf("delete %d: %v", i, err)
		}
	}
	if runs := backupRuns(t, s); len(runs) != 20 {
		t.Errorf("want exactly 20 backup runs, got %d", len(runs))
	}
}

func TestFileStore_TrashResolveByNumberAndTitle(t *testing.T) {
	s := NewFileStore(t.TempDir())
	seedProject(t, s, "pa", 11, "Alpha Service")
	seedProject(t, s, "pb", 12, "Alpha Worker")
	if err := s.DeleteProject("pa"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteProject("pb"); err != nil {
		t.Fatal(err)
	}

	items, err := s.TrashList()
	if err != nil || len(items) != 2 {
		t.Fatalf("TrashList = %v, %v", items, err)
	}
	if items[0].Title == "" || items[0].TrashName == "" {
		t.Errorf("trash items carry no usable identifiers: %+v", items[0])
	}

	// Ambiguous title: both match "alpha", nothing restored.
	if err := s.TrashRestore("alpha"); err == nil {
		t.Fatal("ambiguous title must fail")
	} else if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("ambiguity error must say so, got: %v", err)
	} else if !strings.Contains(err.Error(), "Alpha Service") || !strings.Contains(err.Error(), "Alpha Worker") {
		t.Errorf("ambiguity error must list candidates, got: %v", err)
	}
	if live, _ := s.ListProjects(); len(live) != 0 {
		t.Fatalf("ambiguous restore moved something: %d live", len(live))
	}

	// Unique number restores exactly one.
	if err := s.TrashRestore("12"); err != nil {
		t.Fatalf("restore by number: %v", err)
	}
	live, _ := s.ListProjects()
	if len(live) != 1 || live[0].Title != "Alpha Worker" {
		t.Fatalf("wrong project restored: %+v", live)
	}

	// Unknown value names itself.
	if err := s.TrashRestore("nope"); err == nil {
		t.Error("unknown restore target must fail")
	} else if !strings.Contains(err.Error(), `"nope"`) {
		t.Errorf("error must name the requested value, got: %v", err)
	}
}

func TestFileStore_TrashRestoreRefusesNumberCollision(t *testing.T) {
	s := NewFileStore(t.TempDir())
	seedProject(t, s, "pa", 21, "Original")
	if err := s.DeleteProject("pa"); err != nil {
		t.Fatal(err)
	}
	// A live project takes the same number while the deleted one sits in trash.
	raw := "id: pretend\nnumber: 21\ntitle: Squatter\nstatus: active\ncreated_at: \"2026-01-01T00:00:00Z\"\nupdated_at: \"2026-01-01T00:00:00Z\"\n"
	dir := filepath.Join(s.root, "pretend")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.yaml"), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	items, _ := s.TrashList()
	if len(items) != 1 {
		t.Fatalf("want 1 trash item, got %d", len(items))
	}
	if err := s.TrashRestore(items[0].TrashName); err == nil {
		t.Fatal("restore onto a taken number must refuse")
	} else if !strings.Contains(err.Error(), "#21") {
		t.Errorf("refusal must name the number, got: %v", err)
	}
}

func TestFileStore_ProjectDeleteBacksUpWholeTree(t *testing.T) {
	s := NewFileStore(t.TempDir())
	seedProject(t, s, "p9", 9, "Tree")
	now := types.NowTimestamp()
	if err := s.SaveStep(types.Step{ID: "s1", Title: "S", Status: types.StepTodo, ProjectID: "p9", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveDecision(types.Decision{ID: "d1", Title: "D", Date: now, ProjectID: "p9"}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteProject("p9"); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	runs := backupRuns(t, s)
	if len(runs) != 1 {
		t.Fatalf("want 1 backup run, got %d", len(runs))
	}
	for _, rel := range []string{"project.yaml", "steps/s1.yaml", "decisions/d1.yaml"} {
		if _, err := os.Stat(filepath.Join(s.root, "_meta", "backups", runs[0], "p9", rel)); err != nil {
			t.Errorf("backup missing %s: %v", rel, err)
		}
	}
}
