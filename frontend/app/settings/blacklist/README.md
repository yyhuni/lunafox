# Blacklist Settings Page

The standalone blacklist route keeps the resolved workbench and loading state on
one shared geometry path. The reusable workbench, page shell, rule list,
editor, and action-row geometry live under
`components/settings/blacklist/` so target detail settings can embed the same
presentation without importing from `app/`. The standalone route owns only the
global BlacklistPolicy; embedded Target settings use its Target-local child policy.

- `frontend/components/settings/blacklist/blacklist-settings-workspace.tsx`
  owns `BlacklistSettingsWorkspace`; standalone mode includes the PageHeader
  and page gutter, while embedded mode omits both and keeps the target detail
  shell as the navigation owner.
- `frontend/components/settings/blacklist/blacklist-settings-loading-state.tsx`
  owns `BlacklistSettingsLoadingState`; resolved content and data loading MUST
  reuse that lightweight visual state instead of
  reintroducing `blacklist-settings-skeleton.tsx`.
- Sidebar and the protected shell do not inject a Blacklist skeleton;
  `content.tsx` owns the choice between its query loading state and ready
  content.
- `blacklist-page-content` is the standalone first-screen workspace owner. Its
  `ContentHandoff` must declare `layer="workspace"` and pair `blacklist-header`,
  `blacklist-controls`, and `blacklist-list` on the real page structures in both
  states. `prepareContentBeforeHandoff` is only a
  hidden-content readiness gate; it must not reserve height or mask a geometry
  mismatch with a blacklist-local minimum-height value.
- Small screens let the workbench follow natural page flow, with a stable rule-list
  viewport. On desktop, the rule list fills its card's remaining workbench height.
  Rule counts and empty groups are data-dependent, so they must remain inside this
  shared scrollable viewport rather than changing the page surface when loading resolves.
- Loading action placeholders MUST use `ActionSkeleton` instead of rendering a
  disabled real `Button` or local button-like `h-*` / `rounded-*` blocks.
- Do not restate editor action rows or line-numbered textarea geometry inside the
  route components; update the layout contract and local contract test first.
