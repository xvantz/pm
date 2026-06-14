package cli

import (
	"flag"
	"fmt"
	"log/slog"
	"strings"

	"github.com/xvantz/pm/internal/slug"
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

	id := slug.Of(title)
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
		UpdatedAt: now,
	}

	if err := st.SaveBlocker(blocker); err != nil {
		return fmt.Errorf("save blocker: %w", err)
	}

	// SaveBlocker already set step status to StepBlocked and saved it

	pd.Project.UpdatedAt = now
	if err := st.SaveProject(pd.Project); err != nil {
		slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
	}

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

	// Find the blocker by index — avoid range-copy confusion
	stepIdx := -1
	blockerIdx := -1
	for i := range pd.Steps {
		if pd.Steps[i].ID == stepSlug {
			stepIdx = i
			for j := range pd.Steps[i].Blockers {
				if pd.Steps[i].Blockers[j].ID == blockerID {
					blockerIdx = j
					break
				}
			}
			break
		}
	}

	if stepIdx == -1 {
		return fmt.Errorf("step %q not found in project #%d", stepSlug, pd.Project.Number)
	}
	if blockerIdx == -1 {
		return fmt.Errorf("blocker %q not found in step %q", blockerID, stepSlug)
	}

	// Resolve the blocker in-place (updates pd.Steps so subsequent checks are correct)
	pd.Steps[stepIdx].Blockers[blockerIdx].Status = types.BlockerResolved
	pd.Steps[stepIdx].Blockers[blockerIdx].UpdatedAt = types.NowISO()

	if err := st.SaveBlocker(pd.Steps[stepIdx].Blockers[blockerIdx]); err != nil {
		return fmt.Errorf("save blocker: %w", err)
	}

	// If no more active blockers in the step, unblock it
	stillBlocked := false
	for _, b := range pd.Steps[stepIdx].Blockers {
		if b.Status == types.BlockerWaiting || b.Status == types.BlockerActive {
			stillBlocked = true
			break
		}
	}
	if !stillBlocked {
		pd.Steps[stepIdx].Status = types.StepTodo
		if err := st.SaveStep(pd.Steps[stepIdx]); err != nil {
			return fmt.Errorf("save step (unblock): %w", err)
		}
	}

	pd.Project.UpdatedAt = types.NowISO()
	if err := st.SaveProject(pd.Project); err != nil {
		slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
	}

	fmt.Printf("Blocker %q resolved in step %q (project #%d).\n", blockerID, stepSlug, pd.Project.Number)
	return nil
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
