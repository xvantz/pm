# Briefing Specification (delta)

## ADDED Requirements

### Requirement: Daily digest sections

`Generate` SHALL build a dated briefing from the store: per-status sections (active, blocked, idea, completed) plus summary counts and recommendations. Blocked sections SHALL include only `active` projects with unresolved blockers; `completed` projects never contribute blockers. Full scoring rules arrive before implementation.

#### Scenario: Completed blockers stay out

- **WHEN** a `completed` project holds unresolved blockers
- **THEN** no blocked section or recommendation mentions them
