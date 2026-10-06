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

Every git process of the backup SHALL run with a sterile config
(`GIT_CONFIG_NOSYSTEM=1`, `GIT_CONFIG_GLOBAL=/dev/null`): only the
repo-local config applies. A user-global `url.insteadOf` (e.g. an
https-to-SSH rewrite) MUST NOT hijack the transport away from the
configured remote. Identity stays pinned locally, so nothing needed is lost.

Rationale: fail-closed refusal sounded safe but guaranteed a stuck backup
on every legitimate remote change (observed 2026-10-06: SSH origin never
followed the HTTPS config). A loud `set-url` keeps the backup moving while
naming both URLs, and the single-writer daemon stays the only component
allowed to mutate this repo. Sterile env followed the next day, when a
host-global insteadOf pushed an https origin over SSH despite the config
file saying otherwise.

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

#### Scenario: Host rewrite ignored

- **WHEN** a user-global `url.insteadOf` rewrites the configured scheme
- **THEN** backup git processes ignore it and push to the configured URL
  unchanged
