# cli-lifecycle Specification (delta)

## MODIFIED Requirements

### Requirement: Doctor

`pm doctor` SHALL verify storage integrity (projects dir exists, YAML parses, counter consistent) and print actionable repair hints (`pm init` when the store is missing).

It SHALL also split the verdict explicitly: the file half names the local root it checked (`PM_DIR`), the live half names the daemon address it probed (`PM_API`) and the data dir the daemon reports. When the two roots differ, doctor SHALL warn naming both sides instead of staying silent. When the daemon is unreachable the comparison is skipped, not failed: absence of a daemon is its own reported problem.

#### Scenario: Missing store

- **WHEN** the user runs `pm doctor` with no storage directory present
- **THEN** doctor reports the store as missing and suggests running `pm init`

#### Scenario: Path divergence is loud

- **WHEN** `PM_DIR` points at `/a` while the daemon serves `/b`
- **THEN** doctor prints both paths with a mismatch warning
