# Organization Loading Notes

## Add Organization Drawer

- The optional target-entry disclosure begins after a light top divider, so the
  primary organization fields stay visually distinct from optional target setup.
  Keep the divider on the disclosure's owning `FormItem` rather than adding a
  second border inside the expanded target editor.

## Shared Selection Workspace

- `organization-selection-workspace.tsx` owns the reusable organization
  selection presentation used by target creation and scheduled-scan creation:
  header, search, all/selected scope, responsive cards, loading and empty states,
  and the three-zone pagination footer.
- The workspace is controlled and supports `single` and `multiple` selection.
  Callers own selected IDs, organization queries, pagination tokens, and request
  payloads. Do not import services or React Query hooks into the workspace.
- Selection count is visible by default for multi-select association workflows.
  Fixed single-select callers may disable it with `showSelectionCount={false}`
  while keeping selected-card feedback and the clear action.
- Selected organization cards use the shared `interaction-accent` border;
  unselected cards retain the neutral border. Callers must not override this
  selection feedback with page-local colors.
- Target creation is the multi-select adapter and keeps submitting
  `organizationIds`. Scheduled-scan creation is the single-select adapter and
  keeps submitting `organizationId`.
- Keep the card grid as the only result presentation. Do not add caller-local
  card renderers, organization popovers, or list/grid view switches.
- The shared organization pagination footer is cursor-only: it presents the
  service-returned result total, page-size selector, first-page reset, and
  adjacent previous/next controls. Callers derive next availability from the
  active response `nextPageToken` and prior availability from the current-query
  token cache. The first-page action clears that cache and requests page one;
  do not render or infer last, numbered, current, or total-page controls from
  `totalSize` / `totalPages`.

The organization list and organization detail routes are route-critical
direct-import pages:

- `app/organizations/page.tsx`
- `app/organizations/[id]/page.tsx`

The organization-name column uses the shared `DataTableColumnHeader` status
menu. Do not add an organization-only sort or column-visibility trigger: the
shared owner provides explicit ascending/descending actions and conditionally
offers hiding when the TanStack column permits it.

Required rules:

- The route files must directly import `OrganizationList` and
  `OrganizationDetailView`.
- Do not hide either route-critical first-screen surface behind
  `next/dynamic(..., { loading: () => null })` or `lazyPage(..., null)`.
- Reason: `organizations/` would otherwise show the page header and then leave
  the workspace blank until the list skeleton mounts, while
  `organizations/[id]` would otherwise show a blank main region before the
  detail skeleton appears.
- Keep list/detail loading ownership inside the components:
  - `organization-list-content`
  - `organization-detail-view-content`
  - `organization-detail-drawer-content`
- The route-critical detail owner declares `layer="workspace"` and is listed in
  `loading-route-contracts.mjs` as a paired first-screen surface. The drawer
  remains a local section interaction and must not be promoted into that route
  contract.
- Drawer-only and dialog-only subviews may stay dynamic when they are not the
  route-critical first-screen owner.
- Dynamic subviews still need browser-reviewed interaction ownership:
  - if the drawer/dialog appears immediately after click, it may stay as-is
  - if a dynamic chunk or local data wait creates `click -> blank wait ->
    drawer/dialog`, add an interaction-local loading owner instead
- The organization list preloads only the detail and create-panel chunks after
  an explicit local intent signal: row pointer/focus for details and add-button
  pointer/focus for creation. Keep the panels dynamically imported; do not
  preload scan, scheduling, or other optional workbenches from this route.
- The organization list initial loading state must reuse the resolved
  `OrganizationDataTable` columns, pagination state, search state, and actions
  from `OrganizationListState`; only the volatile rows should be skeletonized
  through the table `loading` / `loadingRowCount` props.
