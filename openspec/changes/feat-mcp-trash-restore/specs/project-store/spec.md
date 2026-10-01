# project-store Specification (delta)

## ADDED Requirements

### Requirement: Delete backups

`FileStore` SHALL write a backup copy before every deletion
(`DeleteProject`, `DeleteStep`, `DeleteBlocker`, `DeleteDecision`) under
`_meta/backups/<unix-timestamp>/`, keeping only the 20 most recent backup
runs. The backup SHALL contain the exact bytes being removed, so a hard
delete of a step, blocker or decision - which has no trash - stays manually
recoverable from files under git.

Backups are written on delete only, never on save. Nothing reads them back
automatically; recovery is a human copying files, documented in the README.

#### Scenario: Step delete leaves a backup

- **WHEN** a step is deleted
- **THEN** its YAML bytes exist under the newest `_meta/backups/` run

#### Scenario: Rotation keeps twenty runs

- **WHEN** 21 deletions happen in a row
- **THEN** exactly 20 backup run directories remain
