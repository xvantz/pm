# project-store Specification (delta)

## ADDED Requirements

### Requirement: Creation returns the stored project

The `Store` SHALL expose `CreateProject(title, goal, tags, id)
(types.Project, error)` as the single creation path: the store assigns the
project number (file counter, daemon mutex, or live max+1 in test doubles)
and returns the stored project, so the caller never prints an advisory
number. An empty id means the store generates a UUIDv7; a non-empty id MUST
be a valid UUID (400 on garbage) and MUST NOT collide (409 on collision).
`NextNumber`/`AdvanceNextNumber` SHALL NOT be part of the creation path;
they stay only as deprecated shims until every caller migrates, then are
removed from the interface.

Rationale: the advisory-then-save dance prints a stale number whenever the
counter runs ahead of the live max (trash consumes numbers permanently;
observed 2026-10-05: printed #10, real #14). The daemon response already
carries the truth, but `SaveProject(...) error` has nowhere to put it, so
every caller today re-reads by UUID. One creation method returning the truth
makes the bug class impossible instead of worked around per caller.

#### Scenario: Created number is printable as-is

- **WHEN** the caller creates a project through `CreateProject` while the
  counter sits ahead of the live max (trash gaps or races)
- **THEN** the returned project carries the number under which it is stored,
  and resolving by the returned UUID yields the same number with no second read

#### Scenario: Bad id is loud

- **WHEN** the caller passes a non-UUID id or an already-used UUID
- **THEN** creation fails with a `bad id` / `already exists` error and nothing
  is stored
