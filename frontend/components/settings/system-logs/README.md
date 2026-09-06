# System Logs UI Notes

## Loading Geometry

`system-logs-loading-state.tsx` owns `SystemLogsLoadingState` so the resolved
terminal page and workspace data loading stay on one
lightweight visual path. Page and loading-overlay geometry lives in
`system-logs-layout.ts`; the shared terminal body, viewport, jump-to-latest
control, and footer shell live in `LiveLogSurface`; search and level-filter
chrome lives in `TerminalLogToolbar`.

- Sidebar and the protected shell do not inject a System Logs skeleton; the
  destination workspace owns the choice between `SystemLogsLoadingState` and
  ready content.
- `SystemLogsWorkspace` directly imports `SystemLogsView`. Its route boundary
  mounts the resolved view hidden while the first request settles, so a
  `next/dynamic(..., { loading: () => null })` wrapper would create a readiness
  deadlock: the visible route skeleton would wait for a child that has not
  committed a node. Keep the route boundary as the only first-screen loader.
- The view MUST NOT call its route `onReady` callback from the hook's `idle`
  pre-request render. It may signal readiness only after the initial request
  has left `idle` and is no longer initial-loading (including a terminal error);
  otherwise the parent boundary can remount/cancel the request it is waiting
  for.
- `deferInitialSkeleton` does not mean that `SystemLogsView` may return `null`.
  The boundary already hides the mounted resolved branch; the viewer must stay
  mounted to start its request and call `onReady` after the first stable frame.

- Keep the page shell, header overlay, terminal container, and footer overlay group geometry in `system-logs-layout.ts`. Do not add terminal body, viewport, jump-button, or footer-shell classes there; those belong to `LiveLogSurface`.
- Resolved and loading states must both compose `LiveLogSurface`. The page continues to own Server source text, connection/gap/trimming status, auto-refresh state, filtering, and error mapping.
- System Logs 的行数选择只允许 `100/200/500/1000/2000/5000`，由共享 `TerminalLogToolbar` 的 Base UI Select 统一拥有；页面每次进入默认 100，选择不写入 URL、localStorage 或后端配置。N 表示浏览器始终保留的最新日志窗口，footer 必须显示实际行数与当前 N；上滚只停自动滚屏，不停日志更新。
- Resolved System Logs passes the currently filtered visible log lines, formatted with `formatStructuredLogLine`, to `TerminalLogCopyAllButton` through `LiveLogSurface.topRightAction`; the copied text must match the human-readable `[timestamp] [LEVEL] message caller key=value` stream shown in `StructuredLogViewer`, rather than raw JSON. Do not create a local copy button or copy unfiltered rows after a search or level filter is active. With no active filter, the copied text remains the complete loaded log window.
- Resolved log rows inherit the copy-first `[timestamp] [LEVEL] message caller key=value` stream from `StructuredLogViewer`; do not add page-local badges, fixed-width level columns, or field reordering.
- The latest-N Server window remains complete in memory for filtering, counts, and generating the current visible-range copy text. Above the shared virtualization threshold, `StructuredLogViewer` must receive the same viewport ref as `LiveLogSurface`, mount only the visible rows plus finite overscan, and measure wrapped rows at their actual height; do not add a page-local scroll container. Native selection is not guaranteed across unmounted rows, so filtered visible-range copying stays on `TerminalLogCopyAllButton`.
- An empty Server follow poll may update connection, gap, and caught-up metadata, but it must preserve the existing log-lines array reference and must not recommit the body or trigger automatic-scroll layout work.
- Use `TerminalLogToolbar` for the resolved and loading search/filter bar so system and Agent logs keep the same interaction and geometry. Its loading mode must derive from the real disabled `Input` and `ActionSkeleton` controls; do not approximate them with local `Skeleton h-8 ... rounded-sm` rectangles.
- The footer auto-refresh loading control must use the real disabled `Switch` primitive with the resolved `scale-75` density. Do not approximate it with a local rounded `Skeleton` pill.
- 加载态同样传入禁用的行数选择器，保持与 resolved toolbar 一致的控制几何；不要用本地 skeleton 代替该 Select。
- Do not duplicate footer overlay flex/group class strings in the loading state. Use `SYSTEM_LOGS_FOOTER_OVERLAY_STATUS_GROUP_CLASS` and `SYSTEM_LOGS_FOOTER_OVERLAY_REFRESH_GROUP_CLASS` so the loading footer preserves the same structure as the resolved footer.
- The loading state keeps a real `PageHeader` mounted with hidden content and paints the loading header overlay on top. Do not replace it with a separate page-local header layout or reintroduce `system-logs-page-skeleton.tsx`.
- `SystemLogsWorkspace` and both `ContentHandoff` branches share the
  viewport-bound workbench classes from `system-logs-layout.ts`. Log volume and
  terminal state must remain inside the terminal scroll surface instead of
  changing the route owner's first-screen height during handoff.
- The route pairs `system-logs-header`, `system-logs-terminal-toolbar`, and
  `system-logs-log-surface` in both branches. The last slot deliberately wraps
  the shared `LiveLogSurface` body and footer together, so the terminal shell
  remains one flex-owned region rather than gaining test-only wrappers.

The boundary is guarded by `__tests__/system-logs-view.contract.test.ts`.
