package cli

import (
	"fmt"

	"github.com/xvantz/pm/internal/types"
)

func cmdDel(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm del <project|step|blocker|decision> [args]")
	}

	sub := args[0]
	switch sub {
	case "project":
		return cmdDelProject(args[1:])
	case "step":
		return cmdDelStep(args[1:])
	case "blocker":
		return cmdDelBlocker(args[1:])
	case "decision":
		return cmdDelDecision(args[1:])
	default:
		return fmt.Errorf("unknown: pm del %s\n  Try: project, step, blocker, decision", sub)
	}
}

func cmdDelProject(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm del project <id>")
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

	if err := st.DeleteProject(pd.Project.ID); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}

	fmt.Printf("Project #%d (%s) deleted.\n", pd.Project.Number, pd.Project.Title)
	return nil
}

func cmdDelStep(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: pm del step <project-id> <step-id>")
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
	if pd == nil {
		return fmt.Errorf("project %q not found", ref)
	}

	if err := st.DeleteStep(pd.Project.ID, stepID); err != nil {
		return fmt.Errorf("delete step: %w", err)
	}

	pd.Project.UpdatedAt = types.NowISO()
	st.SaveProject(pd.Project)

	fmt.Printf("Step %q deleted from project #%d.\n", stepID, pd.Project.Number)
	return nil
}

func cmdDelBlocker(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: pm del blocker <project-id> <step-id> <blocker-id>")
	}

	ref, stepID, blockerID := args[0], args[1], args[2]

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

	if err := st.DeleteBlocker(pd.Project.ID, stepID, blockerID); err != nil {
		return fmt.Errorf("delete blocker: %w", err)
	}

	pd.Project.UpdatedAt = types.NowISO()
	st.SaveProject(pd.Project)

	fmt.Printf("Blocker %q deleted from step %q (project #%d).\n", blockerID, stepID, pd.Project.Number)
	return nil
}

func cmdDelDecision(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: pm del decision <project-id> <decision-id>")
	}

	ref, decisionID := args[0], args[1]

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

	if err := st.DeleteDecision(pd.Project.ID, decisionID); err != nil {
		return fmt.Errorf("delete decision: %w", err)
	}

	pd.Project.UpdatedAt = types.NowISO()
	st.SaveProject(pd.Project)

	fmt.Printf("Decision %q deleted from project #%d.\n", decisionID, pd.Project.Number)
	return nil
}
