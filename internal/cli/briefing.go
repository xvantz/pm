package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/xvantz/pm/internal/briefing"
	"github.com/xvantz/pm/internal/store"
)

func cmdBriefing(args []string) error {
	fs := flag.NewFlagSet("briefing", flag.ExitOnError)
	mock := fs.Bool("mock", false, "use mock data instead of file store")
	date := fs.String("date", "", "ISO date for briefing (default: today)")
	dir := fs.String("dir", "", "path to projects/ (default: ./pm/projects)")
	asJSON := fs.Bool("json", false, "output JSON instead of markdown")
	projectRef := fs.String("project", "", "filter to a single project (number or UUID)")
	_ = fs.Parse(args)

	var st store.Store
	if *mock {
		st = store.NewMockStore()
	} else {
		root := *dir
		if root == "" {
			root = defaultProjectsDir()
		}
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			return fmt.Errorf("projects dir not found: %s\n  Run `pm init` first, or use --mock for testing.", root)
		}
		st = store.NewFileStore(root)
	}

	cfg := briefing.Config{Store: st}
	if *date != "" {
		cfg.Date = *date
	}
	if *projectRef != "" {
		cfg.FilterProject = *projectRef
	}

	b, err := briefing.Generate(cfg)
	if err != nil {
		return fmt.Errorf("generate briefing: %w", err)
	}

	if *asJSON {
		return printJSON(b)
	}
	fmt.Println(b.FormatMarkdown())
	return nil
}
