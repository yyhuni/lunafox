# Shared Data Table Guide

Read this guide before changing `frontend/components/shared/data-table/**` or any production business-list page that renders a reusable data table surface.

This file is the near-code contract for the shared table layer. `frontend/components/ui/README.md` defines the broader UI foundation rules; this guide explains how business-list tables stay aligned in practice.

## Owners

- `BusinessListDataTable` owns the default route-facing baseline for production business-list tables.
- `SmartFilterBusinessListDataTable` owns the default route-facing baseline for smart-filter business-list tables.
- `UnifiedDataTable` owns the shared table shell, row rhythm, column-width strategy hooks, sticky action support, bulk selection wiring, and pagination placement.
- Standard table pagination is a shared natural-flow sibling below the bordered table surface. The table root keeps `min-w-0` so the bordered table surface, rather than a Grid parent, owns horizontal overflow; normal vertical flow stays with the page. Do not wrap pagination into the table border, make it sticky, or add a route-local pagination frame.
- `DataTableToolbar` and `TableToolbar` own search/filter/action composition for reusable business-list toolbars.
- `ColumnVisibilityMenu`, `ToolbarActionMenu`, and `DenseRowActionMenu` own the shared dropdown trigger/content contract for dense business-list menus.
- `DataTableColumnHeader` owns the status-menu interaction for sortable columns: its trigger is limited to the localized title plus current sort icon, and its compact menu provides left-aligned ascending/descending actions plus hide-current-column when the TanStack column is hideable. Static headers stay non-interactive; route modules must not recreate direct-cycle sorting or local sort menus.
- `SimpleSearchToolbar`, `SmartFilterInput`, `DataTableFacetedFilter`, `DataTableFacetedFilterGroup`, `TargetTypeFilterSelect`, `DataTablePagination`, `SharedCompactPagination`, `DenseRowActionOwner`, `DenseRowActionButton`, `QuietCopyButton`, `ExpandableCell`, `TimestampCell`, `MonoValueCell`, `SingleBadgeCell`, and related helpers own the dense-list affordances that appear across routes. Frequent row commands should use `DenseRowActionButton` through `DenseRowActionMenu.leadingActions`; low-frequency commands remain inside the overflow menu.
- `QuietCopyButton` is the dense-table wrapper around `CopyButton` from `frontend/components/shared/feedback/copy-button.tsx`; new non-table copy affordances should use `CopyButton` directly and keep the same quiet icon, copied-state, clipboard fallback, and toast behavior.
- Route modules own domain copy, column definitions, data wiring, and route-specific actions. They do not own a parallel table system.

## Business-List Baseline

`/targets/` via `TargetsDataTable` is the canonical baseline for production business-list tables after migration. `/organizations/` is the third migrated backend-query page and follows the same rule with a narrower sortable set: only organization name and created time have server sorting; description, target count, selection, and actions remain ordinary headers. `/targets/:target/subdomains/` and `/scan/history/:scan/subdomains/` are migrated child-list batches; both use ordinary `dnsName` search, server `dnsName` / `createdAt` sorting, parent-bound page tokens, and no SmartFilter syntax. `/targets/:target/websites/` and `/scan/history/:scan/websites/` are migrated website child-list batches; both use complete raw URL exact search (`url=="<complete URL>"`), structured `statusCode` / `tech` / `webserver` / `contentType` / `vhost` filters, server `statusCode` / `contentLength` / `createdAt` sorting only, parent-bound page tokens, and no SmartFilter syntax. `/targets/:target/endpoints/` and `/scan/history/:scan/endpoints/` are migrated URL child-list batches; both use complete raw URL exact search (`url=="<complete URL>"`), the same structured filters as websites, server `statusCode` / `contentLength` / `createdAt` sorting only, parent-bound page tokens, and no SmartFilter syntax. `/targets/:target/directories/` and `/scan/history/:scan/directories/` are migrated directory child-list batches; both use complete raw URL exact search (`url=="<complete URL>"`), structured `status` / `contentType` filters, server `status` / `contentLength` / `createdAt` sorting only, parent-bound page tokens, and no SmartFilter syntax. `/targets/:target/screenshots/` and `/scan/history/:scan/screenshots/` are migrated screenshot gallery batches; both use complete raw URL exact search (`url=="<complete URL>"`), structured `statusCode` filter, server `statusCode` / `createdAt` sorting only, parent-bound page tokens, and no SmartFilter syntax. Screenshot URL, image, and updated time are not sortable controls. `/vulnerabilities/`, `/targets/:target/vulnerabilities/`, and `/scan/history/:scan/vulnerabilities/` are migrated controlled variants: they use complete raw URL exact search (`url=="<complete URL>"`), structured `severity` / `source` / `vulnType` filters, server `createdAt` sorting only, and no SmartFilter syntax; review state remains tab UI for global/target vulnerabilities only and scan vulnerabilities must not render review tabs.

