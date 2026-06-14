package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	_ = fs.Parse(args)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current dir: %w", err)
	}

	pmDir := filepath.Join(cwd, "pm")
	projectsDir := filepath.Join(pmDir, "projects")

	for _, d := range []string{pmDir, projectsDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("create %s: %w", d, err)
		}
	}

	gitInit := exec.Command("git", "init", pmDir)
	if out, err := gitInit.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: git init failed (%v). You can init manually later.\n  %s\n", err, string(out))
	} else {
		gitignore := "# PM ignores nothing by default — all data is meant to be tracked.\n# Add project-specific ignores below if needed.\n"
		if err := os.WriteFile(filepath.Join(pmDir, ".gitignore"), []byte(gitignore), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not create .gitignore: %v\n", err)
		}
	}

	fmt.Printf("PM initialized at %s\n", pmDir)
	fmt.Println()
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
