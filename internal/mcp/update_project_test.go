package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

func seedProject(t *testing.T, st store.Store, title, goal string) types.Project {
	t.Helper()
	p, err := st.CreateProject(title, goal, []string{"seed"}, "")
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}
	return p
}

func callUpdateProject(t *testing.T, st store.Store, args string) (string, error) {
	t.Helper()
	return handleUpdateProject(st, context.Background(), json.RawMessage(args))
}

func TestUpdateProject_Rename(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	seedProject(t, st, "Old Name", "goal")

	out, err := callUpdateProject(t, st, `{"project_id":"1","title":"New Name"}`)
	if err != nil {
		t.Fatalf("update_project error: %v", err)
	}
	if !strings.Contains(out, "New Name") {
		t.Errorf("output does not report the new title: %s", out)
	}

	pd, err := st.ResolveProject("1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if pd.Project.Title != "New Name" {
		t.Errorf("title = %q, want %q", pd.Project.Title, "New Name")
	}
	if pd.Project.Goal != "goal" || len(pd.Project.Tags) != 1 || pd.Project.Status != types.StatusIdea {
		t.Errorf("untouched fields changed: %+v", pd.Project)
	}
}

func TestUpdateProject_Status(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	seedProject(t, st, "P", "")

	if _, err := callUpdateProject(t, st, `{"project_id":"1","status":"active"}`); err != nil {
		t.Fatalf("update_project error: %v", err)
	}
	pd, err := st.ResolveProject("1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if pd.Project.Status != types.StatusActive {
		t.Errorf("status = %q, want active", pd.Project.Status)
	}
}

func TestUpdateProject_SeveralFields(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	seedProject(t, st, "P", "")

	if _, err := callUpdateProject(t, st, `{"project_id":"1","title":"T2","goal":"G2","tags":["a","b"],"status":"paused"}`); err != nil {
		t.Fatalf("update_project error: %v", err)
	}
	pd, _ := st.ResolveProject("1")
	if pd.Project.Title != "T2" || pd.Project.Goal != "G2" || pd.Project.Status != types.StatusPaused {
		t.Errorf("fields not applied: %+v", pd.Project)
	}
	if len(pd.Project.Tags) != 2 || pd.Project.Tags[0] != "a" {
		t.Errorf("tags = %v, want [a b]", pd.Project.Tags)
	}
}

func TestUpdateProject_EmptyTagsClears(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	seedProject(t, st, "P", "")

	if _, err := callUpdateProject(t, st, `{"project_id":"1","tags":[]}`); err != nil {
		t.Fatalf("update_project error: %v", err)
	}
	pd, _ := st.ResolveProject("1")
	if len(pd.Project.Tags) != 0 {
		t.Errorf("tags = %v, want empty", pd.Project.Tags)
	}
}

func TestUpdateProject_EmptyTitleRefused(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	seedProject(t, st, "Keep", "")

	_, err := callUpdateProject(t, st, `{"project_id":"1","title":"  "}`)
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("error = %v, want a title complaint", err)
	}
	pd, _ := st.ResolveProject("1")
	if pd.Project.Title != "Keep" {
		t.Errorf("title = %q after refused update, want Keep", pd.Project.Title)
	}
}

func TestUpdateProject_NoFieldsRefused(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	seedProject(t, st, "P", "")

	_, err := callUpdateProject(t, st, `{"project_id":"1"}`)
	if err == nil || !strings.Contains(err.Error(), "nothing to update") {
		t.Fatalf("error = %v, want 'nothing to update'", err)
	}
}

func TestUpdateProject_UnknownProject(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	seedProject(t, st, "P", "")

	_, err := callUpdateProject(t, st, `{"project_id":"99","title":"X"}`)
	if err == nil || !strings.Contains(err.Error(), "99") {
		t.Fatalf("error = %v, want it to name the missing reference", err)
	}
}

func TestUpdateProject_BadStatus(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	seedProject(t, st, "P", "")

	_, err := callUpdateProject(t, st, `{"project_id":"1","status":"archived"}`)
	if err == nil || !strings.Contains(err.Error(), "bad status") {
		t.Fatalf("error = %v, want a bad status complaint", err)
	}
	pd, _ := st.ResolveProject("1")
	if pd.Project.Status != types.StatusIdea {
		t.Errorf("status = %q after refused update, want idea", pd.Project.Status)
	}
}
