package cli

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/xvantz/pm/internal/types"
)

func cmdProject(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm project <create|list|show|goal|tag|status> [args]")
	}

	sub := args[0]
	switch sub {
	case "create":
		return cmdProjectCreate(args[1:])
	case "list":
		return cmdProjectList(args[1:])
	case "show":
		return cmdProjectShow(args[1:])
	case "goal":
		return cmdProjectGoal(args[1:])
	case "tag":
		return cmdProjectTag(args[1:])
	case "status":
		return cmdProjectStatus(args[1:])
	default:
		return fmt.Errorf("unknown project subcommand: %s", sub)
	}
}

func cmdProjectCreate(args []string) error {
	if len(args) < 1 || args[0] == "" {
		return fmt.Errorf("usage: pm project create <title>")
	}

	title := strings.Join(args, " ")
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title cannot be empty")
	}

	st, err := openStore()
	if err != nil {
		return err
	}

	id := uuid.Must(uuid.NewV7()).String()
	number, err := st.NextNumber()
	if err != nil {
		return fmt.Errorf("next number: %w", err)
	}

	now := types.NowISO()
	p := types.Project{
		ID:        id,
		Number:    number,
		Title:     title,
		Status:    types.StatusIdea,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := st.SaveProject(p); err != nil {
		return fmt.Errorf("save project: %w", err)
	}

	fmt.Printf("Created project #%d: %q\n", number, title)
	fmt.Println()
	fmt.Printf("  pm project goal %d \"...\"    # add a goal\n", number)
	fmt.Printf("  pm project tag %d ...        # add tags\n", number)
	fmt.Printf("  pm project show %d           # view details\n", number)
	fmt.Printf("  pm add step %d \"...\"        # add first step\n", number)
	return nil
}

func cmdProjectList(args []string) error {
	st, err := openStore()
	if err != nil {
		return err
	}

	projects, err := st.ListProjects()
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects found. Create one:")
		fmt.Println("  pm project create \"My Project\"")
		return nil
	}

	fmt.Printf("%-4s %-30s %-12s %s\n", "#", "Title", "Status", "Tags")
	fmt.Println(strings.Repeat("-", 80))
	for _, p := range projects {
		tags := strings.Join(p.Tags, ", ")
		fmt.Printf("%-4d %-30s %-12s %s\n", p.Number, p.Title, string(p.Status), tags)
	}
	return nil
}

func cmdProjectShow(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm project show <id>")
	}

	st, err := openStore()
	if err != nil {
		return err
	}

	pd, err := st.ResolveProject(args[0])
	if err != nil {
		return err
	}

	p := pd.Project
	fmt.Printf("#%d  %s\n", p.Number, p.Title)
	fmt.Printf("ID:      %s\n", p.ID)
	fmt.Printf("Status:  %s\n", p.Status)
	if p.Goal != "" {
		fmt.Printf("Goal:    %s\n", p.Goal)
	}
	if len(p.Tags) > 0 {
		fmt.Printf("Tags:    %s\n", strings.Join(p.Tags, ", "))
	}
	fmt.Printf("Created: %s\n", p.CreatedAt)
	fmt.Printf("Updated: %s\n", p.UpdatedAt)
	if p.CompletedAt != "" {
		fmt.Printf("Done:    %s\n", p.CompletedAt)
	}

	if len(pd.Steps) > 0 {
		fmt.Println()
		fmt.Println("Steps:")
		for _, s := range pd.Steps {
			fmt.Printf("  [%s] %s — %s\n", s.Status, s.Title, s.ID)
			for _, b := range s.Blockers {
				fmt.Printf("     🚫 [%s] %s — %s\n", b.Status, b.Title, b.ID)
			}
		}
	}
	if len(pd.Decisions) > 0 {
		fmt.Println()
		fmt.Println("Decisions:")
		for _, d := range pd.Decisions {
			r := d.Reason
			if r == "" {
				r = "—"
			}
			fmt.Printf("  %s (%s) — %s\n", d.Title, d.Date, r)
		}
	}
	return nil
}

func cmdProjectGoal(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: pm project goal <id> <goal text>")
	}
	return updateProject(args[0], func(p *types.Project) {
		p.Goal = strings.Join(args[1:], " ")
	})
}

func cmdProjectTag(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: pm project tag <id> <tag> [tag ...]")
	}

	return updateProject(args[0], func(p *types.Project) {
		existing := make(map[string]bool)
		for _, t := range p.Tags {
			existing[t] = true
		}
		for _, t := range args[1:] {
			if !existing[t] {
				p.Tags = append(p.Tags, t)
				existing[t] = true
			}
		}
	})
}

const validStatuses = "idea, active, paused, completed"

func cmdProjectStatus(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: pm project status <id> <%s>", validStatuses)
	}

	status := types.ProjectStatus(args[1])
	switch status {
	case types.StatusIdea, types.StatusActive, types.StatusPaused, types.StatusCompleted:
		// valid
	default:
		return fmt.Errorf("invalid status %q, use one of: %s", status, validStatuses)
	}

	return updateProject(args[0], func(p *types.Project) {
		p.Status = status
		if status == types.StatusCompleted {
			p.CompletedAt = types.NowISO()
		} else {
			p.CompletedAt = ""
		}
	})
}

func updateProject(ref string, fn func(p *types.Project)) error {
	st, err := openStore()
	if err != nil {
		return err
	}

	pd, err := st.ResolveProject(ref)
	if err != nil {
		return fmt.Errorf("resolve %q: %w", ref, err)
	}

	fn(&pd.Project)
	pd.Project.UpdatedAt = types.NowISO()

	if err := st.SaveProject(pd.Project); err != nil {
		return fmt.Errorf("save project: %w", err)
	}

	fmt.Printf("Project #%d updated.\n", pd.Project.Number)
	return nil
}
