# Target Components

## Add Target Drawer

- `AddTargetDialog` uses the shared `FormDrawer` with the responsive workbench
  layout so batch entry has a wider desktop canvas while narrow viewports stay
  full-width.
- The shared `BulkLineValidationInput` remains the primary task and owns line
  numbering, validation highlights, and input feedback. Do not rebuild its
  textarea shell locally.
- Organization association is optional and adapts the organization-domain
  `OrganizationSelectionWorkspace` in `multiple` mode. The target module owns
  organization queries and the submitted `organizationIds` state, but it must
  not recreate organization cards, search/scope controls, loading/empty states,
  or pagination presentation.
- The association workspace is collapsed by default behind the optional
  association trigger. The trigger owns the optional-field label, keeps the
  text aligned to the primary input, and reveals the unchanged shared workspace
  only on demand.
- The all/selected scope changes the visible current-page card grid without
  changing selection or resetting the result viewport. Do not import the
  organization-list business table or recreate its sorting and row actions.

## Target Selection Workspace

- `TargetSelectionWorkspace` owns the target-domain card selection presentation
  used by scheduled-scan creation: header, live search, all/selected scope,
  responsive target cards, loading/empty states, and pagination footer.
- The workspace is controlled and single-select. Scheduled-scan state owns the
  target query, page tokens, selected ID, validation, and submitted `targetId`.
- Organization and target picker footers are cursor-only. They retain the
  filtered result total and page-size selector, while next availability comes
  from the active response token and previous availability comes from the
  current-query token cache. The first-page control resets that cache and
  requests page one. Do not reintroduce page numbers, last
  controls, or total-page calculations.
- The header omits a numeric selection-count badge because scheduled scan permits
  one target; selected-card feedback and the clear action remain required.
- Target cards show target identity and linked organizations. Do not replace the
  workspace with a caller-local command list or move request construction into it.
- Selected target cards use the shared `interaction-accent` border; unselected
  cards retain the neutral border. Do not add page-local selection colors.

## Target List Workspace Handoff

- `/targets/` keeps the page header as stable route chrome and lets
  `AllTargetsDetailView` own the first-screen workspace wait through
  `ContentHandoff owner="all-targets-detail-view-content" layer="workspace"`.
- The owner remains registered for the automatic paired `surface` geometry
  contract; header chrome does not become a second first-screen owner.
- The initial list wait should read as `header -> workspace skeleton -> workspace content`,
  not as a page-wide full-screen loader and not as `header -> blank region -> table`.
- Keep the visible owner on the workspace itself. Do not add a route-level
  full-page skeleton just because the table is loading, and do not add a second
  same-layer visible owner inside the table before the initial handoff finishes.
- The target list initial loading state must reuse the resolved `TargetsDataTable`
  columns, search, type filter, pagination, sorting, actions, and visibility
  flags from `AllTargetsDetailViewState`; only the volatile rows should be
  skeletonized through the table `loading` / `loadingRowCount` props.
- `AllTargetsDetailViewLoadingState` and `AllTargetsDetailViewTable` must pass
  the same page-size-derived `stableSurfaceRowCount`; the shared table reserves
  its real header and row rhythm only during loading. Resolved sparse pages use
  their real rows or empty-state row without spacer rows or a target-local
  height patch.
- `/targets/` uses the shared natural table flow. The page owns normal vertical
  scrolling, the column-header row scrolls with table rows, and shared
  pagination remains below the bordered table surface. Target detail, drawer,
  tab, selection, and embedded callers must not introduce a route-owned
  height-fill, sticky header, or framed pagination variant.
- `AllTargetsDetailViewLoadingState` and `AllTargetsDetailViewTable` pass the
  same `all-targets-toolbar`, `all-targets-table-body`, and
  `all-targets-pagination` `loadingSlots` set to `TargetsDataTable`. The shared
  table renders each region once per phase; do not create a second named table
  shell around it.
- Do not reintroduce a target-list-only data-table skeleton wrapper that builds
  fake columns or fake translations for the normal query loading path.
- The target-list organization cell uses the shared `singleLinePreview` mode
  with an explicit expand control. It preserves the first visible organization
  and makes additional linked organizations available without letting narrow
  table rows grow after the skeleton handoff; wider cells still show as many of
  the first three as fit before the disclosure.

## Target Detail Overview Handoff

- `/targets/[id]/overview/` stays a route-critical direct import so the detail
  shell skeleton owns the first visible route geometry instead of introducing a
  route chunk gap.
- The target detail layout must keep `TargetDetailShellLoadingState` visible
  until `TargetOverview` reports a stable first real frame through
  `useDetailShellReadySignal`. Both loading and resolved chrome consume
  `app/targets/[id]/target-detail-shell-layout.tsx`; update its header,
  primary-tab, secondary-tab, and child-content slots together rather than
  adding a parallel shell skeleton.
