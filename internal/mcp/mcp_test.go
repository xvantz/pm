package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/xvantz/pm/internal/store"
	"github.com/xvantz/pm/internal/types"
)

// readMCPResponse parses a Content-Length framed MCP response from buf.
func readMCPResponse(t *testing.T, buf *bytes.Buffer) jsonrpcMessage {
	t.Helper()
	data := buf.Bytes()
	re := regexp.MustCompile(`Content-Length: (\d+)\r\n\r\n`)
	m := re.FindSubmatch(data)
	if m == nil {
		t.Fatalf("no Content-Length header in: %s", string(data))
	}
	length := 0
	fmt.Sscanf(string(m[1]), "%d", &length)
	bodyStart := len(m[0])
	body := data[bodyStart : bodyStart+length]

	var resp jsonrpcMessage
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal response: %v\nbody: %s", err, string(body))
	}
	return resp
}

// --- Server protocol tests ---

func TestServer_Initialize(t *testing.T) {
	s := NewServer("test", "1.0")
	var buf bytes.Buffer
	initialized := false

	s.handleMessage(
		json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`),
		&buf, &initialized,
	)

	resp := readMCPResponse(t, &buf)
	if resp.ID == nil || *resp.ID != 1 {
		t.Errorf("id = %v, want 1", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("result is nil")
	}
}

func TestServer_ToolsList(t *testing.T) {
	s := NewServer("test", "1.0")
	s.AddTool(Tool{
		Name:        "echo",
		Description: "Echo test",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		Handler: func(_ context.Context, args json.RawMessage) (string, error) {
			return string(args), nil
		},
	})

	var buf bytes.Buffer
	initialized := true // already initialized

	s.handleMessage(
		json.RawMessage(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`),
		&buf, &initialized,
	)

	resp := readMCPResponse(t, &buf)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Tools) != 1 {
		t.Fatalf("got %d tools, want 1", len(result.Tools))
	}
	if result.Tools[0].Name != "echo" {
		t.Errorf("tool name = %q, want %q", result.Tools[0].Name, "echo")
	}
}

func TestServer_ToolsCall(t *testing.T) {
	s := NewServer("test", "1.0")
	s.AddTool(Tool{
		Name:        "hello",
		Description: "Say hello",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`),
		Handler: func(_ context.Context, args json.RawMessage) (string, error) {
			var p struct{ Name string }
			json.Unmarshal(args, &p)
			return fmt.Sprintf("Hello, %s!", p.Name), nil
		},
	})

	var buf bytes.Buffer
	initialized := true

	s.handleMessage(
		json.RawMessage(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"hello","arguments":{"name":"World"}}}`),
		&buf, &initialized,
	)

	resp := readMCPResponse(t, &buf)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("got %d content items, want 1", len(result.Content))
	}
	if result.Content[0].Text != "Hello, World!" {
		t.Errorf("got %q, want %q", result.Content[0].Text, "Hello, World!")
	}
}

func TestServer_NotInitialized(t *testing.T) {
	s := NewServer("test", "1.0")
	var buf bytes.Buffer
	initialized := false

	// Should reject tools/list before initialized
	s.handleMessage(
		json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`),
		&buf, &initialized,
	)

	resp := readMCPResponse(t, &buf)
	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != -32000 {
		t.Errorf("error code = %d, want -32000", resp.Error.Code)
	}
}

func TestServer_UnknownTool(t *testing.T) {
	s := NewServer("test", "1.0")
	var buf bytes.Buffer
	initialized := true

	s.handleMessage(
		json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nonexistent","arguments":{}}}`),
		&buf, &initialized,
	)

	resp := readMCPResponse(t, &buf)
	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- Slug tests ---

func TestSlug_Simple(t *testing.T) {
	if slug("Hello World") != "hello-world" {
		t.Errorf("slug = %q, want %q", slug("Hello World"), "hello-world")
	}
}

func TestSlug_Cyrillic(t *testing.T) {
	s := slug("Настроить Caddy reverse proxy")
	want := "настроить-caddy-reverse-proxy"
	if s != want {
		t.Errorf("slug = %q, want %q", s, want)
	}
}

