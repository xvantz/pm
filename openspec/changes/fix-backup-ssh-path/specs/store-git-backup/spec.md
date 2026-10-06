# store-git-backup Specification (delta)

## MODIFIED Requirements

### Requirement: Daemon owns the data-dir git state

On startup with backup enabled, the daemon SHALL ensure the data directory
is a git repository pointing at the configured remote: `init` when `.git`
is absent (adopting existing files and history when present), and sync
`remote origin` to the configured URL when it differs. A stale origin
(e.g. SSH from before the HTTPS switch) follows the config via `set-url`
with a warning naming both URLs; the daemon MUST NOT stay stuck on an
obsolete remote. The operator owns both config and repo, and a typo'd URL
fails loudly at push with an auth error rather than pushing silently
elsewhere. The check SHALL run at startup; pushes re-resolve the remote
through git itself.

Rationale: fail-closed refusal sounded safe but guaranteed a stuck backup
on every legitimate remote change (observed 2026-10-06: SSH origin never
followed the HTTPS config). A loud `set-url` keeps the backup moving while
naming both URLs, and the single-writer daemon stays the only component
allowed to mutate this repo.

#### Scenario: Fresh directory bootstraps

- **WHEN** the daemon starts with backup enabled on a data dir without `.git`
- **THEN** it initializes a repo, sets the configured remote, and subsequent
  actions produce commits

#### Scenario: Adopted repo keeps history

- **WHEN** the data dir already holds a git repo (e.g. created by hand)
- **THEN** the daemon keeps its history, verifies the remote, and continues
  committing on top

#### Scenario: Wrong remote disables push loudly

- **WHEN** the configured remote differs from the repo's origin
- **THEN** the daemon updates origin to the configured URL, the log carries
  a warning naming both URLs, and pushes continue to the configured remote
