# MCP Server Specification

## Purpose

`pm-mcp` exposes Project Memory operations as MCP tools over stdio transport so LLM agents can read, create and update project data. Source: `cmd/pm-mcp/main.go`, `internal/mcp/server.go`, `internal/mcp/tools.go`.

## Requirements

### Requirement: Stdio JSON-RPC transport

The server SHALL communicate over stdio using newline-delimited JSON-RPC 2.0 (NDJSON, one JSON object per line, no `Content-Length` framing) on both read and write, matching the Hermes Python MCP client. If the official MCP Go SDK is adopted, its stdio transport SHALL be used instead of any hand-rolled framing.

#### Scenario: Initialize handshake

- **WHEN** the client sends `initialize` with any params
- **THEN** the server responds with `protocolVersion: "2024-11-05"`, empty `capabilities.tools` and `serverInfo` (name, version), without requiring prior state

#### Scenario: Initialize handshake (Hermes interop)

- **WHEN** Hermes sends `initialize` as a single `\n`-terminated JSON line followed by `notifications/initialized`
- **THEN** the server responds with `protocolVersion`, `capabilities` and `serverInfo` as single `\n`-terminated JSON lines, with no `Content-Length` headers on the wire

#### Scenario: Calls before initialization are rejected

- **WHEN** the client sends `tools/list` or `tools/call` before `notifications/initialized`
- **THEN** the server responds with error code `-32000` ("Not initialized")

#### Scenario: Unknown methods

- **WHEN** the client sends an unknown method with an id
- **THEN** the server responds with error `-32601`; notifications without id are silently ignored

### Requirement: Tool catalogue

The server SHALL expose the tool set registered in `RegisterPMTools` (`internal/mcp/tools.go`): the live `tools/list` response IS the catalogue, and no tool count is hardcoded in this spec. Every entry SHALL carry `name`, `description` and `inputSchema`. Integrity invariants (non-empty set, unique names, every `inputSchema.type == "object"`) are asserted by `TestToolsList_AllSchemasAreObjects`, which runs against all registered tools: newly registered tools are covered without spec edits.

#### Scenario: List tools

- **WHEN** an initialized client sends `tools/list`
- **THEN** the server returns all registered tools with `name`, `description` and `inputSchema`

#### Scenario: New tool needs no catalogue edit

- **WHEN** a tool is added to `RegisterPMTools` with a valid schema and unique name
- **THEN** `tools/list` includes it and all spec validations still pass unchanged

### Requirement: Tool input schemas

Every tool inputSchema SHALL be a JSON Schema object with `"type": "object"` explicitly set. Empty schemas MUST use `{"type": "object", "properties": {}}` - a bare `{}` is forbidden because strict MCP clients (Hermes pydantic validation of `ListToolsResult`) reject the entire `tools/list` response when any single schema lacks `type`.

#### Scenario: List tools passes strict validation

- **WHEN** an initialized client sends `tools/list`
- **THEN** every tool entry carries `inputSchema.type == "object"` and the response validates against the MCP `ListToolsResult` schema

#### Scenario: Call list_projects

- **WHEN** the client calls `list_projects` with `{}` arguments
- **THEN** the handler returns the project list as text content

### Requirement: Tool call envelope

`tools/call` SHALL dispatch by `name`, pass `arguments` to the handler, and wrap the handler's string result as `content: [{type: "text", text}]`. Unknown tool names SHALL return `-32602`, handler failures SHALL return `-32603`.

#### Scenario: Unknown tool

- **WHEN** the client calls `tools/call` with an unregistered `name`
- **THEN** the server responds with error `-32602` naming the unknown tool

### Requirement: Bulk close tool

The `close_project` tool SHALL require explicit consent from the caller, in two
calls. Called without `confirm`, it SHALL NOT close: it returns a plan — the
steps that would move to `done` with their current statuses, and the unresolved
blockers on those steps — and the project is untouched. Called with
`confirm: true`, it SHALL require a non-empty `reason` and then bulk-close:
force-complete every non-done step, set status `completed` with `CompletedAt`,
and record a `Closed: <reason>` decision. Input schema is `{"type": "object"}`
with required `project_id` and optional `reason` and `confirm`.

