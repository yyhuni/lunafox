# Shared Loading Guide

## Progressive Loading Boundary Standard

Loading ownership is selected by the user's waiting intent, not by component
boundaries or by whichever async primitive happens to be closest.

The standard layered sequence is:

1. `boot`
2. `protected shell warmup`
3. `route or page skeleton`
4. `section ContentHandoff`
5. `content-only`

Layered loading is allowed. Staged loading inside the same visual layer is not.
Do not reveal a route as `title -> tabs -> first card -> table -> chart` when
those pieces are perceived as one first-screen page. The same layer must have
one visible owner at a time.

Sectioned dashboards and workspaces are the narrow exception to the "one owner"
shortcut phrasing. They MAY keep multiple initial section owners when all of
the following are true:

- each owner maps to a visually distinct stable section, such as overview,
  toolbar, and results
- those sections share the same initial readiness gate instead of resolving
  one by one
- browser timeline evidence shows synchronous section handoff rather than
  `section A -> section B -> section C` staged reveal

The anti-pattern is sequential reveal, not the raw number of section owners.
`/settings/agents` is the current audited example: overview, toolbar, and
results keep separate owners, but desktop and mobile timeline evidence show
them entering `loading` together and reaching `content` together.

When one route-level or section-level owner already owns the whole first-screen
dashboard, do not reintroduce staged loading with delayed inner reveal classes.
`overview-delay-*` style choreography on first paint is still same-layer staged
loading even if the outer owner is singular.

## Loading Owner Metadata

Every production loading owner must expose shared control-plane metadata:

- `data-loading-owner`: stable owner name used by smoke diagnostics
- `data-loading-layer`: one of `boot`, `auth-shell`, `app-shell`, `route`,
  `workspace`, `section`, `interaction`, `compact-pending`, or
  `domain-status`
- `data-loading-intent`: optional user-waiting intent such as `boot`, `auth`,
  `app-shell`, `route`, `data`, `navigation`, `interaction`, `pending`, or
  `status`
- `data-loading-phase`: required for owners that transition through loading;
  settled route/content owners should report `content`

Use `getLoadingOwnerAttributes` from
`@/components/shared/loading/loading-owner` instead of hand-writing owner
attributes in production components. The only direct production
`data-loading-owner` exception is the server-rendered root boot layer in
`app/layout.tsx`; tests keep the rest of the app behind the helper.

Layer ownership follows the user-visible waiting boundary:

- `boot`: server-rendered root boot only
- `auth-shell`: reserved for a future reviewed auth entry owner. The current
  login flow keeps root boot as the only visible auth-entry owner until the
  login surface is ready, so do not add a visible auth warmup card or revive
  `AuthWarmupShell`.
- `app-shell`: protected shell warmup
- `route`: first-screen route shell or whole-page fallback
- `workspace`: first-screen table, list, editor, or workbench area
- `section`: independently stable dashboard or page section
- `interaction`: dialog, drawer, preview, or post-click local wait
- `compact-pending`: inline control pending state
- `domain-status`: live runtime status, not placeholder loading

Inline metric strips are a separate allowed pattern. A fixed-height strip such
as `StatMetricRow` MAY keep its resolved shell mounted and replace only the
numeric value slot with an inline skeleton when the shell geometry, labels, and
footer copy are already stable. That is a stable-shell value swap, not a second
loading scene.

Direct-client list/workspace routes are another allowed pattern. When the page
header is already stable route chrome, the route MAY render that header first
and let the first-screen workspace own its own `ContentHandoff`. The intended
sequence is `header -> workspace skeleton -> workspace content`, not
`header -> blank workspace -> table` and not `full-page loader -> header + table`.
`/targets` is the current audited example.

For protected-shell and route-child suspense on a direct-client detail
workspace, keep the server boot layer visible with a non-visual
`data-boot-handoff-pending` marker until the resolved chrome and child
workspace commit together. Do not insert an app-shell warmup or detail-shell
fallback before that workspace skeleton; the sequence must be
`boot -> stable chrome + workspace skeleton -> content`.

