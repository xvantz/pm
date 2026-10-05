# store-git-backup Specification (delta)

## MODIFIED Requirements

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