- Both list branches pass the same page-size-derived `stableSurfaceRowCount` to
  `OrganizationDataTable`. This is the reviewed shared-table first-frame
  reservation: it applies only while `loading` is true, then the resolved table
  uses its real rows or empty-state row. It is not a data-derived page-local
  `min-h-*` patch and does not create fake rows.
- `OrganizationDataTable` sets the shared `loadingRowHeightEstimate` to the
  measured multiline-row upper bound. The description overflow action can add a
  second control line after data resolves, so the loading table must reserve
  that shared-row geometry instead of applying a route-local height patch.
- The `/organizations/` primary list uses the shared natural table flow: the
  page owns normal vertical scrolling, the table header scrolls with its rows,
  and shared pagination is rendered below the bordered table surface. Do not
  add a route-owned height-fill, sticky header, or framed pagination variant.
- The description cell keeps its initial preview to one line. Its explicit
  expand action remains available after the handoff; allowing a multi-line
  preview plus an overflow action would exceed the shared comfortable-row
  rhythm on narrow screens and move the pagination during the first frame.
- `OrganizationDataTable` owns the paired `organization-list-toolbar`,
  `organization-list-body`, and `organization-list-pagination` slots through
  its shared-table `loadingSlots` configuration. Both initial and resolved
  table branches must retain that exact trio; do not add a parallel wrapper
  solely to label the skeleton.
- `organization-list-content` opts into the shared
  `prepareContentBeforeHandoff` readiness gate because populated table rows
  must settle before their first visible frame. It MUST NOT reserve a
  data-derived height or weaken the route's intrinsic geometry tolerance; fix
  table/skeleton parity instead of adding an organization-local `min-h-*`
  patch.
- Do not put `space-y-*` or other sibling-margin layout classes on the list
  `ContentHandoff` itself. Its skeleton and content wrappers share one grid
  area during handoff, so sibling spacing would inflate the content state and
  create a visible geometry jump. Put any persistent vertical rhythm inside
  each state owner instead.
- Do not reintroduce an organization-list-only data-table skeleton wrapper that
  builds fake columns or fake translations for the normal query loading path.
- The current audited organization surfaces are:
  - list-page detail drawer: reviewed in browser; the real sheet opens
    immediately and does not need an extra loading shell
  - detail-page scheduled scan create/edit dialogs: use shared
    `InteractionLoadingDialog`

## Organization List Identity

- The `/organizations/` identity cell is the only list surface covered by the
  initials treatment. It renders one Unicode code point from the trimmed,
  required organization name in a transparent `size-9` circle, using the
  shared semantic border/text roles. An unselected, non-hovered row uses a
  dashed border; hovering the row or selecting it through its checkbox
  (`data-state="selected"`) changes it to solid. Moving the pointer away from
  an unselected row restores the dashed border.
- The identity cell stacks the shared primary name role above the shared
  secondary description role. The description keeps `ExpandableCell`'s
  single-line initial preview and explicit expansion action. The standalone
  description column is intentionally absent, but `Organization.description`
  remains available to detail, edit, and drawer flows.
- Do not propagate this marker treatment to
  `organization-selection-workspace.tsx`; its card avatar and selection
  layout are a separate shared contract.

## Organization Detail Target Pagination

- The organization target tables use the endpoint's opaque `pageToken` /
  `nextPageToken` contract. Keep the current-query token cache in the owning
  state, reset it when search, type filters, or page size change, and only
  allow first, previous, and next transitions the service has authorized.
  Do not derive a last page from `totalSize` or restore the numbered `page`
  query parameter.
- The overflow-menu "view details" action must invoke the owning navigation
  callback. The menu is portalled outside `DetailDrawer`, so a route action
  must not rely on an inline anchor surviving the drawer's outside-interaction
  handling.

## Detail Skeleton Geometry

- `OrganizationDetailViewLoadingState` is used by both the route detail page
  and the list-page detail drawer. Its header and summary strip must extend the
  resolved `OrganizationDetailHeader` and `OrganizationSummaryStrip` structures
  with loading value slots instead of drawing a separate header/metric skeleton.