- Standard route-facing business-list wrappers SHOULD start from `BusinessListDataTable` instead of mounting `UnifiedDataTable` directly.
- Smart-filter resource pages SHOULD start from `SmartFilterBusinessListDataTable` instead of treating `SmartFilterDataTable` as the long-term route-facing abstraction.
- `UnifiedDataTable` remains the lower-level shared shell for baseline wrappers and reviewed controlled variants.
- Business-list rows use the shared `ui.rowDensity="dense"` rhythm by default. Routes with a persistent identity marker may opt into `ui.rowDensity="comfortable"`, which owns the 64px row / 65px measured-box rhythm, matching cell padding, loading rows, and virtual-scroll estimates together. Do not reproduce that rhythm through route-local row-height utilities.
- Dense table/list toolbars use one compact 32px control system end to end: inline-icon search input, filter select, column visibility, page-level bulk add/import actions, add actions that remain inside the toolbar, page-size select, and pagination icon buttons.
- When dense toolbar filters wrap to a second line on desktop, right-side actions such as export and add stay aligned to the first search/filter row instead of centering against the full wrapped left block.
- Resolved paginated tables MUST NOT pad sparse or empty pages with spacer rows. Loading height belongs to `ui.loading` and shared loading rows only; once data has resolved, the table renders exactly the returned rows plus the single empty-state row when there is no data. `ui.stableSurfaceRowCount` is a loading-only shared-shell reservation: while `ui.loading` is true it reserves the shared header plus fixed row rhythm, then releases its `min-height` after resolution. It never inserts fake rows.
- Initial loading for business-list tables is owned by the same shared table surface as resolved content. Use `ui.loading`, `ui.loadingRowCount`, and, when resolved rows are intentionally multiline, `ui.loadingRowHeightEstimate` on `UnifiedDataTable` / route-facing wrappers instead of route-local skeletons that copy toolbar buttons, table headers, row heights, column layout, or pagination.
- Cold route entry uses `ui.loadingPresentation="initial"`: the shared table keeps production column/action geometry, marks the surface busy, and replaces toolbar controls, column labels, totals, and page numbers with non-interactive structural skeletons. Later query/refetch loading keeps the default `rows` presentation so already revealed chrome is not replaced.
- `initialLoadingToolbarFilterCount` renders filter placeholders as siblings of `SearchToolbarSkeleton` inside the shared left toolbar flex row. This preserves the same narrow-screen wrap boundary as resolved `toolbarLeft` filters; do not pass these slots through the search skeleton's `after` content when the resolved filter is a sibling.
- Initial loading MUST NOT expose metadata-derived `0` totals or an empty-state conclusion before the owning list query succeeds. Empty state is a resolved success state, not a pending state.
- `UnifiedDataTable` emits exactly one `data-loading-slot` for its toolbar, body, and pagination in both loading and resolved states. The defaults are `data-table-toolbar`, `data-table-body`, and `data-table-pagination`. A route geometry contract passes one complete `ui.loadingSlots` set when it needs feature-specific names; an owner with multiple tables must give every table a distinct complete set rather than reusing a slot or adding a lookalike placeholder.
- Route-local table skeleton components may remain as thin wrappers for owner metadata, columns, empty data, and row count. They should not import `ActionSkeleton`, `SearchToolbarSkeleton`, `CompactPaginationSkeleton`, table layout constants, or dense row height constants to recreate the shared table chrome. Thin wrappers SHOULD require `rows` from the caller rather than defaulting to `rows = N`; the caller derives that value from the resolved page size with `getDataTableSkeletonRowCount(...)`, while the wrapper keeps the real table `pageSize` shell instead of setting pagination `pageSize` to the loading row count.
- Smart-search surfaces must choose density through the shared `toolbarDensity` owner on `SmartFilterInput`; use the compatibility `density` alias only while migrating existing callers. Dense smart filters default to an inline search icon and no adjacent submit button; explicit icon or attached primary submit buttons are opt-in for search-heavy surfaces.
- Search-style pagination, picker pagination, and compact result feeds should reuse `SharedCompactPagination` instead of rebuilding their own 32px select/button geometry.
- `SimpleSearchToolbar` defaults to the shared inline-icon `SearchInput` pattern. Explicit search submit buttons are opt-in and, when retained, must stay adjacent to the input through `SimpleSearchToolbar`; do not invent page-local split-control markup. Use its default fixed width for the standard dense table search field, and use `inputWidthMode="fill"` only when the search field is intentionally controlled by a parent flex/max-width track. Do not restate the fixed default through `inputClassName="w-full sm:w-72 lg:w-80"`.
- Dense row actions should collapse into shared menu or icon-button patterns through `DenseRowActionMenu` / `DenseRowActionOwner`, and quiet copy affordances should route through `QuietCopyButton` / `CopyButton` instead of page-local tooltip/timer/clipboard bundles.
- Row-detail activation and row actions MUST stay separated. `UnifiedDataTable` owns row activation through `behavior.onRowClick`; row action controls and dropdown menu portal content must be ignored by the shared row activation guard instead of adding page-local row click patches. A route that has a high-probability detail panel MAY supply idempotent chunk warming through `behavior.onRowIntent`; it fires for row pointer/focus intent only and MUST NOT activate the row, fetch detail data, or add another visual state.
- Sticky action and support columns are a positioning affordance, not a separate visual surface. Use `meta.stickyRight` only for reviewed tables where horizontal overflow is expected and the right-side action/support column must remain reachable while scrolling. Do not make action columns sticky by default on ordinary business-list pages; if a new production file needs `stickyRight: true`, add it to the reviewed allowlist in `components/shared/data-table/__tests__/unified-data-table.contract.test.ts` with a narrow reason in the local change. `TableHeader` owns the visible header-row surface, body row states come from `TableRow`, and sticky header/body cells use `bg-inherit` so their opaque scroll backdrop exactly matches the owning row. Do not add page-local `bg-card`, `bg-background`, `bg-secondary`, `bg-muted`, `border-l`, local shadows, gradients, or hover/selected background overrides to sticky action cells.
- Stable taxonomy and multi-select filters use shared wrappers such as `TargetTypeFilterSelect` or the shared `DataTableFacetedFilter`; do not duplicate select, popover, badge, or checkmark styling per route.
- Production faceted filters must declare their option source before they are shown. Allowed sources are stable enum, parent-scoped backend options / aggregation, complete local dataset faceting, or current-page temporary supplement. Backend-paginated business lists must not present current-page row values as the full option source; current-page values may only preserve selected values or supplement missing edge values while a stable source owns the menu.
- Parent-scoped backend options / aggregation responses use `results[{ value, label, count? }]`. Nested resource lists must bind options to the same parent and permission boundary as the list, for example target websites, scan websites, target hostPorts, or scan hostPorts.
- `DataTableFacetedFilter` options MAY include an optional `count` when the route already owns matching summary totals. The shared menu renders that count as the right-aligned numeric affordance; routes should pass their existing summary data instead of rebuilding option rows locally.
- `DataTableFacetedFilter` popovers SHOULD include the shared `CommandInput` search field, following the shadcn Tasks faceted-filter pattern, so longer option sets such as tags, sources, statuses, and vulnerability types remain searchable without rebuilding page-local inputs. The popover search field should stay visually quiet: its 16px search icon aligns to the option checkbox column, the input text starts on the same column as option labels, it has no inner focus ring, and selection emphasis is owned by the option row background plus checkbox state.
- `DataTableFacetedFilter` trigger selected-state display follows the shared LunaFox dense-toolbar pattern: on desktop, one selected option renders as a text badge; two or more selected options collapse to a numeric count badge. On narrow viewports the trigger may show only the numeric count to preserve toolbar space. Text badges must left-align their label and truncate from the right edge inside the badge. The trigger width SHOULD follow the currently visible summary only; do not add hidden placeholder labels, longest-option reserves, or invisible grid layers that make the empty or numeric state wider than its visible content. Do not replace the desktop one-label case with an always-numeric badge.
- `DataTableFacetedFilterGroup` owns the visible active-filter reset affordance. Every toolbar section that renders one or more `DataTableFacetedFilter` controls MUST wrap them in one `DataTableFacetedFilterGroup`; when any filter in that group has selected values, the group renders exactly one compact adjacent reset action for that whole group. Single-filter pages still use the group wrapper so the reset placement and copy stay shared. Do not add page-local faceted-filter reset buttons next to targets, vulnerabilities, IP addresses, scan history, agent status, wordlist tags, or severity filters.
- `DataTableFacetedFilter` popover width is owned by the popover content, not by the selected-state trigger summary. Do not let the popover inherit `PopoverContent`'s anchor minimum width; the default popover width stays fixed at 192px so short option sets do not shrink and long option sets do not make the menu jump. Wider standard menus MUST use the shared `contentSize="wide"` API instead of page-local width utilities such as `contentClassName="w-56"`.
- `DataTableFacetedFilter` popover scrolling is owned by the option list only. The search input stays at the top and the shared “重置” and “应用” actions stay fixed at the bottom; option toggles are local drafts and must not query until “应用” is clicked. Do not put footer controls inside the scrolling `CommandList`.
- Related structured facets that would otherwise occupy multiple dense-toolbar controls use `DataTableFacetPanel` as one aggregate entrypoint. Multi-facet panels own the selected count, a left-hand category rail headed by the shared “筛选维度” label, and a direct right-hand option list; a single-facet panel omits the redundant rail while retaining the same fixed-height draft, reset, and apply behavior. Do not nest `DataTableFacetedFilter` popovers inside the panel. Draft changes do not query until the operator chooses `应用筛选`; `重置` clears that draft without querying. Option overflow scrolls inside the option list so content cannot resize the outer panel. The aggregate trigger's count is the only applied-filter summary; do not add chips, individual removal controls, or a second toolbar row below it.
- Column visibility, export, refresh, selected-row actions, page-level bulk add/import actions, and pagination remain shared table responsibilities.
- Selected-row business actions render through `SelectedRowActionBar` or `UnifiedDataTable.actions.selectedRowActions`. Do not place selected-row delete, unlink, review, acknowledge, archive, or status-change commands in `TableActions` or toolbar dropdown menus.
- `SelectedRowActionBar` owns the final action feedback: default, `success`, and `muted` actions rest at the sidebar's subdued foreground and use its neutral accent surface on hover, while semantic status color stays on the icon. `destructive` actions keep their destructive text and tinted hover background. Call sites MUST NOT add page-local hover classes or infer a persistently emphasized action from array order.
- When a route stores selected rows outside the table, pass that selected-row array back through `state.selectedRows` so confirmation flows, successful mutations, pagination resets, and explicit clear-selection actions keep external state, internal row selection, pagination summaries, and action bars synchronized.
- Column visibility menus SHOULD route through `ColumnVisibilityMenu`, which owns the compact trigger and `DropdownMenuContent width="content-fit"` policy for localized column titles.
- Column visibility is enabled by default on shared business-list tables. Minimal controlled tables with no optional columns MAY set `ui.showColumnVisibility=false`; keep the decision at the route wrapper and do not rebuild the toolbar locally.
- Dense toolbar action menus SHOULD route through `ToolbarActionMenu` instead of page-local `DropdownMenuTrigger` / `DropdownMenuContent` markup.
- Dense row action menus SHOULD route through `DenseRowActionMenu` instead of page-local `MoreHorizontal` trigger markup.
- Business-list tables default to semantic width allocation: fixed narrow support columns plus bounded flex columns declared next to their size bounds. Multi-flex routes designate one primary fill column for spare desktop width.
- New route definitions MUST use `columnDef.meta.widthPolicy`; `expandColumnIds` remains a compatibility input for existing callers and maps its first ID to the legacy fill column.
- Overflow is solved inside the cell first through truncation, badge budgets, expandable disclosures, or stable summaries before the whole table is allowed to scroll horizontally.
- Standard route-facing business-list tables SHOULD keep their default visible column set within the shared desktop width budget enforced by contract tests; if a route intentionally exceeds it, that deviation needs a reviewed near-code note or equivalent exception path.
- Dense badge-list cells SHOULD prefer `ExpandableBadgeList singleLinePreview` over route-local `flex-wrap` strips when the row belongs to the 48px business-list rhythm. A comfortable-row route with an explicit preview-count requirement may use `wrapPreview` to render every capped item before disclosure; it must remain bounded by `maxVisible` and still use the shared expander for the remainder.
- The `default compact` desktop width state is the baseline. A route should not spend width casually just because a column can hold more content.
- `user-driven width pressure` is not an available control: shared business-list tables keep route-declared and automatic column widths, then use disclosure owners and horizontal overflow for exceptional content pressure.

