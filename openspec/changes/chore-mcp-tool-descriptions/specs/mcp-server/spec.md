# mcp-server Specification (delta)

## ADDED Requirements

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
