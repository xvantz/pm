package cli

import (
	"fmt"
)

func cmdAdd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm add <project|step|blocker|decision> [args]")
	}

	sub := args[0]
	switch sub {
	case "project":
		return cmdAddProject(args[1:])
	case "step":
		return cmdAddStep(args[1:])
	case "blocker":
		return cmdAddBlocker(args[1:])
	case "decision":
		return cmdAddDecision(args[1:])
	default:
		return fmt.Errorf("unknown: pm add %s\n  Try: project, step, blocker, decision", sub)
	}
}

func cmdAddProject(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pm add project <title>")
	}
	return cmdProjectCreate(args)
}

func cmdAddStep(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: pm add step <project-id> <title>")
	}
	return cmdStepAdd(args)
}

func cmdAddBlocker(args []string) error {
	// Matches the signature: pm blocker add [--reason ...] <project-id> <step-slug> <title>
	return cmdBlockerAdd(args)
}

func cmdAddDecision(args []string) error {
	// Matches the signature: pm decision add [--reason ...] <project-id> <title>
	return cmdDecisionAdd(args)
}