## Natural Table Flow

Standard business-list tables keep their header, rows, and pagination in
document flow at every breakpoint. The bordered table surface contains only the
table, so the browser/page owns normal vertical scrolling and the header scrolls
with its rows. Do not introduce a route-owned height-fill prop, an internal
vertical scroll viewport for ordinary paginated rows, a sticky baseline header,
or fixed/sticky pagination.

- `ui.stableSurfaceRowCount` reserves the actual shared header and row rhythm
  only while `ui.loading` is true. The resolved surface releases that
  reservation and follows its real rows or empty-state row without rendering
  fake rows. It does not establish a viewport, fixed footer, or internal
  normal-table scroll region.
- Explicit virtual scrolling for large local datasets remains separate from the
  ordinary paginated-table baseline. It may retain its existing bounded scroll
  container but must not turn pagination into a table-frame footer.

## Column Width Taxonomy

Business-list tables use semantic width roles instead of treating every `size / minSize / maxSize` triple as a route-local guess.

- `fixed support`
  - Selection, row actions, icon-only utilities, and similarly tiny support columns.
  - These columns are usually fixed width, with `minSize` and `maxSize` matching.
- `stable metric`
  - Dates, status codes, counts, lengths, response time, and similarly short scan-friendly values.
  - These columns may allow limited widening, but they should not become the main consumer of spare table width.