The same rule applies to layout-level metadata such as tab counts. If the shell
chrome is already stable, count queries should use inline pending treatment and
must not block the child workspace from mounting its own first-screen handoff.
`/tools/fingerprints/ehole` is the current audited example.

Duplicate same-layer owners include:

- a visible `lazyPage` fallback plus a component-owned `ContentHandoff`
- a visible `next/dynamic` fallback plus a route-owned `ContentHandoff`
- a route skeleton that exits before the first-screen child content is
  geometrically stable
- a `ContentHandoff` skeleton subtree that declares its own `data-loading-owner`
  for the same visual workspace or section
- a local reveal wrapper on top of an existing shared skeleton handoff

Skeleton components that can be rendered both standalone and inside
`ContentHandoff` SHOULD accept an optional `owner`. Pass the owner only for the
standalone skeleton surface; when the skeleton is passed to `ContentHandoff`,
the handoff owner owns the loading metadata and the skeleton should remain a
visual structure with stable `data-slot` markers.

The default `transitionMode="crossfade"` remains appropriate for local section
handoffs. A complete route/detail shell MAY use `transitionMode="replace"` when
crossfading would expose resolved controls underneath unfinished route chrome.
Replacement mode keeps hidden readiness measurement during loading, then swaps
directly to content without a mixed `handoff` frame.

Skeletons are geometry-preserving loading variants, not independent mockups.
It is fine to replace volatile values, chart marks, row text, and business
labels with coarse placeholders, but the surrounding shell, padding, grid
breakpoints, row/container rhythm, and first-frame height must be derived from
the resolved component or a shared layout owner. If the resolved height depends
on data, document the stable first-frame contract, such as default page size,
measured overview-card height, or reviewed empty/result container height. Do
not tune route-local `min-h-[...]` values or fixed row counts by eye; measure
desktop and narrow/mobile geometry first, then add a contract test or a narrow
reviewed exception.

Search control skeletons must match the resolved search control's width owner.
Use `SearchToolbarSkeleton`'s default fixed width when the resolved toolbar uses
the standard `sm:w-72 lg:w-80` search field. Use `inputWidthMode="fill"` when
the resolved input fills a parent-owned width such as a page-specific `max-w-*`
or flex track; do not patch over the fixed defaults with breakpoint-specific
`inputClassName` overrides.
Generic shared skeletons such as `CardGridSkeleton`, `MasterDetailSkeleton`,
and `PageSectionSkeleton` must consume `SearchToolbarSkeleton` for search-like
toolbar slots instead of drawing local `Skeleton h-9 ... rounded-lg` input
lookalikes.
Interaction loading shells such as `InteractionLoadingDialog` must use disabled
`Input` for form-like placeholders and `ActionSkeleton` for footer actions;
large content preview blocks may remain ordinary `Skeleton` geometry. Auth
entry loading is not an interaction shell: keep it owned by the root boot and
login readiness gate instead of rendering a local auth-card skeleton.

Route-local skeleton containers must not own fixed `h-[...]` or `w-[...]`
geometry when the matching resolved component also depends on that size. Move
card, table viewport, dialog, drawer, and workbench dimensions into a shared
layout owner consumed by both resolved content and the skeleton. Raw `Skeleton`
line/shape placeholders may still use measured text or icon dimensions, but
outer containers and section/table shells should not repeat those dimensions
inside a skeleton file.

When a page needs more than one loading state, split by user intent. A route
skeleton may own the initial first-screen wait, while a table can later own a
compact refetch state after content already exists.

Global route progress is a navigation-gap fallback, not a second skeleton
layer. Once a visible `route`, `workspace`, `section`, `app-shell`, or
`auth-shell` loading owner appears for the next frame, `RouteProgress` should
cancel its top bar so users do not see `top progress -> page skeleton -> top
progress` for one navigation. The top bar should also keep a short handoff
grace window before it appears, so ordinary sidebar or mobile-drawer navigation
can hand directly to the next route owner without a 100ms-class progress flash.