Closing does NOT resolve blockers: the blocker records stay on the completed
steps. The plan states this, so a human is not left believing a blocker was
settled when only the step around it was.

Rationale: the tool deliberately bypasses the step lifecycle, so it is the one
operation that can mark unfinished work finished. Requiring a plan first is what
keeps the `review` stop meaningful — the confirmation is the human's claim that
the work is done, relayed by the agent, not the agent's own judgement. For
finished work it still replaces N lifecycle calls; for active work the strict
lifecycle stays.

#### Scenario: Close with reason

- **WHEN** the client calls `close_project` with a valid `project_id` and `reason`
- **THEN** all open steps become `done`, status is `completed`, and a `Closed:` decision carries the reason

#### Scenario: Close already completed

- **WHEN** the client closes a project whose status is already `completed`
- **THEN** the call fails with an `already completed` error

#### Scenario: First call plans instead of closing

- **WHEN** the client calls `close_project` without `confirm` on a project with
  open steps and an unresolved blocker
- **THEN** the project stays `active`, steps keep their statuses, and the
  response lists the steps, the blocker, and that the blocker stays on the record

#### Scenario: Confirm without a reason is refused

- **WHEN** the client calls `close_project` with `confirm: true` and an empty reason
- **THEN** the call fails, the project stays `active`, and no decision is written

### Requirement: Delete tool

The `delete_project` tool SHALL expose the existing trash move over MCP: the project leaves the list and lands in `.trash`, recoverable via `trash restore`. For finished work `close_project` is preferred over delete.

#### Scenario: Delete moves to trash

- **WHEN** the client calls `delete_project` with a valid `project_id`
- **THEN** the project disappears from the list and appears in trash

### Requirement: Reads are sized to the question

`get_project` SHALL return a project summary by default and SHALL NOT return
the step list unless asked. The summary SHALL carry: number, title, status,
goal, tags, step counts grouped by status, the count of unresolved blockers,
and the most recently completed step (title and when). It SHALL NOT carry the
full step objects or the decision list.

Called with `detail: true`, it SHALL return the full project data as before.

`list_steps` SHALL return per step `id`, `title`, `status`, `updated_at` and
`blocker_ids` (ids of blockers on that step, empty when none); it SHALL NOT
embed blocker objects, reasons, or artifacts.

`list_blockers` SHALL accept an optional `step_id`. Without it the response
carries every step group as before. With it the response carries only the
group for that step; an unknown step SHALL be an error naming the step.

Rationale: measured, one detailed project cost more than the list of all
projects (2 195 B vs 1 755 B), which inverts the expected ratio — a detail
read must cost more than an overview, not less. Most calls want "where does
this stand", which the summary answers in a fraction of the bytes.

The `blocker_ids` list keeps the brief listing navigable: a `blocked` step
names its blockers without carrying their reasons, so the caller knows which
`get_step` or `resolve_blocker` to call next. A separate `get_blocker` tool
is rejected on purpose: blockers live inside steps (usually 0-2 per step)
and `get_step` already returns them with reasons, so a new tool would grow
the catalogue to save ~200 B.

#### Scenario: Default read is a summary

- **WHEN** the client calls `get_project` with only `project_id`
- **THEN** the response carries the project fields, step counts by status and
  the unresolved blocker count, and carries no step objects

#### Scenario: Detail flag restores the full dump

- **WHEN** the client calls `get_project` with `detail: true`
- **THEN** the response carries the full project data including every step

#### Scenario: Summary is markedly smaller

- **WHEN** a project is read both ways
- **THEN** the summary response is substantially smaller than the detailed one

#### Scenario: Step listing omits heavy fields

- **WHEN** the client calls `list_steps`
- **THEN** each step carries id, title, status, updated_at and blocker_ids, and no blockers
  or artifacts

#### Scenario: Step listing names its blockers

- **WHEN** the client calls `list_steps` on a project with a blocked step
- **THEN** that step carries the blocker id in `blocker_ids`, and a step with no
  blockers carries an empty list

#### Scenario: Blocker listing filters by step