- `flexible summary`
  - Technology tags, badge summaries, owner/taxonomy summaries, and similar dense multi-value columns.
- These columns start compact; shared disclosure owners reveal additional summary content without changing the column width.
  - When they share remaining width with another flex column, declare `widthPolicy: { mode: "flex", flex: <weight> }` beside the column bounds.
- `primary flexible text`
  - Names, URLs, hosts, titles, and other columns that carry the main scanning payload for a row.
  - The primary column in a multi-flex route declares `widthPolicy: { mode: "flex", flex: <weight>, fill: true }` and receives spare width after bounded flex columns cap.

Routes do not need identical numbers, but they do need the same role semantics.

## Semantic Width Allocation

`UnifiedDataTable` derives default widths from visible columns, their `size` / `minSize` / `maxSize` bounds, `meta.widthPolicy`, and the measured table container. It does not measure cell content, so paging, filtering, and refreshes do not make columns jump.

- `fixed` is the default: the column keeps its declared size within its bounds.
- `flex` starts at `minSize`; remaining width is distributed by `flex` weight.
- A flex column that reaches `maxSize` stops growing and its unused share is redistributed to uncapped flex columns.
- After every bounded flex column caps, the one `fill: true` flex column receives any remaining desktop width.
- If visible minimum widths exceed the container, columns do not shrink below `minSize`; the shared horizontal overflow container remains the fallback.

