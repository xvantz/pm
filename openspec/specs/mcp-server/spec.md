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

The confirm path SHALL NOT assume the store answered with a plan: a store
that returns nil (any present or future file-backed or daemon-backed store)
SHALL yield a success message without a moved-count, never a crash. After the
serve-api change the daemon always sends a plan, so the guard is a second line,
not the fix.

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

#### Scenario: Confirm survives a nil plan

- **WHEN** the underlying store answers confirm with a nil plan
- **THEN** the tool still succeeds with a closed message naming the project
  and reason, and the server process stays alive

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

### Requirement: Actionable responses

Every MCP tool success response that implies a next action SHALL name the next
tool by its exact catalogue name with its key parameters, never a CLI command:
the MCP agent has tools, not a shell with `pm`. Every tool error for a
recoverable caller mistake (unknown step id, duplicate id, missing argument,
unmet lifecycle precondition, consent without reason) SHALL state what
happened, what is needed instead, and which call fixes it.

Rationale: a hint in a language the caller cannot execute (`Next: pm step
start 1 x` to an agent with no shell) is zero signal at full byte cost. An
error without the fix (`step_id is required`) buys a second failed call where
one sentence would do. Sources: Anthropic effective-tools (actionable errors
with a correct-call example) and Merge.dev (workflow in text, ops in schema).

#### Scenario: Hints name tools

- **WHEN** any write tool succeeds with a next action
- **THEN** the response names the next tool (e.g. `start_step`,
  `list_blockers`) and carries no `pm <cli ...>` command text

#### Scenario: Step hint routes by state

- **WHEN** the client calls `get_step` on a blocked step
- **THEN** the response hints at `resolve_blocker`/`list_blockers` for the
  reason; on a plain step it hints at the next lifecycle call for its status

#### Scenario: Unknown step names the fix

- **WHEN** the client calls with a step id that does not exist
- **THEN** the error names the step and points at `list_steps` as the source
  of live ids

#### Scenario: Duplicate names the way out

- **WHEN** the client adds a step, blocker or decision whose id already exists
- **THEN** the error names the conflict and says to reuse or rename

#### Scenario: Unmet precondition names the repair call

- **WHEN** the client calls `done_step` on a non-review step
- **THEN** the error states review is required and names `review_step` as the
  fix; `close_project` confirm without reason names `reason` and the two-call
  flow

### Requirement: Project list filtering

The `list_projects` tool SHALL accept an optional `status` parameter with
values `active`, `completed` or `all`, defaulting to `active` when absent or
empty. The system SHALL filter in the MCP handler (tool layer); the store
contract and the daemon HTTP API SHALL stay unfiltered.

The response SHALL always carry `projects`, `count` (number of returned
entries), `active_count`, `completed_count` and `total`, so the cheap filtered
answer still COUNTs what it hid. When the default filtered view hides
completed projects (`completed_count > 0`), the response SHALL carry a hint
naming `status=all`.

An unknown `status` value SHALL be a loud error naming the accepted values,
never a silent default.

Rationale: measured on the live store, one third of the unfiltered list is
closed-project noise (3 of 9), growing linearly with every close. Sorting
reorders the bytes but does not remove them; only server-side filtering cuts
agent context and the `get_project` fan-out behind it. The counters keep the
honesty invariant from the sizing rule: filtered-out records are counted,
never silently dropped. `status=all` is the full-fidelity escape hatch, the
same shape as `detail:true` on `get_project`.

#### Scenario: Default lists active only but counts all

- **WHEN** the client calls `list_projects` with `{}` on a store with active and completed projects
- **THEN** `projects` carries only active entries, and `active_count + completed_count == total` with `count == len(projects)`

#### Scenario: Explicit all restores the full list

- **WHEN** the client calls `list_projects` with `status=all`
- **THEN** `projects` carries active and completed entries as before the change

#### Scenario: Completed-only view

- **WHEN** the client calls `list_projects` with `status=completed`
- **THEN** `projects` carries only completed entries with correct counts

#### Scenario: Unknown status is a loud error

- **WHEN** the client calls `list_projects` with a `status` outside `active|completed|all`
- **THEN** the call fails naming the accepted values instead of returning a silent default

### Requirement: Project creation echoes the server-assigned number

The `add_project` tool SHALL print the server-assigned project number,
re-read by stable UUID after save: the advisory `NextNumber` known before
save may be stale whenever the daemon counter ran ahead of the live max
(trashed projects keep their numbers consumed, concurrent creates race),
and the follow-up hint MUST point at the real project. The `Next:` hint
SHALL carry the client-assigned UUID (`add_step {project_id: "<uuid>"}`),
which is stable across every backend; the `Project #N` headline SHALL carry
the re-read number. A failed re-read SHALL be a loud error, never a
confirmation with the advisory number.

Rationale: the remote daemon assigns numbers from a file counter that counts
trashed projects, while the advisory number is live max+1. Printing the
advisory value lies permanently after any delete (observed 2026-10-05:
printed #10, real #14, `add_step` 404, gateway circuit breaker parked the
server). The CLI already re-reads (`cli-lifecycle / Creation echoes the
server-assigned number`); this requirement closes the MCP half of the same
contract.

#### Scenario: Confirmation survives trash gaps

- **WHEN** the client creates a project while the daemon counter sits ahead
  of the live max (numbers consumed by trash)
- **THEN** the response prints the daemon-assigned number and an `add_step`
  call with the printed `project_id` succeeds on the first try with no
  manual `list_projects`

#### Scenario: Failed re-read is loud

- **WHEN** the project cannot be read back by its UUID right after a
  successful save
- **THEN** the tool returns a `resolve created project` error instead of a
  confirmation carrying the advisory number