- The shared detail-shell handoff, both state wrappers, and the content slot
  are one viewport-bounded flex surface. A child route may own a taller
  scrollable table or panel, but that natural height must not resize the outer
  `target-detail-shell` surface between its skeleton and first resolved frame.
  Keep the breadcrumb placeholder on the same 20px `text-sm` line box as the
  resolved header.
- When the layout owns the visible shell skeleton, `TargetOverview` must
  suppress its own initial visible skeleton by returning `null` while
  `deferInitialSkeleton` is active.
- The ready signal should represent the first stable overview frame or error
  frame, not merely that the module mounted.
- Do not move scheduled scan or scan history query timing earlier just to hide
  loading sooner. This pattern is presentation-only.

## Target Detail Child Workspaces

- The target detail shell owns the dynamic breadcrumb hierarchy: target detail collection, target display name, and the active primary workspace. It intentionally stops at the primary workspace, because secondary asset tabs and website-detail identity are already explicit in the page itself. Reuse the layout's canonical primary path map and localized tab labels; ancestors are links, while the current primary workspace remains a non-interactive location indicator. Do not add child-page breadcrumb parsing, website context, or target queries to expose deeper route levels.
- `/targets/[id]/{websites,subdomains,ip-addresses,endpoints,directories,screenshots,vulnerabilities,settings}/`
  are also route-critical direct-import pages.
- Reason: the target detail shell already reveals breadcrumb, primary tabs, and
  sometimes the secondary asset tabs. If the child route still hides its first
  workspace behind `lazyPage(..., null)` or `dynamic(..., { loading: () => null })`,
  the user sees `shell -> blank child region -> child workspace skeleton`.
- Keep the visible loading owner inside each child view component. Website,
  subdomain, IP address, and URL route pages should use the shared
  `DetailAssetContentFrame` for the standard shell gutter and otherwise only
  pass route ids.
- Do not reintroduce route-level chunk fallbacks for these child pages unless a
  new approved design also changes the visible owner model.
- Child workspace skeletons that mirror real buttons, icon buttons, toolbar actions,
  or action-menu triggers must use shared structural placeholders such as
  `ActionSkeleton`; do not hand-roll button-sized `Skeleton h-* w-* rounded-*`
  blocks inside route skeleton files.
- `TargetSettings` routes `blacklist` to the embedded shared BlacklistPolicy
  workbench and routes `scheduled-scans` to the target-scoped scheduled-scan
  state. The blacklist workspace renders global rules only as read-only inherited
  content and edits only the Target-local `patterns`/`etag`; it must never mutate
  the global policy. The base settings URL is the blacklist workspace; the
  scheduled scan workspace is `/targets/[id]/settings/scheduled-scans/`.
- `TargetSettingsLoadingState` receives `TargetSettingsState` and renders only
  the border-owned scheduled-scan table through `ScheduledScanDataTable` in
  loading mode. Target settings must not wrap it in a redundant card or repeat
  the title already represented by the secondary settings tab.
  `TargetSettingsRouteFallback` is the state-less target-detail layout loading
  shell and renders only the active settings workspace. For scheduled scans it
  must build the production columns, omit the target-inapplicable scope column,
  hide quick filters, and render `ScheduledScanDataTable` in loading mode. For
  blacklist it delegates to the embedded Target-local blacklist loading state. Do not
  reintroduce `target-settings-skeleton.tsx` or a fixed-column `DataTableSkeleton`.
- 目标详情 settings route fallback 不得借用通用六行数量；scheduled-scan 总数无法从目标摘要得到，因此使用经过桌面/窄屏测量的稳定两行首帧结构，并保持与目标 page 相同的 gutter。
- 目标详情完整 shell 骨架必须保持当前 URL 对应的 primary tab 激活位置，并使用已知的本地化 tab 标签做不可见几何占位；不能固定激活概览，也不能用一组固定宽度让中文/英文 tab 轨在 handoff 时整体伸缩。
- `/targets/[id]/websites/` 的目标站点列表属于 Assets 主工作区，并使用 website evidence controlled variant fallback；`/targets/[id]/websites/[websiteId]/...` 使用站点详情 fallback。站点详情已有自己的 identity 与 section tabs，因此 shell 在该路由隐藏资产类型 secondary rail，避免重复二级导航。
- 站点详情的 `overview`、`ip-addresses`、`urls`、`directories`、`vulnerabilities` 只是站点关联内容的局部 section；目标 shell 必须先识别完整 website-detail 路由，再计算 primary tab 和 breadcrumb。它们不能激活目标级“漏洞”等 primary tab，也不能把该局部 section 名称写入目标面包屑。
- `/targets/[id]/relations/` 及其详情子路由只承担向 canonical website 路由的兼容重定向，不得恢复一级“关联”Tab、独立 breadcrumb 或第二套 loading owner。