For now, the shared contract stops at width roles plus contract tests. Do not introduce a shared width-preset module until multiple route families need the same numeric bands and the extra abstraction removes real drift instead of hiding it.

## Resize And Disclosure Rules

- `fixed support` and `stable metric` columns keep their declared width band; shared tables do not expose direct header resizing.
- `stable metric` and other fixed-band headers should reserve at least their shared header title floor, including table-head padding, sortable trigger padding, sort icon width, icon gap, and measurement buffer when the column is sortable. Route-local `minSize` / `size` values must not undercut that floor; if a default header title truncates, fix the shared floor calculation before adding page-local width patches.
- Single-label badge columns should reserve at least one full badge width for the longest currently visible label. When the badge has a finite categorical label set and the table can first render without rows, declare every rendered local label through `meta.singleBadgeValues`; this keeps initial loading, empty data, pagination, and resolved rows on the same column axis. Route-local `size` / `minSize` / `maxSize` values must not undercut that badge floor, and page modules should prefer `SingleBadgeCell` over raw table-local `<Badge>` markup.
- `flexible summary` columns should keep a single-line preview by default and use shared disclosure owners such as `ExpandableBadgeList` / `ExpandableTagList` before asking the whole table to overflow.
- `primary flexible text` columns should absorb spare desktop width intentionally and use truncation or expandable text owners before defaulting to multiline dense rows.
- Do not make “show every tag/value by stretching the table” the standard behavior. Dense business-list tables prefer explicit disclosure.

