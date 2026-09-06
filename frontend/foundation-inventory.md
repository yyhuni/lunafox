# UI Foundation Baseline Inventory

This file is the readable index for the approved initial UI foundation baseline.
The machine-readable ledger is `foundation-exceptions.json`; guardrail scripts use
that ledger to approve only the counted baseline in `--mode verify`.

## Source Of Truth

- Raw inventory commands: `check:typography-foundation`, `check:color-foundation`,
  `check:spacing-foundation`, `check:button-foundation`,
  `check:component-foundation`, `check:a11y-foundation`,
  `check:motion-foundation`, and `check:ui-boundary` in inventory mode.
- Approved baseline: `foundation-exceptions.json`.
- Blocking rollout command: `check:ui-foundation` in verify mode.

## Status Classes

| Status | Meaning |
| --- | --- |
| `keep` | Intentional foundation owner or bounded production behavior. |
| `narrow` | Acceptable only in a small scope; future reuse should move to shared helpers. |
| `remove` | Known drift that should be cleaned in an approved cleanup batch. |
| `deferred` | Historical debt retained until a focused migration batch. |
| `excluded` | Prototype, demo, visual lab, or non-production boundary excluded from default production enforcement. |

## Baseline Categories

| Finding family | Ledger field | Current disposition |
| --- | --- | --- |
| Readable typography utility drift | `check=typography` | No remaining production violations in inventory or verify mode. |
| Raw color literals and color functions | `check=color` | `keep`, `narrow` |
| Raw CSS variable fallbacks | `check=color` | `keep`, `narrow` |
| Arbitrary spacing, dimensions, radius, shadow, and `!important` drift | `check=spacing` | `keep`, `narrow` |
| Native buttons and ad-hoc shared Button sizing | `check=button` | No remaining production violations in inventory or verify mode. |
| Direct icon imports and component boundary drift | `check=component` / `check=ui-boundary` | `excluded`; no production boundary findings. |
| Icon-only a11y gaps and focus drift | `check=a11y` | No remaining production violations in inventory or verify mode. |
| Layout-shifting hover motion, transition-all, and missing reduced-motion ownership | `check=motion` | `narrow` |
| Bounded sidebar content hover motion | `class=bounded-hover-motion` | `keep` |
| Loading-state drift and route-progress engine ownership | `check=loading` | No unclassified loading snippets; retained non-loading motion is recorded below. |

## Verify Behavior

Verify mode must fail when a finding has no matching ledger entry or exceeds the
approved count for that entry. Cleanup work should reduce or remove ledger
counts in approved batches; it should not widen scope just to make a check pass.

## Migration Progress

- Batch 3 Typography: inventory and verify mode report no production typography
  violations. No typography ledger entry remains to remove or narrow.
- Batch 4 Color And Status: removed `color-scan-history-running-var` after
  `scan-history-stat-cards` moved from a local raw running color to shared
  scan/status tone helpers. Retained bounded chart, support canvas, sidebar
  runtime, and vulnerability floating-bar entries as reviewed `keep` / `narrow`
  exceptions.
- Agent log drawer color guardrail: recorded
  `color-agent-log-drawer-terminal-log-palette` as a narrow terminal-log
  renderer exception because the drawer derives structured log level badges and
  error-row emphasis from `--terminal-log-*` variables rather than product UI
  status colors.
- Batch 5 Spacing, Sizing, Shape, And Elevation: promoted repeated bulk-add,
  organization, target, and shared data-table dialog/editor geometry into
  shared helpers. Removed `spacing-organization-dialog-baseline`,
  `spacing-shared-data-table-baseline`, and `spacing-subdomains-baseline`;
  narrowed `spacing-common-dialog-baseline`.
- Batch 6 Button And Action: replaced overview, workflow, distributed,
  target-overview, and tools-page action drift with shared `Button` sizes and
  accessible icon action names. Removed all `check=button` ledger entries and
  `a11y-scan-history-icon-button-baseline` after verify mode reported no
  button or a11y findings.
- Batch 7 Component Boundary And Icon: `check:component-foundation` and
  `check:ui-boundary` reported no production findings. No component-boundary
  ledger entries required removal or narrowing.
- Batch 8 A11y And Motion: added accessible names to icon-only row actions,
  moved scan and target hover feedback away from layout-shifting transforms and
  widths, and placed login trace animation behind reduced-motion controlled CSS
  variables. Removed `motion-login-trace-baseline`,
  `motion-scan-history-overview-baseline`, `motion-quick-scan-baseline`, and
  `motion-target-overview-baseline`; retained only reviewed GitHub/support
  `narrow` motion entries.
- Final spacing and route convergence: reviewed every remaining spacing entry
  after verify-mode scans. All remaining spacing findings are bounded route or
  visualization/editor geometry with owner, trigger, and recovery path recorded
  as `narrow`; no `remove` or `deferred` ledger status remains.
- Loading-state standardization: route progress is owned by
  `components/route-progress.tsx` backed directly by `nprogress`; shared loading
  entrypoints are `Skeleton`, shared skeleton templates, `ContentReveal`,
  `Spinner`, `Button` loading props, `AppWarmupLoader`, root boot plus login
  readiness gating for public auth entry, `Toaster`, and toast helpers. Legacy
  `AuthWarmupShell`, `ShieldLoader`, `LoadingState`, `LoadingSpinner`, and
  `LoadingOverlay` are not approved production loading policy and have been
  hard-cut from the codebase. Inventory mode reports
  `unclassified loading snippets = 0`.

## Retained Non-Loading Status Motion

| Call site | Semantic owner | Why not loading | Approved helper / exception | Review trigger | Recovery path |
| --- | --- | --- | --- | --- | --- |
| `components/nudges/nudge-minimal.tsx` | Nudge terminal cursor | Decorative terminal cursor after message content has rendered. | `loading-status-nudge-terminal-cursor` | When nudge terminal visuals or motion policy are reworked. | Move the cursor animation into a shared terminal/status affordance or remove the blink. |
| `components/auth/terminal-login-sections.tsx` | Auth terminal cursor | Auth shell terminal affordance, not a data fetch placeholder. | `loading-status-auth-terminal-cursor` | When auth terminal boot visuals change. | Move the cursor animation into the auth terminal primitive or remove the blink. |
| `components/settings/system-logs/system-logs-view.tsx` | System log auto-refresh status | Live auto-refresh heartbeat for an active log viewer. | `loading-status-system-logs-auto-refresh` | When system log polling or toolbar status changes. | Move the heartbeat into a shared live-status indicator helper. |
| `components/scan/scan-status-badge.tsx` | Scan runtime status | Running scan icon/progress animation describes domain execution state. | `loading-status-scan-badge-running` | When scan status badges or progress visualization change. | Move running-state motion into scan status helpers or make the badge static. |
| `components/scan/history/scan-overview-sections.tsx` | Scan history runtime status | Running scan and auto-refresh pulses describe active scan telemetry. | `loading-status-scan-history-runtime` | When scan history runtime telemetry changes. | Move scan runtime motion into shared scan status helpers. |
| `components/scan/history/scan-runtime-detail-drawer.tsx` | Scan runtime task status | Task-level running indicators describe active scan execution. | `loading-status-scan-runtime-detail` | When scan runtime drawer task state changes. | Move running task motion into shared scan status helpers. |
