# Search Loading Notes

`app/search/page.tsx` is a route-critical direct-import page.

- The search route must directly import `SearchPage`.
- `search-initial-page-shell.tsx` owns the resolved initial-state geometry and quick-search tag constants. It is part of `SearchPage`, not a protected-shell navigation fallback.
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

# Global Search Contract

- `GET /v1/assets:search` accepts `q`, `assetType`, optional `pageSize`, and
  optional `pageToken`; the page defaults to Website and 10 rows.
- Plain text searches URL only. Structured input allows `url`, `host`, `title`,
  `statusCode`, and `tech` with flat `&&`; the shared frontend parser rejects
  unsupported syntax before fetching.
- Focusing or clicking the resolved search input opens `SearchSyntaxGuidance`.
  It derives its fields from `GLOBAL_ASSET_SEARCH_FIELDS`, shows only `=`, `==`,
  and `&&`, and offers draft-only field/example shortcuts. Its popover aligns
  to the input's left edge and width, rather than the surrounding type selector
  or submit button. At the end of a draft, it renders strict inline completion
  for field prefixes, quotes, and `&&`; `Tab` and the right arrow key accept
  that draft-only continuation. Quick tags and recent-search entries also only
  replace the draft. The form submit path remains the only way to start a
  query; do not reuse the broader
  `SmartFilterInput` DSL here.
- The result response contains `results` and optional `nextPageToken`, never a
  page number or total. Query, asset type, and page-size changes reset token
  history.
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
  Endpoint results remain table-owned; response headers and bodies are optional
  columns and start hidden.
# Search Components

## Result Table

- 表格结果将响应头和响应体作为共享“显示列”菜单中的可选列，默认隐藏；该状态仅保留在当前结果表实例内。
- `SearchResultCard` remains available for its focused response contract, but
  Website search results are rendered by `SearchWebsitesDataTable`; the
  evidence-list table owns selection, screenshot/response presentation, and
  CSV export state while the Search page keeps the only search control and
  renders the export trigger beside it.