Sidebar soft navigation owns only immediate, single-item optimistic selection.
It MUST NOT register or inject a route skeleton, hide the committed workspace,
or require a destination `loading.tsx` merely to make a click visibly pending.
After activation, the router commits the destination and that page or workspace
decides whether its own component/query state renders a skeleton or ready
content directly. A page-owned `ContentHandoff` remains valid, but do not put a
shell-owned fallback in front of it and create `route skeleton -> component
skeleton -> content` for the same wait.

## Root Boot Motion

The server-rendered root boot layer exists before React can decide whether a
route, auth surface, or workspace is ready. A full document reload creates a new
document, so any boot animation that starts from document-local `performance.now()`
will visibly reset to its first frame even when the user is refreshing an
already-known app route.

Root boot animations that can appear on reload MUST derive their visual phase
from a wall-clock-derived phase, for example `Date.now() - performance.now()`
plus the current `performance.now()`, instead of calling `draw(0)` or subtracting
a per-document `startedAt`. This keeps decorative boot motion continuous across
hard reloads while still allowing the boot owner to cover real startup gaps.

The boot layer may still exit through opacity once a real owner is ready, but it
must not hand off to a second same-layer visual loader or a whole-shell entry
animation. Route `ContentReveal` and workspace `ContentHandoff` own the later
handoff motion.

## Route-critical Chunks

A route-critical component is the component that owns the first visible route
surface: page header, primary tabs, first summary band, first cards, or the
first table/list workspace. If a route shell skeleton already represents that
surface, avoid adding a second visible chunk fallback for the same surface.

Use these defaults:

- Directly import route-critical overview/detail/workspace/tool content when
  either:
  - the route shell skeleton already covers the first-screen route geometry, or
  - the route would otherwise expose a blank first-screen region before a
    `lazyPage(..., null)` / `dynamic(..., { loading: () => null })` child
    resolves, whether that gap is `boot -> blank region -> route surface` or
    `shell -> blank child region -> child workspace skeleton`
- Do not hide a route-critical first-screen owner behind a null chunk fallback
  and assume boot timing will cover the gap. If the first-screen surface lives
  inside the lazy child, the route import is part of the visible loading path.
- Use `lazyPage(..., null)` or `dynamic(..., { loading: () => null })` when the
  lazy child does not own a new route-critical first-screen surface, for
  example detail child routes whose shell or existing workspace owner already
  covers the child geometry, or subviews whose visible loading owner already
  exists outside the lazy boundary.
- Keep visible dynamic fallbacks for low-frequency local surfaces such as
  opened dialogs, editor panes, canvas panes, or other reviewed subviews.
- Those reviewed local surfaces still need immediate interaction-local feedback
  after the trigger fires. Do not allow `click -> blank wait -> dialog/drawer`.
  If the dynamic surface or action-local data would otherwise create a visible
  empty beat, the trigger must either:
  - enter a real pending state itself, or
  - hand off immediately to a local loading owner such as
    `InteractionLoadingDialog` / `InteractionLoadingDrawer`
- Current audited examples:
  - scheduled scan create/edit dialogs in `scan/scheduled`,
    `target/settings`, and `organization/detail`
  - fingerprint add/import dialogs in `tools/fingerprints/*`
  - vulnerability detail dialogs in target/scan detail child routes
  - search vulnerability detail dialog after result-row interaction
- Do not change API calls, query keys, cache/stale timing, mock routing, or
  service routing to make loading look faster. Loading fixes in this layer are
  presentation-only unless a separate approved spec changes data behavior.

Route-critical direct import examples are captured in:

- detail overview pages:
  - `app/targets/[id]/overview/page.tsx`
  - `app/scan/history/[id]/overview/page.tsx`
- top-level first-screen workspaces:
  - `app/vulnerabilities/page.tsx`
  - `app/vulnerabilities/vulnerabilities-workspace.tsx`
  - `app/search/page.tsx`
  - `app/organizations/page.tsx`
  - `app/organizations/[id]/page.tsx`
  - `app/scan/config/workflows/page.tsx`
  - `app/scan/config/engines/page.tsx`
  - `app/tools/nuclei/page.tsx`
  - `app/tools/wordlists/page.tsx`