func TestSlug_SpecialChars(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"Test/Path", "test-path"},
		{"test_file.yaml", "test-file-yaml"},
		{"key:value", "key-value"},
		{"a,b,c", "a-b-c"},
		{"it's ok", "its-ok"},
		{`"quoted"`, "quoted"},
		{"(parens)", "parens"},
		{"back`tick`", "backtick"},
	}
	for _, c := range cases {
		got := slug(c.input)
		if got != c.want {
			t.Errorf("slug(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestSlug_TrimDashes(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"-leading", "leading"},
		{"trailing-", "trailing"},
		{"-both-", "both"},
	}
	for _, c := range cases {
		got := slug(c.input)
		if got != c.want {
			t.Errorf("slug(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestSlug_CollapseMultipleDashes(t *testing.T) {
	s := slug("foo   bar___baz")
	if s != "foo-bar-baz" {
		t.Errorf("slug = %q, want %q", s, "foo-bar-baz")
	}
}

func TestSlug_Empty(t *testing.T) {
	cases := []string{"", "'", `"`, "`", "'\"`"}
	for _, c := range cases {
		if slug(c) != "" {
			t.Errorf("slug(%q) should be empty, got %q", c, slug(c))
		}
	}
}

func TestSlug_NoTruncation(t *testing.T) {
	long := "abcdefghij-abcdefghij-abcdefghij-abcdefghij-abcdefghij-xxx"
	s := slug(long)
	if len(s) <= 50 {
		t.Errorf("slug truncated: len=%d, want > 50", len(s))
	}
	if s != long {
		t.Errorf("slug = %q, want %q", s, long)
	}
}

func TestSlug_Lowercase(t *testing.T) {
	if slug("HELLO WORLD") != "hello-world" {
		t.Errorf("slug = %q, want %q", slug("HELLO WORLD"), "hello-world")
	}
}

// --- Handler tests (with MockStore) ---

func TestHandleListProjects(t *testing.T) {
	st := store.NewMockStore()
	result, err := handleListProjects(st, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("handleListProjects error: %v", err)
	}
	if !strings.Contains(result, "Projects: 6 total") {
		t.Errorf("result missing '6 total': %s", result)
	}
	if !strings.Contains(result, "AdGuard Home") {
		t.Errorf("result missing 'AdGuard Home': %s", result)
	}
}

func TestHandleGetProject(t *testing.T) {
	st := store.NewMockStore()
	result, err := handleGetProject(st, json.RawMessage(`{"project_id":"1"}`))
	if err != nil {
		t.Fatalf("handleGetProject error: %v", err)
	}
	if !strings.Contains(result, "#1") {
		t.Errorf("result missing '#1': %s", result)
	}
	if !strings.Contains(result, "AdGuard Home") {
		t.Errorf("result missing 'AdGuard Home': %s", result)
	}
	if !strings.Contains(result, "configure-dns") {
		t.Errorf("result missing step 'configure-dns': %s", result)
	}
	if !strings.Contains(result, "router") {
		t.Errorf("result missing blocker 'router': %s", result)
	}

	// Not found
	_, err = handleGetProject(st, json.RawMessage(`{"project_id":"999"}`))
	if err == nil {
		t.Error("expected error for non-existent project, got nil")
	}
}

func TestHandleAddProject(t *testing.T) {
	st := store.NewMockStore()
	result, err := handleAddProject(st, json.RawMessage(`{"title":"New Test Project","goal":"Testing","tags":["test"]}`))
	if err != nil {
		t.Fatalf("handleAddProject error: %v", err)
	}
	if !strings.Contains(result, "Project #7") {
		t.Errorf("result missing '#7': %s", result)
	}
	if !strings.Contains(result, "New Test Project") {
		t.Errorf("result missing title: %s", result)
	}

	// Verify it was saved
	projects, _ := st.ListProjects()
	found := false
	for _, p := range projects {
		if p.Title == "New Test Project" {
			found = true
			if p.Goal != "Testing" {
				t.Errorf("goal = %q, want %q", p.Goal, "Testing")
			}
			if len(p.Tags) != 1 || p.Tags[0] != "test" {
				t.Errorf("tags = %v, want [test]", p.Tags)
			}
			if p.Number != 7 {
				t.Errorf("number = %d, want 7", p.Number)
			}
			break
		}
	}
	if !found {
		t.Error("new project not found in store")
	}

	// Empty title
	_, err = handleAddProject(st, json.RawMessage(`{"title":""}`))
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestHandleAddStep(t *testing.T) {
	st := store.NewMockStore()
	result, err := handleAddStep(st, json.RawMessage(`{"project_id":"1","title":"New Step"}`))
	if err != nil {
		t.Fatalf("handleAddStep error: %v", err)
	}
	if !strings.Contains(result, "new-step") {
		t.Errorf("result missing slug 'new-step': %s", result)
	}
	if !strings.Contains(result, "#1") {
		t.Errorf("result missing project #1: %s", result)
	}

	// Duplicate
	_, err = handleAddStep(st, json.RawMessage(`{"project_id":"1","title":"New Step"}`))
	if err == nil {
		t.Error("expected error for duplicate step")
	}

	// Non-existent project
	_, err = handleAddStep(st, json.RawMessage(`{"project_id":"999","title":"Step"}`))
	if err == nil {
		t.Error("expected error for non-existent project")
	}
}

func TestHandleStartStep(t *testing.T) {
	st := store.NewMockStore()
	result, err := handleStartStep(st, json.RawMessage(`{"project_id":"1","step_id":"vpn-access"}`))
	if err != nil {
		t.Fatalf("handleStartStep error: %v", err)
	}
	if !strings.Contains(result, "in_progress") {
		t.Errorf("result missing in_progress: %s", result)
	}

	// Already done step
	_, err = handleStartStep(st, json.RawMessage(`{"project_id":"1","step_id":"setup-caddy"}`))
	if err == nil {
		t.Error("expected error for starting a done step")
	}
}

func TestHandleReviewStep(t *testing.T) {
	st := store.NewMockStore()
	result, err := handleReviewStep(st, json.RawMessage(`{"project_id":"2","step_id":"review-spec"}`))
	if err != nil {
		t.Fatalf("handleReviewStep error: %v", err)
	}
	if !strings.Contains(result, "review") {
		t.Errorf("result missing review: %s", result)
	}
}

func TestHandleDoneStep(t *testing.T) {
	st := store.NewMockStore()

	// Must go through review first
	pd, _ := st.ResolveProject("2")
	for i, s := range pd.Steps {
		if s.ID == "review-spec" {
			pd.Steps[i].Status = types.StepReview
			st.SaveStep(pd.Steps[i])
			break
		}
	}

	result, err := handleDoneStep(st, json.RawMessage(`{"project_id":"2","step_id":"review-spec"}`))
	if err != nil {
		t.Fatalf("handleDoneStep error: %v", err)
	}
	if !strings.Contains(result, "done") {
		t.Errorf("result missing done: %s", result)
	}

	// Blocked step should fail
	_, err = handleDoneStep(st, json.RawMessage(`{"project_id":"1","step_id":"configure-dns"}`))
	if err == nil {
		t.Error("expected error for blocked step")
	}
}

func TestHandleAddBlocker(t *testing.T) {
	st := store.NewMockStore()
	result, err := handleAddBlocker(st, json.RawMessage(`{"project_id":"1","step_id":"test-dns","title":"No access","reason":"Need admin credentials"}`))
	if err != nil {
		t.Fatalf("handleAddBlocker error: %v", err)
	}
	if !strings.Contains(result, "no-access") {
		t.Errorf("result missing slug 'no-access': %s", result)
	}

	// Verify blocker was added
	pd, _ := st.ResolveProject("1")
	found := false
	for _, s := range pd.Steps {
		if s.ID == "test-dns" {
			for _, b := range s.Blockers {
				if b.ID == "no-access" {
					found = true
					if b.Reason != "Need admin credentials" {
						t.Errorf("reason = %q, want %q", b.Reason, "Need admin credentials")
					}
				}
			}
		}
	}
	if !found {
		t.Error("blocker not found in step")
	}

	// Non-existent step
	_, err = handleAddBlocker(st, json.RawMessage(`{"project_id":"1","step_id":"nonexistent","title":"Blocker"}`))
	if err == nil {
		t.Error("expected error for non-existent step")
	}
}

func TestHandleResolveBlocker(t *testing.T) {
	st := store.NewMockStore()

	// Resolve the router blocker on AGH's configure-dns step
	result, err := handleResolveBlocker(st, json.RawMessage(`{"project_id":"1","step_id":"configure-dns","blocker_id":"router"}`))
	if err != nil {
		t.Fatalf("handleResolveBlocker error: %v", err)
	}
	if !strings.Contains(result, "router") {
		t.Errorf("result missing blocker id: %s", result)
	}

	// Verify it's resolved
	pd, _ := st.ResolveProject("1")
	for _, s := range pd.Steps {
		if s.ID == "configure-dns" {
			if s.Status != types.StepTodo {
				t.Errorf("step should be unblocked (todo), got %s", s.Status)
			}
			for _, b := range s.Blockers {
				if b.ID == "router" {
					if b.Status != types.BlockerResolved {
						t.Errorf("blocker status = %s, want resolved", b.Status)
					}
				}
			}
		}
	}

	// Non-existent blocker
	_, err = handleResolveBlocker(st, json.RawMessage(`{"project_id":"1","step_id":"configure-dns","blocker_id":"nonexistent"}`))
	if err == nil {
		t.Error("expected error for non-existent blocker")
	}
}

func TestHandleAddDecision(t *testing.T) {
	st := store.NewMockStore()
	result, err := handleAddDecision(st, json.RawMessage(`{"project_id":"3","title":"Use Python","reason":"Better ecosystem"}`))
	if err != nil {
		t.Fatalf("handleAddDecision error: %v", err)
	}
	if !strings.Contains(result, "use-python") {
		t.Errorf("result missing slug: %s", result)
	}

	// Duplicate
	_, err = handleAddDecision(st, json.RawMessage(`{"project_id":"3","title":"Use Python"}`))
	if err == nil {
		t.Error("expected error for duplicate decision")
	}
}

func TestHandleGetBriefing(t *testing.T) {
	st := store.NewMockStore()

	// Full briefing
	result, err := handleGetBriefing(st, json.RawMessage(`{"date":"2026-06-14"}`))
	if err != nil {
		t.Fatalf("handleGetBriefing error: %v", err)
	}
	if !strings.Contains(result, "PM Briefing") {
		t.Errorf("result missing 'PM Briefing': %s", result)
	}
	if !strings.Contains(result, "AdGuard Home") {
		t.Errorf("result missing project: %s", result)
	}

	// Single project briefing
	result, err = handleGetBriefing(st, json.RawMessage(`{"date":"2026-06-14","project_id":"1"}`))
	if err != nil {
		t.Fatalf("handleGetBriefing(single) error: %v", err)
	}
	if !strings.Contains(result, "AdGuard Home") {
		t.Errorf("result missing project: %s", result)
	}
}

func TestHandleListSteps(t *testing.T) {
	st := store.NewMockStore()
	result, err := handleListSteps(st, json.RawMessage(`{"project_id":"1"}`))
	if err != nil {
		t.Fatalf("handleListSteps error: %v", err)
	}
	if !strings.Contains(result, "Steps for #1") {
		t.Errorf("result missing header: %s", result)
	}
	if !strings.Contains(result, "configure-dns") {
		t.Errorf("result missing step: %s", result)
	}

	// Non-existent project
	_, err = handleListSteps(st, json.RawMessage(`{"project_id":"999"}`))
	if err == nil {
		t.Error("expected error for non-existent project")
	}
}

func TestHandleListBlockers(t *testing.T) {
	st := store.NewMockStore()
	result, err := handleListBlockers(st, json.RawMessage(`{"project_id":"1"}`))
	if err != nil {
		t.Fatalf("handleListBlockers error: %v", err)
	}
	if !strings.Contains(result, "Blockers for #1") {
		t.Errorf("result missing header: %s", result)
	}
	if !strings.Contains(result, "router") {
		t.Errorf("result missing blocker: %s", result)
	}
}

func TestHandleListDecisions(t *testing.T) {
	st := store.NewMockStore()
	result, err := handleListDecisions(st, json.RawMessage(`{"project_id":"1"}`))
	if err != nil {
		t.Fatalf("handleListDecisions error: %v", err)
	}
	if !strings.Contains(result, "Decisions for #1") {
		t.Errorf("result missing header: %s", result)
	}
	if !strings.Contains(result, "migrate-docker") {
		t.Errorf("result missing decision: %s", result)
	}
}

func TestHandleListBlockers_None(t *testing.T) {
	st := store.NewMockStore()
	result, err := handleListBlockers(st, json.RawMessage(`{"project_id":"2"}`))
	if err != nil {
		t.Fatalf("handleListBlockers(no blockers) error: %v", err)
	}
	if !strings.Contains(result, "No blockers") {
		t.Errorf("result mismatch: %s", result)
	}
}

func TestHandleStartStep_AlreadyInProgress(t *testing.T) {
	st := store.NewMockStore()
	// Start a todo step (vpn-access in project #1)
	result, err := handleStartStep(st, json.RawMessage(`{"project_id":"1","step_id":"vpn-access"}`))
	if err != nil {
		t.Fatalf("first start should succeed: %v", err)
	}
	if !strings.Contains(result, "in_progress") {
		t.Errorf("result: %s", result)
	}
	// Starting again should fail
	_, err = handleStartStep(st, json.RawMessage(`{"project_id":"1","step_id":"vpn-access"}`))
	if err == nil {
		t.Error("expected error for starting already in_progress step")
	}
}

func TestHandleDoneStep_BlockedStep(t *testing.T) {
	st := store.NewMockStore()
	// configure-dns has an unresolved blocker
	_, err := handleDoneStep(st, json.RawMessage(`{"project_id":"1","step_id":"configure-dns"}`))
	if err == nil {
		t.Error("expected error for done on blocked step")
	}
}

// --- Full RegisterPMTools integration test ---

func TestRegisterPMTools(t *testing.T) {
	st := store.NewMockStore()
	s := NewServer("pm-mcp", "0.1.0")
	RegisterPMTools(s, st)

	if len(s.tools) != 14 {
		t.Fatalf("got %d tools, want 14", len(s.tools))
	}

	// Check all tool names
	names := make(map[string]bool)
	for _, tool := range s.tools {
		names[tool.Name] = true
	}
	expected := []string{
		"list_projects", "get_project", "add_project", "add_step",
		"start_step", "review_step", "done_step",
		"add_blocker", "resolve_blocker", "add_decision",
		"get_briefing",
		"list_steps", "list_blockers", "list_decisions",
	}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestHandleListProjects_Empty(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	result, err := handleListProjects(st, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("handleListProjects error: %v", err)
	}
	if !strings.Contains(result, "No projects") {
		t.Errorf("result mismatch: %s", result)
	}
}

func TestHandleGetProject_NotFound(t *testing.T) {
	st := store.NewFileStore(t.TempDir())
	_, err := handleGetProject(st, json.RawMessage(`{"project_id":"1"}`))
	if err == nil {
		t.Error("expected error for non-existent project")
	}
}

func TestHandleAddProject_EmptyTitle(t *testing.T) {
	st := store.NewMockStore()
	_, err := handleAddProject(st, json.RawMessage(`{"title":""}`))
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestHandleAddProject_InvalidTitle(t *testing.T) {
	st := store.NewMockStore()
	_, err := handleAddProject(st, json.RawMessage(`{"title":"'\"\""}`))
	if err == nil {
		t.Error("expected error for invalid title")
	}
}

func TestHandleAddBlocker_Duplicate(t *testing.T) {
	st := store.NewMockStore()
	// Add, then duplicate
	handleAddBlocker(st, json.RawMessage(`{"project_id":"1","step_id":"vpn-access","title":"Test Blk"}`))
	_, err := handleAddBlocker(st, json.RawMessage(`{"project_id":"1","step_id":"vpn-access","title":"Test Blk"}`))
	if err == nil {
		t.Error("expected error for duplicate blocker")
	}
}
