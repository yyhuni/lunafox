# Scan History Components

## Column Semantics

The scan history list separates the scan's bound `scanWorkflow` from the
executed task/engine summary shown in the table. A scan is created with one
canonical scan workflow, while the table's execution column displays the
engine identities captured from `scan_task.engine_id` through
`plannedEngineIds`.

The browser resolves each ID through the installed-engine catalog's current
locale resource (`engine.displayName` and `engine.description`). The history
table keeps the localized name as the Badge text and discloses the matching
description through the shared Tooltip; the runtime task summary shows that
same description inline. Do not derive either value from the mutable workflow,
persist localized labels on a scan, render a raw engine ID, or fall back to a
different locale when a catalog entry is unavailable.

Runtime task status remains task-level: a planning-time skipped task renders
the primary `skipped` state plus its stable `skipReason` (for example,
`user_disabled` or `target_not_applicable`). Its log section remains visible
for audit context, but a task without a saved plan or execution session must
show the planning-skip empty state and must not issue a runtime-log request.

The history table keeps the target on a compact bounded flex band because most
values are short domains, while scan summary and executed-engine badges need
more horizontal room. The executed-engine column owns spare desktop width after
the three flexible columns reach their reviewed bands; it avoids unnecessary
engine Badge wrapping when the table has room. Do not make target or summary
the fill owner or reintroduce route-level `expandColumnIds` for this table.

The scan detail runtime header separately shows the bound workflow name when
its stable `scanWorkflow` reference resolves through the workflow catalog. This
is a configuration identity, not a substitute for the executed-engine summary;
if the referenced workflow is unavailable, omit the metadata item instead of
rendering its internal resource name.

Assignment provenance belongs to Scan detail/runtime only. The Scan detail
projection is the sole source for canonical Agent reference, read-time current
name, transport status, health state, automatic/pinned assignment, and explicit
`agentDeleted`; runtime UI must not query an Agent collection to fill them.
Agent rename therefore appears on the next detail read. A deleted Agent keeps
its canonical reference and assignment provenance but omits current name and
runtime state. Missing optional runtime state is rendered as unavailable and
must not be inferred as offline or deleted. The dense history list must not gain
an Agent column or another Agent runtime projection.

## Trigger Provenance

Scan history renders one fixed, localized trigger-source marker from the
server-owned required `triggerType`: `manual`, `scheduled`, or reserved `ai`.
It is a persisted Scan fact, not a current-user, Schedule, or occurrence
lookup, so Schedule updates and deletion cannot change a historical row. In the
dense history list, the compact semantic icon is placed immediately before the
target name; when the target column is intentionally hidden in a target-scoped
embedded list, the same marker leads the summary cell instead. Hovering or
focusing the icon reveals the localized trigger label, while the detail view
continues to show the full text label. The marker is display-only: do not add
trigger filters, search, sorting, query parameters, or Schedule/occurrence links
in this phase.

## Execution Input Source Visibility

Each Scan row and detail projection retains its required persisted
`inputSource` (`scanSnapshot` or `targetInventory`). The overview header and
runtime-detail drawer show its localized value so an operator can explain what
the execution actually used. The history table deliberately does not add an
input-source column, filter, sort, query control, or another launch control;
the field remains transport data for this list and detail-only context here.

Frontend transport must reject a missing, blank, or unknown source rather than
guessing `scanSnapshot`. `targetInventory` describes a Server request-time
Target asset read, not a newly frozen history snapshot, so the detail label is
the durable explanation for the Scan's input policy rather than a record-count
claim.

## Query Control Contract

`/scan/history/` uses the shared `BusinessListQuery` lifecycle as its only
search, filter, sorting, and pagination state. Do not reintroduce separate
`searchQuery`, `statusFilter`, page state, or scan-history-local filter string
builders.

Search compiles to fuzzy `targetName` filtering. Status is a stable enum
multi-select facet that compiles as exact `status` predicates, with OR semantics
inside the facet and AND semantics with search. Sorting is server-owned and is
limited to `createdAt`; the default is `createdAt desc`, and status remains
filter-only.

