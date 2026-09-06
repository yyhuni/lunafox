# Scheduled Scan UI Contracts

## Management Overview Contract

The page reads the bounded five-item horizontal timeline and its next-24-hour
task count from the independent `GET /v1/scheduledScans:summarize` query. The
count is compact timeline context rather than a separate metric strip. The
service sends an empty query and maps the server's
persisted `nextRunTime` projection; it does not request a large List page,
expand Cron, or make the overview follow table search, pagination, or quick
filters.

The overview owns its loading and failure lifecycle separately from the table.
An initial failure shows a local retry state without replacing the usable List;
after a successful response, a refresh failure keeps that last snapshot, marks
it stale, and shows its last successful time until a later response recovers.
Unknown overview values are never rendered as zero or rebuilt from cached List
rows. Window labels describe tasks that will execute/next execute rather than
Cron occurrences or completed Scan executions.

Scheduled scan page loading must preserve the resolved workbench geometry.

## Management List Pagination

The scheduled-scan management list is backed by the AIP cursor contract:
requests use `pageSize` and the current query's opaque `pageToken`, while
responses expose `nextPageToken` and `totalSize`. The page caches only tokens
returned for already visited adjacent pages and resets that cache when search,
quick filters, page size, or the first-page action changes the query position.
The shared table renders the result total, page-size selector, first-page reset,
and previous/next actions; it must not derive `totalPages` or expose numbered,
last, or direct-page controls.

Other scheduled-scan surfaces that still pass a numbered `page` parameter are
legacy callers outside this management-list contract and must not be used as a
pattern for the `/scan/scheduled` route.

## Batch Status Updates

Only the global `/scan/scheduled` management page passes batch status actions
to `ScheduledScanDataTable`. The selected-row bar exposes explicit Enable and
Disable commands for mixed selections; it never infers an opposite state from
each row. The service sends one `POST /v1/scheduledScans:batchUpdate` request
with canonical `scheduledScans/{id}` names and `updateMask: "isEnabled"` for
each of the 1-100 selected resources.

Disable uses the shared `AlertDialog` because it clears each Schedule's future
cursor and removes unattempted occurrences. A successful batch mutation
invalidates the list and overview, then clears the selection. A failure keeps
the selected rows so the operator can retry. Organization-detail callers do
not pass these actions and retain their existing single-row controls.

## Execution Input Source

A Scheduled Scan persists the shared required `inputSource`:
`scanSnapshot` or `targetInventory`. It is independent of the Schedule's
`scanMode: target|organization`, Workflow reference, and optional Agent
assignment. New create state starts at `scanSnapshot` but explicitly submits
the selected source. Edit state initializes from the persisted value, never
reapplies the create default, and includes `inputSource` in its update only
when the operator changed it.

The saved value applies only to future Schedule handoffs. Each occurrence
copies the source into the newly created Scan; later edits do not rewrite
historical Scans. Missing or unknown values are transport errors, not a reason
to infer a default.

Scheduled Scan editing keeps task name, Workflow, scope, and Cron controls in
the initial basic-information path. Its persisted `inputSource` and optional
Agent assignment reuse `InitiateScanExecutionOptions`, which is closed whenever
the edit drawer opens. Expanding this disclosure only changes visibility; it
must not reset the controlled values or change partial-update submission.

Scheduled scan creation is a right-side `workbench` panel. Its header uses the
shared `EdgePanelHeader` variant with the semantic scheduled-scan icon, visible
title/description, step indicator, and one explicit close action. The four-step
state machine and footer remain owned by this module; do not move them into the
shared header.

The four creation steps deliberately split by product ownership: step 1 owns
scheduled-scan-specific scope; step 2 reuses `InitiateScanWorkflowSelection`
and `ScanAgentSelector`; step 3 reuses `InitiateScanConfigStep`; step 4 owns
Cron inputs and UTC execution preview. Do not restore a
scheduled-scan-local workflow card list or reduced configuration editor.
Workflow selection still initializes only from its parent-scoped Profile, while
scan-options validation runs before leaving step 3 through the shared editor's
`validateAndReveal` operation.

## Schedule Rule And Execution Projection

- Create and Edit submit only the five-field Cron expression. The Server and
  preview interpret it in UTC; the UI does not expose or persist a configurable
  time zone.
- `cronExpression` accepts exactly five fields in minute, hour, day-of-month,
  month, and day-of-week order. Do not accept seconds, `@daily`, `@every`,
  embedded `TZ`/`CRON_TZ`, or silently normalize another Cron dialect.
- Cron validation and upcoming-time preview must use UTC.
  The preview is explanatory only; the Server's persisted `nextRunTime` remains
  authoritative.
- Disabled schedules project `nextRunTime: null` and render the existing empty
  value. The UI must not calculate or invent a replacement cursor.
- `runCount` means committed occurrence trigger-attempt count. The required
  `successfulHandoffCount` and `failedHandoffCount` values respectively count
  complete and observed non-complete Schedule-to-normal-Scan creation handoffs;
  they do not describe final Scan execution results and do not have to sum to
  `runCount` after a process interruption. `lastRunTime` means last
  trigger-attempt time. Keep table/detail copy aligned with those meanings.
- A later occurrence uses the latest committed Schedule inputs at attempt start
  and makes one normal Scan Create attempt. The UI must not imply automatic
  occurrence retry, partial-batch compensation, Scan state tracking, or that
  saving a Schedule reserves current resources.
- First-phase management remains Schedule-only. Do not add occurrence history,
  scheduling provenance to Scan History, Run Now, a global scheduler setting,
  runtime tuning, a scheduler dashboard, metrics/alerts, or scheduler health