## Controlled Variants

Not every table in the product must be pixel-identical, but the default path is intentionally narrow.

- `business-list` is the default variant. Most resource index pages, list pages, and admin tables should start here.
- Split-pane or master/detail tables may layer extra detail behavior on top of the shared row rhythm and table shell, but they should still reuse shared toolbar, pagination, and cell-disclosure helpers whenever possible.
- Dashboard preview tables, result feeds, and content-tab detail panes should reuse shared `TabsList` / `TabsTrigger` geometry (`variant` + `size`) instead of reintroducing page-local `h-*`, bottom-rail, or pagination owners.
- Log, evidence, matrix, or inspection-heavy tables may need a reviewed route wrapper when their interaction model is materially different. Even then, reuse shared actions, pagination, search, or row rhythm before introducing page-local replacements.
- A route-specific table system is allowed only when the shared baseline truly does not fit and the reason is captured through a near-code note, contract test, or approved exception path.

## Anti-Drift Strategy

The standard is enforced by shared implementation first, then by docs and tests.

- New or migrated business-list pages MUST start from `UnifiedDataTable` or a shared wrapper built on top of it.
- Prefer `BusinessListDataTable` / `SmartFilterBusinessListDataTable` when the route is a standard business-list page, and reach for direct `UnifiedDataTable` only when building a reviewed controlled variant or lower-level shared wrapper.
- Prefer extending shared table helpers over copying markup into a page component.
- Prefer extending shared table loading mode over tuning route-local placeholder heights or button counts. A mismatch between table skeleton and resolved table is a shared-owner issue first, not a page-by-page sizing task.
- A first-screen `ContentHandoff` whose resolved result can contain fewer rows
  than the loading page MAY opt into `ui.stableSurfaceRowCount`. While loading,
  this keeps the shared table shell at its actual rendered `shared header
  rhythm + N shared row rhythms`; after resolution it deliberately returns to
  intrinsic content height without fake rows. Virtual-scroll measured-box
  estimates are separate and must not inflate this visual reservation. The
  route MUST pass the same documented row count to both branches; do not use
  mock result counts or a page-local `min-h-*` workaround.
- If multiple routes need the same new table affordance, move it into `frontend/components/shared/data-table/**` and add or update contract tests in the same area.
- Route modules should describe what data the table shows, not how a generic table system is built.
- README guidance, contract tests, and UI-foundation checks work together. Do not rely on prose alone to keep tables aligned.

## 查询控件与排序职责

后端分页业务列表的搜索、筛选、排序和分页必须作为一个查询状态处理。页面不要再分别维护 `search`、分面数组、排序字段和分页 token 后临时拼请求；迁移页面应使用 `BusinessListQuery` 以及 `createBusinessListQuery`、`applyBusinessListControlChange`、`setBusinessListPage`、`compileBusinessListFilter`、`compileBusinessListOrderBy`、`toggleBusinessListSorting` 这些共享 helper。