- child workspaces under an already visible shell:
  - `app/tools/fingerprints/*`
  - `app/targets/[id]/{websites,subdomains,ip-addresses,endpoints,directories,screenshots,vulnerabilities,settings}/page.tsx`
  - `app/scan/history/[id]/{websites,subdomains,ip-addresses,endpoints,directories,screenshots,vulnerabilities}/page.tsx`

High-visible route examples that keep chunk fallbacks invisible are captured in
the loading audit contract. Top-level first-screen workspaces should be direct
imports; route-level hidden-readiness pages keep only their hidden dynamic child
fallbacks invisible.

## Detail Shell Overview Handoff

The overview route is the readiness authority for a complete detail shell.
Do not start child queries earlier merely to dismiss the shell skeleton: the
overview must report readiness only after it has a stable visible frame.

## Detail Shell First-Entry Handoff

Direct import prevents a shell skeleton from cutting to a blank or delayed
route chunk, but it is not enough if the detail shell still reveals as
`title -> tabs -> content` inside the same first-screen layer.

For route-critical detail pages whose shell skeleton already covers the
breadcrumb, applicable tab rails, and active child workspace:

- keep the route page directly imported
- keep one feature-local shell loading state that consumes the same layout
  primitives as the resolved detail chrome; it is the only visible shell
  structure
- use `HiddenReadinessRouteBoundary` in the detail layout so the resolved shell
  mounts hidden until the overview page reports ready
- pass the ready signal through `DetailShellReadyProvider` or an equivalent
  route-owned handoff channel
- have every supported child workspace call `useDetailShellReadySignal` only
  after its first stable data, error, or empty frame exists
- while `deferInitialSkeleton` is active, the active child must suppress its own
  initial visible skeleton and return `null`
- keep the same `HiddenReadinessRouteBoundary` root mounted across every detail
  child route; only its first mount may defer the complete shell skeleton. Do not add
  or remove the boundary when the active tab changes, because that remounts the
  detail rail and breaks the tab-to-workspace handoff.
- capture the first-entry deferral intent on mount. Later pathname changes MUST
  NOT turn the resolved shell back into a loading shell; they only replace
  the child workspace.
- while target/scan detail metadata is unresolved, keep the global boot layer
  active with a non-visual `data-boot-handoff-pending` blocker. Do not render a
  standalone shell skeleton and then replace it with an equivalent boundary
  skeleton, because that remount restarts the shimmer.
- `ContentReveal` containing a pending boot-handoff blocker stays non-visible;
  do not let the route subtree count as a second visible owner underneath boot.
- while the complete shell owns first entry, nested child fallbacks reuse its
  geometry with the explicit `nested` contract and without declaring another
  `data-loading-owner`; after the shell is
  ready, each child keeps its existing local `ContentHandoff` for later
  navigation, refreshes, filtering, and pagination.
- do not render speculative tab labels, fallback resource identifiers,
  temporary zero counts, placeholder count badges, or a separate tab-rail
  skeleton. The heading, real rails, real counts, and child workspace reveal
  together.
- the complete shell skeleton MUST keep the URL-selected primary tab active.
  When localized labels are already available, use them invisibly to size the
  shared tab placeholders so the rail does not contract or expand only because
  the resolved locale replaces fixed placeholder widths. The labels remain
  visually hidden until the shell handoff; do not add speculative count badges.
- websites fallbacks MUST render `WebSitesDataTable` through its production
  loading mode and shared column/visibility contract. Do not keep a generic
  fixed-column `DataTableSkeleton` approximation beside that real path.

Current audited examples:

- `app/targets/[id]/layout.tsx` + all target detail child workspaces
- `app/scan/history/[id]/layout.tsx` + all scan detail child workspaces

Do not start child queries earlier or move data ownership upward just to make
the shell skeleton disappear sooner. This pattern is presentation-only and
exists to prevent shell-level `title -> tabs -> content` stair-step loading.

## Hidden Readiness Route Handoff

Some client-only route pages cannot mark the route ready until a dynamic child
mounts and its first-screen queries settle. In that case the route owns the
visible skeleton, and the child mounts hidden only to emit readiness.

The required pattern is:

