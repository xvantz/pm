# store-git-backup Specification

## Purpose
PM project data SHALL survive host disk loss and stay auditable: the daemon
keeps its data directory in git and pushes it off-host, one commit per
action, without ever letting git failures break data writes.

## Requirements

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

### Requirement: One commit per mutating action

Every mutating API call SHALL leave exactly one commit whose message names
the action and its target (e.g. `add_step <project> <slug>`). Hooks live at
the API handler level, not in the file store, so timestamp touches do not
produce commits. The data write SHALL land before the commit, and a failed
git operation SHALL be a log warning, never a failed API response: the next
commit-all picks the data up.

Rationale: handler-level commits give a readable audit trail (`git log`
shows what the agent did); store-level hooks would drown it in noise. Data
durability must not depend on git health.

#### Scenario: Action is traceable

- **WHEN** the client adds a step through the daemon
- **THEN** `git log` shows one commit naming the action, project and step

#### Scenario: Broken git keeps the API alive

- **WHEN** the git binary fails (e.g. lock, corrupt index) during a write
- **THEN** the API call still succeeds and the data is on disk; the failure
  is a warning in the log

### Requirement: Push is asynchronous and retried

Push SHALL happen off the request path with retries. The daemon's
availability SHALL NOT depend on network reachability: offline periods
accumulate local commits that leave on reconnect. Unpushed state MUST be
observable (ahead count via `git status`).

#### Scenario: Offline accumulates, reconnect drains

- **WHEN** the remote is unreachable during actions and returns later
- **THEN** no action fails and all accumulated commits eventually arrive

### Requirement: Nix options carry repo and key

The NixOS module SHALL expose `services.pm.backup = { enable, repoUrl,
keyFile, tokenFile }`. The key and token files SHALL be readable only by the
service user, never enter the nix store or logs. The token travels as an
`Authorization: Bearer` header injected via git env config
(`GIT_CONFIG_COUNT` / `GIT_CONFIG_KEY_0=http.extraHeader` /
`GIT_CONFIG_VALUE_0`), never on the command line (invisible in `ps`) and
never baked into the remote URL or written to disk (no `.netrc`). `keyFile`
serves SSH remotes, `tokenFile` serves HTTPS remotes; both set means each
travels its own channel and git picks the one matching the remote scheme.
With backup disabled nothing in daemon behavior changes (no repo touched,
no process spawned).

#### Scenario: Disabled means untouched

- **WHEN** the service runs with backup disabled
- **THEN** the data directory gains no `.git` from the daemon and no git
  process runs
