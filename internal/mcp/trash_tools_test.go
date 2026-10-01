package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

func TestTrashTools_ListEmpty(t *testing.T) {
	t.Parallel()
	st := store.NewMockStore()
	out, err := handleTrashList(st, context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("trash_list error = %v", err)
	}
	if !strings.Contains(out, "empty") {
		t.Errorf("empty trash should say so, got: %s", out)
	}
}

func TestTrashTools_RestoreMissingTarget(t *testing.T) {
	t.Parallel()
	st := store.NewMockStore()
	if _, err := handleTrashRestore(st, context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("missing target must fail")
	}
	if _, err := handleTrashRestore(st, context.Background(),
		json.RawMessage(`{"target":"   "}`)); err == nil {
		t.Fatal("blank target must fail")
	}
}

// Happy path runs against FileStore: MockStore TrashRestore is a stub that
// always errors, so a mock test would only prove the error branch.
func TestTrashTools_RestoreRoundTrip(t *testing.T) {
	s := store.NewFileStore(t.TempDir())
	now := types.NowTimestamp()
	p := types.Project{ID: "tp1", Number: 41, Title: "Trash Roundtrip", Status: types.StatusActive, CreatedAt: now, UpdatedAt: now}
	if err := s.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteProject("tp1"); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}

	out, err := handleTrashList(s, context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("trash_list error = %v", err)
	}
	if !strings.Contains(out, "Trash Roundtrip") {
		t.Errorf("trash_list must show the title, got: %s", out)
	}
	if !strings.Contains(out, "#41") {
		t.Errorf("trash_list must show the number, got: %s", out)
	}

	// Restore by number: the friendly path an agent actually uses.
	restored, err := handleTrashRestore(s, context.Background(), json.RawMessage(`{"target":"41"}`))
	if err != nil {
		t.Fatalf("trash_restore by number: %v", err)
	}
	if !strings.Contains(restored, "Restored") {
		t.Errorf("restore output should confirm, got: %s", restored)
	}
	if _, err := s.GetProject("tp1"); err != nil {
		t.Errorf("project not back after restore: %v", err)
	}
}

func TestTrashTools_RestoreAmbiguitySurfaces(t *testing.T) {
	s := store.NewFileStore(t.TempDir())
	now := types.NowTimestamp()
	for _, tc := range []struct {
		id    string
		num   int
		title string
	}{
		{"ta", 51, "Gamma Service"}, {"tb", 52, "Gamma Worker"},
	} {
		if err := s.SaveProject(types.Project{ID: tc.id, Number: tc.num, Title: tc.title, Status: types.StatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
		if err := s.DeleteProject(tc.id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := handleTrashRestore(s, context.Background(),
		json.RawMessage(`{"target":"gamma"}`)); err == nil {
		t.Fatal("ambiguous title must fail")
	} else if !strings.Contains(strings.ToLower(err.Error()), "ambiguous") {
		t.Errorf("ambiguity must be named, got: %v", err)
	} else if !strings.Contains(err.Error(), "Gamma Service") {
		t.Errorf("candidates must be listed, got: %v", err)
	}
}
