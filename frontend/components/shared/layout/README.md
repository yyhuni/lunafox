# Detail Asset Content Frame

## Page shell density

`page-shell-density.ts` owns the compact vertical rhythm for authenticated
production route shells and ordinary section stacks. Use the shared constants
for route and loading owners instead of repeating page-local `gap`/`py` values.
The compact rhythm is `12px` at every breakpoint. Horizontal gutters are owned by `PageHeader` and
`DetailAssetContentFrame` through `COMPACT_CONTENT_GUTTER_CLASS`
(`px-3`). Generic content regions that sit below route chrome should
use the same shared constant rather than restating the utility bundle.

`COMPACT_CONTENT_FRAME_CLASS` bounds the authenticated route frame to 1624px
at the default spacing scale (`max-w-406`): 1600px of usable content plus the
existing two 12px gutters. It is fluid below that limit and centered within the
main workspace after the sidebar. Apply it once through the protected shell,
not to individual pages or nested sections. It adds no padding and does not
constrain the top bar, scroll viewport, or portaled overlays.

An active page-level canvas may declare `data-workspace-width="full"` to opt
the enclosing frame out via CSS `:has()`. The workflow composition canvas is
the current caller; its management list stays bounded. Do not mark ordinary
tables, inline charts, or hidden pre-mounted canvases. New graph/canvas owners
must document the use here and verify restoring the cap when leaving them.

The constants only govern page/section whitespace. Controls retain their shared
hit areas, while charts, terminals, editors, long evidence, complex identity
rows, and detail overlays keep their content-driven or controlled internals.

When a route-owned content region must remain viewport-bound and scroll on its
own, render the shared `ScrollArea` with
`COMPACT_PAGE_SCROLL_AREA_CLASS`,
`COMPACT_PAGE_SCROLL_AREA_CONTENT_CLASS`, and
`COMPACT_PAGE_SCROLL_AREA_VIEWPORT_CLASS`. Do not put native
`overflow-y-auto` on that same gutter owner: a reserved native scrollbar makes
the visible right page inset wider than the left one. This does not apply to
controlled internal scroll surfaces such as tables, logs, editors, and canvases.

Search's initial command-style state and the community support page are bounded
route exceptions. They still use the authenticated shell and responsive gutter;
their centered composition and editorial content stack retain intentional
breathing room so an empty search state and a contribution call-to-action do
not read like data tables. Their outer padding and primary stack gaps remain
on the compact baseline.

`DetailAssetContentFrame` owns the shared responsive content gutters and bottom
breathing room for the website, subdomain, IP address, and URL workspaces under
target detail and scan history detail routes.

Use it in the resolved route page and in the state-less route fallback. Do not
add it inside a domain `*LoadingState`; those loading states already render
inside the page-owned frame and must keep their domain-specific DataTable
geometry.

The default frame is `px-3 pb-3`. Route-specific attributes and
classes may be forwarded when they do not create another gutter owner.
# Selection workspace sizing

`selection-workspace-layout.ts` owns the organization/target selection toolbar scope-filter width. Both domain workspaces use `SELECTION_WORKSPACE_SCOPE_TRIGGER_CLASS` (128px, non-shrinking) with the shared compact Select size. Search inputs take the remaining width; translated labels and scope changes must not change the filter slot. Pagination selectors retain their separate size. Base UI supplies selection behavior, not this domain-level width policy.
