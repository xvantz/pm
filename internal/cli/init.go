package cli

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	_ = fs.Parse(args)

	pmDir := os.Getenv("PM_DIR")
	if pmDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get current dir: %w", err)
		}
		pmDir = filepath.Join(cwd, "pm")
	}

	projectsDir := filepath.Join(pmDir, "projects")

	for _, d := range []string{pmDir, projectsDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("create %s: %w", d, err)
		}
	}

	gitInit := exec.Command("git", "init", pmDir)
	if out, err := gitInit.CombinedOutput(); err != nil {
		slog.Warn("git init failed", "error", err, "output", string(out))
		slog.Warn("you can git init the pm directory manually")
	}

	gitignore := "# PM ignores nothing by default — all data is meant to be tracked.\n# Add project-specific ignores below if needed.\n"
	if err := os.WriteFile(filepath.Join(pmDir, ".gitignore"), []byte(gitignore), 0644); err != nil {
		slog.Warn("could not create .gitignore", "error", err)
	}

	fmt.Printf("PM initialized at %s\n\n", pmDir)
	fmt.Println("  Structure:")
	fmt.Println("    pm/")
	fmt.Println("    ├── .git/")
	fmt.Println("    ├── .gitignore")
	fmt.Println("    └── projects/     ← project YAML files go here")
	fmt.Println()
	fmt.Println("First project:")
	fmt.Println("  pm add project \"My Project\"")
	return nil
}
