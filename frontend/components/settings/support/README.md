# Support Page

Support page loading mirrors the value-first resolved surface through
`support-page-loading-state.tsx`, `support-page-content.tsx`, and
`support-page-layout.tsx`.

- `support-page-loading-state.tsx` owns `SupportPageLoadingState`; resolved
  content and workspace data loading MUST reuse that lightweight
  visual state instead of reintroducing `support-page-skeleton.tsx`.
- Sidebar and the protected shell do not inject a Support skeleton; the
  destination workspace owns the choice between `SupportPageLoadingState` and
  ready content.
- The resolved content MUST be directly readable on first render. Do not add a
  scratch card, locked reveal, forced identity input, or other interaction before
  the user can understand the project's value.
- `SupportPageLayout`, `SupportPageHeader`, `SupportPageValueBand`,
  `SupportPageActions`, and the shared repeated-value-band helpers are the one
  feature-local structural layout owner. Both loading and resolved states MUST
  consume them; neither state may recreate an independent page grid, gutters,
  responsive breakpoints, or action-row layout.
- `SupportPageLayout` owns the route-level footer slot. Both loading and
  resolved states MUST pass `SupportPageFooter` through that slot so the support
  note remains at the route bottom after content is disclosed.
- The loading and resolved trees each expose exactly one
  `data-loading-slot="support-page-header"`,
  `data-loading-slot="support-page-value-band"`, and
  `data-loading-slot="support-page-actions"`, and
  `data-loading-slot="support-page-tier-options"`. The tier-options slot keeps
  its geometry while hidden until the contribution action reveals it, so the
  page does not reflow. `ContentHandoff` owns the route `surface` slot, so
  Support components MUST NOT add their own `surface` slot.
- The loading owner MUST mirror the value-first geometry closely enough that the
  handoff does not show a second, unrelated interaction model. It preserves the
  resolved text line boxes and shared button sizing beneath low-fidelity shared
  skeleton primitives, so localization and narrow-screen wrapping remain stable.
- Reviewed support-page spacing exceptions are bounded to this custom responsive
  page. When the value-first geometry changes, update the layout contract and
  local contract test before changing resolved or loading markup.
- Tier-card fixed geometry belongs to `SUPPORT_TIER_CARD_CLASS`; do not put
  `min-h-[160px]` or local radius/focus bundles back into the page JSX.
- `SupportPageContent` owns the value-first contribution disclosure. The
  resolved page MUST present maintenance, growth, and open-collaboration value
  before the `join contributors` action reveals explicit amounts. Payment
  methods and QR details stay in the selected-amount dialog; do not move them
  into the initial value narrative.
- Support-tier labels and descriptions use neutral contribution terminology.
  Do not reintroduce food, feeding, wallet, or consumer-purchase metaphors.
- `SupportGrowingBranch` precomputes a random tree so tier changes reveal a
  stable shape. Each completed branch has a 22% chance to add a terminal flower;
  that flower must reveal only when its branch has finished revealing.
- The route surface clips horizontal decoration only. Keep vertical overflow
  available so the disclosed tier list remains scrollable on narrow viewports.
- The route surface aligns content to the top on narrow viewports and centers it
  from `md` upward; do not restore unconditional centering or mobile content can
  be positioned above the scroll origin after tier disclosure.