The status column remains titled `Status` because its primary value is the scan
lifecycle state. On the history list, runtime-detail disclosure belongs to the
whole row: rows expose shared hover/focus feedback and keyboard activation,
while the status cell remains a static lifecycle display. Selection checkboxes,
target links, and row action menus remain independent controls and must not
open the runtime drawer. Do not rename the column to a mixed `Status / View`
label or add a trailing disclosure arrow. The shared column factory may still
support an explicit status-cell action for other list consumers.

Control changes such as search, status, page size, and sorting must reset to the
first page and clear cached `pageToken` values. Plain page movement must preserve
the current search/filter/sort query and use the cached token returned by the
backend.

History pagination is cursor-only: the result total remains a summary, the
active response `nextPageToken` alone enables next, and the current-query token
cache alone enables previous. The first-page control clears the cache and
requests page one. Do not infer page numbers, last controls,
or a total-page count from the result total.

## Selected-row Actions

When the history table is not embedded with `hideToolbar`, a non-empty
selection renders Delete and Stop through the shared `SelectedRowActionBar`.
Stop remains visible for terminal-only selections but is disabled with localized
feedback; it is enabled when at least one selected Scan is `pending` or
`running`. The page opens the count-aware confirmation dialog from
`useScanHistoryActions`, sends all selected IDs to `useBatchStopScans`, and
clears selection only after the mutation has completed list/statistics
invalidation. A rejected mutation leaves the current selection intact.

## Page Refresh Contract

`/scan/history/` owns one manual refresh action on the right side of the page
header description row. The compact ghost control shows the current client page
mount time, then the latest coordinated refresh completion time after a manual
or automatic refresh; it may show the explicit not-yet-updated state only during
the server-to-client handoff before the first client effect. The action refetches
the active scan-statistics query and the currently mounted scan-list query
together. It must preserve the current search, status filters, sorting,
pagination, and resolved content while those requests are pending.

When the latest statistics contain a `pending` or `running` scan, the page-level
refresh owner also coordinates the same two active queries every three seconds
while the document is visible. It pauses the timer when the document is hidden,
refreshes once when visibility returns, and stops polling after the statistics
show no active scans. Automatic and manual refreshes share one in-flight guard.

Do not restore a second refresh action inside the scan-history table toolbar.
Embedded scan-history lists keep their existing toolbar visibility rules and do
not gain page-level refresh ownership. The page-level polling does not add
polling to embedded lists or subscribe to task-completion events.

## Runtime Detail Refresh

An open runtime detail drawer polls the Scan detail and runtime logs every three
seconds only while the server-owned Scan status is `pending` or `running`.
The refreshed detail is the source for the drawer's status, runtime task
lifecycle, Agent projection, and result summary. Skip a tick while the prior
detail request is in flight, and stop the timer immediately when the drawer
closes, the Scan reaches a terminal status, or no Scan ID is available.

This behavior is scoped to the runtime detail drawer. The ordinary Scan
overview retains its manual refresh behavior; the scan history list has its own
page-level foreground polling described above and must not gain polling from
this runtime detail state hook.

## Retention Notice

`/scan/history/` displays a compact retention summary beside the page-header
description text. It is an information icon that opens a shared
Tooltip on hover or keyboard focus. The Tooltip reads the existing
scan-statistics query's backend-owned `retentionPolicy` and describes whether
automatic cleanup is active. The header's right side remains reserved for the
manual refresh control.
This is explanatory product copy, not a setting: do not add a retention-duration
input, cleanup action, or backend-mode selector to the page.

## Loading Contract

The scan history list uses `ContentHandoff` as the single visible loading owner
for the initial table load. Keep the table toolbar, header, body, and pagination
geometry inside the skeleton so the resolved table can take over the same
surface.

The stats strip above the list is a different pattern. It is a fixed-height
metric strip that keeps its real shell mounted and swaps only the numeric value
slot through the shared `StatMetricRow` inline-loading path. Do not wrap that
strip in a second page-wide handoff just because the counts are pending.

Route timing evidence for `/scan/history/`:

- desktop `1280x720`: `1300ms` shows `scan-history-stats-skeleton` plus
  `scan-history-list-view-content` in `loading`; `2200ms` the stat-strip
  skeleton slot is gone and the list owner is `content`
- mobile `390x844`: `1300ms` shows `scan-history-stats-skeleton` plus
  `scan-history-list-view-content` in `loading`; `2200ms` the stat-strip
  skeleton slot is gone and the list owner is `content`

