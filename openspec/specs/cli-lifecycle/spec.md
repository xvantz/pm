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