- 搜索、筛选或排序变化必须回到第一页，并清空旧 `pageToken`。
- 后端分页列表的文本搜索 SHOULD 将输入态和查询态分离。默认使用约 300ms debounce 自动提交，Enter 立即提交当前输入；只有接口很轻、数据量明确很小的本地受控列表，才可以每次输入都直接改后端查询状态。
- 共享搜索 hook 执行 hard cut：业务列表搜索输入态使用 `searchInput` / `handleSearchInputChange`，提交态使用 `commitSearch` 或 `commitNow`。不要在业务列表生产路径继续暴露 `handleSearchChange`、`setValue`、`submit`、`setLocalSearchValue` 或 `handleSearchSubmit` 这类兼容 API；旧命名会模糊“输入 draft”和“已提交查询”的边界。
- 多选分面筛选默认选择即生效，因为点击筛选项是明确意图且频率低于键盘输入。若一个面板承载多个高成本筛选维度或会触发重查询，应提供“应用筛选”这类显式提交入口，而不是每次勾选都打重请求。
- 翻页只能改变 `pageIndex` / `pageToken`，必须保留当前搜索、筛选和排序。
- 使用 `DataTablePagination` 呈现的 AIP `pageToken` 列表必须声明 `paginationNavigation: { mode: "cursor" }`，提供清空当前 token 并重新请求第一页的首页 reset，以及服务端已返回 token 支持的上一页和下一页。不得显示末页或任意页码跳转，因为这些操作会承诺一个尚未取得的 opaque token。
- 多选分面筛选的同一分面内是 OR/IN，不同分面和搜索之间是 AND；空数组省略，不表示匹配空集合。
- 普通分面筛选编译器只处理页面声明的字段和值编码器；`SmartFilterInput` 或类 FOFA 全局搜索语法属于智能查询字符串，不由普通分面筛选编译器隐式解析。
- 后端分页排序点击循环为：端点默认排序 -> 列声明的首次方向 -> 相反方向 -> 端点默认排序。

排序能力必须声明归属。`UnifiedDataTable` 支持 `state.sortingMode`：`client` 用于完整前端数据集或受控本地变体，`server` 用于已接入后端 `orderBy` 的分页列表，`none` 用于不可排序或尚未完成后端排序迁移的列表。`BusinessListDataTable` 在传入 `paginationInfo` 且未显式声明 `sortingMode` 时默认使用 `none`，避免把 TanStack 当前页排序展示成全量排序。

后端分页列表的查询 affordance 必须跟真实查询语义绑定。`UnifiedDataTable` 和 `useTableState` 在收到 `paginationInfo` 且调用方没有显式声明 `sortingMode` 时默认按 `none` 处理；页面不得靠当前页本地排序来显示表头排序。搜索框、`DataTableFacetedFilter`、`TargetTypeFacetedFilter` 等筛选控件也必须只在父级确实把搜索/筛选值编译进后端请求，或列表被明确分类为完整本地数据集/受控变体时展示。未迁移页面应直接隐藏这些控件，而不是显示禁用态或只过滤当前页。

筛选生效语义和筛选选项来源是两层契约。后端分页页面可以把多选筛选编译进后端 `filter`，但筛选菜单的完整选项必须来自稳定枚举、父资源范围后端 options / aggregation，或已登记的完整本地数据集。当前页 rows 派生值只能作为已选值保留或临时补充，不能被命名、测试或文档化为完整选项来源。网站列表的状态码、技术栈、Web 服务器、内容类型、虚拟主机，以及 IP 地址列表的端口号，必须走目标或扫描范围内的稳定 options；完成迁移后要 hard cut 当前页完整选项派生生产路径。

聚合型业务列表必须把“用户看到的一行”作为分页和排序对象。若页面行由多条明细记录聚合而来，例如 host-port 聚合成 IP 地址列表，后端必须先按父资源限定并应用筛选，再形成聚合业务行，然后计算 `totalSize`、排序和分页；前端不得对已分页明细行做二次聚合来模拟列表。

后端分页列只有同时具备 `meta.orderBy` 和 `meta.serverSortPerformance` 时，才允许在 `server` 模式下显示排序表头。操作列、选择列、组合展示列、本地计算列、后端未支持字段、没有索引/查询计划/规模上限依据的字段，都必须渲染为普通表头。已迁移页面要硬切旧路径：删除旧查询别名、旧请求构造、当前页本地排序回退和临时双写/双读；共享客户端排序能力只能留给明确分类的本地数据集。

新增或恢复后端分页表格的排序表头时，必须在同一变更中补齐后端 `orderBy` 白名单、稳定兜底排序、索引/查询计划/规模上限依据，以及能机械拦截回归的契约测试。默认优先给公开排序字段建立贴近实际 `ORDER BY` 的组合索引，例如 `(排序字段, id)`；如果不建索引，必须在近代码文档或 OpenSpec inventory 中写明可接受的数据规模上限或查询计划依据。

## Review Triggers

Revisit the shared table layer when any of the following happens:

- More than one route needs the same table customization.
- A page starts mixing 36px page-header controls with 32px dense-table controls in the same toolbar.
- Column widths reflow when data, locale, or badge counts change.
- A route introduces page-local pagination, page-local search controls, or a second row-action system because the shared layer was not extended first.