This is a valid split: one stable metric strip shell with inline value
placeholders above one table workspace owner. It is not `title -> stats ->
table` same-layer staged reveal.

Skeleton rows are not a prediction of the final result count. They should use
`getDataTableSkeletonRowCount(pageSize)` from the shared loading layer so they
follow the active table `pageSize` with a first-screen cap of 10 rows. This keeps
`/scan/history/` aligned with its default 10-row request while embedded views
such as target overview can still render smaller first-page skeletons through
their own `pageSize`.

The initial and resolved `ScanHistoryDataTable` branches must also receive that
same page-size-derived `stableSurfaceRowCount`. It reserves the shared header
and row rhythm only while loading; after resolution the surface follows real
rows or the existing empty-state row. It does not render synthetic data rows or
use a route-local height approximation.

The first-screen history table keeps its loading skeleton on the shared dense
row rhythm. After resolution, summary and executed-engine Badge groups wrap
complete Badge items at their allocated column boundaries, and that row may
grow to its actual content height. The executed-engine column owns spare
desktop width before any wrapping occurs. Do not hide later Badge items, add an
internal cell scroller, widen fixed support columns, or predict data-ordered
row heights in the skeleton. A single engine name that alone exceeds its Badge
can retain its existing bounded truncation because the Badge title and Tooltip
disclose the full value.

`ScanHistoryListLoadingState` and `ScanHistoryListTable` must pass the same
`scan-history-list-toolbar`, `scan-history-list-body`, and
`scan-history-list-pagination` `loadingSlots` set to `ScanHistoryDataTable`.
The shared table owns those regions in both phases; do not add a route-local
lookalike toolbar, body, or pagination solely for geometry probing.

Do not switch the initial table load to an empty state, spinner, dynamic import
fallback skeleton, or page-local pulse placeholders. Empty state belongs after
the request resolves with zero rows; compact refresh/search pending states
belong in the resolved toolbar controls.

## Detail Overview Shell Handoff

`/scan/history/[id]/overview/` is a route-critical direct-import overview. The
detail shell must keep `ScanHistoryDetailShellLoadingState` visible until
`ScanOverview` reports its first stable real frame through
`useDetailShellReadySignal`. Both loading and resolved chrome consume
`app/scan/history/[id]/scan-history-detail-shell-layout.tsx`; update its
header, primary-tab, secondary-tab, and child-content slots together rather
than adding a parallel shell skeleton.

The shared detail-shell handoff, both state wrappers, and the content slot are
one viewport-bounded flex surface. A child route may own a taller scrollable
table or panel, but that natural height must not resize the outer
`scan-history-detail-shell` surface between its skeleton and first resolved
frame. Keep the breadcrumb placeholder on the same 20px `text-sm` line box as
the resolved header.

When the layout owns the visible shell skeleton:

- `ScanOverview` must suppress its own initial visible skeleton while
  `deferInitialSkeleton` is active
- the ready signal must mean the first stable overview frame or error frame is
  present, not just that the module mounted
- loading cleanup must remain presentation-only; do not move overview child
  queries earlier just to make the shell exit sooner

The overview route fallback skeleton is owned by
`scan-overview-loading-state.tsx`. Keep that module lightweight: it may depend
on shared skeleton primitives and `scan-overview-layout.ts`, but it must not
import the runtime drawer, log panels, overview state hooks, or
`scan-overview-sections`. Those modules belong to the resolved overview
workspace and can otherwise pull large client-only runtime dependencies into
every scan-history detail route.

`scan-overview-layout.ts` owns geometry shared by the lightweight fallback and
the resolved runtime overview: workbench shell, primary column, side-panel
width/sticky variant, and responsive summary strip row rhythm. If the resolved
summary item count changes, update the loading state placeholder count and the
layout contract together; do not leave the fallback on a stale column count.

Runtime detail tab loading must derive from the shared detail drawer tabs. The
overview fallback and runtime drawer loading state should render the two-tab
`DetailDrawerTabsList` / `DetailDrawerTabsTrigger` shell used by the resolved
runtime panel, not a local row of rectangular button placeholders or stale
four-tab guesses.

## Runtime Failure Diagnostics

