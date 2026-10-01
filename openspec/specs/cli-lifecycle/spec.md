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

Deleted projects SHALL go to trash: `trash list` shows trashed items with
their titles and deletion time, `trash restore <name>` brings them back by
trash name, project number or title. The same trash is exposed over MCP via
`trash_list` and `trash_restore`; permanent erase (`trash clean`, `pm del`)
stays CLI-only and is never exposed over MCP.

#### Scenario: Delete and restore

- **WHEN** the user deletes a project and runs `trash restore` with its name, number or title
- **THEN** the project reappears in the project list with its steps and decisions intact

#### Scenario: Ambiguous restore lists candidates

- **WHEN** the restore argument matches several trashed items
- **THEN** the call fails with the candidate list and nothing is restored

### Requirement: Doctor

`pm doctor` SHALL ask the daemon for its integrity verdict (`GET /api/doctor`)
and print it: counts, orphans, unreadable files and timestamp stats, plus
actionable repair hints. There SHALL be no local file scan: the daemon is the
single reader, so there is nothing to compare and nothing to diverge.

When the daemon is unreachable, doctor SHALL fail with an error naming the
daemon, not fall back to a local scan. A fallback would reintroduce the second
reader this change removes. Fix the daemon.

#### Scenario: Missing store

- **WHEN** the user runs `pm doctor` with no storage directory present
- **THEN** the daemon reports the store as missing and doctor suggests running `pm init`

#### Scenario: Daemon down is the verdict

- **WHEN** the daemon is unreachable
- **THEN** doctor fails naming the daemon and performs no local file scan

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
