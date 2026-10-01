package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

// No dead language: a hint naming a `pm` shell command is unusable to an MCP
// agent, which has tools, not a shell. Every Next/Hint must name a real tool.
func TestResponsesNameToolsNotCLI(t *testing.T) {
	for _, tl := range registeredTools(t) {
		if strings.Contains(tl.Description, "pm ") {
			t.Errorf("%s: description speaks CLI, got %q", tl.Name, tl.Description)
		}
	}

	st := store.NewMockStore()
	ctx := context.Background()
	responses := map[string]string{}

	call := func(tool, args string) string {
		t.Helper()
		var out string
		var err error
		switch tool {
		case "add_project":
			out, err = handleAddProject(st, ctx, json.RawMessage(args))
		case "add_step":
			out, err = handleAddStep(st, ctx, json.RawMessage(args))
		case "start_step":
			out, err = handleStartStep(st, ctx, json.RawMessage(args))
		case "review_step":
			out, err = handleReviewStep(st, ctx, json.RawMessage(args))
		case "done_step":
			out, err = handleDoneStep(st, ctx, json.RawMessage(args))
		case "add_blocker":
			out, err = handleAddBlocker(st, ctx, json.RawMessage(args))
		case "resolve_blocker":
			out, err = handleResolveBlocker(st, ctx, json.RawMessage(args))
		case "add_decision":
			out, err = handleAddDecision(st, ctx, json.RawMessage(args))
		case "get_step":
			out, err = handleGetStep(st, ctx, json.RawMessage(args))
		case "get_project":
			out, err = handleGetProject(st, ctx, json.RawMessage(args))
		case "delete_project":
			out, err = handleDeleteProject(st, ctx, json.RawMessage(args))
		default:
			t.Fatalf("unknown tool %s", tool)
		}
		if err != nil {
			t.Fatalf("%s %s: error = %v", tool, args, err)
		}
		return out
	}

	responses["add_project"] = call("add_project", `{"title":"Hint Probe"}`)
	pn := projectNumber(t, responses["add_project"])
	pid := `"` + pn + `"`

	responses["add_step"] = call("add_step", `{"project_id":`+pid+`,"title":"probe step"}`)
	responses["get_step_todo"] = call("get_step", `{"project_id":`+pid+`,"step_id":"probe-step"}`)
	responses["start_step"] = call("start_step", `{"project_id":`+pid+`,"step_id":"probe-step"}`)
	responses["review_step"] = call("review_step", `{"project_id":`+pid+`,"step_id":"probe-step"}`)
	responses["done_step"] = call("done_step", `{"project_id":`+pid+`,"step_id":"probe-step"}`)
	responses["get_step"] = call("get_step", `{"project_id":`+pid+`,"step_id":"probe-step"}`)
	responses["get_project"] = call("get_project", `{"project_id":`+pid+`}`)

	call("add_step", `{"project_id":`+pid+`,"title":"blocked probe"}`)
	responses["add_blocker"] = call("add_blocker", `{"project_id":`+pid+`,"step_id":"blocked-probe","title":"no budget","reason":"waiting"}`)
	responses["get_project_blocked"] = call("get_project", `{"project_id":`+pid+`}`)
	responses["get_step_blocked"] = call("get_step", `{"project_id":`+pid+`,"step_id":"blocked-probe"}`)
	responses["resolve_blocker"] = call("resolve_blocker", `{"project_id":`+pid+`,"step_id":"blocked-probe","blocker_id":"no-budget"}`)
	responses["get_project_open"] = call("get_project", `{"project_id":`+pid+`}`)
	responses["add_decision"] = call("add_decision", `{"project_id":`+pid+`,"title":"probe why"}`)
	responses["delete_project"] = call("delete_project", `{"project_id":`+pid+`}`)

	names := map[string]bool{}
	for _, tl := range registeredTools(t) {
		names[tl.Name] = true
	}
	mustName := map[string][]string{
		"add_project": {"add_step"}, "add_step": {"start_step"},
		"start_step": {"review_step"}, "review_step": {"done_step"},
		"done_step": {"get_project"}, "add_blocker": {"resolve_blocker"},
		"resolve_blocker": {"start_step"}, "add_decision": {"list_decisions"},
		"get_step": {"get_project"}, "get_step_todo": {"start_step"}, "get_step_blocked": {"resolve_blocker"},
		"get_project": {"close_project"}, "get_project_open": {"list_steps"},
		"get_project_blocked": {"list_blockers"},
		"delete_project":      {"trash_restore"},
	}
	for tool, out := range responses {
		for _, cli := range []string{"pm step", "pm list_", "pm blocker", "pm project", "pm add"} {
			if strings.Contains(out, cli) {
				t.Errorf("%s: response speaks CLI (%q): %q", tool, cli, out)
			}
		}
		for _, want := range mustName[tool] {
			if !names[want] {
				t.Fatalf("test bug: %q is not a registered tool", want)
			}
			if !strings.Contains(out, want) {
				t.Errorf("%s: response must name %q, got %q", tool, want, out)
			}
		}
	}
}