Failed runtime tasks keep the stable failure kind beside the task status. The
expanded failure area renders one customer-facing explanation: a single fixed
failure summary (prefer the task summary and otherwise use the canonical
scan-level summary) followed by a non-empty `failureDetail` as controlled
customer guidance. Do not render task and scan summaries separately, and do
not derive guidance from task logs, raw errors, or the scan-level failure
object; legacy tasks without this optional field must not render an empty
diagnostic region.

## Terminal Delivery Diagnostics

The existing expanded `RuntimeTaskList` is the only frontend surface for the
Server-projected `runtimeTasks[].diagnostics` snapshot. It renders explicit
availability, delivery state, closed failure classifications, and bounded
per-result-type watermarks. `complete` and `none` describe typed-result
delivery only; neither claims that an Engine found or did not find assets.
When failed tasks are present, the first failed task opens by default so its
controlled failure summary, diagnostics, and progress logs are immediately
visible. The existing Collapsible trigger remains the sole owner of subsequent
open/closed interaction.

When availability is `unavailable`, show that state and the Agent-proven task
facts without inventing zero counters or a root cause from raw logs. Do not add
a diagnostic route, navigation item, history-list column, log parser, nested
diagnostic card, or client-side reconstruction from task progress logs. The
snapshot is retained with Scan history by the Server; it is not an Agent
workspace or 72-hour local-log retention view.

## Runtime Status Presentation

扫描运行详情继续使用后端稳定的 `pending`、`running`、`succeeded`、`failed`、`cancelled` 五态模型，但所有可见标签必须本地化，不能直接暴露枚举值。状态颜色和 Badge 外观统一复用 `status-config`，状态图标统一从 `@/components/icons` 入口获取，不得在运行详情维护局部颜色映射。

扫描级 `cancelled` 显示为通用“已取消”。任务级 `cancelled` 根据运行事实细分展示：存在 `startedAt` 时显示“已中止”，没有 `startedAt` 时显示“未执行”。这只是前端 presentation，不增加 API 状态，也不改变统计、筛选或生命周期判断；两种派生展示都继续使用 `cancelled` 的 muted 语义色。

## Runtime Progress Logs

扫描运行详情中的“全部进度”、单任务展开进度和独立任务进度日志都是 operator-facing 的原始输出，不是 Agent/Server 的结构化诊断日志。三个入口都使用 shared visualization 的 `RawLogViewer` 和共享 scan-log formatter，以保留 ANSI、连续的 `[YYYY-MM-DD HH:mm:ss] [LEVEL] message` 文本顺序、时间戳/级别着色、自然换行、内容转义和靠近底部时的自动跟随。扫描进度统一使用本地秒精度，不显示毫秒；`RawLogViewer` 仍兼容旧毫秒输入。单任务展开区域可以保留自己的高度上限，但不得重新拆成时间列、级别 Badge 和消息列。

扫描模块不得从 `settings/system-logs` 反向导入 renderer，也不要因为共享终端视觉而给原始进度日志增加结构化日志专用的搜索/级别工具栏、连接状态或 footer。

任务运行清单的“开始时间”使用本地时间并固定显示为 `YYYY-MM-DD HH:mm:ss`，不得只显示时分秒；这样跨日运行和历史任务仍保留完整日期上下文。

## Runtime Overview Side Panel

`/scan/history/[id]/overview/` uses a two-column workbench layout on ordinary
desktop widths and a stacked layout below that. The runtime details, asset
result summary, and scan modules side panel should enter the right column at
Tailwind `xl` (`1280px`) with a fixed auxiliary width of `w-80`; below `xl`, it
must remain a full-width block after the primary runtime workspace. Do not push
this breakpoint back to `2xl`, because that makes common laptop and admin
console widths read as a mobile-style stacked detail page.

## Detail Child Workspaces

- `/scan/history/[id]/{websites,subdomains,ip-addresses,endpoints,directories,screenshots,vulnerabilities}/`
  are route-critical direct-import child pages.
- Reason: once the scan-history detail shell reveals breadcrumb, primary tabs,
  and secondary asset tabs, the next first-screen owner is the child workspace
  itself. Hiding that workspace behind a null chunk fallback produces
  `shell -> blank child region -> child workspace skeleton`, which is not a
  standard progressive-loading handoff.
- Keep each child view's `ContentHandoff` as the only visible workspace owner.
  The route page should stay a thin `useParams` + shell gutter wrapper.
