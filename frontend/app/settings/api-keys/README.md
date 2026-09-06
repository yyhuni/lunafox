# API Keys Settings Page

This route keeps the resolved credential editor and initial loading state on one
shared geometry path. Shared page, master-detail, provider row, credential field,
and footer action layout classes live in `api-keys-settings-layout.ts`.

- `frontend/components/settings/api-keys/api-keys-settings-loading-state.tsx`
  owns `ApiKeysSettingsLoadingState`; resolved content and workspace data loading
  MUST reuse that lightweight visual state instead of
  reintroducing `api-keys-settings-skeleton.tsx`.
- Sidebar and the protected shell do not inject an API Keys skeleton; the
  destination workspace owns the choice between `ApiKeysSettingsLoadingState`
  and ready content.
- Loading button-like placeholders MUST use `ActionSkeleton` so loading actions
  follow the shared `Button` structural size contract.
- The provider-list header is a search-only toolbar. Resolved and loading states
  MUST reuse the shared inline-icon `SearchInput` geometry and MUST NOT restore a
  redundant provider-list title beside it. This compact search toolbar owns its
  40px height independently; the provider-detail header keeps the shared compact
  `CardHeader` rhythm.
- The provider search field keeps a transparent light-theme surface and uses
  `bg-card` in dark mode, overriding the shared input's muted dark fill so the
  header reads as one card surface. Focus feedback remains ring-only.
- Do not restate password input row classes, footer action groups, or local
  `h-*` / `size-*` button placeholders inside the route components.
- The initial route boundary and both handoff branches use the shared
  viewport-bound workbench classes from `api-keys-settings-layout.ts`. Provider
  and credential form growth must scroll inside their card/content regions, not
  change the first-screen route surface during loading handoff.
- The route pairs `api-keys-header`, `api-keys-provider-list`,
  `api-keys-provider-detail`, and `api-keys-notice` between its loading and
  resolved branches. `ContentHandoff` owns the surrounding `surface` slot.
