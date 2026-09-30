# briefing Specification

## Purpose

`internal/briefing/briefing.go` builds a daily digest across all projects: what
exists, what moved today, what is blocked, and what to pick up next. It answers
"what happened and what now" in one call, so an agent does not have to read
every project to know where the work stands.

Event times follow the `project-store` capability (`Event timestamps`), and the
calendar day is bucketed in the reader's zone per `Calendar-day bucketing`
there. This capability does not restate those rules; it describes only what the
briefing computes from them.

## Requirements

### Requirement: Sections and their order

A briefing SHALL present, in this display order, the sections that have
members: `active`, `blocked`, `idea`, `completed`. A section with no projects
SHALL be omitted rather than emitted empty. Each section SHALL carry a title
including its project count and a machine-readable `type`.

#### Scenario: Only non-empty sections appear

- **WHEN** no project is `idea`
- **THEN** the briefing has no `idea` section and does not mention zero

#### Scenario: Section order is stable

- **WHEN** projects exist in all four states
- **THEN** sections appear in the order active, blocked, idea, completed

### Requirement: Blocked section membership

The `blocked` section SHALL contain only `active` projects that hold at least
one unresolved blocker (`active` or `waiting`). A project with unresolved
blockers SHALL NOT also appear in the `active` section — the states are
mutually exclusive. Projects in any other status SHALL NOT contribute blockers
to the briefing.

Rationale: blockers on a `completed` project are history, not work in progress.
Counting them as blockers would demand action on something already closed.

#### Scenario: Completed blockers stay out

- **WHEN** a `completed` project holds an unresolved blocker
- **THEN** no blocked section and no recommendation mentions it

#### Scenario: Blocked and active are exclusive

- **WHEN** an `active` project has an unresolved blocker
- **THEN** it appears in `blocked` and does not appear in `active`

#### Scenario: Paused projects are counted but not shown

- **WHEN** a project is `paused`
- **THEN** it contributes to `paused_projects` in the summary and appears in
  no section

### Requirement: Daily and weekly movement

The summary SHALL report `steps_today` (steps whose status is `done` and whose
`updated_at` falls on the requested calendar day), `steps_this_week` (same,
from seven days before that day inclusive) and `projects_moved` (projects with
at least one step counted in `steps_today`). The three are distinct: many steps
in one project move that project once.

#### Scenario: Several steps move one project

- **WHEN** two steps are completed today in one project
- **THEN** `steps_today` is 2 and `projects_moved` is 1

#### Scenario: Weekly counts the whole window

- **WHEN** steps were completed today and four days ago
- **THEN** both count toward `steps_this_week`

### Requirement: Long-lived blockers

The summary SHALL list `long_lived_blockers`: unresolved blockers on steps of
`active` projects whose age exceeds seven calendar days, measured in the
reader's zone. Each entry SHALL carry the project, the blocker, its reason and
the age in days. Ordering SHALL be by descending age so the worst first.

#### Scenario: Fresh blockers are not listed

- **WHEN** a blocker was created yesterday
- **THEN** it does not appear in `long_lived_blockers`

#### Scenario: Worst first

- **WHEN** two long-lived blockers have ages 9 and 20 days
- **THEN** the 20-day blocker is listed first

### Requirement: Recommendations

The briefing SHALL produce ranked recommendations derived from the sections:
for each project in `active` with an unfinished next step, a recommendation to
continue it, carrying the remaining-step count; for each project in `blocked`, a
recommendation to resolve its blockers, carrying the blocker count. Priorities
SHALL be sequential and dense, starting at 1.

#### Scenario: Blocked projects are recommended first-class

- **WHEN** a project is blocked with two blockers
- **THEN** a recommendation exists naming it with 2 blockers

#### Scenario: Priority is dense and ordered

- **WHEN** recommendations are produced
- **THEN** priorities run 1, 2, 3 with no gaps

### Requirement: Requested day, not today

`Generate` SHALL accept an optional calendar day and SHALL generate the
briefing for that day rather than for today. An absent day means the current
calendar day in the reader's zone. An unparsable day SHALL fall back to the
current day rather than failing: the briefing is a report, not a transaction.

The `date` argument means "which day to report on". It is not an event time and
SHALL NOT be parsed as one.

#### Scenario: Explicit day is honoured

- **WHEN** the briefing is requested for a past date
- **THEN** movement counts cover that day and the week ending on it

#### Scenario: Unparsable day falls back

- **WHEN** the requested day is not a valid date
- **THEN** the briefing is generated for the current day

### Requirement: Single-project filter

`Generate` SHALL accept an optional project reference and SHALL then build the
briefing from that project alone: summary counts reflect only it, and it
contributes sections on the same terms as in a full run.

#### Scenario: Filter narrows the briefing

- **WHEN** the briefing is filtered to one project
- **THEN** every section and count describes that project only

### Requirement: Degraded reads are visible

When an event time cannot be read, the briefing SHALL NOT silently count the
event as absent; it SHALL log a warning and append a message to
`data_warnings`, per `Unreadable timestamps are loud` in `project-store`. The
briefing SHALL still be returned. `data_warnings` SHALL be omitted from the
response entirely when nothing was unreadable.

#### Scenario: Clean store has no warnings

- **WHEN** every event time parses
- **THEN** `data_warnings` is absent from the response

#### Scenario: Degraded counts are marked

- **WHEN** one step carries an unreadable timestamp
- **THEN** the response carries `data_warnings` naming the step and the raw
  value, and the briefing still returns
