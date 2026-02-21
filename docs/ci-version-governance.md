# CI Version Governance

## Scope

The repository now uses a simplified CI baseline:

- CI jobs run directly on GitHub-hosted runners.
- Language toolchains are installed by official setup actions.
- Worker build toolchain is maintained in `worker/Dockerfile`.
- Quarterly reminder-based version governance is retained.

## CI Baseline Contracts

Mandatory contracts:

- `.github/workflows/test.yml` must run without job-level container images.
- `.github/workflows/check-generated-files.yml` must run without job-level container images.
- `.github/workflows/ci.yml` must support change-detection and conditional module testing.
- The legacy tool-image workflows must not be reintroduced.

Guard policy:

- `test.yml` contains a contract-guard step that fails CI if forbidden legacy keywords or removed workflow names reappear.
- This guard is part of drift prevention and should remain enabled.

## Workflow Topology

Core CI workflows:

- `.github/workflows/ci.yml`: entry pipeline with change detection and conditional fan-out.
- `.github/workflows/test.yml`: module tests (frontend/scripts, worker, server, agent, installer).
- `.github/workflows/check-generated-files.yml`: generated-file consistency for worker metadata.

Release workflow:

- `.github/workflows/release.yml`: installer build, multi-registry image publish/merge/promote, release finalization, channel update.

## Release Contracts

Release contracts that must not drift:

- Keep dual channels (`stable`, `canary`) and `SCHEMA_VERSION=2`.
- Keep digest lists for `AGENT_IMAGE_REFS` and `WORKER_IMAGE_REFS` in channel files.
- Keep four registry digests emitted and verified: DockerHub, GHCR, Tencent TCR, Alibaba ACR.
- Keep both architectures for published images: `linux/amd64` and `linux/arm64`.

## Version Inventory and Upgrades

Governance is intentionally conservative:

- Version upgrades are manual via pull request.
- Quarterly review remains advisory, not a merge/release blocker.
- `scripts/ci/version_inventory.sh` is the canonical source used for version snapshots and reports.

## Quarterly Reminder Workflow

- Workflow: `.github/workflows/quarterly-version-review.yml`
- Issue title pattern: `[Quarterly] Version Review YYYY-QN`

Report sections:

- current versions
- latest versions
- version gap and risk
- UNPINNED high-priority list
- suggested actions

## UNPINNED Rule

`UNPINNED` entries are highlighted in quarterly output with explicit pinning suggestions.

Current expected candidates:

- `nmap` / `masscan` apt installs in `worker/Dockerfile`
- `pipx install` packages in `worker/Dockerfile` without strict `==<version>`

These findings are reminders only and do not fail workflows.
