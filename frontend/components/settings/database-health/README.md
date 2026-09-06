# Database Health UI Notes

## Connection Metric Semantics

`coreSignals.connectionsUsed` is the count of established `sql.DB`
connections, including both in-use and idle sessions. Its percentage is the
established-connection count divided by `connectionsMax`; it is not an
active-query count.

- UI labels must use `已建立连接` / `Open connections` and `已建立连接占比` /
  `Open connection ratio`; do not describe this metric as active or currently
  occupied connections.
- The existing payload keys remain stable for compatibility. A future
  active-connection display must use backend fields derived from `InUse` and
  `Idle`, not infer either value from `connectionsUsed`.

## Loading Ownership

The database health workspace uses one visible page-owned loading owner for the
dynamic view import and the first database-health query. Loading geometry is
shared by the resolved view and its lightweight loading-state module, not by a
separate lookalike skeleton.

- The workspace's dynamic child fallback in
  `frontend/app/settings/database-health/database-health-workspace.tsx` MUST
  stay `null`.
- The workspace MUST wrap `DatabaseHealthView` in
  `HiddenReadinessRouteBoundary` with owner `database-health-page-route`,
  `mountContentWhileLoading`, and `DatabaseHealthLoadingState` as the only
  visible loading state.
- Initial query loading in `database-health-view.tsx` MAY render `DatabaseHealthLoadingState` with owner `database-health-page` only when the view is used without the workspace handoff. When `deferInitialSkeleton` is true, the view MUST return `null` during initial query loading and report `onReady` only after the first data, error, or empty frame is available.
- Do not route this page through `lazyPage()`, because its default `PageSectionSkeleton` creates a second visible skeleton scene before the page-specific loading state.
- Do not add inline `loading ? <Skeleton /> : ...` branches for the first-screen
  page body. Update the exported `DatabaseHealthLoadingState` in
  `database-health-loading-state.tsx` so workspace and direct query loading remain
  visually identical.
- `database-health-loading-state.tsx` owns the lightweight
  `DatabaseHealthLoadingState`; `database-health-view.tsx` and workspace data
  loading MUST reuse that export. Sidebar and the protected shell do not own a
  database-health fallback.
- Keep the snapshot, core metrics, optional signals, and alerts skeleton geometry aligned with the resolved MVP sections so the page handoff does not imply unsupported database-history or storage panels.
- Resolved database-health content and `DatabaseHealthLoadingState` MUST share page shell, section panel, header, grid, finding-row, and table-content geometry through `database-health-layout.ts`; do not reintroduce duplicated route-local card/table class strings in either path.
- Do not reintroduce `database-health-page-skeleton.tsx` or
  `DatabaseHealthPageSkeleton`; this page follows the shared lightweight
  loading-state model used by the migrated settings pages.
- Section body flush padding and section headers with descriptions are also layout contract facts. Use `DATABASE_HEALTH_SECTION_BODY_FLUSH_CLASS` and `DATABASE_HEALTH_SECTION_HEADER_WITH_DESCRIPTION_CLASS` instead of repeating `className="p-0"` or local `border-b px-6 py-*` strings in the resolved view or loading state.
- Snapshot header loading placeholders must mirror the resolved header slots exactly: title/description plus the auto-check cadence. Auto-check cadence is refresh-policy status, not database instance metadata, so it belongs in the snapshot header and MUST NOT reappear as a context-stat cell. The header MUST NOT repeat a broad overall-health badge; core metrics, findings, and alerts carry the actionable health state. Do not add action-like squares or button placeholders unless the resolved header grows a matching shared `Button` action.
- Snapshot title and auto-check text placeholders MUST retain the resolved typography line boxes through `DatabaseHealthTextPlaceholder`; fixed-height skeleton bars change the mobile header height and cause the snapshot slot to jump at handoff.
- Core-metric titles, values, and detail rows follow the same typography-line-box rule so the single-column mobile metric stack has the same height before and after handoff.
- The first findings placeholder preserves the two-line mobile description, evidence-wrap, and recommendation rhythm of the first resolved finding; its description collapses back to one line at `lg` to match the wider resolved layout. Keep that representative first row aligned when changing the diagnostics mock or finding layout.
- Snapshot metadata contains six database-backed values and uses the shared 2/3/6-column grid from `database-health-layout.ts`. Keep the loading count and resolved item count aligned so wide layouts form one complete row without an empty slot.
- The route boundary, resolved shell, and loading shell share a viewport-bound
  workspace contract from `database-health-layout.ts`. Snapshot, diagnostics,
  and alert text may vary by locale and payload, so the content region scrolls
  internally; neither branch may expand the route owner's first-screen height.
- The page also pairs `database-health-header`, `database-health-snapshot`,
  `database-health-metrics`, and `database-health-findings` across loading and
  resolved branches. `ContentHandoff` provides the outer `surface` slot.

The contract tests in `__tests__/page.contract.test.ts`, `__tests__/database-health-view.contract.test.ts`, and `__tests__/database-health-loading-state.contract.test.ts` guard this boundary.