- route file uses `HiddenReadinessRouteBoundary`
- dynamic import uses `loading: () => null`
- `HiddenReadinessRouteBoundary` owns a `ContentHandoff` with
  `mountContentWhileLoading` and `prepareContentBeforeHandoff`; when it is nested inside the protected app route
  owner, pass `layer="workspace" intent="data"` so it does not create a second
  visible `route` owner beside `auth-layout-route-content`
- child accepts `onReady?: () => void`
- child accepts `deferInitialSkeleton?: boolean`
- child calls `onReady?.()` only after the first visible real frame is ready
- child returns `null` for its own initial skeleton while
  `deferInitialSkeleton` is active

The route file must still keep the dynamic import, route owner, route skeleton,
and any viewport-fill wrapper geometry explicit. The helper only owns
route-level hidden-readiness state and the shared `ContentHandoff` wiring.

This pattern prevents a deadlock where `isLoading` prevents the child from
mounting while the child is responsible for clearing `isLoading`. It also keeps
the route skeleton as the only visible owner during the initial wait. While the
real branch is hidden, `ContentHandoff` clips its scrollable overflow: invisible
late content must not create a scrollbar or alter the skeleton's available
width before the handoff starts.

The current audited examples include:

- `app/settings/api-keys/page.tsx`
- `app/settings/database-health/page.tsx`
- `app/settings/notifications/page.tsx`
- `app/settings/support/page.tsx`
- `app/settings/system-logs/page.tsx`
- `app/scan/scheduled/page.tsx`

## Ready Semantics

`ready`, `onReady`, and `isLoading=false` mean the first visible content frame
is geometrically stable. They do not mean "the module imported" or "one query
returned while first-screen placeholders are still present".

Before clearing the route or section skeleton, verify:

- the first visible real header/tab/table/card/chart shell is mounted
- first-screen queries needed for that shell have settled enough to avoid a
  second skeleton swap
- wrapper geometry is stable at both desktop and narrow viewport widths
- hidden mounted content has been measured when using
  `mountContentWhileLoading`

For dynamic children whose readiness is checked outside the rendered child
tree, `ContentHandoff` keeps the outgoing skeleton visible during `handoff`
until committed content has positive natural geometry. A loaded chunk, settled
query, empty content wrapper, or zero-height shell is not the first visible
content frame. During that not-ready overlap, the content branch remains
visually and accessibility hidden; it must not paint above the skeleton merely
because its wrapper has mounted.

`ContentHandoff` retains the same one-track Grid in `loading`, `handoff`, and
`content`. Do not rely on a Flex-only settled phase to grow a `flex-1` wrapper:
the layout primitive must not change simply because the skeleton has been
removed.

Hidden content mounted through `mountContentWhileLoading` exists only to emit
or verify readiness. It MUST stay out of the loading Grid track and MUST NOT
write a late measurement, `min-height`, or other dynamic geometry back to the
visible owner or skeleton. If late metadata would change the first visible
surface, keep the skeleton visible until the resolved layout itself has reached
a stable first frame, then fix the shared layout contract rather than expanding
the visible skeleton.

### Hidden Readiness Gate

`prepareContentBeforeHandoff` is a narrow, explicit `ContentHandoff` option
for a data-backed first resolved frame whose real child needs to settle before
the visible handoff starts. It implies hidden content mounting during `loading`,
keeps the skeleton as the only visible owner, and waits for two stable animation
frames after committed content reaches positive natural geometry before a
`crossfade` or `replace` handoff begins.

It is a readiness gate, not a geometry mode. It MUST NOT apply a data-derived
`min-height`, resize the visible skeleton, or introduce a route-level height
exception. If stable content is taller or otherwise moves relative to its
skeleton, fix the shared layout contract; do not retain extra whitespace or add
a second geometry contract that masks the mismatch. The sole narrow exception
is a shared paginated table that has an explicitly declared
`resolvedIntrinsicTableSettlement` in `loading-route-contracts.mjs`: a sparse
resolved result may shorten only the named table body, surface, owner, and
natural-flow pagination position. It does not authorize a page-local height
patch, a broader tolerance, or drift in another slot.

