package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xvantz/pm/internal/apistore"
	"github.com/xvantz/pm/internal/store"
)

func openStore() (store.Store, error) {
	// Remote mode: everything goes through the single-writer daemon.
	// Set PM_API=http://127.0.0.1:8472 (PM_TOKEN for auth).
	if remote, ok := apistore.NewFromEnv(); ok {
		return remote, nil
	}
	root := defaultProjectsDir()
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("projects dir not found: %s\n  Run `pm init` first.", root)
	}
	return store.NewFileStore(root), nil
}

func defaultProjectsDir() string {
	if dir := os.Getenv("PM_DIR"); dir != "" {
		return filepath.Join(dir, "projects")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "./pm/projects"
	}
	return filepath.Join(cwd, "pm", "projects")
}
