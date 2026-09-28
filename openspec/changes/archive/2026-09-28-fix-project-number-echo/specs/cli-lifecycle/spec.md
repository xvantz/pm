# CLI Lifecycle Specification (delta)

## ADDED Requirements

### Requirement: Creation echoes the server-assigned number

`pm project create` SHALL print the server-assigned project number, re-read after save: the advisory `NextNumber` shown before save may be stale under concurrency, and follow-up hints must point at the real project.

#### Scenario: Hints match the created project

- **WHEN** the user creates a project while another create races it
- **THEN** the printed number and hints resolve to the just-created project