## Browser Probe

Use the loading smoke probe for route ownership and geometry evidence:

```bash
cd frontend
LOADING_SMOKE_LOCALES=en \
LOADING_SMOKE_BASE_URL=http://127.0.0.1:3000 \
node scripts/run-loading-handoff-smoke.mjs
```

For targeted route and interaction checks, pass `LOADING_SMOKE_PRIORITY` and
`LOADING_SMOKE_INTERACTIONS` explicitly. Without
`LOADING_SMOKE_INTERACTIONS`, the probe also runs its default reviewed
interaction targets for safe local dynamic fallbacks. Set
`LOADING_SMOKE_INTERACTIONS=off` to disable those interaction probes. The
generated report is written to `frontend/test-plan/loading-handoff-smoke.json`.

For targeted route audits, prefer filtering by smoke target id instead of
editing the route inventory. The filter matches a target's own id plus only
its explicitly declared aliases; it never follows a redirect scenario's
destination contract. The authenticated login redirect is independently named
`smoke-login-authenticated-redirect` and intentionally aliases `route_login`
in skip-auth mode, while the canonical public login route remains its own
contract:

```bash
cd frontend
LOADING_SMOKE_TARGET_IDS=route_settings_agents \
LOADING_SMOKE_INTERACTIONS=off \
node scripts/run-loading-handoff-smoke.mjs
```

For staged-reveal investigations, capture a route timeline directly from the
shared probe:

```bash
cd frontend
LOADING_SMOKE_TARGET_IDS=route_settings_agents \
LOADING_SMOKE_TIMELINE_MS=700,1300,2200,4000 \
LOADING_SMOKE_INTERACTIONS=off \
node scripts/run-loading-handoff-smoke.mjs
```

Each selected route report then includes `timeline[]` snapshots with requested
time, actual sampled time, heading text, loading owners, visible `data-slot`
markers, phases, and owner geometry. Use that timeline evidence before
collapsing a sectioned dashboard into one page-wide skeleton, before promoting
a local section owner into a route-wide owner, or before misclassifying an
inline metric strip as a staged scene cut.

### Structural Geometry Slots

`ContentHandoff` automatically gives both direct wrappers the paired
`data-loading-slot="surface"` marker. A feature whose first-screen header,
toolbar, value band, grid, table shell, pagination, or primary content region
can move independently MUST give the skeleton and resolved branches one matching
`data-loading-slot` for that region. Do not reuse ordinary `data-slot` test
selectors for this purpose, and do not add a second `surface` marker.

Every paired owner must list its complete, route-specific `requiredSlots` in
`frontend/scripts/loading-route-contracts.mjs`. `surface` is never an automatic
contract fallback. A surface-only contract is valid only when the route inventory
explicitly documents that no first-frame subregion can move independently; normal
P0/P1/P2 workspace and section owners should name their header, toolbar, body,
footer, pagination, or feature-local workbench regions.

The loading foundation guard rejects a direct `data-loading-slot` declaration
outside `ContentHandoff` unless its feature-local shared template is explicitly
reviewed in `check-loading-foundation.mjs`. A ledger exception cannot waive this:
add the slot to the real handoff/template and update its route contract instead.

The smoke script reads `frontend/scripts/loading-route-contracts.mjs` for every
active P0/P1/P2 route expectation. `expectedOwners` records semantic loading
ownership and layer transitions; it is not automatically a paired-geometry
owner. A `geometry.disposition: "verify"` contract lists only owners that expose
the paired skeleton/content structure states, while `excludedOwners` records
each semantic owner that deliberately cannot be paired and why. In particular,
`auth-layout-route-content` is a resolved-only `ContentReveal`, so it MUST stay
excluded from paired geometry. A `geometry.disposition: "not-applicable"`
contract MUST declare no paired owners and MUST explain why its first screen has
no comparable handoff.

