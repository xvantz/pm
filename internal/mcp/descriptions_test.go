package mcp

import (
	"strings"
	"testing"
	"unicode"

	"github.com/xvantz/pm/internal/store"
)

func registeredTools(t *testing.T) []Tool {
	t.Helper()
	s := NewServer("test", "1.0")
	RegisterPMTools(s, store.NewMockStore())
	if len(s.tools) == 0 {
		t.Fatal("no tools registered")
	}
	return s.tools
}

// descriptionsSnapshot pins the router texts: the spec (tool descriptions
// navigate) promises each description carries its route, so a revert to a bare
// one-liner must fail loudly here instead of silently degrading the catalogue.
var descriptionsSnapshot = map[string][]string{
	"list_projects":   {"Start here", "get_project"},
	"get_project":     {"get_step"},
	"get_step":        {"list_steps"},
	"add_project":     {"add_step", "close_project"},
	"add_step":        {"start_step", "review_step", "done_step"},
	"start_step":      {"review_step"},
	"review_step":     {"done_step", "add_blocker"},
	"done_step":       {"review_step", "close_project"},
	"add_blocker":     {"resolve_blocker"},
	"resolve_blocker": {"list_steps", "get_step"},
	"add_decision":    {"list_decisions", "close_project"},
	"get_briefing":    {"get_project"},
	"list_steps":      {"get_step"},
	"list_blockers":   {"list_steps", "get_step"},
	"list_decisions":  {"add_decision"},
	"close_project":   {"confirm"},
	"delete_project":  {"trash_restore", "close_project"},
	"trash_list":      {"trash_restore"},
	"trash_restore":   {"trash_list"},
}

func TestToolDescriptionsNavigate(t *testing.T) {
	for _, tl := range registeredTools(t) {
		name := tl.Name
		desc := tl.Description
		if desc == "" {
			t.Errorf("%s: empty description", name)
			continue
		}
		// Front-load: starts with a capital verb, not lowercase filler.
		if r := rune(desc[0]); !unicode.IsUpper(r) {
			t.Errorf("%s: description must start with a capital verb, got %q", name, desc)
		}
		// Catalogue budget: tools/list carries every description on each call.
		if len(desc) > 300 {
			t.Errorf("%s: description %d chars exceeds 300 cap", name, len(desc))
		}
		for _, want := range descriptionsSnapshot[name] {
			if !strings.Contains(desc, want) {
				t.Errorf("%s: description must mention %q, got %q", name, want, desc)
			}
		}
	}
}

func TestStepIdParamsNameSource(t *testing.T) {
	for _, tl := range registeredTools(t) {
		schema := string(tl.InputSchema)
		for _, param := range []string{"step_id", "blocker_id"} {
			if strings.Contains(schema, `"`+param+`"`) && !strings.Contains(schema, "list_steps") && !strings.Contains(schema, "get_step") {
				t.Errorf("%s: schema for %s must point at list_steps or get_step", tl.Name, param)
			}
		}
	}
}
