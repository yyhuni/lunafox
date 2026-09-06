# Tool Workspaces

## Wordlists Pagination

The Wordlists footer is cursor-only. Keep the service result summary and
page-size selector, but derive next availability solely from the active
`nextPageToken` and previous availability solely from the current-query token
cache. The first-page control clears the cache and requests page one. Do not
calculate total pages or add numbered or last navigation.

## Engines

- Engine management is presented at `/scan/config/engines/` inside the shared
  Scan Configuration workspace. `EngineInstallationPage` supports an embedded
  presentation so the workspace owns the single title and primary Tab rail;
  `/tools/engines/` remains a redirect-only compatibility alias and must not
  regain an independent page header.
- `engine-catalog-content` owns the first-screen Engines handoff. Both loading
  and resolved content must place the real search control inside that owner and
  pair `engine-catalog-controls`, `engine-catalog-grid`, and
  `engine-catalog-card-rhythm` on the controls region, semantic results region,
  and cards grid respectively. Do not move the controls outside the handoff or
  place the card-rhythm slot on every individual card.
- `engine-catalog-controls` owns the compact-density search and install-command
  row. On `sm` and wider layouts, the search control remains left while the
  install trigger is right-aligned; narrower layouts stack the trigger below the
  input. Loading keeps the same disabled shared control shell so the catalog
  handoff does not move the grid.
- The built-in catalog's first-frame window contains eight cards. Keep the
  loading grid on that window so it does not add an extra mobile card row that
  disappears when the static catalog resolves. A catalog-size behavior change
  must update this shared first-frame contract and its cold-load evidence.
- Resolved cards render only user-facing data from the Server catalog summary:
  localized name and description, package version, publisher, targets, input
  types, and platform-resource count. Input roles render through localized
  labels; never show their internal IDs in the card grid or detail drawer. The
  Engine ID remains an internal lookup/search key and is shown in the detail
  drawer. Do not infer builtin, health, enabled, or update state from publisher
  or package identity.
- Each resolved card is one accessible action. It opens the shared
  `DetailDrawer` and loads the exact selected Engine detail through the catalog
  hook; do not preload every detail or add nested actions to the card.
- `engine-catalog-detail` owns detail-query pending feedback at the
  `interaction` layer inside the already-open drawer. The list remains owned by
  `engine-catalog-content`; detail loading must not add a second workspace
  owner or replace the catalog grid.

## Wordlists

- The Wordlists page shell and its page-owned `ContentHandoff` use the shared
  viewport-bound catalog classes in `wordlists-page-layout.ts`. Loading cards,
  results, and pagination must scroll inside that surface; they must not alter
  the first-screen route height as data becomes available.
- Its paired first-screen slots are `wordlists-controls` and `wordlists-list`;
  `ContentHandoff` owns the common `surface` slot. A selected wordlist opens in
  the shared `DetailDrawer` as an interaction-local detail surface rather than
  reserving a persistent inspector in the first screen.
- `wordlist-catalog-card` is one accessible action that opens the selected
  wordlist's detail drawer. Keep editing and deletion in that drawer so the
  catalog remains a browsing surface without nested card actions.
- Loading reuses `WordlistCatalogCardLoadingState` in the same responsive grid
  as resolved cards. Recheck desktop and narrow frames before changing the card
  count or geometry.
- The loading footer is cursor-pagination too: pass `mode="cursor"` and retain
  the summary in `CompactPaginationSkeleton`, so narrow layouts do not reserve
  a numbered-page row that resolved content never renders.
- `WordlistTagPicker` is shared by the upload and edit dialogs. Its add control
  belongs beside the selected tags, while recommended tags only select existing
  server-backed suggestions. Recommendations remain visible while a custom tag
  is being entered; a non-empty, unique draft commits on blur, Enter, or a
  recommendation click. Escape and the cancel action discard the draft instead.
  Do not use the draft text to filter the visible recommendation set.
- Wordlist cards, drawers, edits, cache keys, and list deduplication use the
  canonical API `name` (`wordlists/{id}`) as identity. The uploaded basename is
  displayed through read-only `fileName`; there is no rename control. Resource
  selectors serialize the canonical `name` while rendering `fileName` labels.

## Nuclei POC Catalog

- `/tools/nuclei/` is a backend-owned, single-source POC catalog. The page uses
  `nuclei-poc.service.ts` through `use-nuclei-pocs.ts`; components must not call
  transport services directly.
- A successful sync atomically replaces the committed collection. Current
  source metadata is rendered above the table, while the table exposes only
  server metadata and the persisted `isEnabled` runtime overlay.
- Initial source loading reserves the same narrow-screen structure as a
  committed source: one source-type line and a wrapped two-line URL area. Keep
  this inside `NucleiPocSyncSourceStatus`; a shorter no-source placeholder must
  not collapse the catalog before source metadata resolves.
- The tag facet reads `GET /v1/nucleiPocs/filterOptions?field=tags` through the
  hook boundary. Options represent the complete committed catalog, use
  canonical lowercase values and labels, and include catalog-wide counts;
  search, severity, tag filters, and cursor pagination never narrow this
  option source. Selected values remain visible while the projection refreshes.
- `NucleiPocSyncDialog` accepts exactly one public HTTPS Git source at a time
  (official Git, operator-provided `gitee.com`, or custom Git), shows durable
  task progress, and lets the mounted catalog page own one-second polling while
  the task is non-terminal. Dialog visibility controls presentation only;
  closing the dialog never cancels the server task or stops page monitoring.
- A `SYNC_ALREADY_RUNNING` conflict with canonical `ErrorInfo.metadata.task` is
  adopted as the active task. Malformed conflict metadata remains a localized
  recoverable error and is never used as a request path.
- POC content is read-only in this surface: no create, edit, delete,
  repository commit, or scan controls are allowed. Row selection is permitted
  only for persisted activation of the exact selected POCs; it must not expose
  a content-lifecycle operation. Details load by canonical
  `nucleiPocs/{templateId}` name and provide YAML viewing/copying only.
- The left-toolbar `批量操作` menu exposes only `全部开启` and `全部关闭`.
  Both commands confirm against the entire committed catalog, call the
  server-owned `POST /nucleiPocs:setActivation` method, and never include the
  visible page, search, or filter scope. Success feedback uses the server's
  actual `affectedCount`; definitive failures preserve the current view and
  transport-uncertain results refresh list/detail projections without retrying.
- The selected-row action bar exposes only enable and disable. Its confirmation
  sends the exact selected canonical names, and selection clears before cursor
  page, search, filter, or source-sync transitions so no stale scope can be
  submitted.
- This is a natural-flow business-list table. Keep its table and pagination out
  of viewport-fill `flex-1` / `min-h-0` wrappers so long pages retain the route
  bottom padding. Its loading geometry contract permits the table surface to
  shrink after a sparse result resolves; derive initial loading rows from
  `getDataTableSkeletonRowCount(PAGE_SIZE)`.
- The initial catalog skeleton must render through the same `BusinessListDataTable`
  and column definitions as the resolved catalog. Use the shared initial toolbar
  and cursor-pagination shells so column allocation, filter/action slots, and
  pagination controls stay on the same axes during the handoff.
