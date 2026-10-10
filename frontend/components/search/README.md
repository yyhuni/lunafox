# Search Loading Notes

`app/search/page.tsx` is a route-critical direct-import page.

- The search route must directly import `SearchPage`.
- `search-initial-page-shell.tsx` owns the resolved initial-state geometry and quick-search tag constants. It is part of `SearchPage`, not a protected-shell navigation fallback.
- Its centered command composition is intentional, but its outer page whitespace
  must use the shared compact page rhythm and `COMPACT_CONTENT_GUTTER_CLASS`;
  do not reintroduce a local full-page `px-*` convention for the initial or
  result state.
- Sidebar or protected-shell pending state must not inject a search skeleton. Once navigation commits, `SearchPage` decides whether to show its initial tool surface, its local result `ContentHandoff`, or already-ready content.
- Do not wrap the route in `lazyPage(..., null)` or `next/dynamic` with
  `loading: () => null`.
- Reason: the route has no parent visible first-screen owner. Hiding the search
  tool behind a null chunk fallback creates a `boot -> blank main region ->
  tool surface` gap.
- `SearchPageContent` still owns query-result loading locally through
  `ContentHandoff owner="search-results-content"`.
- A valid URL query initializes the search state and canonical request parameters
  synchronously as `searching`; do not wait for an effect to create the first result owner.
  Otherwise a fast cached response can commit before the skeleton/content pair
  is observable on cold entry.
- Its first query-result handoff must use the paired
  `search-results-toolbar`, `search-results-body`, and
  `search-results-pagination` regions. The loading branch reuses the same
  region wrappers and shared control shells; do not restore a whole-page
  `PageSectionSkeleton` for results.
- The global route has no vulnerability summary or target-scoped management
  action. It owns the selected asset type, strict query validation, token
  history, and the single result-search entrypoint. The Website adapter owns
  CSV data and row selection; its export trigger renders beside that global
  search bar instead of creating a second search toolbar in the result list.
  The Endpoint adapter column visibility trigger renders beside that same global
  search bar.

## Search Guidance Contract

- `frontend/lib/global-asset-search-query.ts` owns the typed
  `GLOBAL_ASSET_SEARCH_GUIDANCE` catalog. Every catalog entry is parsed by the
  strict parser before it is exported; `GLOBAL_ASSET_SEARCH_FEATURED_GUIDANCE`
  is the six-item homepage projection (`url`, `host`, `title`, `statusCode`,
  `tech`, and one `&&` scenario). Components must consume these projections
  rather than defining independent query strings.
- Field/example shortcuts in the Popover, manual, and homepage only replace
  the draft. They must not call the search handler, reset cursor tokens, or
  write recent searches. The form submit remains the only draft-to-submitted
  transition.
- Parser failures expose a stable
  `GlobalAssetSearchDiagnosticCode`; the state hook maps that code to localized
  message and repair text. Parser messages, positions, and stack details must
  never be rendered. A first diagnostic appears only after an invalid submit
  or an invalid URL `q`; once active, it re-parses on edit and clears as soon as
  the draft is valid. Invalid submissions fast-fail before `useAssetSearch` and
  recent-search persistence.
- The Popover links to `SearchSyntaxManualContent`, a stateless shared body.
  `SearchAssetBar` owns its controlled overlay: desktop (`useIsMobile() ===
  false`) uses the shared `Dialog`, and the `<768px` branch uses the shared
  `Drawer` with its standard downward swipe direction. Opening, scrolling,
  selecting an example, and closing preserve the draft; closing restores the
  search input focus on the next animation frame.
- The manual uses the shared `ScrollArea` for its long body and documents the
  field-specific matching exceptions: URL/host/title `=` contains and `==`
  full-field matching, while `statusCode` and `tech` keep typed exact semantics
  for both operators. It also documents Unicode length limits, the ten-condition
  cap, JSON-style quoted values, and rejected syntax. Keep Chinese and English
  entries in `messages/zh.json` and `messages/en.json` in sync when adding a
  parser diagnostic or catalog example.
