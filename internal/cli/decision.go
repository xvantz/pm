package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/xvantz/pm/internal/types"
)

func cmdDecision(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm decision <add|list> [args]")
	}

	sub := args[0]
	switch sub {
	case "add":
		return cmdDecisionAdd(args[1:])
	case "list":
		return cmdDecisionList(args[1:])
	default:
		return fmt.Errorf("unknown decision subcommand: %s", sub)
	}
}

func cmdDecisionAdd(args []string) error {
	fs := flag.NewFlagSet("decision add", flag.ExitOnError)
	reason := fs.String("reason", "", "обоснование решения")
	_ = fs.Parse(args)

	positional := fs.Args()
	if len(positional) < 2 {
		return fmt.Errorf("usage: pm decision add [--reason ...] <project-id> <title>")
	}

	ref, title := positional[0], strings.Join(positional[1:], " ")

	st, err := openStore()
	if err != nil {
		return err
	}

	pd, err := st.ResolveProject(ref)
	if err != nil {
		return fmt.Errorf("resolve %q: %w", ref, err)
	}
	if pd == nil {
		return fmt.Errorf("project %q not found", ref)
	}

	id := slug(title)
	if id == "" {
		return fmt.Errorf("invalid decision title: %q", title)
	}

	// Check for duplicate slug in this project
	existing, err := st.GetDecisions(pd.Project.ID)
	if err != nil {
		return fmt.Errorf("get decisions: %w", err)
	}
	for _, d := range existing {
		if d.ID == id {
			return fmt.Errorf("decision %q already exists in project #%d", id, pd.Project.Number)
		}
	}

	now := types.NowISO()
	dec := types.Decision{
		ID: id, Title: title,
		Reason: *reason, Date: now,
		ProjectID: pd.Project.ID,
	}

	if err := st.SaveDecision(dec); err != nil {
		return fmt.Errorf("save decision: %w", err)
	}

	pd.Project.UpdatedAt = now
	st.SaveProject(pd.Project)

	fmt.Printf("Decision %q recorded in project #%d.\n", id, pd.Project.Number)
	fmt.Printf("  pm decision list %d\n", pd.Project.Number)
	return nil
}

func cmdDecisionList(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm decision list <project-id>")
	}

	st, err := openStore()
	if err != nil {
		return err
	}

	pd, err := st.ResolveProject(args[0])
	if err != nil {
		return fmt.Errorf("resolve %q: %w", args[0], err)
	}
	if pd == nil {
		return fmt.Errorf("project %q not found", args[0])
	}

	decisions, err := st.GetDecisions(pd.Project.ID)
	if err != nil {
		return err
	}

	if len(decisions) == 0 {
		fmt.Printf("No decisions in project #%d (%s).\n", pd.Project.Number, pd.Project.Title)
		return nil
	}

	fmt.Printf("Decisions for #%d %s:\n", pd.Project.Number, pd.Project.Title)
	fmt.Println(strings.Repeat("-", 60))
	for _, d := range decisions {
		r := d.Reason
		if r == "" {
			r = "—"
		}
		fmt.Printf("  %s — %s  (%s)\n", d.Title, d.Date, d.ID)
		if d.Reason != "" {
			fmt.Printf("    └─ %s\n", d.Reason)
		}
	}
	return nil
}
