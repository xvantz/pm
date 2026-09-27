package cli

import (
	"os"
	"path/filepath"

	"github.com/xvantz/pm/internal/apistore"
	"github.com/xvantz/pm/internal/store"
)

// openStore always talks to the daemon: remote-only, no file fallback.
// Address defaults to the serve convention (PM_API overrides), token from PM_TOKEN.
func openStore() (store.Store, error) {
	remote, _ := apistore.NewFromEnv()
	return remote, nil
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
