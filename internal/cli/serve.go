package cli

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xvantz/pm/internal/api"
	"github.com/xvantz/pm/internal/apistore"
	"github.com/xvantz/pm/internal/gitbackup"
	"github.com/xvantz/pm/internal/store"
)

// pm serve [--addr 127.0.0.1:8472] [--dir PATH] [--token ...]
// [--backup-repo URL] [--backup-key PATH]
// The daemon is the only process touching YAML: single writer.
// Token falls back to PM_TOKEN env. Missing token is a startup error.
// Backup falls back to PM_BACKUP_REPO/PM_BACKUP_KEY env; unset repo means
// no backup. A broken backup degrades to serving without it, loudly.
func cmdServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", apistore.DefaultServeAddr, "listen address (keep localhost unless behind Tailscale)")
	dir := fs.String("dir", "", "PM root directory (overrides PM_DIR env)")
	token := fs.String("token", "", "Bearer token (overrides PM_TOKEN env)")
	backupRepo := fs.String("backup-repo", "", "git remote for data-dir backup (overrides PM_BACKUP_REPO env)")
	backupKey := fs.String("backup-key", "", "SSH key for the backup remote (overrides PM_BACKUP_KEY env)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	tok := *token
	if tok == "" {
		tok = os.Getenv("PM_TOKEN")
	}
	if tok == "" {
		return fmt.Errorf("missing token: pass --token or set PM_TOKEN env")
	}

	root := *dir
	if root == "" {
		root = os.Getenv("PM_DIR")
	}
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		root = filepath.Join(cwd, "pm")
	}
	projectsDir := filepath.Join(root, "projects")
	if info, err := os.Stat(projectsDir); err != nil || !info.IsDir() {
		return fmt.Errorf("projects dir not found: %s\n  Run `pm init` first.", projectsDir)
	}

	repo := *backupRepo
	if repo == "" {
		repo = os.Getenv("PM_BACKUP_REPO")
	}
	key := *backupKey
	if key == "" {
		key = os.Getenv("PM_BACKUP_KEY")
	}
	bk, err := gitbackup.New(projectsDir, gitbackup.Config{Enabled: repo != "", RepoURL: repo, KeyFile: key})
	if err != nil {
		// Fail-closed on push, open on serving: data matters more than
		// backup. Loud log, degraded mode, same API.
		slog.Error("git backup degraded: serving without backup", "error", err)
		bk, _ = gitbackup.New(projectsDir, gitbackup.Config{})
	}
	defer bk.Stop()

	srv := api.NewWithBackup(store.NewFileStore(projectsDir), tok, bk)
	httpSrv := &http.Server{Addr: *addr, Handler: srv}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("pm-serve listening on %s\n", *addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return httpSrv.Shutdown(shutdown)
}
