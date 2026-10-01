// Package client is a typed HTTP client for `pm serve`.
//
// It mirrors every daemon endpoint so CLI and MCP can run without touching
// YAML files: set PM_API=http://127.0.0.1:8472 and PM_TOKEN, and all
// operations go through the single-writer daemon.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/xvantz/pm/internal/briefing"
	"github.com/xvantz/pm/internal/types"
	"io"
	"net/http"
	"net/url"
)

// Client talks to one `pm serve` instance.
type Client struct {
	base  string
	token string
	http  *http.Client
	ver   string // cached /healthz version
}

// New returns a Client for base (e.g. http://127.0.0.1:8472).
func New(base, token string) *Client {
	return &Client{base: trimSlash(base), token: token, http: &http.Client{}}
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

// APIError is a non-2xx response from the daemon.
type APIError struct {
	Status int
	Msg    string
}

func (e *APIError) Error() string { return fmt.Sprintf("pm api: %d %s", e.Status, e.Msg) }

func (c *Client) do(method, path string, body any, out any) error {
	var buf io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		buf = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.base+path, buf)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("pm api: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var em map[string]string
		if json.Unmarshal(raw, &em) == nil && em["error"] != "" {
			return &APIError{Status: resp.StatusCode, Msg: em["error"]}
		}
		return &APIError{Status: resp.StatusCode, Msg: string(raw)}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("pm api: decode: %w", err)
	}
	return nil
}

// Health pings the daemon (no auth needed).
func (c *Client) Health() (string, error) {
	var v map[string]string
	if err := c.do("GET", "/healthz", nil, &v); err != nil {
		return "", err
	}
	c.ver = v["version"]
	return v["status"], nil
}

// Version returns the daemon version from the last Health call.
func (c *Client) Version() string { return c.ver }

// --- projects ---

func (c *Client) ListProjects() ([]types.Project, error) {
	var out []types.Project
	err := c.do("GET", "/api/projects", nil, &out)
	return out, err
}