During a cold load the probe records samples from document start and makes two
strict comparisons. It first compares intrinsic skeleton/content slots,
preferring a same-frame hidden-content pair. It then compares the last observed
skeleton frame to every later `content`-only frame for every required slot and
for the owner outer rectangle. Owner `top` and `left` are stored in document
coordinates, so scrolling cannot disguise a handoff displacement. Missing
owners, missing/duplicate slots, a missing content-only pair, or either kind of
drift fail the smoke. The automatic `surface` slot is measured at its intrinsic
grid-item height while a crossfade overlays both branches, so Grid stretch cannot
make two differently high branches pass merely because they share the larger
track. `ResizeObserver` supplements DOM mutation sampling, which catches a
late CSS or layout size change that does not mutate the loading subtree.

An approved `bounded` exception may describe only a genuinely non-paired region
outside an owner's required slots. It cannot replace the owner's paired geometry
or waive `surface`. It must include `owner`, `affectedSlots`, `reason`,
`reviewTrigger`, `recoveryPath`, and a testable `firstFrame` record with a named
measurement, at least two stable frames, and `contentOnlyStable: true`.

`resolvedIntrinsicTableSettlement` is separate from `bounded`. It is available
only to an owner that names its exact table-body slot and optional natural-flow
pagination slot. The smoke contract still requires stable owner top/left/width,
toolbar geometry, and every unlisted slot. It permits only a shorter owner,
surface, and named body, plus an upward pagination movement; a taller result,
downward pagination movement, width change, or unrelated drift remains a
failure.

The probe intentionally ignores unconfigured local owners for paired geometry while
continuing the existing semantic layer and ownership diagnostics. It also fails
geometry drift, missing layer metadata, same-layer visible owner overlap, long
boot handoff overlap, visible route shell with a missing workspace owner,
expected owner timeline gaps, and unsettled route owners.

Manual browser sampling should inspect:

- `data-loading-owner`
- `data-loading-layer`
- `data-loading-intent`
- `data-loading-phase`
- `.loading-handoff__skeleton`
- `.loading-handoff__content`
- `data-loading-slot` rectangles relative to the owner (`top`, `left`, `width`, and `height`)

Always sample at least one desktop viewport and one narrow/mobile viewport
before claiming a route loading handoff is stable.

## Data Table Skeleton Rows

Initial data-table skeletons are a first-screen geometry contract, not a
prediction of the final result count. When a table owns an active pagination
state, derive the skeleton row count from that `pageSize` through
`getDataTableSkeletonRowCount(pageSize)`.

When the resolved surface is a `UnifiedDataTable`, `BusinessListDataTable`, or
a route wrapper built on those owners, the loading skeleton should render that
same owner in loading mode. Route-local skeleton components may wrap the owner
to provide `data-loading-owner`, columns, empty data, and row count, but they
must not rebuild the shared toolbar, action slots, table rows, column layout,
or pagination with local skeleton markup.

If a resolved table intentionally hides toolbar or pagination chrome, the
loading skeleton must hide the same chrome through shared switches such as
`withSearch={false}`, `toolbarButtonCount={0}`, or `withPagination={false}`.
Do not show a skeleton search row or pagination row for preview tables whose
resolved state omits those controls.

The helper caps the first visible skeleton body at 10 rows. This keeps ordinary
10-row business-list pages aligned with the first request, lets compact embedded
tables render fewer rows, and prevents large page sizes such as 50 or 100 from
creating oversized loading surfaces.

Do not hard-code `rows={3}`, `rows={4}`, `rows={6}`, or `rows={8}` for an
initial paginated table load. If a route shell fallback renders a full table
skeleton before the table state exists, derive the row count from that route's
default table `pageSize` through the same helper. Fixed row counts are only
appropriate for non-table preview geometry that does not own the resolved
table's pagination state.

If a dense table-like skeleton cannot use the shared table loading owner, mark
it as a controlled variant near the component and explain why the shared owner
does not fit. The loading-foundation check treats route-local skeletons that
combine a hand-built table shell with toolbar/action/pagination placeholders as
drift unless they are reviewed through the exception path.

## Action Placeholders

Initial loading action controls are geometry hints, not miniature fake buttons.

- If a toolbar, header, card footer, or detail footer needs to preserve an
  action slot during initial loading, use `ActionSkeleton`.