- The compact syntax Popover also uses the shared `ScrollArea`; its width and
  maximum height must use the positioning layer's `--available-width` and
  `--available-height` variables so narrow viewports keep the full guide
  reachable without escaping the viewport.

# Global Search Contract

- `GET /v1/assets:search` accepts `q`, `assetType`, optional `pageSize`, and
  optional `pageToken`; the page defaults to Website and 10 rows.
- Plain text searches URL only. Structured input allows `url`, `host`, `title`,
  `statusCode`, and `tech` with one flat connector: `&&` or the word `and` when
  every condition must match, or `||` or the word `or` when any condition may
  match. The shared frontend parser rejects mixed connectors and other
  unsupported syntax before fetching.
- `=` contains values for URL, host, and title require at least two Unicode
  characters. `==` exact values are not subject to this minimum; status codes
  remain integers and technology values remain exact elements.
- Focusing or clicking the resolved search input opens `SearchSyntaxGuidance`.
  It derives its fields from `GLOBAL_ASSET_SEARCH_FIELDS`, shows `=`, `==`,
  `&&`, and `||`, and offers draft-only field/example shortcuts. Its popover aligns
  to the input's left edge and width, rather than the surrounding type selector
  or submit button. At the end of a draft, it renders strict inline completion
  for field prefixes, quotes, and `&&`; `Tab` and the right arrow key accept
  that draft-only continuation. Quick tags and recent-search entries also only
  replace the draft. The form submit path remains the only way to start a
  query; do not reuse the broader
  `SmartFilterInput` DSL here.
- The result response contains `results`, the capped match total (`totalSize`
  with `totalSizeCapped` marking "10000+" style capped counts), and an optional
  `nextPageToken`; never a page number. Query, asset type, and page-size
  changes reset token history. The capped total is display-only: it never
  authorizes a page transition, and the pagination area shows "共 N 条" or
  "共 10000+ 条" via the shared `common.pagination` keys.
- Website results use `SearchWebsitesDataTable`, a read-only adapter over the
  canonical target website evidence-list table surface. It reuses the
  `RelationEvidenceListFrame`, three-column header, website identity, HTTP
  response, screenshot empty state, checkbox selection, and CSV export
  behavior while preserving the Search shell and Search-owned cursor
  pagination. The global result toolbar owns the only search field and the
  website export trigger; the evidence list must not render a duplicate local
  search toolbar. Its loading adapter uses that same evidence-list frame and
  renders the Search pagination region inside the evidence footer in both
  phases, with the current page-size row budget before handoff. Bulk add/delete
  stay unavailable, and rows without `targetId` omit the target details link.
  Endpoint results remain table-owned; default column visibility matches
  `/targets/:target/endpoints/` (`host`, `location`, `responseHeaders`, and
  `responseBody` hidden by default), with on-demand toggle owned by the column
  menu beside the global search bar. Row activation reuses `EndpointDetailDrawer`
  so clicking an endpoint row displays full HTTP metadata and response bodies.
  Loading state uses `SearchResultsTableLoadingState`,
  preserving exact table header and column widths through `UnifiedDataTable`.
# Search Components

## Result Table

- 表格结果隐藏冗余的本地内部工具栏（`hideToolbar: true`）；“显示列”菜单置于顶部全局搜索栏右侧（与网站导出的位置保持对称与一致），默认隐藏主机名、跳转地址、响应头和响应体（与 `/targets/:target/endpoints/` 保持一致），用户可通过顶部“显示列”按需勾选启用。点击任意行复用 `EndpointDetailDrawer` 弹出详细侧边栏抽屉展示完整详情（与 `/targets/:target/endpoints/` 一致）。加载状态通过 `SearchResultsTableLoadingState` 原生渲染表头与行骨架，消除卡片与表格之间的布局抖动。
- `SearchResultCard` remains available for its focused response contract, but
  Website search results are rendered by `SearchWebsitesDataTable`; the
  evidence-list table owns selection, screenshot/response presentation, and
  CSV export state while the Search page keeps the only search control and
  renders the export trigger beside it.
