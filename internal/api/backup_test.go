package api

import (
	"net/http"
	"os/exec"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/gitbackup"
	"github.com/xvantz/pm/internal/store"
)

// Actions through the daemon leave one commit naming the action. Push points
// at an invalid remote: commits land, push failures stay warnings.
func TestDaemonBackup_CommitsPerAction(t *testing.T) {
	dir := t.TempDir()
	bk, err := gitbackup.New(dir, gitbackup.Config{
		Enabled: true, RepoURL: "git@example.invalid:x/pm-data.git",
	})
	if err != nil {
		t.Fatalf("backup New: %v", err)
	}
	defer bk.Stop()

	srv := NewWithBackup(store.NewFileStore(dir), testToken, bk)

	w := doReq(t, srv, "POST", "/api/projects", map[string]string{"title": "Backed"}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: code = %d, body = %s", w.Code, w.Body.String())
	}
	w = doReq(t, srv, "POST", "/api/projects/1/steps", map[string]string{"title": "First"}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("step: code = %d, body = %s", w.Code, w.Body.String())
	}

	out, err := exec.Command("git", "-C", dir, "log", "--oneline").CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v: %s", err, out)
	}
	log := string(out)
	if !strings.Contains(log, "add_project") || !strings.Contains(log, "add_step") {
		t.Fatalf("log = %q, want one commit per action", log)
	}
	if n := len(strings.Split(strings.TrimSpace(log), "\n")); n != 2 {
		t.Fatalf("commits = %d, want 2 (no touch noise)", n)
	}
}

// A server without backup answers identically: handlers never branch on it.
func TestDaemonBackup_DisabledSameAnswers(t *testing.T) {
	srv := New(store.NewFileStore(t.TempDir()), testToken)
	w := doReq(t, srv, "POST", "/api/projects", map[string]string{"title": "Plain"}, testToken)
	if w.Code != http.StatusCreated {
		t.Fatalf("create without backup: code = %d, body = %s", w.Code, w.Body.String())
	}
}
