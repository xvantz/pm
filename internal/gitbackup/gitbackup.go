// Package gitbackup keeps the daemon data directory in git and pushes it
// off-host. The daemon is the only writer, so no other component touches
// this repo. Git failures are advisory: data writes land first and succeed
// regardless; a failed commit or push is retried or picked up later.
package gitbackup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config wires the backup. Zero Config disables everything: New returns a
// Backup whose methods are no-ops, so callers never branch on enablement.
type Config struct {
	Enabled bool
	RepoURL string
	// KeyFile is an SSH private key for the remote. Token is a Bearer
	// token for HTTPS remotes. Both set means each travels its own
	// channel; git picks the one matching the remote scheme.
	KeyFile   string
	Token     string
	Author    string
	Email     string
	PushTries int
}

const (
	defaultAuthor = "pm-serve"
	defaultEmail  = "pm-serve@localhost"
	cmdTimeout    = 30 * time.Second
	pushRetryWait = 10 * time.Second
)

// Backup owns the git side of one data directory.
type Backup struct {
	dir string
	cfg Config
	git string // git binary, overrideable in tests to force failures

	pushCh chan struct{}
	stopCh chan struct{}
	wg     sync.WaitGroup
	once   sync.Once
}

// New ensures the git state and starts the push loop. With a zero Config
// it returns a disabled Backup: every method is a no-op returning nil.
func New(dir string, cfg Config) (*Backup, error) {
	b := &Backup{dir: dir, cfg: cfg, git: "git"}
	if !cfg.Enabled {
		return b, nil
	}
	if strings.TrimSpace(cfg.RepoURL) == "" {
		return nil, fmt.Errorf("gitbackup: repo URL is required when backup is enabled")
	}
	if err := b.ensure(); err != nil {
		return nil, err
	}
	b.pushCh = make(chan struct{}, 1)
	b.stopCh = make(chan struct{})
	b.wg.Add(1)
	go b.pushLoop()
	return b, nil
}

// ensure inits or adopts the repo and verifies the remote. Fail-closed:
// a wrong remote is an error, never a silent push elsewhere.
func (b *Backup) ensure() error {
	if _, err := os.Stat(filepath.Join(b.dir, ".git")); os.IsNotExist(err) {
		if err := b.run("init", "-b", "main"); err != nil {
			return fmt.Errorf("gitbackup: init: %w", err)
		}
		if err := b.run("remote", "add", "origin", b.cfg.RepoURL); err != nil {
			return fmt.Errorf("gitbackup: remote add: %w", err)
		}
		b.pinIdentity()
		return nil
	}
	// Adopted repo: same local identity, so commits never depend on the
	// user's global gitconfig (the daemon user may have none).
	b.pinIdentity()
	out, err := b.output("remote", "get-url", "origin")
	if err != nil {
		return fmt.Errorf("gitbackup: read origin: %w", err)
	}
	if current := strings.TrimSpace(out); current != strings.TrimSpace(b.cfg.RepoURL) {
		// The operator owns both config and repo: a stale origin (e.g. SSH
		// from before the HTTPS switch) must follow the config, or the
		// backup stays stuck forever. Loud log names both URLs; a typo'd
		// URL fails at push with auth error, not silently elsewhere.
		slog.Warn("gitbackup: origin differs, updating to configured remote",
			"current", current, "configured", strings.TrimSpace(b.cfg.RepoURL))
		if err := b.run("remote", "set-url", "origin", strings.TrimSpace(b.cfg.RepoURL)); err != nil {
			return fmt.Errorf("gitbackup: set-url origin: %w", err)
		}
	}
	return nil
}

func (b *Backup) identity() (string, string) {
	author := b.cfg.Author
	if author == "" {
		author = defaultAuthor
	}
	email := b.cfg.Email
	if email == "" {
		email = defaultEmail
	}
	return author, email
}

// pinIdentity fixes the repo-local author so commits never depend on the
// user's global gitconfig. Best-effort: a failure surfaces on commit.
func (b *Backup) pinIdentity() {
	author, email := b.identity()
	_ = b.run("config", "user.name", author)
	_ = b.run("config", "user.email", email)
	_ = b.run("config", "commit.gpgsign", "false")
}

// Commit stages everything and commits with the action message. Clean tree
// is a nil no-op. Disabled backup is a nil no-op. Errors go to the caller,
// which logs them as warnings: git must never fail a data write.
func (b *Backup) Commit(message string) error {
	if b == nil || !b.cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("gitbackup: empty commit message")
	}
	if err := b.run("add", "-A"); err != nil {
		return fmt.Errorf("gitbackup: add: %w", err)
	}
	if err := b.run("diff", "--cached", "--quiet"); err == nil {
		return nil // nothing staged
	}
	if err := b.run("commit", "-m", message); err != nil {
		return fmt.Errorf("gitbackup: commit: %w", err)
	}
	b.kickPush()
	return nil
}

// kickPush coalesces: at most one pending push signal.
func (b *Backup) kickPush() {
	select {
	case b.pushCh <- struct{}{}:
	default:
	}
}

// pushLoop drains push signals until Stop.
func (b *Backup) pushLoop() {
	defer b.wg.Done()
	for {
		select {
		case <-b.stopCh:
			return
		case <-b.pushCh:
			b.pushWithRetry()
		}
	}
}

// pushWithRetry pushes until success or tries run out. Leftovers stay local
// and leave on the next kick: offline accumulates, reconnect drains.
func (b *Backup) pushWithRetry() {
	tries := b.cfg.PushTries
	if tries <= 0 {
		tries = 5
	}
	for i := 0; i < tries; i++ {
		if err := b.run("push", "origin", "HEAD:main"); err == nil {
			return
		} else {
			slog.Warn("gitbackup: push failed, retrying", "attempt", i+1, "error", err)
		}
		select {
		case <-b.stopCh:
			return
		case <-time.After(pushRetryWait):
		}
	}
	slog.Warn("gitbackup: push still failing, will retry on next commit")
}

// Stop ends the push loop. In-flight retry waits abort; committed data stays.
func (b *Backup) Stop() {
	if b == nil || !b.cfg.Enabled {
		return
	}
	b.once.Do(func() {
		close(b.stopCh)
		b.wg.Wait()
	})
}

func (b *Backup) env() []string {
	env := os.Environ()
	if b.cfg.KeyFile != "" {
		cmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new -o BatchMode=yes", b.cfg.KeyFile)
		env = append(env, "GIT_SSH_COMMAND="+cmd)
	}
	if b.cfg.Token != "" {
		// Env config, not -c args: the token stays out of the process list.
		env = append(env, "GIT_CONFIG_COUNT=1")
		env = append(env, "GIT_CONFIG_KEY_0=http.extraHeader")
		env = append(env, "GIT_CONFIG_VALUE_0=Authorization: Bearer "+b.cfg.Token)
	}
	return env
}

func (b *Backup) run(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, b.git, args...)
	cmd.Dir = b.dir
	cmd.Env = b.env()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (b *Backup) output(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, b.git, args...)
	cmd.Dir = b.dir
	cmd.Env = b.env()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
