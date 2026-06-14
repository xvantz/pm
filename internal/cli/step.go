package cli

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/xvantz/pm/internal/domain"
	"github.com/xvantz/pm/internal/slug"
	"github.com/xvantz/pm/internal/types"
)

func cmdStep(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm step <add|start|review|done|list> [args]")
	}

	sub := args[0]
	switch sub {
	case "add":
		return cmdStepAdd(args[1:])
	case "start":
		return cmdStepStart(args[1:])
	case "review":
		return cmdStepReview(args[1:])
	case "done":
		return cmdStepDone(args[1:])
	case "list":
		return cmdStepList(args[1:])
	default:
		return fmt.Errorf("unknown step subcommand: %s", sub)
	}
}

func cmdStepAdd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: pm step add <project-id> <title>")
	}

	ref, title := args[0], strings.Join(args[1:], " ")

	st, err := openStore()
	if err != nil {
		return err
	}

	pd, err := st.ResolveProject(ref)
	if err != nil {
		return fmt.Errorf("resolve %q: %w", ref, err)
	}

	id := slug.Of(title)
	if id == "" {
		return fmt.Errorf("invalid step title: %q", title)
	}

	for _, s := range pd.Steps {
		if s.ID == id {
			return fmt.Errorf("step %q already exists in project #%d", id, pd.Project.Number)
		}
	}

	now := types.NowISO()
	step := types.Step{
		ID: id, Title: title, Status: types.StepTodo,
		ProjectID: pd.Project.ID, CreatedAt: now, UpdatedAt: now,
	}

	if err := st.SaveStep(step); err != nil {
		return fmt.Errorf("save step: %w", err)
	}

	pd.Project.UpdatedAt = now
	if err := st.SaveProject(pd.Project); err != nil {
		slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
	}

	fmt.Printf("Step %q added to project #%d.\n", id, pd.Project.Number)
	fmt.Printf("  pm step start %d %s       # mark as in_progress\n", pd.Project.Number, id)
	fmt.Printf("  pm step review %d %s\n", pd.Project.Number, id)
	fmt.Printf("  pm step done %d %s    # only after review\n", pd.Project.Number, id)
	return nil
}

func cmdStepStart(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: pm step start <project-id> <step-id>")
	}

	ref, stepID := args[0], args[1]

	st, err := openStore()
	if err != nil {
		return err
	}

	pd, err := st.ResolveProject(ref)
	if err != nil {
		return fmt.Errorf("resolve %q: %w", ref, err)
	}

	for i, s := range pd.Steps {
		if s.ID == stepID {
			if err := domain.ValidateStepStart(s); err != nil {
				return err
			}

			domain.StepStatusChange(&pd.Steps[i], types.StepInProgress, types.NowISO())

			if err := st.SaveStep(pd.Steps[i]); err != nil {
				return fmt.Errorf("save step: %w", err)
			}

			pd.Project.UpdatedAt = types.NowISO()
			if err := st.SaveProject(pd.Project); err != nil {
				slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
			}

			fmt.Printf("Step %q started (in_progress) in project #%d.\n", stepID, pd.Project.Number)
			return nil
		}
	}

	return fmt.Errorf("step %q not found in project #%d", stepID, pd.Project.Number)
}

func cmdStepReview(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: pm step review <project-id> <step-id>")
	}

	ref, stepID := args[0], args[1]

	st, err := openStore()
	if err != nil {
		return err
	}

	pd, err := st.ResolveProject(ref)
	if err != nil {
		return fmt.Errorf("resolve %q: %w", ref, err)
	}

	for i, s := range pd.Steps {
		if s.ID == stepID {
			if err := domain.ValidateStepReview(s); err != nil {
				return err
			}

			domain.StepStatusChange(&pd.Steps[i], types.StepReview, types.NowISO())

			if err := st.SaveStep(pd.Steps[i]); err != nil {
				return fmt.Errorf("save step: %w", err)
			}

			pd.Project.UpdatedAt = types.NowISO()
			if err := st.SaveProject(pd.Project); err != nil {
				slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
			}

			fmt.Printf("Step %q sent to review in project #%d.\n", stepID, pd.Project.Number)
			return nil
		}
	}

	return fmt.Errorf("step %q not found in project #%d", stepID, pd.Project.Number)
}

func cmdStepDone(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: pm step done <project-id> <step-id>")
	}

	ref, stepID := args[0], args[1]

	st, err := openStore()
	if err != nil {
		return err
	}

	pd, err := st.ResolveProject(ref)
	if err != nil {
		return fmt.Errorf("resolve %q: %w", ref, err)
	}

	for i, s := range pd.Steps {
		if s.ID == stepID {
			if err := domain.ValidateStepDone(s); err != nil {
				return err
			}

			domain.StepStatusChange(&pd.Steps[i], types.StepDone, types.NowISO())

			if err := st.SaveStep(pd.Steps[i]); err != nil {
				return fmt.Errorf("save step: %w", err)
			}

			pd.Project.UpdatedAt = types.NowISO()
			if err := st.SaveProject(pd.Project); err != nil {
				slog.Warn("update project timestamp", "project", pd.Project.ID, "error", err)
			}

			fmt.Printf("Step %q marked done in project #%d.\n", stepID, pd.Project.Number)
			return nil
		}
	}

	return fmt.Errorf("step %q not found in project #%d", stepID, pd.Project.Number)
}

func cmdStepList(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm step list <project-id>")
	}

	st, err := openStore()
	if err != nil {
		return err
	}

	pd, err := st.ResolveProject(args[0])
	if err != nil {
		return fmt.Errorf("resolve %q: %w", args[0], err)
	}

	if len(pd.Steps) == 0 {
		fmt.Printf("No steps in project #%d (%s).\n  pm step add %d \"...\"\n",
			pd.Project.Number, pd.Project.Title, pd.Project.Number)
		return nil
	}

	fmt.Printf("Steps for #%d %s:\n", pd.Project.Number, pd.Project.Title)
	fmt.Println(strings.Repeat("-", 60))
	for _, s := range pd.Steps {
		fmt.Printf("  [%-11s] %s  (%s)\n", s.Status, s.Title, s.ID)
		for _, b := range s.Blockers {
			fmt.Printf("     🚫 [%s] %s\n", b.Status, b.Title)
		}
	}
	return nil
}
