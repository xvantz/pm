# store-git-backup Specification

## Purpose
PM project data SHALL survive host disk loss and stay auditable: the daemon
keeps its data directory in git and pushes it off-host, one commit per
action, without ever letting git failures break data writes.

## Requirements

### Requirement: Daemon owns the data-dir git state

On startup with backup enabled, the daemon SHALL ensure the data directory
is a git repository pointing at the configured remote: `init` when `.git`
is absent (adopting existing files and history when present), and verify
`remote get-url origin` equals the configured URL. A mismatch SHALL disable
pushing and fail loudly in the log; the daemon MUST NOT push to an
unexpected remote. The check SHALL run at startup and before every push.

Rationale: a silent push to the wrong repo leaks project data. Fail-closed
is the only safe default, and the single-writer daemon is the only component
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
- **THEN** no push happens and the log carries an error naming both URLs

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