// Every covered error states the fix, not just the fact.
func TestErrorsAreActionable(t *testing.T) {
	st := store.NewMockStore()
	ctx := context.Background()

	cases := []struct {
		name string
		call func() error
		want []string
	}{
		{"unknown step names list_steps", func() error {
			_, err := handleGetStep(st, ctx, json.RawMessage(`{"project_id":"1","step_id":"ghost"}`))
			return err
		}, []string{"ghost", "list_steps"}},
		{"missing step_id points at list", func() error {
			_, err := handleGetStep(st, ctx, json.RawMessage(`{"project_id":"1"}`))
			return err
		}, []string{"step_id", "list_steps"}},
		{"duplicate step offers reuse", func() error {
			_, err := handleAddStep(st, ctx, json.RawMessage(`{"project_id":"1","title":"Setup Caddy"}`))
			return err
		}, []string{"already exists", "reuse"}},
		{"done without review names review_step", func() error {
			_, err := handleDoneStep(st, ctx, json.RawMessage(`{"project_id":"1","step_id":"setup-caddy"}`))
			return err
		}, []string{"review", "review_step"}},
		{"unknown blocker names list_blockers", func() error {
			_, err := handleResolveBlocker(st, ctx, json.RawMessage(`{"project_id":"1","step_id":"configure-dns","blocker_id":"ghost"}`))
			return err
		}, []string{"ghost", "list_blockers"}},
		{"missing project_id names list_projects", func() error {
			_, err := handleCloseProject(st, ctx, json.RawMessage(`{"confirm":true,"reason":"x"}`))
			return err
		}, []string{"project_id", "list_projects"}},
		{"confirm without reason names the flow", func() error {
			_, err := handleCloseProject(st, ctx, json.RawMessage(`{"project_id":"1","confirm":true}`))
			return err
		}, []string{"reason", "confirm"}},
		{"malformed args name what is needed", func() error {
			_, err := handleGetStep(st, ctx, json.RawMessage(`{"project_id":`))
			return err
		}, []string{"Need:", "step_id"}},
		{"unknown project names list_projects", func() error {
			_, err := handleGetProject(st, ctx, json.RawMessage(`{"project_id":"9999"}`))
			return err
		}, []string{"list_projects"}},
	}

	for _, tc := range cases {
		err := tc.call()
		if err == nil {
			t.Errorf("%s: expected an error, got success", tc.name)
			continue
		}
		for _, want := range tc.want {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: error must contain %q, got %q", tc.name, want, err.Error())
			}
		}
	}
}

// The all-done summary routes to close_project: the terminal state still
// names its one next action.
func TestAllDoneSummaryNamesClose(t *testing.T) {
	pd := types.ProjectData{
		Project: types.Project{Number: 99, Title: "T", Status: types.StatusCompleted},
		Steps:   []types.Step{{ID: "a", Title: "A", Status: types.StepDone}},
	}
	data, err := json.Marshal(buildProjectSummary(pd))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "close_project") {
		t.Errorf("all-done summary must name close_project, got %s", data)
	}
}

func projectNumber(t *testing.T, addProjectOut string) string {
	t.Helper()
	// add_project answers "Project #N ...", read N back for later calls.
	var n string
	for _, tok := range strings.Fields(addProjectOut) {
		if strings.HasPrefix(tok, "#") {
			n = strings.Trim(tok, "#\",.")
			break
		}
	}
	if n == "" {
		t.Fatalf("cannot read project number from %q", addProjectOut)
	}
	return n
}