func (c *Client) GetProject(ref string) (*types.ProjectData, error) {
	var out types.ProjectData
	err := c.do("GET", "/api/projects/"+url.PathEscape(ref), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateProject creates a project (status idea, server-assigned number).
// Empty id means the server generates one; pass a UUIDv7 to keep client
// confirmation texts pointing at the real project.
func (c *Client) CreateProject(title, goal string, tags []string, id string) (*types.Project, error) {
	var out types.Project
	err := c.do("POST", "/api/projects", map[string]any{
		"title": title, "goal": goal, "tags": tags, "id": id,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// PatchProject updates goal/status/tags (nil = leave alone).
func (c *Client) PatchProject(ref string, goal *string, status *types.ProjectStatus, tags []string) (*types.Project, error) {
	var out types.Project
	err := c.do("PATCH", "/api/projects/"+url.PathEscape(ref), map[string]any{
		"goal": goal, "status": status, "tags": tags,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteProject moves a project to trash.
func (c *Client) DeleteProject(ref string) (string, error) {
	var out map[string]string
	err := c.do("DELETE", "/api/projects/"+url.PathEscape(ref), nil, &out)
	if err != nil {
		return "", err
	}
	return out["trashed"], nil
}

// closeReq is the body of POST /api/projects/{ref}/close.
//
// Confirm is what separates the two calls: without it the daemon returns a
// plan and changes nothing, with it the close proceeds. Reason is mandatory
// on confirm — it is the only durable trace of why the project was closed.
type closeReq struct {
	Reason  string `json:"reason,omitempty"`
	Confirm bool   `json:"confirm,omitempty"`
}

// closeResponse is what the daemon returns from POST /close. Exactly one of
// the two is populated, decided by the request's confirm: a plan when the
// caller has not consented yet, the closed project once they have.
type closeResponse struct {
	Confirmed bool             `json:"confirmed"`
	Project   *types.Project   `json:"project,omitempty"`
	Plan      *types.ClosePlan `json:"plan,omitempty"`
}

// CloseProject bulk-closes: open steps done, status completed, reason kept.
// With confirm=false it returns the plan and touches nothing; the returned
// project is nil in that case.
func (c *Client) CloseProject(ref, reason string, confirm bool) (*types.Project, *types.ClosePlan, error) {
	var out closeResponse
	err := c.do("POST", "/api/projects/"+url.PathEscape(ref)+"/close",
		closeReq{Reason: reason, Confirm: confirm}, &out)
	if err != nil {
		return nil, nil, err
	}
	if out.Plan != nil {
		return nil, out.Plan, nil
	}
	if out.Project == nil {
		return nil, nil, fmt.Errorf("close %s: response carried neither project nor plan", ref)
	}
	return out.Project, nil, nil
}

// ClosePlan previews the close without touching the project.
func (c *Client) ClosePlan(ref string) (*types.ClosePlan, error) {
	var out types.ClosePlan
	err := c.do("GET", "/api/projects/"+url.PathEscape(ref)+"/close-plan", nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// --- steps ---

func (c *Client) ListSteps(ref string) ([]types.Step, error) {
	var out []types.Step
	err := c.do("GET", "/api/projects/"+url.PathEscape(ref)+"/steps", nil, &out)
	return out, err
}

// AddStep creates a step (slug id derived server-side).
func (c *Client) AddStep(ref, title string) (*types.Step, error) {
	var out types.Step
	err := c.do("POST", "/api/projects/"+url.PathEscape(ref)+"/steps",
		map[string]string{"title": title}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// StepAction runs start|review|done with server-side validation.
func (c *Client) StepAction(ref, step, action string) (*types.Step, error) {
	var out types.Step
	err := c.do("POST", "/api/projects/"+url.PathEscape(ref)+
		"/steps/"+url.PathEscape(step)+"/"+action, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteStep(ref, step string) error {
	return c.do("DELETE", "/api/projects/"+url.PathEscape(ref)+
		"/steps/"+url.PathEscape(step), nil, nil)
}

// --- blockers ---

func (c *Client) ListBlockers(ref string) ([]types.Blocker, error) {
	var out []types.Blocker
	err := c.do("GET", "/api/projects/"+url.PathEscape(ref)+"/blockers", nil, &out)
	return out, err
}

// AddBlocker creates a blocker (default status waiting).
func (c *Client) AddBlocker(ref, step, title, reason string) (*types.Blocker, error) {
	var out types.Blocker
	err := c.do("POST", "/api/projects/"+url.PathEscape(ref)+
		"/steps/"+url.PathEscape(step)+"/blockers",
		map[string]string{"title": title, "reason": reason}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ResolveBlocker marks a blocker resolved (unblocks the step when none remain).
func (c *Client) ResolveBlocker(ref, step, blk string) (*types.Blocker, error) {
	var out types.Blocker
	err := c.do("POST", "/api/projects/"+url.PathEscape(ref)+
		"/steps/"+url.PathEscape(step)+"/blockers/"+url.PathEscape(blk)+"/resolve",
		nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteBlocker(ref, step, blk string) error {
	return c.do("DELETE", "/api/projects/"+url.PathEscape(ref)+
		"/steps/"+url.PathEscape(step)+"/blockers/"+url.PathEscape(blk), nil, nil)
}

// --- decisions ---

func (c *Client) ListDecisions(ref string) ([]types.Decision, error) {
	var out []types.Decision
	err := c.do("GET", "/api/projects/"+url.PathEscape(ref)+"/decisions", nil, &out)
	return out, err
}

func (c *Client) AddDecision(ref, title, reason string) (*types.Decision, error) {
	var out types.Decision
	err := c.do("POST", "/api/projects/"+url.PathEscape(ref)+"/decisions",
		map[string]string{"title": title, "reason": reason}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteDecision(ref, dec string) error {
	return c.do("DELETE", "/api/projects/"+url.PathEscape(ref)+
		"/decisions/"+url.PathEscape(dec), nil, nil)
}

// --- trash ---

func (c *Client) TrashList() ([]types.TrashItem, error) {
	var out []types.TrashItem
	err := c.do("GET", "/api/trash", nil, &out)
	return out, err
}

func (c *Client) TrashRestore(name string) error {
	return c.do("POST", "/api/trash/"+url.PathEscape(name)+"/restore", nil, nil)
}

func (c *Client) TrashClean() error {
	return c.do("DELETE", "/api/trash", nil, nil)
}

// Doctor fetches the integrity verdict from the daemon. An unreachable
// daemon is an error naming it: there is no local fallback scan, fixing the
// daemon is the answer.
func (c *Client) Doctor() (*types.DoctorReport, error) {
	var out types.DoctorReport
	if err := c.do("GET", "/api/doctor", nil, &out); err != nil {
		return nil, fmt.Errorf("daemon unreachable: %w (fix the daemon - no local check exists)", err)
	}
	return &out, nil
}

// --- briefing ---

func (c *Client) Briefing(date, project string) (*briefing.Briefing, error) {
	q := url.Values{}
	if date != "" {
		q.Set("date", date)
	}
	if project != "" {
		q.Set("project", project)
	}
	path := "/api/briefing"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	var out briefing.Briefing
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
