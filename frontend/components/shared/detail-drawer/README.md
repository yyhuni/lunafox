# DetailDrawer

`DetailDrawer` is the shared right-side drawer for read-only inspection and diagnostic detail workflows.

## Use When

- The user opens details from a table row, status badge, finding, scan record, or runtime object.
- The user needs to inspect structured information without leaving the list page.
- The content benefits from tabs, sections, summaries, logs, or tables.

## Do Not Use When

- The primary task is creating or editing a record; use `FormDrawer`.
- The content only asks for a short confirmation; use the shared confirm dialog.
- The drawer needs a persistent submit/cancel footer; use or extend `FormDrawer`.

## Layout Contract

- Shell: shared `Sheet` with `detailDrawerContentClassName`.
- Backdrop: inherits the shared shadcn base-nova overlay from `SheetContent`; detail drawers do not own lighter backdrop variants.
- Motion: `DetailDrawer` always uses the shared edge-panel transform transition. Its backdrop transitions only opacity. The shared Sheet primitive stages a dynamically loaded controlled detail drawer's first `open=true` frame before opening, so callers cannot opt out of directional motion or add a second drawer-entry animation; workbench variants own only content layout and width.
- Header: `DETAIL_DRAWER_COMPACT_INSET_CLASS` gives the shell the same 12px edge inset as the compact workbench. It keeps the bottom border, shared `EdgePanelHeader` `detail` variant, `min-h-8` title rhythm, `textRole.panelTitle` title, optional `titleMeta`, one close action, and optional secondary `headerMeta` row. `titleMeta` is for one compact primary signal beside the title, not a cluster of business metadata. The title row must keep `min-w-0`; plain string titles truncate with a native `title` attribute so long localized labels do not push the close action or drawer width.
- Content: ordinary structured read-only bodies use `DETAIL_DRAWER_COMPACT_BODY_CLASS`, fields use `DETAIL_DRAWER_COMPACT_FIELD_GRID_CLASS`, and fixed read-only action rows use `DETAIL_DRAWER_COMPACT_FOOTER_CLASS`. This keeps header, tabs, body, and footer on the same horizontal axis. The caller still owns detail semantics and must keep `min-w-0` on flex/grid roots and prevent horizontal overflow from escaping to the drawer or page.
- Tabs: use `DetailDrawerTabs`, `DetailDrawerTabsList`, `DetailDrawerTabsTrigger`, and `DetailDrawerTabsContent`. Content-tab rails automatically use the shared compact horizontal gutter; compact secondary subviews may pass `variant="minimal"` to both to use the response-style underline tabs.
- Sidecar: detail workflows may pass `sidecar` when a secondary form should appear as a sibling column to the left of the detail drawer instead of splitting the detail body or opening a nested overlay. On desktop the sidecar is anchored to the detail drawer's left edge and enters with transform/opacity motion while the detail drawer stays stable. On narrower screens the sidecar covers the detail drawer content so users complete one task at a time instead of reading two cramped panels. Use `onSidecarClose` so Escape closes the sidecar before closing the drawer.

## Styling Contract

- The shared shell owns edge placement, width tier, backdrop, header rhythm, and close affordance.
- Detail content should use theme tokens, typography roles, shared tabs, shared tables, and shared badges.
- The shared inner shell uses `bg-card`, matching compact feedback drawers such as notifications. Workbench-like detail regions may use `bg-background` inside their own content only; they must not override the drawer shell.
- Avoid route-local `SheetContent`, raw overlay colors, custom close buttons, and one-off drawer widths.
- Code viewers, logs, editors, long evidence, and multi-panel diagnostic workbenches retain their own inner scroll and reading geometry. They may share the shell header edge but must not be forced into the ordinary structured-body classes.

## Route Usage

Route modules pass title, optional description, and detail content. They should not compose raw `SheetContent` for record details. If a detail workflow needs a distinct repeated layout, extend `DetailDrawer` or introduce a named shared owner instead of local patches.
