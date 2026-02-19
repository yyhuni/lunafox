# CI Version Governance

## Scope

This repository uses layered tool images for CI and quarterly reminder-based version governance.

- `base-tools`: lightweight common utilities.
- `ci-tools`: Go/Node/pnpm on top of `base-tools` for CI jobs.
- `scanner-tools`: heavy scanner toolchain in `worker/Dockerfile.tools`.

## Repository Variables

Set these repository variables to immutable image digests:

- `BASE_TOOLS_IMAGE=ghcr.io/<owner>/lunafox-base-tools@sha256:<digest>`
- `CI_TOOLS_IMAGE=ghcr.io/<owner>/lunafox-ci-tools@sha256:<digest>`

Use digest references to avoid tag drift.

## Build Flows

Image build workflows support automatic and manual trigger:

- `.github/workflows/build-base-tools.yml`
- `.github/workflows/build-ci-tools.yml`

Trigger policy:

- `push` on `main/develop` when relevant Dockerfiles/workflow files change
- `workflow_dispatch` manual fallback
- `build-ci-tools` also runs after `Build Base Tools Image` completes

Promotion policy:

- `BASE_TOOLS_IMAGE` variable auto-updates on `main` push or manual dispatch.
- `CI_TOOLS_IMAGE` auto-updates only after all verification jobs pass.
- Verification failure fails workflow and blocks promotion (no auto issue creation).

## CI Job Usage

`CI_TOOLS_IMAGE` is used by these jobs:

- `.github/workflows/test.yml`
- `.github/workflows/check-generated-files.yml`
- `.github/workflows/release.yml` (`build-installers` job)

## Upgrade Policy

- Runtime/tool version upgrades are manual via pull request.
- No automatic major-version bump bot is used.
- Quarterly workflow provides reminder/report only and does not block CI/release.
- CI image promotion requires successful validation jobs.

## Quarterly Reminder Workflow

- Workflow: `.github/workflows/quarterly-version-review.yml`
- Outputs/updates issue title: `[Quarterly] Version Review YYYY-QN`

Report sections:

- current versions
- latest versions
- version gap and risk
- UNPINNED high-priority list
- suggested actions

## UNPINNED Rule

`UNPINNED` items are highlighted as high priority in quarterly issue with concrete pinning suggestions.

Current expected candidates:

- `nmap` / `masscan` apt installs in `worker/Dockerfile`
- `pipx install` packages in `worker/Dockerfile` without `==<version>`

These findings are reminders only and do not fail workflows.
