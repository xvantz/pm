package cli

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xvantz/pm/internal/api"
	"github.com/xvantz/pm/internal/apistore"
	"github.com/xvantz/pm/internal/store"
)

// pm serve [--addr 127.0.0.1:8472] [--dir PATH] [--token ...]
// The daemon is the only process touching YAML: single writer.
// Token falls back to PM_TOKEN env. Missing token is a startup error.
func cmdServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", apistore.DefaultServeAddr, "listen address (keep localhost unless behind Tailscale)")
	dir := fs.String("dir", "", "PM root directory (overrides PM_DIR env)")
	token := fs.String("token", "", "Bearer token (overrides PM_TOKEN env)")
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

	srv := api.New(store.NewFileStore(projectsDir), tok)
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