- **WHEN** the client calls `list_blockers` with `project_id` and `step_id`
- **THEN** the response carries only that step group; without `step_id` it carries
  every group as before

### Requirement: Single-step reads

The server SHALL expose a `get_step` tool taking `project_id` and `step_id`,
returning that step in full: its fields, its blockers, its artifacts and its
timestamps.

Rationale: without it, answering "what is on this one step" costs the whole
project, which makes the two-level split above useless for large projects —
the detail level would still return 20 steps when the caller wanted one.

#### Scenario: One step in full

- **WHEN** the client calls `get_step` for an existing step
- **THEN** the response carries that step with its blockers and artifacts

#### Scenario: Missing step is an error

- **WHEN** the client calls `get_step` for a step that does not exist
- **THEN** the call fails naming the step and the project

#### Scenario: Unknown step in a real project

- **WHEN** the client calls `get_step` with a step id from another project
- **THEN** the call fails naming the step; it does not fall back to a list

### Requirement: Trash tools

The server SHALL expose `trash_list` (trashed items with trash name, project
id, number, title and deletion time) and `trash_restore` (restore by trash
name, project number or title). Ambiguous matches MUST return an error listing
candidates instead of restoring anything. Unknown names MUST fail naming the
requested value. Both tools use `inputSchema` with `"type": "object"`.

#### Scenario: Restore by unambiguous name

- **WHEN** the client calls `trash_restore` with a name matching exactly one trashed item
- **THEN** the item is restored to its project and reported

#### Scenario: Ambiguous restore is rejected

- **WHEN** the client calls `trash_restore` with a name matching several trashed items
- **THEN** the call fails with the candidate list and nothing is restored

#### Scenario: Unknown restore target fails

- **WHEN** the client calls `trash_restore` with a name matching nothing
- **THEN** the call fails naming the requested value

### Requirement: No permanent wipe over MCP

The server SHALL NOT expose `trash_clean` or any permanent-delete tool over
MCP. `delete_project` moves to trash; recovery is via `trash_restore`. Only
the CLI may permanently erase (`pm del`, `trash clean`).

#### Scenario: Catalogue has no wipe tool

- **WHEN** the client lists tools
- **THEN** no tool name implies permanent deletion of trashed items

### Requirement: Tool descriptions navigate

Every tool `description` in `tools/list` SHALL tell the agent three things in
1-2 sentences, most important first: what the tool does, which call comes
before or after it, and when NOT to call it (naming the tool to use instead).
Parameter descriptions SHALL say where an id comes from when it is not obvious
(`list_steps` / `get_step` for step and blocker ids).

Rationale: `tools/list` is the only catalogue the agent reads before every
call. A description without a route ("Mark a step as done") forces discovery
calls; a description with a route ("Fails unless in review - call review_step
first. To finish everything at once use close_project") answers in zero extra
calls. Sources: Merge.dev tool-description guide (front-load, workflow
predecessor, disambiguate vs neighbor, ops details in schema) and Anthropic
effective-tools guide (write for the agent as a new hire, unambiguous inputs).

#### Scenario: Lifecycle step names its predecessor

- **WHEN** the client lists tools
- **THEN** `done_step` says it requires review and names `review_step`, and
  `review_step` names `done_step` as next

#### Scenario: Cheap read points at detail

- **WHEN** the client lists tools
- **THEN** `list_steps` names `get_step` for reasons/artifacts, and `get_step`
  is framed as the one-step read cheaper than a full project dump

#### Scenario: Destructive names the way back

- **WHEN** the client lists tools
- **THEN** `delete_project` names `trash_restore` and prefers `close_project`
  for finished work; `trash_list` states there is no wipe tool on purpose

#### Scenario: Blocker pair is navigable both ways

- **WHEN** the client lists tools
- **THEN** `list_blockers` names `list_steps`/`get_step` as the cheap/full
  counterparts, and `resolve_blocker` says the step keeps its status

#### Scenario: Id params say where they come from

- **WHEN** the client inspects input schemas
- **THEN** every `step_id`/`blocker_id` description points at `list_steps` or
  `get_step` as the id source
