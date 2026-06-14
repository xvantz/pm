package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/xvantz/pm/internal/types"
)

func cmdBlocker(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm blocker <add|resolve|list> [args]")
	}

	sub := args[0]
	switch sub {
	case "add":
		return cmdBlockerAdd(args[1:])
	case "resolve":
		return cmdBlockerResolve(args[1:])
	case "list":
		return cmdBlockerList(args[1:])
	default:
		return fmt.Errorf("unknown blocker subcommand: %s", sub)
	}
}

func cmdBlockerAdd(args []string) error {
	fs := flag.NewFlagSet("blocker add", flag.ExitOnError)
	reason := fs.String("reason", "", "причина блокера")
	_ = fs.Parse(args)

	positional := fs.Args()
	if len(positional) < 3 {
		return fmt.Errorf("usage: pm blocker add [--reason ...] <project-id> <step-slug> <title>")
	}

	ref, stepSlug, title := positional[0], positional[1], strings.Join(positional[2:], " ")

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

	// Find the step
	var targetStep *types.Step
	for i, s := range pd.Steps {
		if s.ID == stepSlug {
			targetStep = &pd.Steps[i]
			break
		}
	}
	if targetStep == nil {
		return fmt.Errorf("step %q not found in project #%d", stepSlug, pd.Project.Number)
	}

	id := slug(title)
	if id == "" {
		return fmt.Errorf("invalid blocker title: %q", title)
	}

	// Check for duplicate blocker slug in this step
	for _, b := range targetStep.Blockers {
		if b.ID == id {
			return fmt.Errorf("blocker %q already exists in step %q", id, stepSlug)
		}
	}

	now := types.NowISO()
	blocker := types.Blocker{
		ID: id, Title: title,
		Status:    types.BlockerWaiting,
		Reason:    *reason,
		ProjectID: pd.Project.ID,
		StepID:    stepSlug,
		CreatedAt: now,
	}

	if err := st.SaveBlocker(blocker); err != nil {
		return fmt.Errorf("save blocker: %w", err)
	}

	// SaveBlocker already set step status to StepBlocked and saved it

	pd.Project.UpdatedAt = now
	st.SaveProject(pd.Project)

	fmt.Printf("Blocker %q added to step %q in project #%d.\n", id, stepSlug, pd.Project.Number)
	fmt.Printf("  pm blocker resolve %d %s %s\n", pd.Project.Number, stepSlug, id)
	return nil
}

func cmdBlockerResolve(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: pm blocker resolve <project-id> <step-slug> <blocker-id>")
	}

	ref, stepSlug, blockerID := args[0], args[1], args[2]

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

	// Find the blocker in the step using index (avoid range-copy confusion)
	for _, s := range pd.Steps {
		if s.ID == stepSlug {
			for j := range s.Blockers {
				if s.Blockers[j].ID == blockerID {
					s.Blockers[j].Status = types.BlockerResolved
					if err := st.SaveBlocker(s.Blockers[j]); err != nil {
						return fmt.Errorf("save blocker: %w", err)
					}

					// If no more active blockers in the step, unblock it
					stillBlocked := false
					for _, remaining := range pd.Steps {
						if remaining.ID == stepSlug {
							for _, rb := range remaining.Blockers {
								if rb.ID != blockerID && (rb.Status == types.BlockerWaiting || rb.Status == types.BlockerActive) {
									stillBlocked = true
									break
								}
							}
							break
						}
					}
					if !stillBlocked {
						// Reload step and set to todo (SaveBlocker set it to blocked)
						steps, _ := st.GetSteps(pd.Project.ID)
						for i, step := range steps {
							if step.ID == stepSlug {
								steps[i].Status = types.StepTodo
								st.SaveStep(steps[i])
								break
							}
						}
					}

					pd.Project.UpdatedAt = types.NowISO()
					st.SaveProject(pd.Project)

					fmt.Printf("Blocker %q resolved in step %q (project #%d).\n", blockerID, stepSlug, pd.Project.Number)
					return nil
				}
			}
			return fmt.Errorf("blocker %q not found in step %q", blockerID, stepSlug)
		}
	}

	return fmt.Errorf("step %q not found in project #%d", stepSlug, pd.Project.Number)
}

func cmdBlockerList(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm blocker list <project-id>")
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

	hasBlockers := false
	for _, s := range pd.Steps {
		if len(s.Blockers) > 0 {
			if !hasBlockers {
				fmt.Printf("Blockers for #%d %s:\n", pd.Project.Number, pd.Project.Title)
				fmt.Println(strings.Repeat("-", 60))
				hasBlockers = true
			}
			fmt.Printf("  Step %q (slug: %s):\n", s.Title, s.ID)
			for _, b := range s.Blockers {
				fmt.Printf("    [%-8s] %s  (%s)\n", b.Status, b.Title, b.ID)
				if b.Reason != "" {
					fmt.Printf("           └─ %s\n", b.Reason)
				}
			}
		}
	}

	if !hasBlockers {
		fmt.Printf("No blockers in project #%d (%s).\n", pd.Project.Number, pd.Project.Title)
	}

	return nil
}