- The detail page-owned loading state must keep the page wrapper
  `flex flex-col gap-4 py-4 md:gap-6 md:py-6`, and the page loading state must
  keep the resolved workbench `py-5` inset. The drawer passes
  `summarySurface="drawer"` because its scroll body already owns that inset.
- Reason: the drawer can open from a selected list row before the detail query
  resolves. Reusing the resolved structure keeps avatar, title, metadata,
  action slot, metric grid, tabs, and embedded table geometry aligned through
  `ContentHandoff`; hand-written fixed `h-*` / `w-*` placeholder blocks drift
  when text, locale, or drawer width changes.
- `organization-detail-view-content` uses the shared prepared first-frame
  handoff because a real detail payload can add summary and table height after
  the shared loading structure has mounted. Keep its prepared `surface`
  contract instead of introducing a detail-local height reservation.
- The route and drawer must keep that initial handoff loading until the
  organization, primary target list, summary target list, and scheduled-scan
  queries have all settled. Those four sources contribute visible first-frame
  summary or table geometry; allowing one delayed source through after the
  handoff creates a narrow-screen height shift.
- The route-level detail pair exposes `organization-detail-summary` and
  `organization-detail-primary-table` through the same local region wrappers in
  loading and resolved branches. Its target-table loading variant uses the
  current page-size row budget, so the first visible page cannot expand after
  the handoff.
- The drawer loading state should pass the selected row name and description as
  hidden `previewName` / `previewDescription` values so the title and
  description slots reserve the known row geometry while the real detail
  payload is still loading, including narrow drawer widths where descriptions
  can wrap.
- The drawer body and its `ContentHandoff` must mount in the same render as the
  open `DetailDrawer`. Do not defer the body with an effect, animation frame, or
  timeout; the handoff skeleton is the immediate visible content owner.
- The detail header description slot is a resolved header contract, not a
  skeleton-only height patch. On narrow screens it reserves the two-line
  `line-clamp-2` height in both loading and content states so an unknown
  organization description cannot push the summary strip and table down when it
  resolves. Keep the description as direct text inside the clamping paragraph
  and overlay its loading skeleton absolutely; wrapping it in an inline loading
  component can bypass the clamp under Linux/CJK font metrics and add a third
  line during handoff.
- Date metadata and the latest-scan metric must give `InlineSlotLoadingState`
  the same localized text that the resolved branch will render whenever the
  data is available. The latest-scan metric must also opt out of
  `OrganizationMetric`'s default fixed numeric width so that text can define
  the inline slot width. Do not use fixed-width date placeholders: Linux and
  CJK font fallback can otherwise change the metadata wrap threshold and shift
  the summary height during handoff.
- The embedded target table loading state should reuse the resolved target
  table columns, pagination state, filters, and actions from
  `OrganizationDetailViewState`; only the volatile rows should be skeletonized.
- The standalone organization targets detail page follows the same rule:
  `TargetsDetailViewLoadingState` receives `TargetsDetailViewState` and renders
  `TargetsDataTable` in loading mode. Do not pass only a row count into a
  generic table skeleton that guesses the organization target toolbar or column
  shape.
- The scheduled-scan tab data loading state should also reuse the resolved
  `ScheduledScanDataTable` columns, pagination state, filters, and actions from
  `OrganizationDetailViewState`. Do not replace it with a generic
  `DataTableSkeleton` that guesses scheduled-scan column or toolbar counts.
- Full-project audit note: other reviewed production `ContentHandoff` skeletons
  either delegate to shared templates such as `DataTableSkeleton`,
  `CardGridSkeleton`, `MasterDetailSkeleton`, `PageSectionSkeleton`, or reuse
  route-specific layout owners such as overview and settings skeleton wrappers.
  Do not broaden this organization-specific fix into unrelated visual rewrites
  without a scoped migration change.