- `ActionSkeleton` stays structural: one button-shaped block only, with no
  nested label strip, icon chip, chevron, counter, or spinner placeholder.
- Structural does not mean static: `ActionSkeleton` SHOULD still carry the
  shared skeleton shimmer owner so button placeholders move with the rest of
  the loading scene.
- Even when the resolved control is a primary CTA, the initial placeholder
  SHOULD stay low-fidelity and neutral instead of reusing branded primary-fill
  colors that make the skeleton read like a half-finished real button.
- `ActionSkeleton` MUST reuse the shared `Button` structural size contract for
  `default`, `sm`, `lg`, `icon-sm`, `icon`, and `icon-lg` geometry instead of
  restating local `h-*`, `size-*`, or `rounded-*` button-like values.
- If an action slot is not important for geometry continuity, omit it instead
  of inventing fake control detail.
- Real post-click pending states belong on the resolved control via shared
  `Button` loading or shared `Spinner`, not via `ActionSkeleton`.

## Control Shells

Initial loading control shells should reuse shared control geometry instead of
rebuilding route-local lookalikes.

- Search and input shells SHOULD use `SearchToolbarSkeleton` or a real disabled
  `Input` shell when the control geometry affects the layout.
- `SearchToolbarSkeleton` defaults to the same no-submit-button geometry as
  `SimpleSearchToolbar` and preserves inline search icon spacing with a neutral
  invisible leading spacer, not a visible icon placeholder.
- Select-like shells such as taxonomy filters and rows-per-page SHOULD use
  `SelectShellSkeleton` or an equivalent shared select-shell owner.
- Shared loading control shells SHOULD stay text-first. Do not add decorative
  leading icon placeholders inside ordinary search, filter, or page-size
  skeleton controls.
- When a resolved search or select control owns a fixed leading slot that
  changes the text start position, the skeleton SHOULD preserve that spacing
  with a neutral spacer instead of drawing a visible icon placeholder.
- Dense pagination SHOULD use `CompactPaginationSkeleton`, which keeps the
  page-size trigger on `SelectShellSkeleton` and keeps icon-only navigation
  actions on `ActionSkeleton size="icon-sm"`. Cursor-paginated surfaces MUST
  pass `mode="cursor"` so the skeleton omits numbered-summary and page-value
  rows that the resolved cursor control does not render.
- Tabs SHOULD prefer real `TabsList` / `TabsTrigger disabled` instead of
  skeleton-only faux tab rails.
- Do not hand-draw route-local control shells with local
  `border border-input bg-background ... <Skeleton />` markup when a shared
  control-shell owner already fits.

## Content Handoff Geometry

`ContentHandoff` owns the wrapper geometry for both skeleton and resolved
content. Viewport-fill routes, split panes, terminals, log viewers, and other
surfaces that depend on `flex-1 min-h-0` MUST pass matching `skeletonClassName`
and `contentClassName` values so the skeleton wrapper and content wrapper
participate in the same height contract. Do not rely on only the inner skeleton
or inner content node to stretch; the handoff wrapper is the shared layout
owner during `loading`, `handoff`, and `content` phases.

## Route Chunk Fallbacks And Detail Skeletons

When a route chunk loads a component that owns its own first-screen data
skeleton through `ContentHandoff`, pass an explicit invisible chunk fallback
such as `null` to the route lazy loader. Do not let the route chunk fallback
render `PageSectionSkeleton` or another title-like generic skeleton in front of
the component-owned skeleton. The same rule applies to direct `next/dynamic`
usage in `app/`: route files should use `loading: () => null` and let the route
`ContentHandoff` own the visible skeleton. A visible chunk fallback and a
visible data skeleton are the same visual layer; stacking them creates a
sequential loading scene and can leave legacy heading or breadcrumb placeholders
that do not exist in the resolved content.

Detail skeletons should mirror the resolved detail surface they replace. If the
resolved detail view does not render a breadcrumb, route title, or page subtitle
inside that component, its component-owned skeleton must not keep one as a
legacy placeholder. Target and scan-history detail shells use their local shared
layout modules instead of a generic detached shell. A generic shell primitive is
only appropriate when it remains the same layout owner for both states.