state. The existing four-step workbench remains the complete create flow.

## Organization Scope Selection

- Scheduled-scan creation adapts the organization-domain
  `OrganizationSelectionWorkspace` in `single` mode. The scheduled-scan module
  owns organization query and pagination state, validation, and the request
  payload; the shared workspace owns only selection presentation.
- Selecting an organization continues to set one `selectedOrgId`, and request
  construction continues to submit `organizationId`. Do not move that payload
  contract into the shared component.
- Do not reintroduce a scheduled-scan-specific organization popover, command
  list, card renderer, or translation namespace for generic workspace controls.
- Because this scope is fixed single-select, pass `showSelectionCount={false}`;
  keep selected-card feedback and the clear action.

## Target Scope Selection

- Scheduled-scan target mode adapts the target-domain
  `TargetSelectionWorkspace` as a controlled single-select workspace with the
  same header, toolbar, responsive card grid, selection treatment, and footer
  geometry as organization mode.
- The scheduled-scan state hook owns target live search, page tokens, pagination,
  and the selected target ID. Request construction continues to submit one
  `targetId`; the workspace must not construct scheduled-scan payloads.
- Do not reintroduce the target `CommandList`, a separate selected-target badge,
  or caller-local target card markup.
- Both organization and target modes omit numeric selection counts because each
  accepts exactly one entity; both modes retain selected-card feedback and clear.

## Workflow Reference

- A scheduled scan stores the selected canonical `scanWorkflows/{id}` reference,
  not a workflow revision or Profile snapshot.
- The dialog loads the selected workflow's parent-scoped Profile and submits the
  complete displayed Step configuration. It must not merge presets or multiple
  workflow configurations.
- Step 3 uses the same configuration-surface validate-and-reveal operation as
  ordinary Scan submission before moving to step 4. The drawer stays on step 3
  when any enabled Step's enabled configSection has an empty resource field, marks all such fields, switches
  valid YAML to form mode when needed, and expands/scrolls/focuses the first
  field. Selecting a resource clears only that field's error; Server save-time
  validation remains authoritative for current Catalog/file availability.
- A later trigger resolves the then-current workflow through ordinary Scan
  creation. Existing scheduled Scans keep their own frozen plans; executor
  retry/disablement policy is not decided by this UI. A failed occurrence is
  not automatically retried or allowed to disable the Schedule; a later
  occurrence may recover against repaired current inputs.

## Agent Assignment

- The create and edit workbenches reuse the Scan Agent selector. An optional
  Agent selection persists on the scheduled rule and applies only to future
  triggers; clearing it restores automatic assignment for future Scans.
- The selector must keep offline and unhealthy Agents selectable with warnings.
  A pinned Scan is not eligible to fall back to another Agent, so do not imply
  that saving a rule reserves capacity or can reassign an existing Scan.
- Create and edit use the same lazy canonical Agent collection as ordinary and
  quick Scan launch: no closed-state collection request, a bounded 50-row first
  page, normalized server search, and opaque-token incremental loading. A saved
  selected Agent is restored independently through Agent detail, so a page-out
  or unavailable resource never silently changes the scheduled assignment.

- `scheduled-scan-page-layout.ts` owns route-level layout constants that affect both resolved content and loading content: default page size, timeline body height, and table loading row height. Its table loading row height must alias the shared dense table's actual row rhythm rather than restating a numeric value. The timeline stays vertical below the wide-desktop breakpoint and projects up to five entries as a horizontal sequence above it. For multiple entries, the wide connector spans the full content width while the time group sits immediately above the node track; each text field is width-bounded and truncated so an entry cannot widen a timeline column. Its resolved and loading states must import the same height contract.
- `scheduled-scan-page-sections.tsx`, `scheduled-scan-page.tsx`, and `scheduled-scan-data-table.tsx` must import those constants instead of repeating `min-h-[...]`, `pageSize = 10`, or table loading row numbers locally.
- `ScheduledScanPageLoadingState` and `ScheduledScanDataTableLoadingState` live in `scheduled-scan-page-sections.tsx`, next to the resolved section shells. Do not reintroduce `scheduled-scan-page-skeleton.tsx` or another page-local loading duplicate.
- `ScheduledScanDataTableLoadingState` is a thin wrapper around the real `ScheduledScanDataTable`. It requires `rows` from the caller; callers derive rows with `getDataTableSkeletonRowCount(SCHEDULED_SCAN_PAGE_SIZE)` so the loading row count follows the resolved page-size contract.
- `ScheduledScanPageHeader`, `ScheduledScanInsightGrid`, and
  `ScheduledScanOverviewSectionShell` own the visible first-screen chrome for
  both resolved content and skeleton content. They pair
  `scheduled-scan-header` and `scheduled-scan-insight-grid` on the real feature
  regions; skeletons must import those owners instead of restating outer
  `px-*`, `border`, `bg-card`, grid, or heading markup locally.
- `ScheduledScanTableSectionShell` owns the table section's padding and the paired `scheduled-scan-table` loading-geometry slot. Both `ScheduledScanPageLoadingState` and `ScheduledScanPage` must use it. The table may use the shared, documented `stableSurfaceRowCount` contract with the same fixed row count in both states; it reserves height only while loading, and resolved content follows its real rows or empty-state row. Do not add a page-local min-height or a second table-level loading owner.
- `/scan/scheduled` enters through a route-level readiness boundary. Once that
  boundary commits ready content, do not nest `scheduled-scan-page-content`
  around the same initial DOM or the route owner will not observe its paired
  page slots.
- Overview skeletons should stay coarse. They may replace volatile timeline content with a simple placeholder, but the shell, heading, body height, and page/table rhythm must come from the resolved layout owner.
