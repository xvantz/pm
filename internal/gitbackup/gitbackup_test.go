package gitbackup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func logLines(t *testing.T, dir string) []string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "log", "--oneline").CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v: %s", err, out)
	}
	outStr := strings.TrimSpace(string(out))
	if outStr == "" {
		return nil
	}
	return strings.Split(outStr, "\n")
}

// Fresh dir bootstraps: init, remote, action commit lands with its message.
func TestGitBackup_CommitOnAction(t *testing.T) {
	dir := t.TempDir()
	b, err := New(dir, Config{Enabled: true, RepoURL: "git@example.invalid:x/pm-data.git"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Stop()

	write(t, dir, "projects/p1/project.yaml", "title: P1\n")
	if err := b.Commit("add_project P1"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	lines := logLines(t, dir)
	if len(lines) != 1 || !strings.Contains(lines[0], "add_project P1") {
		t.Fatalf("log = %q, want one add_project commit", lines)
	}
}

// Clean tree is a no-op: no empty commits.
func TestGitBackup_CleanTreeNoCommit(t *testing.T) {
	dir := t.TempDir()
	b, err := New(dir, Config{Enabled: true, RepoURL: "git@example.invalid:x/pm-data.git"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Stop()

	write(t, dir, "a.yaml", "x: 1\n")
	if err := b.Commit("first"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := b.Commit("nothing new"); err != nil {
		t.Fatalf("second Commit: %v", err)
	}
	if n := len(logLines(t, dir)); n != 1 {
		t.Fatalf("commits = %d, want 1 (no empty commit)", n)
	}
}

// Existing hand-made repo is adopted: history kept, commits continue on top.
func TestGitBackup_AdoptsExistingRepo(t *testing.T) {
	dir := t.TempDir()
	must := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	must("init", "-b", "main")
	must("remote", "add", "origin", "git@example.invalid:x/pm-data.git")
	write(t, dir, "old.yaml", "old: true\n")
	must("add", "-A")
	must("commit", "-m", "hand-made")

	b, err := New(dir, Config{Enabled: true, RepoURL: "git@example.invalid:x/pm-data.git"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Stop()

	write(t, dir, "new.yaml", "new: true\n")
	if err := b.Commit("add_project New"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	lines := logLines(t, dir)
	if len(lines) != 2 || !strings.Contains(lines[0], "add_project New") || !strings.Contains(lines[1], "hand-made") {
		t.Fatalf("log = %q, want adopted history + new commit", lines)
	}
}

// Wrong remote is fail-closed: no push target switch, loud error.
func TestGitBackup_RemoteMismatchRefuses(t *testing.T) {
	dir := t.TempDir()
	must := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	must("init", "-b", "main")
	must("remote", "add", "origin", "git@example.invalid:evil/other.git")

	_, err := New(dir, Config{Enabled: true, RepoURL: "git@example.invalid:x/pm-data.git"})
	if err == nil || !strings.Contains(err.Error(), "refusing to push elsewhere") {
		t.Fatalf("New err = %v, want fail-closed remote refusal", err)
	}
}

// Push reaches a local bare remote (file URL, no key needed).
func TestGitBackup_PushToBare(t *testing.T) {
	bare := t.TempDir()
	if out, err := exec.Command("git", "init", "--bare", "-b", "main", bare).CombinedOutput(); err != nil {
		t.Fatalf("init bare: %v: %s", err, out)
	}
	dir := t.TempDir()
	b, err := New(dir, Config{Enabled: true, RepoURL: bare})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Stop()

	write(t, dir, "p.yaml", "v: 1\n")
	if err := b.Commit("first"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		out, _ := exec.Command("git", "-C", bare, "log", "--oneline", "main").CombinedOutput()
		if strings.Contains(string(out), "first") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("bare never received the commit, log = %q", out)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// Broken git never fails the caller: Commit surfaces the error for logging,
// the data file is already on disk.
func TestGitBackup_BrokenGitReports(t *testing.T) {
	dir := t.TempDir()
	b, err := New(dir, Config{Enabled: true, RepoURL: "git@example.invalid:x/pm-data.git"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Stop()

	write(t, dir, "p.yaml", "v: 1\n")
	b.git = "/bin/false"
	if err := b.Commit("action"); err == nil {
		t.Fatal("Commit with broken git = nil, want error for the warn log")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "p.yaml")); statErr != nil {
		t.Fatalf("data file must survive git failure: %v", statErr)
	}
}

// Disabled backup is a full no-op: no repo created, Commit nil.
func TestGitBackup_DisabledNoop(t *testing.T) {
	dir := t.TempDir()
	b, err := New(dir, Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Stop()
	if err := b.Commit("whatever"); err != nil {
		t.Fatalf("disabled Commit = %v, want nil", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Fatal("disabled backup must not init a repo")
	}
}
