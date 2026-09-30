# CLI Lifecycle Specification

## Purpose

The `pm` CLI (`cmd/pm`, `internal/cli/`) manages projects through a strict step lifecycle plus blockers, decisions, briefings, trash and a `doctor` integrity check.

## Requirements

### Requirement: Four-stage step lifecycle

Steps SHALL move strictly `todo -> in_progress -> review -> done`. Transitions use `add_step`, `start_step`, `review_step`, `done_step`. `done_step` on a step that is not in `review` MUST fail.

#### Scenario: Happy path

- **WHEN** the user runs `add`, `start`, `review`, `done` in order for a step
- **THEN** the step ends in `done` state

#### Scenario: Skipping review is rejected

- **WHEN** the user runs `done_step` on an `in_progress` step
- **THEN** the command fails and the step stays `in_progress`

### Requirement: Blockers and decisions

Steps SHALL support `blocker add` (with reason) and `blocker resolve`. Projects SHALL support `decision add` (title plus optional rationale) and listing commands for steps, blockers and decisions.

#### Scenario: Blocker roundtrip

- **WHEN** the user adds a blocker with a reason and then resolves it
- **THEN** the blocker ends in `resolved` state and the step is unblocked

### Requirement: Trash

Deleted items SHALL go to trash: `trash list` shows trashed items, `trash restore <name>` brings them back. Trash is CLI-only; no MCP tool exposes it.

#### Scenario: Delete and restore

- **WHEN** the user deletes an item and runs `trash restore` with its name
- **THEN** the item reappears in its project

### Requirement: Doctor

`pm doctor` SHALL verify storage integrity (projects dir exists, YAML parses, counter consistent) and print actionable repair hints (`pm init` when the store is missing).

#### Scenario: Missing store

- **WHEN** the user runs `pm doctor` with no storage directory present
- **THEN** doctor reports the store as missing and suggests running `pm init`

### Requirement: Creation echoes the server-assigned number

`pm project create` SHALL print the server-assigned project number, re-read after save: the advisory `NextNumber` shown before save may be stale under concurrency, and follow-up hints must point at the real project.

#### Scenario: Hints match the created project

- **WHEN** the user creates a project while another create races it
- **THEN** the printed number and hints resolve to the just-created project

### Requirement: Bulk close stays a single CLI command

`pm project close <project-id> <reason>` SHALL close in one command, unchanged
by the consent requirement that applies to agent callers. The person typing it
is the consent; there is nothing to preview for a human who is already at the
terminal deciding. `reason` SHALL be required, since it is the only durable
trace of why the project was closed.

An empty or missing reason SHALL be refused rather than defaulted: the default
`bulk close` recorded by the store is acceptable for API callers that pass an
empty string, but the CLI has a human present and can require the real reason.

#### Scenario: Human closes with one command

- **WHEN** a user runs `pm project close 3 "shipped in v1.2"` on an active project
- **THEN** open steps become `done`, the project becomes `completed`, and a
  `Closed: shipped in v1.2` decision exists

#### Scenario: Reason is required at the terminal

- **WHEN** a user runs `pm project close 3` with no reason
- **THEN** the command refuses and the project stays `active`
