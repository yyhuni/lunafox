# UI Foundation Rules

`frontend/components/ui` is the near-code entrypoint for LunaFox frontend UI foundation. Humans and AI agents MUST read this file before changing production frontend UI, shared primitives, theme tokens, shell layout, or guardrail scripts.

Concrete component contracts and implementation rules live near code: this README, local README files, and contract tests. `openspec/specs/frontend-ui-foundation-guardrails/spec.md` is a high-level foundation facts reference for approved domains and migration context, not the source for component-level API rules. Historical plans and completed changes are useful context only; stale `bauhaus` references are not active production guidance. Runtime themes are registered through a flat registry, and the user-facing selection model is appearance mode only. The current built-in runtime entries are `lunafox-light` as the default light theme and `lunafox-dark` as the default dark theme. The top bar exposes a compact appearance menu for system, light, and dark; it uses one shared radio selection group and does not expose a separate palette picker. Removed or legacy palette/theme storage values are migration inputs only and resolve to the default light/dark theme.

## Foundation Protocol

### Preflight

- Read this README, the nearest local README or contract test, and `openspec/specs/frontend-ui-foundation-guardrails/spec.md`.
- When the change touches a reusable business-list table, also read `frontend/components/shared/data-table/README.md` before editing route-level table markup or shared table helpers.
- Identify affected domains from the coverage matrix below.
- List approved entrypoints before editing: typography roles, theme tokens, shared components, variants, sizes, icons, overlay helpers, status helpers, trend helpers, and exception ledger records.
- When adding or changing a concrete component API or usage rule, update the near-code README and contract tests first. Update OpenSpec only for approved high-level foundation scope or behavior changes.

### Decision Tree

- Existing shared entrypoint fits: use it.
- Existing shared entrypoint almost fits: extend the shared component or helper when reuse is likely.
- One-off value is truly local: add a bounded exception with owner, reason, status, review trigger, and recovery path.
- Static checks cannot prove visual behavior: add route, interaction, screenshot, or visual evidence to verification.

### Forbidden Shortcuts

- Do not add raw color literals in production UI to make a local surface look right.
- Do not create local button or control systems with native `button` plus utility bundles.
- Do not use arbitrary `text-[...]`, `p-[...]`, `h-[...]`, `rounded-[...]`, `shadow-[...]`, or similar values as the default styling path.
- Do not add allowlist or ledger entries without owner, reason, status, review trigger, and recovery path.
- Do not claim frontend UI work is complete without running matching guardrails.

### Done Definition

Frontend UI changes must report changed foundation domains, shared entrypoints used or extended, exceptions added or touched, verification commands and outcomes, and any skipped verification with a concrete reason.

### Post-Convergence Governance

The current foundation posture is convergence-by-guardrail, not repository-wide cleanup by default. A passing `check:ui-foundation` plus a ledger with only reviewed `keep`, `narrow`, or `excluded` entries means the change is not blocked by foundation drift. Existing reviewed ledger entries are controlled historical debt, not an automatic instruction to touch every matching file.

- Do not run broad standardization sweeps unless the task explicitly asks for a migration batch or an approved OpenSpec change scopes the work.
- When a task touches a file that already has a matching ledger entry, first try to reduce or remove the entry within that same task boundary.
- Do not chase unrelated ledger entries in neighboring route groups, shared components, or visual surfaces just because they appear in inventory output.
- Do not broaden an existing ledger entry to make a new finding pass. Prefer code migration, then a narrower exception only when the value is intentionally retained.
- A successful foundation pass is not permission to skip local UX review. Dialogs, shell changes, loading handoff, route composition, and interaction surfaces still need the browser or contract evidence required by `docs/harness/verification.md`.

AI agents should treat this section as a scope brake: read it before editing, report touched foundation domains after editing, and leave unrelated reviewed debt alone unless the user asked for that cleanup.

## Rule Levels

- `MUST`: blocking. Verify mode fails when new production drift is not approved.
- `SHOULD`: default policy. Exceptions are allowed only when reviewed and recorded.
- `MAY`: constrained extension. Prefer extending shared foundation APIs over page-local utility bundles.

## Base UI-First Primitive Standards

Shared primitives under `frontend/components/ui/**` are the adapter layer for third-party primitive behavior. New or materially changed shared primitives MUST use Base UI or a native/platform equivalent when a safe equivalent exists. Radix-backed wrappers are migration exceptions, not the default baseline.

- Production business code MUST import primitive behavior from `@/components/ui/*` or a higher-level shared owner. Do not import raw `@base-ui/*` or `@radix-ui/*` from `frontend/app/**`, feature components, hooks, or ordinary libs.
- Raw `@base-ui/*` and `@radix-ui/*` imports are limited to `frontend/components/ui/**`, focused contract tests, this README, and explicitly approved exceptions.
- Migrated primitives MUST remove stale Radix-only imports, `--radix-*` CSS variables, `data-radix-*` selectors, and compatibility props unless a caller-backed compatibility record states the current caller, reason, risk, and removal condition.
- Migrated primitives currently include `Select`, `Drawer`, `Avatar`, `Label`, `Progress`, `Separator`, `ScrollArea`, `Checkbox`, `Switch`, `RadioGroup`, `Tabs`, `Collapsible`, `Toggle`, `ToggleGroup`, `Dialog`, `AlertDialog`, `Sheet`, `Popover`, `DropdownMenu`, `Tooltip`, and `HoverCard` (implemented with Base UI `PreviewCard`). `Command` is project-owned and uses the shared Base UI-backed `Dialog` wrapper for its dialog shell instead of `cmdk`. `Label` uses a native label wrapper because Base UI has no standalone label primitive.
- Remaining Radix-backed primitive exceptions: none in production shared UI. If a future Radix-backed primitive is retained, record the owner, reason, caller scope, deletion condition, and risk here before adding or keeping the dependency.
- Migrated primitives MUST NOT expose Radix-style compatibility APIs such as `asChild`, `forceMount`, `delayDuration`, `onOpenAutoFocus`, `onCloseAutoFocus`, `AlertDialogAction`, or `AlertDialogCancel`. Use Base UI semantics instead: `render` for custom trigger/control composition, `keepMounted` for tab panels, `delay` for tooltip timing, `initialFocus` / `finalFocus` for popover focus behavior, and `AlertDialogClose` for close buttons. Component-level contract tests MUST fail if those compatibility APIs are reintroduced.
- `AlertDialogClose` owns confirmation-action styling: use `variant="outline"` for cancellation and `variant="destructive"` for irreversible confirmation. Do not nest a `Button` with `render` solely to obtain a destructive color.
- Migrated primitives MUST use Base UI state attributes for styling and tests, not Radix `data-state` selectors. Use `data-checked`, `data-unchecked`, `data-indeterminate`, `data-active`, `data-hidden`, `data-popup-open`, and `data-panel-open` for the matching Base UI states. `Checkbox` indeterminate state MUST be passed through `indeterminate={...}` with boolean `checked={...}`; do not reintroduce `checked="indeterminate"`, `checked?: boolean | "indeterminate"`, or caller patterns such as `checked={all || (some && "indeterminate")}`. Project-owned states such as table row `data-[state=selected]` or sidebar `collapsed` / `expanded` are separate contracts and do not make Radix-style selectors valid for migrated primitives.
- Shared `Checkbox` uses `interaction-accent` for both checked and indeterminate background/border states with the paired foreground icon. Unchecked boxes retain their neutral border, and callers must not add page-local selection colors.
- Shared `Switch` uses `interaction-accent` for its checked track. Its unchecked track, thumb, and adjacent enabled/disabled status text retain their existing tokens.
- `DropdownMenuContent` keeps the project-owned `width` modes because they define LunaFox menu geometry, not Radix compatibility. Width behavior should continue to be implemented through the Base UI `Positioner` / `Popup` wrapper and Base UI geometry variables.
- Base UI checkbox and radio menu items default to staying open after selection, but LunaFox dropdown menus hard-cut that behavior: `DropdownMenuCheckboxItem` and `DropdownMenuRadioItem` MUST close on click, and callers MUST NOT receive or pass `closeOnClick`. Multi-select or long-lived option pickers that intentionally stay open need a dedicated shared picker or popover owner instead of weakening dropdown item semantics.
- Approved exceptions MUST be narrow and reviewable: owner, reason, status, current caller, review trigger, recovery path, deletion condition, and risk. Prefer removing the direct import or moving the behavior into `frontend/components/ui/**` before adding an exception.
- Direct Radix package dependencies may be removed only when repository search shows no remaining production usage outside approved exceptions and the lockfile is updated with pnpm.
- `components.json` uses the shadcn `base-*` style family so future CLI additions resolve Base UI docs and registry paths. The upstream Base `Command` registry item still depends on `cmdk`; do not overwrite this project's `Command` with shadcn CLI output unless a new approved change explicitly reintroduces and records that dependency.

## Coverage Matrix

| Domain | Owner / entrypoint | Forbidden patterns | Exception classes | Verification |
| --- | --- | --- | --- | --- |
| Typography | `frontend/lib/typography.ts`, shared text-owning components | Local readable `text-*`, `font-*`, `tracking-*`, `uppercase` bundles | `typography-role-owner`, `terminal-log`, `chart-rendering`, `skeleton-geometry` | `check:typography-foundation`, contract tests |
| Color | `frontend/styles/themes/**`, `frontend/app/globals.css`, status/severity/chart helpers | Raw hex/rgb/hsl/oklch, raw var fallbacks, local status maps | `theme-token-owner`, `brand-provider-color`, `terminal-log`, `chart-rendering`, `canvas-webgl` | `check:color-foundation` |
| Spacing | shared shells, cards, tables, forms, dialogs | Unclassified arbitrary spacing, dimensions, radius, shadow | `runtime-calculated-geometry`, `radix-dynamic-var`, `skeleton-geometry` | `check:spacing-foundation` |
| Component sizing | `frontend/components/ui/**`, shared data-table/form/status components | Page-local Button/Input/Badge/Tabs/Card/Table/Dialog systems | `legacy-migration`, `runtime-calculated-geometry` | `check:button-foundation`, `check:component-foundation` |
| States | shared components and status/severity helpers | Color-only state semantics, local disabled/loading/error styles | `accessibility-only`, `legacy-migration` | component contracts, a11y checks |
| Icons | `frontend/components/icons` | Direct third-party icon imports in production components | `visual-lab-excluded`, `legacy-migration` | `check:component-foundation` |
| Shape/elevation | theme tokens, Card/Dialog primitives | Page-local radius/shadow systems, nested cards as section layout | `visual-lab-excluded`, `legacy-migration` | `check:spacing-foundation` |
| Layering | overlay helpers and UI primitives | Unowned z-index stacks, custom overlay portals | `radix-dynamic-var`, `legacy-migration` | component contracts |
| Motion | global CSS and shared interactive components | Layout-shifting hover motion, unbounded infinite motion | `visual-lab-excluded`, `canvas-webgl` | `check:motion-foundation` |
| Accessibility | UI primitives, route smoke, interaction smoke | Icon-only controls without names, focus removal, non-keyboard clickable surfaces | `accessibility-only`, `legacy-migration` | `check:a11y-foundation`, smoke tests |
| Responsive | shared shells, table/dialog/form contracts | Fixed dimensions that break mobile or long i18n text | `runtime-calculated-geometry`, `legacy-migration` | route smoke, visual evidence |
| Data visualization | chart helpers and chart components | Unclassified chart colors or legends using color alone | `chart-rendering`, `canvas-webgl` | inventory plus chart contracts |
| Feedback states | Toast, Tooltip, Skeleton, Empty/Error/Loading patterns | Page-local retry/loading systems when shared patterns exist | `skeleton-geometry`, `legacy-migration` | component contracts |

## Iconography Standards

Production iconography is owned by `frontend/components/icons`. Stable product
concepts MUST use `semanticIcons.concept.*` from `@/components/icons` before
reaching for a raw icon name. This applies equally to sidebar navigation,
lists, dialogs, detail views, and metrics: the surface may change tone, size,
or icon-container treatment through shared tokens and primitives, but it must
not select another glyph for the same product noun.

- Use `semanticIcons.concept.*` for organization, target, vulnerability, scan,
  scheduled scan, workflow, agents, tools, system resources, and documented
  asset/configuration concepts. Explicit subtypes such as domain, IP, and CIDR
  keep their own concept entries.
- Use `semanticIcons.navigation.*` only for navigation-only utilities such as
  overview, search, support, and about.
- Use `semanticIcons.action.*` for repeated commands such as add, edit, delete,
  view, run, schedule, search, filter, refresh, and more.
- Use `semanticIcons.status.*` and non-entity `semanticIcons.metric.*` for
  lifecycle indicators and measurements.

Do not pick page-local alternatives or raw stable-concept aliases. When a
stable concept is missing, extend `frontend/components/icons/semantic-icons.tsx`
and its near-code contract instead of importing a raw third-party icon or
inventing another local mapping. The component-foundation guard rejects audited
stable-concept bypasses.

## Visualization Exception Boundaries

Chart, canvas, and WebGL exceptions only cover renderer internals such as chart marks, map pixels, canvas drawing, and runtime geometry variables. Surrounding production UI remains foundation-owned: Card shells, titles, labels, legends, tooltips, captions, metric text, borders, and readable badges MUST use theme tokens, typography roles, shared components, or theme-owned semantic classes.

Do not use `bg-white`, `text-slate-*`, `border-slate-*`, raw gradients, or hard-coded light renderer options for production-readable visualization shells. If a visualization needs theme-specific renderer values, derive them from runtime theme state or move the visual background into `frontend/styles/themes/**` as a semantic class.

## Theme Token Standards

The runtime theme model is a flat registry with exactly two entries: `lunafox-light` and `lunafox-dark`. The user-facing selection model is `system`, `light`, or `dark`, with `system` as the default and the top bar exposing a shared appearance menu. Removed palette/theme IDs and legacy `cherry-cocoa` values resolve to the default light/dark pair instead of remaining selectable. Future theme additions require a separate approved change with independent registry entries and theme stylesheets; do not reintroduce page-local palette branches.

Each runtime stylesheet owns the complete default token matrix for its appearance mode: interaction, chart, product-effect, status, trend, and severity values stay in the shared theme files. Overview risk rings, legends, charts, and globes consume those shared tokens; do not add page-local `data-theme` branches.

- All production-facing color values MUST be owned by `frontend/styles/themes/**` first. `frontend/app/globals.css` may only wire theme variables, animations, scrollbar/logo globals, and Tailwind `@theme inline` mappings from those variables; it must not become a second palette source.
- TypeScript metadata, provider brand helpers, nudge/toast styles, shell widgets, and route components MUST consume CSS variables, semantic Tailwind tokens such as `text-success` / `bg-warning/10`, or shared status/severity/trend/chart helpers. They must not store raw hex/rgb/hsl/oklch values or theme-locked Tailwind families such as `text-emerald-500`, `bg-white`, `bg-slate-100`, or `border-green-200`.
- Repeated product-specific visual tones, including provider brands, nudge tones, loader accents, splash effects, scrollbar thumbs, and logo backgrounds, belong in every runtime theme stylesheet before they are used in components.
- Media preview overlays, screenshot lightboxes, and readable text over media MUST use the runtime theme-owned `--media-overlay-background`, `--media-overlay-gradient`, and `--media-overlay-foreground` tokens instead of page-local `black` / `white` Tailwind utilities.
- The only retained raw color exceptions are bounded renderer internals recorded in `frontend/foundation-exceptions.json`, such as chart marks, canvas/WebGL drawing, terminal/log palettes, runtime CSS variable fallbacks, or approved provider brand escape hatches. Surrounding readable UI chrome still uses theme tokens.
- `check:color-foundation` scans `app`, `components`, `hooks`, and `lib`. Any new color-bearing production directory must be added to the scan roots or have a documented reason why it is not production UI.
- `lunafox-dark` MUST track the shared `neutral` dark semantic token baseline for core UI tokens: background, foreground, card, popover, primary, secondary, muted, accent, destructive, border, input, chart, sidebar surfaces, radius, spacing, and tracking. The runtime `interaction-accent` owns the global `ring`, shared filled Buttons, `sidebar-primary`, `sidebar-ring`, and route `highlight`; these interaction aliases are intentionally excluded from the neutral baseline.
- Product semantic tokens such as `--highlight`, `--success`, `--warning`, `--error`, `--info`, `--trend-positive`, `--trend-negative`, and `--trend-neutral` MAY remain LunaFox-owned extensions, but production UI should consume them through status, severity, trend, chart, or feedback helpers instead of page-local raw color utilities. Trend tokens own numeric deltas and change indicators; status tokens own operation, health, and lifecycle states.
- Dark-mode fixes should update `frontend/styles/themes/lunafox-dark.css` or shared semantic helpers before adding page-local dark-only classes.
- Do not add page-local dark palettes, parallel raw color maps, or local theme branches to compensate for theme mismatch; promote repeated needs into registry-owned theme tokens or shared semantic helpers.
- The default light/dark pair owns the full runtime token matrix, including broad surfaces such as `--background`, `--card`, and `--popover`; keep theme-specific visual identity in those shared tokens instead of adding page-local palette branches.
- The top bar theme control should use one compact shared menu trigger with mode-specific circle-half-2, sun, and moon icons. Its menu should expose `system`, `light`, and `dark` through `DropdownMenuRadioGroup` and visible labels, while keeping palette selection out of the control. Do not add a second page-local theme control.

## Shape Token Standards

Shape ownership is semantic first, not page-local utility first. Shared primitives SHOULD consume the shared radius utility classes wired from theme tokens in `frontend/app/globals.css`.

- `radius-surface`: card, list shell, tabs shell, and other framed surfaces.
- `radius-control`: button, input, select trigger, textarea, toggle, and similar controls.
- `radius-control-prominent`: emphasized joined controls, such as the main search bar, that need a slightly softer shell while preserving normal inner control geometry.
- `radius-overlay`: dialog, popover, dropdown, tooltip, and toast shells.
- `radius-none`: interior segments inside joined controls when the parent owns the outer radius and children only provide dividers.
- `radius-badge`: ordinary badges that should still follow theme corner style.
- `radius-pill`: compact chips, count badges, and explicit capsule labels.
- `radius-round`: avatar, switch thumb, radio, progress, and other intentional circles.

Do not introduce new page-local `rounded-*` geometry for shared production surfaces when one of these owners fits. If a new shape semantic is needed, extend the shared radius token surface first instead of scattering `rounded-[...]` values through route code.

## Component Size Matrix

These dimensions are the migration control plane baseline. They describe the current shared foundation contract; do not fork these values page-locally.

| Component | Standard density | Compact density | Owner / notes |
| --- | --- | --- | --- |
| Button | `default` is 36px high; `action-card` is a full-width/full-height card action with 64px minimum height | `sm` and `icon-sm` are 32px; `icon` is 36px square | `Button` owns action sizing and hover geometry. |
| Input | `size="default"` is 36px high with 12px horizontal padding | `size="sm"` is 32px high | `Input` owns text field geometry and focus ring. |
| NumberStepperInput | 32px high compact integer stepper | same | `NumberStepperInput` owns integer stepper geometry for dense configuration forms; free-form numeric/range strings remain `Input`. Its outer shell follows the default `Input` surface in light and dark themes, while embedded +/- actions stay transparent until hover so the buttons do not read darker than the value field. |
| Select | trigger `default` is 36px high | trigger `sm` is 32px high | `SelectTrigger` owns trigger density; content width follows the shared Base UI-backed `SelectContent` width contract and Base UI anchor vars. |
| Textarea | minimum 64px high | taller editors need shared editor/dialog ownership | `Textarea` owns ordinary multiline input density. |
| Badge | inline, rounded, 8px horizontal padding, 4px vertical padding | count/filter variants stay tabular and compact | `Badge` owns status/severity text density through status helpers. Muted lifecycle badges retain the standard `border-border` outline so cancelled, interrupted, and not-run states remain visible on neutral surfaces. |
| Tabs | list heights are 32px or 36px depending on variant; metric tabs own multiline 80px indicator cells | page/content compact variants stay 32px | `TabsList`, `TabsTrigger`, and `TabsCountBadge` own filter, page-nav, content, split, minimal, metric, and tab count densities. |
| Table row | headers are 40px; ordinary data rows use shared cell padding and content-driven height | dense data rows are 48px content rows, with a 49px measured box when the row border is included | `TableHead`, `TableCell`, and shared data-table components own row rhythm. |
| Card | 24px vertical rhythm and 24px horizontal content padding | stat/metric variants may add bounded decorative geometry | `Card` owns shell, section, stat, and metric container rhythm. |
| Dialog | centered panels use the shared overlay panel helper with 24px padding | narrow dialog variants must be shared or documented | `DialogContent` and `centeredOverlayPanelClassName` own modal width and spacing. |
| Form density | ordinary controls use 36px controls and 8px field gaps | dense toolbars use 32px controls | shared form/field components own label, helper, error, and control rhythm. |

## Selected Bulk Action Label Standards

Selected-row action labels are scoped by their selection surface. When a table toolbar, selected-row action bar, menu, or adjacent pagination summary already shows selected count or selected-row context, action labels SHOULD use the shortest clear verb phrase:

- Use `Delete`, `Archive`, `Export`, `Unlink`, `Mark reviewed`, or their localized equivalents such as `删除`, `归档`, `导出`, `解除关联`, `标记为已审查`.
- Do not repeat selection scope in the action label with `Bulk`, `Selected`, `批量`, or `选中` when the surrounding surface already communicates that scope.
- Keeping a compact count in a menu item is allowed when the selected count is not visible inside that same menu, for example `Delete (3)` / `删除 (3)`. The action name still remains a short verb.
- Confirmation dialogs, destructive descriptions, progress toasts, and success/error feedback MAY state the full scope and count in the description or title, for example `Confirm Bulk Delete`, `将永久删除 3 个漏洞`, or `This will permanently delete 3 vulnerabilities`. The confirmation button itself SHOULD use a short verb phrase (see below).
- Entrypoints that are not inside a selected-row context MAY use explicit labels such as `Bulk Delete` or `Delete selected` when needed to make the scope discoverable.

## Bulk Destructive Confirmation Standards

Bulk destructive confirmation dialogs should focus on the decision the user must make, not on replaying the selected table rows. When a selected-row delete, unlink, or other irreversible/high-risk action opens a confirmation dialog:

- The default dialog content SHOULD show the destructive action, the affected count, and the consequence or recovery model.
- The primary confirmation button SHOULD use a short verb phrase such as `Confirm Delete` / `确认删除` or `Confirm Unlink` / `确认解除`. Do not repeat the count or entity name on the button when the dialog description already states them.
- Do not render a default expanded list of every selected item inside the dialog. Long scroll lists make the confirmation harder to scan and duplicate context the user just selected from the table.
- Because no list is shown by default, copy MUST NOT imply one with words such as `following` or `以下`; say `Delete 3 targets` / `删除 3 个目标` instead of `Delete the following 3 targets` / `删除以下 3 个目标`.
- Item details MAY be offered only as progressive disclosure, such as a collapsed `View selected items` section, when the product has a concrete need for name-by-name verification.
- Single-item destructive confirmations MAY include the item name in the description because the dialog is not competing with a bulk selection surface.

## Form Field Ordering Standards

Create, edit, link, and configuration forms MUST keep the primary task field stable and easy to reach. The primary task field is the field or field group that directly completes the user's intent, such as a target list in an add-target drawer, a name in a create form, or a bulk input editor in an import flow.

- Required primary task fields SHOULD appear before optional association fields, metadata, tags, owners, labels, or secondary configuration unless those fields change validation, permissions, available options, or defaults for the primary task.
- Optional association fields SHOULD appear after the primary task field or inside a clearly secondary section. They must not push the primary task field, validation summary, or active editor when expanded.
- Searchable association selectors SHOULD use a stable trigger with `Popover` plus `Command` content, or an equivalent shared overlay owner, so search results do not reflow the form body.
- If an association selector must expand inline, place the expanding panel after the primary task field, constrain its height, and let the form or panel scroll instead of moving already-visible primary inputs.
- A field may be placed before the primary task only when it materially controls the following fields, for example organization-specific permissions, target taxonomy, validation rules, or defaults. Even then, the control should remain one stable row or use an overlay for long result lists.
- AI agents should choose field order by dependency first, then task importance, then layout stability. When in doubt, keep the main input stable and move optional relationships below it.

## Page Header Spacing Standards

Route-level pages with a visible `PageHeader` MUST let the page shell own the vertical gap between the header and the first content section.

- The standard header-to-content gap is 16px, represented by a shared page shell `gap-4`.
- Production route shells keep that 16px base gap on small screens, then expand to the shared outer rhythm `md:gap-6` on medium and larger breakpoints.
- `PageHeader` owns its internal title, code, divider, and description rhythm only; it MUST NOT add an outer `mb-*` margin that stacks on top of the page shell gap.
- Production pages SHOULD use the standard route shell rhythm `flex flex-col gap-4 py-4 md:gap-6 md:py-6` unless a shared shell variant owns a different density.
- Page-local `PageHeader className="mb-*"` overrides are drift and should be rejected. If a page needs a different header-to-content gap, introduce a shared page-shell density or document a bounded exception.
- Compact `PageHeader` density may tighten internal title-to-description spacing, but it must not change the outer header-to-content gap.

## Bulk Line Validation Fields

Dialogs that accept one item per line and show line-level validation, such as URL, target, subdomain, endpoint, website, directory, blacklist, or import inputs, must not rebuild the input shell locally. Use `LineNumberedTextarea` only for the low-level editor surface, and use or extend `BulkLineValidationInput` for the surrounding label, helper, empty/success/error summaries, line highlights, badges, and collapsible issue list.

`LineNumberedTextarea` keeps the native textarea as the only editing/scroll owner and virtualizes its line-number and highlight presentation rows for large inputs. Callers must pass logical line counts and issue line numbers; use `countLineNumberedTextareaLines` for presentation-only counting instead of repeatedly splitting the full value. This protects every shared caller without changing domain validator ownership.

Domain validators remain owned by their feature modules. For example, URL bulk add may keep URL-only protocol and target mismatch validation, while target creation may keep domain/IP/CIDR validation. The shared owner is the bulk line validation presentation pattern, not the domain parser.

When a new line-level issue type is needed, add it as data passed into `BulkLineValidationInput` instead of adding page-local `text-destructive` summaries, duplicate line-number shells, or route-specific error panels. Add a near-code contract test when a production dialog is migrated to this shared shell.

## Sidebar Navigation Standards

Primary app navigation inside the left shell is owned by `SidebarMenuButton` in `frontend/components/ui/sidebar.tsx`. Route modules and `AppSidebar` supply labels, icons, and active state only; they do not own the menu item's width, height, typography, or active colors.

- The shell uses stable responsive bands instead of fluid viewport-proportional navigation: expanded desktop width is `16rem` (the current application value is `256px`), the `768px` through `1279px` desktop band defaults to icon-only mode, and narrower viewports use the existing mobile sheet at `16rem`. The responsive band always owns the desktop default and is not persisted as a user preference; a manual toggle changes only the current state until the viewport crosses a responsive band. Do not add page-local sidebar widths or continuous `vw` sizing.

- Protected app-shell warmups that include the left shell SHOULD reuse the resolved `AppSidebar` and `UnifiedHeader` chrome instead of drawing a separate warmup-only sidebar or header. Keep warmup-specific differences in content, auth, and interaction gating; do not fork sidebar/header geometry between warmup and resolved shell.

- Standard desktop geometry uses `min-h-8` with `px-2.5 py-1.5`, `gap-3`, and `rounded-lg`. The menu item remains a dense shell control. Do not add page-local sidebar item height, width, or padding overrides such as `h-10`, `px-3`, or one-off rounded values.
- Task-group headings, including the Workspace group that contains Overview and Search, use the shared non-interactive `SidebarGroupLabel` at `h-8` with `px-3`. `SidebarContent` adds no inter-group gap; `SidebarGroup` owns the `py-1` vertical rhythm, creating an 8px gap between adjacent task groups. Do not hand-roll sidebar group headings with custom `div` wrappers, custom padding, or button-level hover states.
- In `AppSidebar`, Nodes, System Logs, and Database Health are direct System-group routes. Scan History, Scheduled Scan, Engine Management, and Workflow are direct Scanning routes. Tools is a task group with direct Wordlists, Fingerprints, and Nuclei POC routes; do not recreate promoted routes under secondary menus, render Runtime Status or Tools disclosures, or render Scan Plans. Support Author is a direct fixed-footer route above About and the account owner, never a System Settings child.
- Readable labels use `textRole.navLabel`, while `SidebarMenuButton` and `SidebarMenuSubButton` own the calmer unselected sidebar foreground, active/hover foreground, and tighter `leading-5` clamp. Unselected navigation text uses `text-sidebar-foreground/65`; primary leading icons use `text-sidebar-foreground/55`. Selected and hover states restore the full semantic foreground. Primary items and expanded secondary subitems use the same 32px dense shell and `rounded-lg` selection geometry. Do not replace sidebar labels with page-local `text-xs`, uppercase, custom tracking patches, or route-local color overrides to chase density or contrast.
- Leading navigation icons are 16px (`[&>svg]:size-4`) and stay optically aligned with the shared gap. If a route needs a different icon density, extend the shared primitive instead of sizing one icon locally.
- Hover, active, and pending states use semantic backgrounds and foreground roles from the shared primitive. The menu row itself keeps stable geometry, while expanded-sidebar hover may translate only its leading icon and label by 2px using the shared 160ms motion token; collapsed icon mode and reduced-motion users remain static. Do not slide a separate background layer or move the menu row. Primary items should not rely on color alone; the shared background plus the `highlight marker` provide the selected affordance. A collapsible parent whose child matches the current route stays background-transparent; `SidebarMenuSubButton` alone owns the child active surface, foreground, and right-side active dot. Secondary subitems retain the low-contrast left hierarchy rail, with an 8px gap between that rail and the rounded child background. Their selection geometry matches primary items at 32px high with `rounded-lg`, while 14px leading and 24px trailing padding keep the child label on the unchanged parent label axis and extend its background further left. The rail's top aligns to the first child background's top edge. The rail communicates nesting only.
- Navigable sidebar links render `SidebarNavigationPendingIndicator`, which uses Next.js `useLinkStatus` for immediate click acknowledgement. While one destination is pending, the shared sidebar navigation scope MUST suppress the pathname-committed item's selected background, marker, weight, and icon emphasis so the clicked destination is the only visual highlight. The former item must resolve directly to the ordinary unselected `font-medium` navigation weight; do not use `font-normal`, which creates a visible weight rebound when the route commits. Keep `data-active` and route semantics pathname-derived; do not predict the pathname or add a timer-backed optimistic route store. A successful navigation commits the new active item, while failure, cancellation, or interruption clears pending and restores the previous committed visuals automatically.
- The pending contract applies to primary links, expanded secondary links, and system settings links. Collapsed-flyout secondary links follow their ordinary close behavior and do not keep a hidden Link subtree mounted merely to preserve pending visuals. Disclosure triggers, About, and other non-route controls do not participate. The marker stays pointer-inert and decorative and MUST NOT add a spinner, reserve trailing spinner space, move the label or icon, or introduce another background geometry. The existing menu button owns the pending surface and its `rounded-lg` clipping; the activated `Link` retains normal focus behavior.
- Sidebar pending MUST NOT register a route fallback, require a `loading.tsx`, or ask the protected shell to replace the content region. After activation, the destination page or workspace decides whether its own component/query state shows a skeleton or commits ready content directly. Do not compose a shell-injected route skeleton in front of that page-owned loading state.
- On mobile, the pending indicator does not own Sheet visibility or keep a hidden Link subtree mounted. The drawer follows its ordinary close/route-commit behavior; content loading ownership remains inside the destination page or workspace.
- `SidebarContent` uses the shared Base UI-backed `ScrollArea` with persistent overlay chrome. Its vertical scrollbar remains visible whenever the sidebar can scroll, while still overlaying the content instead of reserving or releasing menu width. Keep the inner content full-width and at least viewport-height so menu geometry and the bottom `mt-auto` group remain stable.
- Collapsible parent triggers use the ordinary static `SidebarMenuButton`. The background and trailing chevron update immediately from Base UI state attributes; callers MUST NOT add a disclosure motion variant, transition classes, or page-local transform animation.
- The collapsed icon-only mode is a shared shell state. It uses the shared `size-8` square treatment, hides the inline active marker, and must not be patched per item with route-local icon or spacing overrides.
- Touch-friendly or larger sidebar densities, including mobile drawer adjustments, must be introduced as shared `SidebarMenuButton` variants. Do not add route-local `h-11`, `min-h-11`, or custom touch-target patches to a single menu item.

## Control Bar Density Standards

Control bars, filter bars, page headers, and data-table toolbars must choose one density per horizontal control group. Do not mix a 36px `TabsList` with 32px `Input`, `SelectTrigger`, or `Button` controls in the same row.

- Page-level control bars use the standard 36px density by default: `TabsList size="md"`, default `Input`, default `SelectTrigger`, `Button size="default"`, and `Button size="icon"`.
- Dense table/list toolbars may use the compact 32px density when every control in that horizontal group opts into the compact size: `TabsList size="sm"`, compact search helpers, `SelectTrigger size="sm"`, `Button size="sm"`, and `Button size="icon-sm"`.
- `UnifiedDataTable` toolbars MUST choose density through the shared `toolbarDensity` prop. Use `toolbarDensity="standard"` when a shared table surface intentionally follows the 36px page-control rhythm, and use `toolbarDensity="compact"` when the table surface should read as one 32px control system.
- Data-table-adjacent controls such as search, filters, column visibility, bulk actions, row actions, and pagination should use `toolbarDensity="compact"` by default. Move primary page CTAs to the page header when they need standard 36px emphasis.
- `DataTablePagination` is a dense table support surface. Its page-size trigger MUST use `SelectTrigger size="sm"` and pagination icon buttons MUST use `Button size="icon-sm"` instead of page-local height/width overrides.
- Short toolbar selects SHOULD keep the trigger content-fit and use `SelectContent width="content-fit"` so the popover is at least as wide as the trigger without forcing a generic `8rem` menu or page-local fixed widths such as `w-24`.
- Pagination page-size selects are the narrow exception: keep `SelectContent width="content-fit"`, but give the trigger a stable fixed width tier that can hold the widest known option in that menu plus the trigger's own icon, gap, and horizontal padding. Do not let the trigger shrink to only the currently selected value, because `10 -> 100 -> 1000` width jumps make the footer feel unstable.
- Domain-specific select menus with a stable option taxonomy, such as target-type filters, SHOULD use a shared semantic wrapper so labels and width behavior stay aligned across routes instead of duplicating `SelectItem` markup in each page.
- Target-type filters are a stable-width taxonomy control: keep `SelectContent width="content-fit"`, but give the trigger a stable fixed width tier that can hold the longest known option label so the toolbar does not shift when switching between `全部` / `域名` / `IP` / `CIDR`.
- In practice, the width strategy matrix is:
  - short contextual filter selects: trigger may use content-fit width
  - fixed taxonomy selects with a known label set: trigger uses a fixed width tier sized to the longest known label
  - pagination page-size selects: trigger uses a fixed width tier sized to the widest known numeric option
- When a responsive layout wraps controls into separate rows, each row still chooses a single density. Touch-target adjustments must be applied consistently within that row or moved into a shared responsive variant.
- Do not add page-local height overrides such as `md:h-8`, `md:size-8`, or `md:data-[size=sm]:h-8` to force one control to a different density. Extend the shared primitive or document a narrow exception instead.

## Table Row Rhythm Standards

Table row height is a table rhythm decision, not a button/control density decision.

- Data-table shells follow the shared plain table visual baseline: `rounded-md border bg-card` with no page-local accent border, shadow frame, nested card wrapper, or decorative top rule.
- `TableHead` owns header geometry at 40px (`h-10`) with `tableHeader` typography, including an explicit 16px line-height (`leading-4`).
- `Button layout="tableHeader"` owns full-column sortable headers; `layout="tableHeaderInline"` is for table-header menus whose trigger must fit only its text and state icon while retaining the standard pointer and hover affordance. While its Base UI popup is open, the inline trigger keeps the same primary tint and foreground as its ghost hover state so the active menu origin remains clear even over a secondary table header surface.
- Ordinary shared table cells use `TableCell` padding (`p-2`) and content-driven row height.
- `UnifiedDataTable` owns production business-list tables and uses the dense row rhythm: 48px data rows (`h-12`), dense body cell padding (`px-2 py-1`), and a 49px virtual-scroll estimate including the row border.
- Identity-led business lists may explicitly select `ui.rowDensity="comfortable"`; the shared table then owns 64px data rows (`h-16`), `px-2 py-2` cell padding, and a 65px virtual-scroll estimate. Do not recreate this rhythm through page-local row or cell utilities.
- Dense split-pane or master/detail tables may use 48px data rows (`h-12`) when every body cell uses the dense row cell rhythm (`px-2 py-1`) and row actions use `Button size="icon-sm"`.
- A 48px dense row with a 1px row border can measure as 49px in the browser; this is the expected rendered box.
- Cell text uses typography roles such as `tableCellPrimary` or `tableCellSecondary`, which keep dense table body copy at an explicit 20px line-height (`leading-5`); do not force text blocks to 32px or 36px.
- Severity/status badges inside table rows stay compact at 20px (`h-5`) unless a shared badge variant owns another size; shared badge text uses an explicit 16px line-height (`leading-4`).
- Dense tabs and pagination-adjacent text should stay explicit too: `textRole.tab` uses a 20px line-height (`leading-5`), `textRole.helperText` uses a 16px line-height (`leading-4`) for truly compact hints, and pagination footer copy should use `textRole.metadataLabel` / `metadataValue` / `metadataValueStrong` at a 20px line-height (`leading-5`).
- Do not store page-specific row heights in app shell variables such as `--vuln-row-h` or `--*-table-head-h`. Extend `Table`, the shared data-table primitive, or a route-owned table component with tests.
- Multiline, expandable, or nested-detail rows may exceed the dense row height, but the row must be semantically owned by that expandable/multiline component.

## Data Table Visual Standard

Production business-list pages that use `UnifiedDataTable` should treat the targets page as the baseline table pattern after migration.

- Not every table in the app must be pixel-identical, but production business-list pages SHOULD start from the shared baseline documented in `frontend/components/shared/data-table/README.md`. Route-owned deviations need a concrete reason because the baseline does not fit, not because a page happens to want a different visual mood.
- Shared implementation takes precedence over prose-only governance for business-list tables. New or migrated routes SHOULD reuse `UnifiedDataTable`, `DataTableToolbar`, `TableToolbar`, `DataTablePagination`, `SimpleSearchToolbar`, and route-level shared wrappers before adding page-local table controls, page-local pagination, or route-local width-allocation logic.
- Table-adjacent controls use compact 32px density end to end: inline-icon search inputs, filter selects, column visibility, bulk actions, add actions that remain inside the table toolbar, page-size select, and pagination icon buttons.
- Dense table/list search SHOULD default to the shared inline-icon `SearchInput` pattern through `SimpleSearchToolbar`. When a dense table toolbar keeps an explicit search submit button, the search input and icon button MUST render as separate adjacent controls through the shared search toolbar owner: same density, same visual language, and aligned height, but not a fused split control unless a future shared variant explicitly owns that pattern.
- Search fields use the wider shared text-field width tier by default; tighter single-field widths are for inline filter boxes or local search areas with their own reviewed layout constraints.
- Primary page actions that need 36px emphasis should live in the page header action slot, not inside the dense data-table toolbar.
- Toolbars are mobile-first: search stays usable through a minimum width, controls wrap into multiple rows on small screens, and right-side actions wrap instead of squeezing the search input.
- Pagination is mobile-first: selected count, rows-per-page, page label, and navigation buttons may stack into readable rows; pagination copy must not split into vertical characters or overflow the viewport.
- Row actions use one stable `MoreHorizontal` menu entry by default. Do not rely on hover-only action clusters for core actions, because touch users cannot discover them reliably.
- Optional inline affordances such as copy buttons may be hover-revealed on pointer layouts, but they must remain visible or otherwise accessible on touch layouts.
- Dense table inline affordances that only repeat the visible icon meaning, such as a copy icon beside a target name, SHOULD rely on `aria-label` plus click/success feedback instead of adding a hover tooltip. Use a tooltip only when the icon meaning is genuinely ambiguous or needs extra clarification.
- The default desktop column set for a production business-list table SHOULD fit without introducing unnecessary horizontal scrolling. Treat container-level horizontal scroll as a fallback for genuinely column-dense views, small screens, or user-driven expansion, not as the first-line overflow strategy for ordinary desktop usage.
- Tables may use intentional horizontal scrolling for many columns on mobile, but the toolbar and pagination around the table must remain viewport-readable.
- When a production business-list table has one obvious primary text column, that route SHOULD declare it as the shared `expandColumnIds` owner so extra desktop width is absorbed intentionally by that column. Keep secondary columns such as selection, status, taxonomy, date, and row actions on stable `size` / `minSize` / `maxSize` contracts instead of relying on browser auto distribution.
- Semantic-width business-list tables that rely on shared `expandColumnIds` SHOULD opt into the shared `columnLayout: "fixed"` path so cell content cannot back-drive desktop column widths. In that mode, fixed-support columns keep explicit width bands, expand columns share the remaining width, and overflow is handled inside the cell through truncation or expandable helpers instead of widening the whole table unexpectedly.
- Dense-table overflow SHOULD be resolved inside the cell before it spills into the whole table. Prefer truncation, preview budgets, count summaries, or explicit expand/collapse affordances over letting one long value force container-level horizontal scrolling for every row.
- Badge-list cells in dense tables SHOULD use a single-line preview budget by default: show at least one badge, reveal additional badges only while they still fit within the current column width budget, reserve space for the expand/collapse affordance, and keep the row height stable. Do not rely on preview-state wrapping to reveal more badges in a dense table row, and do not wait for the whole table to overflow before collapsing badge content.
- The same disclosure principle applies to other variable-width cell content, but not with the same UI treatment. Multi-value text, tags, owners, and scope summaries MAY use truncation plus expandable helpers; date/time, numeric, status, selection, and row-action columns SHOULD keep stable widths instead of adopting badge-style packing logic.
- User-driven width pressure is allowed. If the user enables more columns, expands hidden cell content, or moves to a smaller viewport, table-container horizontal scroll is acceptable as long as the default route-owned state remains stable and scan-friendly.

### Data Table Column Width Matrix

Business-list tables do not average every column across the available width. The standard is semantic width allocation: the table shell adapts to the container, one primary column absorbs extra desktop width, and the remaining columns keep stable widths that match their information density.

| Column type | Standard width strategy | Notes |
| --- | --- | --- |
| selection checkbox | fixed narrow width | Usually `size/minSize/maxSize` are the same so the column never expands. |
| primary text / name / title | shared `expandColumnIds` owner | This is the default column that absorbs spare desktop width. Keep truncation and inline affordances inside the cell instead of creating a second expanding column by accident. |
| secondary text / taxonomy / owner / scope | stable min/max band | Wide enough for ordinary values, but does not absorb all remaining space. Use truncation, badges, or expandable cell helpers for overflow. |
| date / time / numeric metrics | stable width contract | These columns should stay scan-friendly and predictable across rows; do not let them grow just because one value is longer. |
| status / severity badge | compact stable width band | Keep the badge compact and let the text column own horizontal flexibility. |
| row actions | fixed narrow width | Icon-only or single-menu action columns stay fixed and pinned when needed. |

- Do not treat equal-width columns as the default responsive behavior for admin tables. Equal distribution wastes space on checkbox/action/date columns and reduces room for the primary text column.
- Do not enable content-measurement auto sizing by default on business-list tables with mixed cell renderers. Prefer stable width contracts plus one explicit expand column so the table does not reflow when data, locale, badges, or inline actions change.
- Do not treat "show every value in every row by default" as the standard for dense business tables. The standard is to fill the available width intentionally, then apply cell-level disclosure rules before allowing the whole table to overflow.
- If a table has two genuinely primary long-text columns, choose one as the default expand owner and give the other a reviewed width band or an expandable cell pattern. Only add multiple expand columns when the route can prove the layout remains stable across desktop and tablet widths.
- On mobile and small tablet widths, keep the same semantic width hierarchy and allow intentional horizontal scrolling or a route-owned stacked pattern; do not flatten every column into equal percentages just to avoid overflow.
- `/targets/` is the baseline migration example: selection and actions are fixed, timestamps stay on a wider stable band, and target name plus organization are the paired expand columns.
- If a new business-list route cannot follow this baseline, document the reason in the nearest local README or contract test and extend the shared layer when the pattern will repeat.

## Filter Tabs Active-Indicator Standards

`TabsList variant="filter"` owns one shared active background. The background follows the active trigger by translating and resizing with the shared fast duration and standard easing; callers MUST NOT add a second selected-background, page-local indicator, or route-level tab animation.

The shared owner measures active trigger bounds so long localized labels, count changes, and rail resizing remain aligned. It is pointer-inert, keeps Base UI `data-active` as the selection source, and disables travel under reduced motion. This rule applies only to `filter`; `content`, `page-nav`, `minimal`, `split`, and `metric` keep their existing selection models.

When a filter Tab initiates nested-route navigation, the list moves its indicator immediately and remains mounted. The clicked indicator stays in place until that trigger becomes the committed Base UI `data-active` tab; it MUST NOT use an elapsed-time fallback that can return the indicator to the previous route while navigation is still pending. Route content loading and replacement belong only to the child content region; callers MUST NOT defer the indicator until that work completes or animate the whole route horizontally.

## Tabs Underline Standards

`TabsList variant="content"` owns the content-tab bottom rule. Its triggers align to the bottom edge and use the shared active border to overlap that rule.
Content-tab rails keep `min-w-0`, `max-w-full`, and horizontal overflow ownership in the shared primitive so long localized labels scroll inside the rail instead of widening detail pages or drawers.
Do not add page-local pseudo-elements, duplicate `border-b` rails, negative offsets, arbitrary heights, or `bottom-[...]` guide lines to make content tabs appear aligned.
When a content-tab rail needs different density or underline behavior, extend `TabsList` / `TabsTrigger` in `frontend/components/ui/tabs.tsx` and update `tabs.contract.test.ts` before migrating callers.

`TabsTrigger variant="minimal"` uses the trigger-width underline by default. Compact secondary tabs may opt into `activeIndicator="fixed"`, which centers a fixed 8px underline directly beneath the label rather than at the rail edge. Use `size="md"` when the compact rail needs a 36px control height; `size="sm"` remains 32px.

`TabsList variant="metric"` owns multiline metric selectors such as dashboard resource indicators. Metric tabs use vertical separators and subtle active background only; they MUST NOT add content-tab underline rails, active bottom borders, or page-local absolute indicator lines.

## Overlay And Layer Standards

| Surface | Width tier | z-index | Required behavior |
| --- | --- | --- | --- |
| Dialog / AlertDialog | centered panel, `max-width: calc(100% - 2rem)` unless a shared variant narrows it | `z-50` | use shared overlay backdrop, panel helper, focus trap, title, description, and close affordance. |
| Sheet / Drawer | edge panel, mobile-first width, `sm:max-w-sm` for side sheets unless a shared variant owns another tier | `z-50` | use shared edge overlay helper and preserve keyboard/focus behavior. |
| Popover / Dropdown / HoverCard | content follows trigger or content width tier owned by the primitive | `z-50` | use the owning primitive positioning vars; arbitrary widths need ledger or shared variants. |
| Tooltip | content is fit-content with concise text | `z-50` | tooltips describe controls; they must not carry primary task content. |
| Toast | toast width and placement are owned by `Toaster` / feedback helpers | component-owned | page-local toast width or placement is not a migration path. |

Layering must use shared overlay helpers or primitives. Do not add page-local z-index ladders. New overlay width tier or z-index ownership requires a contract test and README update.

Shared overlay owners live under `frontend/components/shared`. Route code SHOULD use the shared owner before raw primitives:

- Use `FormDrawer` from `frontend/components/shared/form-drawer` for create/edit/link/configuration drawers with a fixed footer.
- Use `DetailDrawer` from `frontend/components/shared/detail-drawer` for read-only details, findings, runtime inspection, and diagnostic drawers.
- Shared right-side panel headers use `frontend/components/shared/edge-panel-header` through explicit `workbench`, `form`, `detail`, or `compact` variants. The variant owns density and slot boundaries; it does not make all side panels the same size. Agent logs, workflow editors, media/code viewers, the app navigation sidebar, and centered dialogs remain outside this header foundation unless a named owner explicitly adopts a variant.
- Raw `Drawer` uses the shared Base UI-backed primitive. Production call sites MUST use `swipeDirection`, `showSwipeHandle`, and `render` composition; do not reintroduce Vaul-only `direction`, `asChild`, `data-vaul-drawer-*`, or Vaul focus props.
- Raw `Select` uses the shared Base UI-backed primitive. Production call sites MUST import from `@/components/ui/select`; do not import raw `@base-ui/react/select` or `@radix-ui/react-select`, and do not rely on Radix-only `--radix-select-*` geometry variables. `SelectContent` defaults to the current item-aligned popup style, where the selected item aligns with the trigger. Item-aligned Select popups MUST portal to the owning overlay event layer, such as Drawer viewport or Dialog/Sheet portal, because Base UI uses fixed positioning for that mode; putting that fixed positioner inside transformed drawers, dialogs, or sheets can shift the overlay's flex layout, while portaling outside a modal overlay can make options unclickable. Use `position="popper"` only when a surface intentionally needs the older trigger-anchored dropdown behavior.
- Default modal backdrops follow the shadcn `base-nova` overlay baseline: `--overlay-backdrop-background` plus `backdrop-blur-xs` through shared overlay helpers. Drawer retains only this light shared blur as a static depth cue; do not reintroduce `backdrop-blur-sm` or caller-level blur variants.
- Sheet and Drawer edge-panel motion MUST come from the shared owners in `frontend/lib/ui/overlay-styles.ts`, using Base UI `data-starting-style` / `data-ending-style` hooks for panel and backdrop transitions. The panel transitions only `transform`; the backdrop transitions only `opacity`; `filter` remains static and outside the transition list; ordinary timing uses the shared shell duration and standard easing. When a controlled Sheet or Drawer first mounts after dynamic loading with `open=true`, the shared primitive MUST commit its closed state and open on the next animation frame so the directional transition is preserved; reduced-motion users resolve directly to the final state. Ordinary Sheet and Drawer panel shells use `bg-card text-card-foreground`; explicitly branded shared shells such as the mobile sidebar may retain their owned surface. `Drawer` retains swipe semantics, while `Sheet` retains dialog semantics; callers must not disable directional motion, delay visible panel content, or add local panel/backdrop variants.
- `SheetContent` owns a visible, localized close icon by default. Production callers MUST NOT set `showCloseButton={false}` unless the same rendered surface provides one equivalent visible and accessible close control. The sole reviewed exception is `NotificationDrawer` in `components/notifications/notification-drawer-sections.tsx`: it hides the default icon by product decision and retains standard backdrop and Escape dismissal. Shared wrappers with behavior-aware header controls must opt out explicitly so the drawer exposes exactly one close owner.
- Use raw `SheetContent` in route modules only when no shared owner fits, and document the exception in the nearest README or a focused contract test.

See `frontend/components/shared/README.md` for the shared component decision matrix and component-specific README files for drawer contracts.

- Short single-column action menus SHOULD use `DropdownMenuContent width="content-fit"` so the menu follows its longest action label, stays at least as wide as its trigger, and avoids page-local fixed widths such as `w-48`.
- Keep the default `min-w-[8rem]` width for generic menus, mixed-content menus, or menus that need a stable baseline across variable copy.
- Non-table dropdown families SHOULD route through shared owners in `frontend/components/shared/dropdown-menu-owners.tsx` instead of page-local `DropdownMenuTrigger` / `DropdownMenuContent` markup.
- `HeaderIconActionMenu` owns compact global or header icon-trigger menus and reserves `width="content-fit"` as the default policy for short single-column actions.
- `SidebarUserMenu` owns the sidebar account menu trigger, mixed-content profile label, desktop/mobile side placement, and the default-width content shell for account actions. Sidebar account loading states MUST use the same owner through `SidebarUserMenuSkeleton` / `NavUserSkeleton`; do not hand-roll a local avatar-plus-lines footer skeleton in `AppSidebar`.
- Generic `DropdownMenuContent` and `PopoverContent` retain their `4px` anchor gap; this is the trigger-to-popup distance, not a Shell-edge clearance. Authenticated header overlays MUST use `shellOverlaySideOffsets.header` directly or through `HeaderIconActionMenu`, while desktop sidebar account menus and collapsed navigation flyouts MUST share `shellOverlaySideOffsets.sidebarDesktop`. These shared offsets preserve approximately `4px` of visible header clearance and the same visible clearance outside the sidebar; mobile account menus keep the generic compact gap.

## Tabs Count Standards

Tab label counts MUST use `TabsCountBadge` from `@/components/ui/tabs`.
Do not place page-local `Badge` geometry, count variants, radius, height, min-width, or padding classes inside `TabsTrigger` labels.
Use ordinary `Badge` for business tags and status chips outside tab labels.
When a new tab variant needs different count density, extend `TabsCountBadge` or the shared tabs primitive before adding page-local classes.

## Feedback State Standards

| State | Required pattern |
| --- | --- |
| Empty | use shared or route-local empty state sections with `textRole` text and a shared `Button` action when actionable. |
| Error | include an accessible message and retry action when recovery is possible; use shared error helpers where available. |
| Loading | use shared loading or skeleton primitives; avoid page-local spinners when a skeleton pattern exists. |
| Skeleton | geometry should mirror the final component through shared skeleton primitives; one-off skeleton dimensions need ledger tracking. |
| Toast | use toast helpers for success, error, loading, and warning copy; do not create page-local toast systems. |
| Tooltip | tooltip text names or clarifies the control; it must not be the only accessible name. |
| Retry | retry actions use shared `Button` sizing and must keep stable geometry. |

### Toast Standards

Toast feedback is a shared feedback primitive, not a page-local notification system.

- Only `frontend/components/ui/sonner.tsx` and `frontend/lib/toast-helpers.ts` may import `sonner` directly.
- Production call sites MUST use `useToastMessages` or `toastFeedback`.
- Use `useToastMessages` when the caller owns i18n message keys and params.
- Use `toastFeedback` when the caller already has resolved display copy, custom toast content, or needs to dismiss a known toast id.
- Repeated user actions such as copy, reset, export, refresh, or singleton nudges MUST pass a stable toast id so the existing toast is updated instead of stacking duplicates.
- One logical async operation MUST use one stable toast id from loading through success, warning, or error. Use domain-qualified ids such as `delete-organization-${id}` so unrelated resources cannot collide in Sonner's global id space.
- Terminal feedback MUST replace the loading toast in place with the same id. Do not dismiss loading and then create an unrelated success, warning, or error toast; that produces a visual gap and leaves lifecycle ownership ambiguous.
- Remote mutation loading MUST remain active through hook-owned query invalidation or refetch work required to reflect success. Replace it only after that refresh completes. Error callbacks may replace loading after their rollback or required recovery work completes.
- Dismiss loading only for cancellation, silent completion, intentionally skipped terminal feedback, or exception cleanup. A stale loading message MUST NOT remain underneath its own terminal result.
- Collapsed stacking is reserved for distinct operations with distinct ids. It is a presentation rule for concurrent feedback, not a history mechanism for multiple states of one operation.
- The owner that creates remote-command loading feedback MUST also own terminal replacement. Hooks and components MUST NOT split one remote operation so a hook dismisses loading before a component creates terminal feedback after `mutateAsync`.
- Multiple toast messages MUST remain readable without taking over the workspace. The global `Toaster` collapses stacked messages until the user interacts with the stack and limits visible messages to three.
- Toasts SHOULD stay visually neutral: semantic state is carried by the toast type icon and copy, while the shell uses `--card`, `--card-foreground`, `--border`, and `--radius-overlay`. Do not enable rich all-card success/error colors for ordinary product feedback.
- The global `Toaster` follows the active LunaFox light/dark theme instead of OS-only system theme or a hard-coded light theme.
- Default transient toasts use a 4000ms lifetime. Loading toasts and long-running async operations use a stable id and are replaced by terminal feedback or dismissed by the operation owner under the rules above.
- Toast placement, maximum visible count, duration, and spacing are global Toaster ownership. Page-local call sites must not tune them unless a reviewed exception documents why.
- Toasts are non-blocking feedback only. Errors that require correction MUST also appear in the page, form, dialog, or table state where the user can act on them.

### Loading Hierarchy

Loading feedback is selected by intent, not by whichever animation is closest at the call site:

| Intent | Approved entrypoint |
| --- | --- |
| Initial table, list, card grid, detail, settings, or page-section load | `Skeleton` from `@/components/ui/skeleton`, `DataTableSkeleton`, `CardGridSkeleton`, `MasterDetailSkeleton`, `PageSectionSkeleton`, `SettingsPageSkeleton`, or another reviewed skeleton owner under `@/components/shared/loading/**`. Shared skeleton templates receive an `owner` when they are the standalone loading owner; when they are passed into `ContentHandoff`, the handoff owns loading metadata and the skeleton omits its own owner. |
| Resolved structured content handoff after skeleton or warmup | `ContentHandoff` from `@/components/shared/loading/content-handoff` or another approved shared handoff owner. The standard local-section handoff is `skeleton-only -> overlap handoff -> content-only`, with a short shared opacity-first overlap and no large translation. Complete route/detail shells may use the bounded `transitionMode="replace"` path when overlap would expose real controls underneath unfinished route chrome. |
| Resolved structured content reveal without a paired skeleton handoff | `ContentReveal` from `@/components/shared/loading/content-reveal` or another approved shared low-level reveal owner. `ContentReveal` is not the preferred owner for skeleton replacement when a shared handoff owner is available. |
| Compact pending action in a button, menu, toolbar, dialog, or inline refresh | Shared `Button` loading props or `Spinner` from `@/components/shared/loading/spinner`. |
| Toast loading | `Toaster` loading icon contract and `@/lib/toast-helpers` / resource mutation feedback helpers. |
| Navigation progress | `RouteProgress` and route-progress helpers from `@/components/route-progress`; direct `nprogress` imports are limited to that adapter and its tests. |
| Public login/auth-entry warmup | Initial public auth entry is owned by the server boot layer plus route-level `data-boot-handoff-pending`; login content clears that blocker only after its first stable visual frame. Do not insert a second visible auth warmup card between boot and the login surface. Later non-initial auth pending exceptions must use a reviewed shared auth loading owner with required `intent` or `owner` props. |
| Protected app shell warmup while the shell cannot render | `AppShellWarmup` or an app-shell-oriented `AppWarmupLoader` from `@/components/shared/loading/**`. Protected app warmup MUST use console shell geometry, not login-form skeleton geometry, and SHOULD stay hidden during very brief auth or hydration waits. Shared implementations default to a short delayed display threshold of about 120ms so users do not perceive a second loading scene for fast resolves. The content region MUST stay generic and high-level: stable shell chrome with an empty content region by default, with neutral title lines allowed only as a restrained exception. It MUST NOT introduce route-specific card grids, fake tables, guessed business layouts, large pseudo-content canvases, or pseudo action/control placeholders by default. |
| Live scan, health, stream, heartbeat, or runtime activity indicators | Domain status helpers or a narrow reviewed exception; these are not data-loading placeholders. |

`LoadingState`, `LoadingSpinner`, `LoadingOverlay`, and `ShieldLoader` have been removed. Do not restore standalone legacy loading wrapper entrypoints outside a reviewed loading owner.

Do not add page-local border spinners, direct loading icon animation, local pulse placeholders, page-local reveal animation, or manual route-progress events on ordinary same-origin `Link` clicks. Programmatic navigation that should show progress uses the LunaFox route-progress helper instead of importing the route-progress engine directly.
When customizing the shared `RouteProgress` template, preserve the structural selector contract required by the underlying `nprogress` engine. The bar node currently needs `role="bar"` so the adapter can find and style it; treat that token as adapter-internal structure, keep the node `aria-hidden`, and do not remove it unless the adapter also changes the matching selector contract.

Progressive loading boundary decisions are near-code owned by
`frontend/components/shared/loading/README.md` and locked by
`frontend/components/shared/loading/__tests__/page-loading-handoff-audit.contract.test.ts`.
Use that guide before changing route chunk fallbacks, `ContentHandoff`
readiness, route-critical direct imports, or hidden readiness handoffs.
The standard is layered loading, not same-layer staged reveal: shell warmup may
hand off to a route skeleton, and a route skeleton may hand off to section
content, but a single first-screen page must not visibly reveal title, tabs,
and content as separate same-layer loaders. A visible `lazyPage` or
`next/dynamic` fallback is a loading owner; when the route or component already
owns the first-screen skeleton, the chunk fallback should be invisible or the
route-critical content should be imported directly.

### Baseline Standard

- `Baseline Standard` is the minimum shared loading model every structured production surface should meet.
- The baseline sequence is `shell warmup -> route/page skeleton -> content handoff -> content-only`.
- The baseline handoff model is `skeleton-only -> overlap handoff -> content-only`.
- Complete route/detail shells are the narrow replacement exception: use `transitionMode="replace"` when the skeleton represents the entire route scene and any overlap would create a visibly mixed skeleton/real frame. Local section owners keep the overlap default.
- The baseline uses one loading owner per visual layer, opacity-first overlap, and reduced-motion-safe fallback.
- For `ContentHandoff`, the shared overlap MAY fade the skeleton out, but the resolved `.loading-handoff__content` container MUST NOT run a handoff enter animation; Chrome can count that container animation as CLS even when measured geometry is stable.
- Loading surfaces MUST NOT add a static entry hold after content is ready. A short delayed display threshold before showing loading is allowed, and a short opacity-first skeleton exit is allowed, but a visible loader that pauses before motion or waits after readiness reads as jank.
- Server-rendered boot or warmup loaders that are visible before hydration SHOULD start approved CSS motion from critical CSS itself. Do not gate first motion on a client effect adding an animation class unless there is a measured reason and a documented exception.
- Global boot loaders MUST be first-paint friendly: run the minimal theme bootstrap before the visible boot layer, keep boot CSS inline and critical, and do not put remote font imports, non-critical package CSS such as editor/flow/terminal styles, or other render-blocking cosmetic resources ahead of the boot layer.
- Boot loader colors should come from theme-owned tokens with safe fallbacks. The global LunaFox boot layer uses `--background` for the surface and `--brand-mark-foreground` with a `--foreground` fallback for the animated mark.
- Repeating loader motion SHOULD prefer compositor-friendly `transform` / `opacity` animation. Avoid animating `background-size`, layout dimensions, filters, or large paint-heavy properties for ordinary boot, warmup, skeleton, and pending indicators unless a bounded exception documents why.

### ContentHandoff Readiness Standard

`ContentHandoff` decides visibility; it must not accidentally become the thing that prevents content readiness from happening.

- If `isLoading` is driven by data owned outside the children, keep the default behavior: children stay unmounted during the `loading` phase, then mount during `handoff` / `content`.
- If `isLoading` is driven by a readiness signal emitted from inside the children, such as a `next/dynamic` child calling `onReady`, the caller MUST pass `mountContentWhileLoading`.
- In the readiness-signal case, the dynamic import fallback SHOULD stay `null` so `ContentHandoff` remains the single visible loading owner.
- If readiness is owned outside the children but the resolved child is lazy or dynamic, `ContentHandoff` MUST keep the outgoing skeleton visible until committed content has positive natural geometry. A loaded module, settled query, mounted-but-empty wrapper, or zero-height shell is not enough to start the skeleton exit animation.
- `mountContentWhileLoading` only mounts hidden content for readiness; it must not become a second geometry owner during the `loading` phase. Hidden mounted content MUST stay visually hidden and out of the visible loading Grid track until handoff begins.
- Hidden readiness content MUST NOT write late measurements, `min-height`, or any other dynamic geometry back to the visible owner or skeleton. A late data commit that would change the first visible frame means readiness is premature or the shared skeleton/layout contract is wrong; do not grow the visible skeleton to absorb it.
- If `prepareContentBeforeHandoff` is used and hidden content can materialize its real subtree after the initial mount, `ContentHandoff` SHOULD continue observing it only to delay readiness until its first frame is stable, never to resize the visible loading surface.
- `prepareContentBeforeHandoff` is a narrow opt-in for a data-dependent first resolved frame. It keeps the resolved branch hidden and waits for two stable animation frames before crossfade or replacement. It MUST NOT become the default response to a skeleton/layout mismatch or reserve a measured external height.
- The prepared path is a readiness gate only. Stable content that is taller is a real geometry mismatch: fix the shared skeleton/layout owner and let ordinary intrinsic smoke comparison fail until it is resolved. The sole narrow exception is a sparse shared paginated table with an explicit, route-owned `resolvedIntrinsicTableSettlement`: its owner, surface, and named body may become shorter, and its named natural-flow pagination may move upward. The contract remains strict for owner top/left/width, toolbar geometry, and every other slot; do not use this exception for `prepareContentBeforeHandoff`, page-local `min-h-*`, or a broad tolerance.
- Never wire a loader so that `isLoading` prevents the child from mounting while the same child is responsible for clearing `isLoading`; that creates a permanent skeleton/loading deadlock.
- `mountContentWhileLoading` is not a shortcut for showing two loading scenes. The mounted child remains `aria-hidden` and visually hidden until the handoff phase.
- `ContentHandoff` owns a paired `data-loading-slot="surface"` wrapper. When a feature has a geometry-critical first-screen region inside that wrapper, both skeleton and resolved branches MUST expose one matching `data-loading-slot` for the region. Use semantic names such as `header`, `toolbar`, `primary-content`, `value-band`, or `pagination`; do not reuse generic `data-slot` selectors and do not add a nested `surface` marker.
- Route loading geometry is verified from a cold browser timeline. A route contract must declare required slots and tolerances for its expected owners; changes to either the resolved layout or its skeleton must update the same shared layout owner and contract in one change.

### Polish Standard

- `Polish Standard` is the follow-up layer for surfaces that are already baseline-compliant but still feel mechanically staged.
- Polish work SHOULD first reduce visible scene cut frequency, especially unnecessary `AppShellWarmup` exposure.
- Polish work SHOULD then improve structure continuity so skeleton surfaces feel like the same container being taken over by resolved content.
- If a loader feels like it appears twice or freezes for a beat, first check for duplicate owners and hydration-gated animation startup before adding new animation. Standard product loading should feel continuous: loading motion, readiness, skeleton exit, content-only.
- Stronger geometric continuity, shared-element treatment, or View-Transition-style motion is a higher-order option and SHOULD NOT be the first response to ordinary stiffness.

### Skeleton Geometry Standards

Skeletons are for initial data loading only. They must mirror the final content closely enough that replacing them with real data does not create an observable layout shift.

- Match the final surface's major regions: page sections, cards, table rows, toolbar slots, tab rails, detail panes, metric blocks, and chart viewport.
- Repeated table rows, card grids, master/detail pages, and route shells SHOULD use shared skeleton templates under `@/components/shared/loading/**`.
- Tables and business-list workspaces with a shared resolved owner MUST derive loading geometry from that owner when available. For `UnifiedDataTable`, `BusinessListDataTable`, and route wrappers built on them, use the shared loading mode instead of route-local skeletons that copy toolbar buttons, table headers, dense row heights, column layout, or pagination.
- A one-off skeleton is allowed only when the geometry is genuinely local; repeated one-off dimensions must become a shared template or a `skeleton-geometry` ledger entry.
- Shared loading templates and shared reveal owners MUST receive their required `owner` or `intent`; missing configuration is a contract failure, not a reason to silently fall back to a spinner or full-screen loader.
- Initial skeleton action controls SHOULD stay structural. If a toolbar, header, or footer needs an action slot for geometry continuity, use a shared action placeholder block such as `ActionSkeleton` instead of simulating internal label, icon, chevron, counter, or spinner detail.
- Initial skeleton action controls SHOULD stay low-fidelity even when the resolved surface contains a primary CTA. Do not restyle loading placeholders into branded filled buttons just to mirror the final emphasis.
- Initial skeleton input, search, select, and page-size shells SHOULD reuse shared control geometry through approved loading owners such as `SearchToolbarSkeleton`, `SelectShellSkeleton`, or `CompactPaginationSkeleton`, or through a real shared control rendered in a disabled shell form.
- Initial skeleton search, select, and page-size shells SHOULD stay text-first and low-fidelity. Do not add decorative leading icon placeholders inside ordinary loading control shells.
- If a resolved search or select control owns a fixed leading slot that shifts the value start position, the skeleton SHOULD preserve that spacing through a neutral leading spacer instead of rendering a visible icon-shaped placeholder.
- Tabs in initial skeletons SHOULD prefer real `TabsList` / `TabsTrigger disabled` geometry over page-local faux tab rails.
- If an initial action slot is not necessary for geometry continuity, omit it rather than guessing fake control content.
- Do not use skeletons for compact pending actions. Use shared `Button` loading props or `Spinner` instead.
- Once a real control exists and the user triggers an operation, the pending state MUST stay on the real control through shared `Button` loading or shared `Spinner`; do not swap back to an initial skeleton-style action placeholder.
- Do not rebuild route-local lookalike control shells by restating local `border`, `background`, `radius`, `shadow`, and inner placeholder detail when a shared loading control-shell owner already exists.
- Do not keep the global app loader mounted until page data, charts, dialogs, or low-priority components finish. App/auth boot hands off to route skeletons and local pending states.
- Do not add page-local `animate-pulse`, shimmer, or local reveal snippets that duplicate `Skeleton`, `ContentHandoff`, `ContentReveal`, or shared skeleton templates.
- `AppShellWarmup` is not a route skeleton. Its content region SHOULD remain intentionally generic and chrome-only by default so it can hand off cleanly to `PageSectionSkeleton`, `DataTableSkeleton`, `SettingsPageSkeleton`, or another route-owned skeleton once the protected shell is ready. Prefer an empty content region by default; if restrained title lines are needed, use them sparingly instead of placeholder buttons, tags, cards, or canvas-like blocks.
- When a protected warmup can mount the resolved shell chrome safely, it SHOULD reuse that chrome and geometry directly rather than building a parallel skeleton shell. The sidebar, top bar, and shell variables should come from the same shared layout path as the resolved app shell; only the content layer should diverge.
- Dashboard and overview routes should keep the same standard route shell rhythm as the rest of the app. Any dashboard-specific skeleton geometry SHOULD stay inside section loaders or shared skeleton owners, not in a second page-level shell.
- Loading progression is layered, not staged within the same layer. The default order is `shell warmup -> route/page skeleton -> content handoff -> content-only`.
- Structured loading surfaces SHOULD keep one loading owner per visual layer. Do not let dynamic fallback, component-level data fallback, and local reveal wrappers compete within the same visible layer.
- A `next/dynamic` fallback skeleton is a visible loading owner. If a table, section, or page already has an explicit data skeleton or is wrapped by `ContentHandoff`, use a static import, `loading: () => null`, or hidden `mountContentWhileLoading` readiness instead of adding another skeleton fallback for the same layer.
- When only one section of a structured page changes, prefer a section-level `ContentHandoff` owner over a single page-wide handoff so geometry stays stable and the reveal feels continuous.
- If resolved content must mount before it can report readiness, for example a `next/dynamic` child that calls `onReady`, use `ContentHandoff mountContentWhileLoading` so hidden content can mount while the skeleton remains the only visible loading owner.
- When a dashboard section already has a stable local layout, extract that geometry into local layout owners and have both the resolved section and the section skeleton consume those same owners. `/overview/` is the baseline reference; see `frontend/components/overview/README.md`.
- When a section skeleton and resolved section still differ after sharing the same layout owner, treat the remaining mismatch as a shared geometry bug first, not a skeleton-only bug. Fix the shared owner, then make the smallest local skeleton adjustment that preserves the same rhythm scale.
- Do not split container-query ownership between the resolved section and the skeleton wrapper. If a container query affects panel splits, keep that query on the shared owner that both states render through.
- The global `loading-skeleton` class owns its own base geometry, including `position: relative`, overflow clipping, background, and the shimmer pseudo-element. Do not treat `loading-skeleton` as a generic positioning utility; prefer the shared `Skeleton` component, or make inline/absolute metric placeholders explicitly block-level and position-owned when a real text line box must be preserved.
- If a page still appears to shift during handoff after geometry matches, inspect motion keyframes before changing layout. For overview, the first suspects are `overview-reveal`, `overview-number-swap`, and `overview-ring-enter`.
- A loader `ready` signal must mean the first visible content frame is geometrically stable. Do not flip `isLoading` just because a dynamic module mounted if that module still renders its own placeholder rows, numbers, or chart shells.
- If the visible handoff is owned by a route skeleton and the mounted child still contains section-local loading placeholders, keep the route skeleton active until those first-screen queries settle; otherwise the user will perceive a second layout swap after handoff.
- Do not sequence shell chrome as `logo -> sidebar -> buttons -> title -> content` or similar staged reveal choreography. When the protected shell becomes visible, its same-layer placeholders SHOULD appear together.
- When resolved content replaces a skeleton or warmup, prefer a shared overlap handoff over a hard replacement. The default handoff should feel like content quietly taking over the same structure, not popping in or waiting for the old skeleton to disappear first.
- Keep handoff motion on the outgoing skeleton or inner non-layout visual details. Do not animate the resolved `ContentHandoff` content wrapper during the handoff; if content needs polish, use stable inner opacity-only owners after the wrapper is already in place.

### Preloading Standards

Preloading is a bounded performance optimization, not a loading-state replacement. Do not globally prefetch every route, data query, or heavy component to hide slow rendering.

| Layer | Standard |
| --- | --- |
| Route code and RSC payloads | Prefer Next.js `Link` automatic prefetch. Manual `router.prefetch` is allowed only for high-probability destinations such as login-to-overview, route shells owned by `useRoutePrefetch`, or explicit user intent such as hover/focus. It must be idle or intent-driven, bounded by named budgets, and network-aware through `saveData` / slow-connection checks. |
| Data | Use React Query `prefetchQuery` only for low-cost, high-probability data that the next route will need. Do not change query keys, stale/cache semantics, mock/service routing, fetch timing rules, or data availability rules to make preloading appear faster. |
| Heavy components | Use `next/dynamic` to split low-frequency editors, dialogs, complex tables, charts, and tool pages. Do not globally preload these chunks; add interaction, viewport, or idle preloading only when the next action is clear and the cost is justified. |
| Critical shell | Do not use preloading to mask layout shift. Route shells, sidebar, header, and stable page geometry should render predictably; data waits belong to skeletons or local pending states. |

New manual route prefetching must update `hooks/use-route-prefetch.ts` or a route-owned must-hit flow plus its contract tests. Avoid page-local one-off `router.prefetch` calls for low-probability navigation.

## Responsive Implementation Standards

Every production route, page section, dialog, table, toolbar, form, chart, and shell change must be designed mobile-first, then expanded for tablet, desktop, and large desktop contexts. Responsive work is not complete when it only looks correct at the current browser width.

| Context | Standard |
| --- | --- |
| phone / small mobile, 320px-767px | Start with a single-column layout, stable vertical rhythm, no clipped controls, and touch targets large enough for touch use. Dense toolbars should wrap or collapse into shared menus/sheets. Tables must either become a readable stacked/list pattern or provide intentional horizontal scrolling with preserved labels; do not let columns overflow invisibly. |
| tablet, 768px-1023px | Use hybrid layouts: two columns, master/detail, side panels, or stacked sections depending on content. Support both touch and pointer input; hover-only affordances need visible touch alternatives. |
| desktop, 1024px+ | Use available width for scan-friendly density: persistent navigation, side-by-side panels, richer tables, and multi-column forms when useful. Keep max-width and grid constraints so content does not stretch without purpose. |
| large desktop / high resolution | Add max-width, container, or grid constraints before content becomes too wide to scan. Important panels may expand, but text lines, tables, charts, and cards must keep readable measures and stable geometry. |

The shared protected application content frame owns the wide-screen canvas. It MUST use the width available after shell chrome and MUST NOT reintroduce a fixed screen-level maximum width. Route-level wrappers may constrain only their own long-form copy, error state, reader, editor, or dialog; they must not cap the primary workspace again.

Prefer container queries and content-driven breakpoints for reusable components; viewport breakpoints are acceptable for route shells and page-level composition. Do not hide critical functionality on mobile or tablet to make a layout fit. Long localized text, large numbers, loading/skeleton states, empty/error states, and permission-limited states must be checked at each responsive tier.

## Responsive Verification Standards

- Route shells, navigation, tables, dialogs, forms, and overlays changed by migration need browser evidence when static checks cannot prove behavior.
- Use `test:e2e:routes` for route shell, navigation, provider, or page composition changes.
- Use `test:e2e:interaction` for forms, dialogs, tables, overlays, navigation, or core interaction changes.
- Fixed dimensions that can break mobile or long localized text must move into shared responsive contracts or be recorded as bounded runtime geometry.
- Migration batches must state when browser smoke is skipped and why.

## Motion Scale

Production motion uses a small shared timing scale. Reuse these buckets before introducing new durations or easing:

| Motion tier | Default duration | Default easing | Standard use |
| --- | --- | --- | --- |
| Fast | `160ms` | `cubic-bezier(0.22, 1, 0.36, 1)` | Hover/focus/selection transitions and compact state changes. |
| Reveal | `180ms` | `cubic-bezier(0.22, 1, 0.36, 1)` | `ContentHandoff`, `ContentReveal`, subtle opacity-first content replacement, and restrained loading handoff. |
| Enter | `220ms` | `cubic-bezier(0.22, 1, 0.36, 1)` | Small enter/swap motion such as metric or value replacement. |
| Shell | `280ms` | `cubic-bezier(0.22, 1, 0.36, 1)` | Protected app shell fade-in after boot warmup exits. |
| Visualization exception | `420ms-520ms` | `cubic-bezier(0.22, 1, 0.36, 1)` | Approved dashboard/chart/visualization reveals only. |

- Loading and reveal motion should be opacity-first by default.
- The top-bar appearance menu uses the shared compact dropdown owner and radio-item close behavior. Theme surfaces still switch immediately when the selected mode changes.
- Ordinary product UI should reuse the shared standard easing instead of introducing page-local curves.
- `linear` is reserved for continuous machine-like motion such as spinners, shimmer, and approved runtime status sweeps.
- Infinite motion belongs only to shared loading/status owners or documented exceptions.
- Shell, sidebar, progress, and status indicator motion MUST avoid animating layout-bound properties such as `width`, `height`, `left`, `right`, `top`, `bottom`, `margin`, `padding`, `max-width`, and `max-height`. Prefer compositor-friendly `transform` and `opacity`; progress fills should use `scaleX()` with left-origin geometry.
- When layout occupancy must change, commit the layout at the state boundary and keep visual polish on inner compositor-friendly layers. Do not hide layout work inside broad `transition-all` utilities.
- Performance claims require same-scope before/after evidence. Static risk reduction, contract coverage, and visual inspection support the claim, but are not substitutes for runtime measurement when the change targets animation or render performance.
- If a new motion pattern does not fit these tiers, extend the shared motion owners and update this table before shipping page-local values.

## Exception Ledger

The unified ledger lives at `frontend/foundation-exceptions.json`. Every retained exception MUST include:

- `status`: `keep`, `narrow`, `remove`, `deferred`, or `excluded`.
- `scope`: file path or glob.
- `pattern`: the bounded value or pattern.
- `class`: approved exception class.
- `owner`, `reason`, `reviewTrigger`, and `recoveryPath`.

Approved classes are `theme-token-owner`, `typography-role-owner`, `chart-rendering`, `terminal-log`, `canvas-webgl`, `skeleton-geometry`, `radix-dynamic-var`, `runtime-calculated-geometry`, `brand-provider-color`, `visual-lab-excluded`, `accessibility-only`, and `legacy-migration`.

The readable baseline index lives at `frontend/foundation-inventory.md`. Verify mode should block unapproved new drift. Inventory mode may report historical debt for later approved cleanup batches.

### Ledger Operating Rules

- Treat `narrow` entries as bounded reviewed exceptions. They SHOULD shrink when the owning surface is already being edited, but they MUST NOT trigger opportunistic cross-route rewrites.
- Treat `keep` entries as stable foundation-owned implementation boundaries such as theme token owners, typography role owners, or renderer internals. Do not try to remove them during unrelated UI work.
- Treat `excluded` entries as intentional non-production or visual-lab boundaries. Do not copy their patterns into production UI.
- New entries MUST be narrower than the production surface they protect whenever possible: exact file before route group, exact pattern before wildcard, exact count before open-ended allowance.
- Reducing ledger counts requires fresh guardrail evidence. Do not decrement counts by inspection only.
- If a retained exception starts repeating in a second independent surface, consider promoting a shared primitive/helper before adding another exception.

## Typography System

Use `textRole` from `@/lib/typography` for production-readable text roles in foundation components and high-frequency production surfaces.

### Reference Strategy

- Do not duplicate exact typography utility strings in documentation as the long-term standard source.
- Treat `@/lib/typography` as the single implementation truth for current role-to-class mappings.
- Treat this README as the stable policy layer: when to reuse roles, when exceptions are allowed, and how to evolve the system.
- If typography values change later, update `@/lib/typography` first; only update docs when the semantic meaning or governance changes.

### Current Role Entry Points

- `pageTitle`: page-level primary title.
- `pageDescription`: page-level supporting description.
- `panelTitle`: high-signal panel or detail header title.
- `sectionTitle`: headings and compact panel titles.
- `metricValueDisplay`: operational metric values; set `data-featured="true"` only for the one decision-driving value in a contained summary surface.
- `body`: primary readable body copy.
- `bodyLarge`: long explanations, onboarding copy, documentation-like guidance, and longer empty-state descriptions.
- `bodyStrong`: emphasized readable body or primary labels inside controls.
- `bodySubtle`: support copy and secondary readable text.
- `helperText`: compact helper copy and support hints.
- `metadataLabel`: labels in key/value rows and compact support labels.
- `metadataValue`: values in key/value rows.
- `navLabel`: readable navigation labels in sidebar and menu surfaces.
- `tableHeader`: readable table headers.
- `tableCellPrimary`: primary list/table content.
- `tableCellSecondary`: secondary list/table content.
- `badge` / `badgeSubtle`: readable badge labels.
- `tab`: readable tab and filter labels.
- `code`: machine-oriented text such as code, IDs, URLs, and snippets.

### Readability Rules

- Do not make Chinese-readable UI text depend on `uppercase` or large letter spacing by default.
- Use mono fonts for code, URLs, IDs, numbers, versions, and machine labels, not for normal Chinese interface labels.
- Do not add remote font `@import` rules to `frontend/app/globals.css`. Global typography must use `--font-sans` / `--font-mono` / `--font-serif` token stacks unless a reviewed font-loading strategy proves it does not delay first paint.
- Prefer `text-foreground` and `text-muted-foreground` through `textRole` over ad hoc opacity such as `text-foreground/80`.
- If a page needs a new repeated typography pattern, add or extend a role instead of hand-writing local `font-* text-* tracking-*` combinations.
- If a role no longer fits the product direction, change the role implementation or rename the role; do not copy the old classes into page-local code.
- Operational hierarchy is semantic, not card-driven: page title comes first, then a focused panel title or one featured metric, then compact section headings, ordinary metrics, metadata, and support copy. Equal peer metrics MUST remain ordinary; do not mark every card-grid value as featured.
- Use `sectionTitle` for compact operational headings. Reserve `panelTitle` for focused detail, dialog, drawer, or error titles that need a stronger local anchor.
- Loading placeholders for a resolved heading or metric MUST reserve the same typography role. A featured metric skeleton must also carry `data-featured="true"`.

### Responsive Typography Rules

- Page titles and descriptions should get responsive behavior from `textRole` or shared shells such as `PageHeader`.
- Do not scatter page-local breakpoint font sizes such as `sm:text-*`, `md:text-*`, or `lg:text-*` on ordinary headings when a shared role or shell can own the hierarchy.
- Dense data surfaces such as tables, filters, pagination, tabs, metadata, and badges should stay stable across breakpoints unless a shared component explicitly owns a responsive rule.
- Long help, onboarding, documentation-like copy, and longer empty-state explanations should use `bodyLarge` instead of local `text-base` combinations.

### Typography Workflow

- Production-readable text MUST reuse `@/lib/typography` first.
- Do not introduce new page-local `font-* text-* tracking-*` combinations when an existing `textRole` fits.
- If no existing role fits, extend `textRole` first and then reuse it from the page or component.
- Treat local one-off typography as an explicit exception that should be documented, not as the default implementation path.

### Recommended Mapping

- Page shell: `pageTitle` + `pageDescription`
- Long explanations/onboarding/empty states: `bodyLarge`
- Split detail headers: `panelTitle`
- Compact section headers: `sectionTitle`
- Sidebar and menu labels: `navLabel`
- Table and filter support text: `helperText`
- Pagination footer labels and counts: `metadataLabel` + `metadataValue`
- Tabs and filter labels: `tab`
- Badge-like status labels: `badge` / `badgeSubtle`

## Button System

Use `Button` from `@/components/ui/button` for production buttons. Do not duplicate primary button styling with raw utility strings such as `bg-primary hover:bg-primary/90 px-4 py-2 rounded-md text-primary-foreground`.

The only approved production native-button exceptions are recorded in `frontend/foundation-exceptions.json`. As of the current convergence baseline, those exceptions cover terminal-styled nudge controls, overview interactive chart marks, an accessibility-only support skip control, the low-level sidebar rail hit area, and the vulnerability vertical resize handle. New ordinary actions, icon-only actions, toolbar actions, table actions, dialog actions, and empty-state actions MUST use `Button` unless a narrower reviewed exception explains why the control is not an ordinary command button.

### Approved Sizes

- `default`: 36px height. Use for form submits, dialog footers, empty-state CTAs, retry actions, and normal-density page actions.
- `sm`: 32px height. Use for dense table/list toolbars, filter strips, and compact operator controls.
- `action-card`: full available height and width with 64px minimum height. Use for card-like shortcut actions inside overview grids or action panels.
- `icon-sm`: 32px square. Use for compact icon-only controls in top bars, data tables, cards, and dense toolbars.
- `icon`: 36px square. Use for standard icon-only controls when the surrounding surface is not dense.
- Shared loading action placeholders MUST reuse this same structural size
  contract through `ActionSkeleton`; do not restate local `h-*`, `size-*`, or
  `rounded-*` button geometry when a skeleton only needs to reserve the action
  slot.
- Shared loading pagination shells SHOULD keep page-size geometry on
  `SelectShellSkeleton` and compact icon-only navigation geometry on
  `ActionSkeleton size="icon-sm"` instead of route-local `h-8 w-24` and
  `h-8 w-8` approximations.

### Interaction Rules

- Primary/default buttons must keep stable geometry on hover.
- `surface` remains opaque in both light and dark themes through `bg-background`; it owns solid secondary controls on canvases, cards, and panels. Do not add translucent `dark:bg-input/*` or `dark:hover:bg-input/*` overrides to this variant.
- `outline` may keep its theme-owned translucent dark fill when a border-first affordance is intentional; `ghost` and `quiet` remain transparent.
- Do not add hover transform motion such as `hover:-translate-y-px`.
- Do not change border thickness on hover, including `hover:border-b-2`.
- Do not add highlight-only bottom borders such as `hover:border-b-highlight`.
- Non-primary variants may use color-only hover feedback when it does not change layout or geometry.
- `link` keeps neutral text by default and changes only its text color to `interaction-accent` on hover; it does not add a hover background or replace ordinary navigation links.
- Opening a generic `DropdownMenu` must not replace a `Button`'s semantic variant colors. A primary action remains primary while its menu is open; only an explicit layout owner such as `tableHeaderInline` may define a popup-open visual state.

### Selected-Row Bulk Actions

Selectable business tables use one selected-row interaction model across the product.

- Selected-row business actions MUST render through `SelectedRowActionBar` or `UnifiedDataTable.actions.selectedRowActions`.
- The ordinary table toolbar is reserved for page-level tools: search, filters, add, bulk add/import, refresh, export, and column controls.
- Do not put selected-row destructive actions in `TableActions`, toolbar dropdown menus, or page-local action strips. Selection creates the contextual action surface.
- Non-destructive actions in the selected-row surface, including `success` and `muted` status actions, MUST share the outline button's neutral hover background. Status tone stays on the semantic icon; it must not suppress the shared hover surface or tint the label/background.
- Action order and `tone` do not imply persistent primary emphasis. Do not give the first action a permanent fill or add page-local action-bar hover classes; a future persistent emphasis requirement needs an explicit reviewed shared API.
- Destructive selected-row actions keep the shared destructive text and destructive-tinted hover treatment. Do not normalize delete, unlink, revoke, or similarly risky commands to the neutral non-destructive background.
- Destructive selected-row actions must open a confirmation dialog before mutation. The dialog copy should summarize count and consequence instead of listing every selected row by default.
- Pages that keep selected rows in route state MUST pass that array back through `state.selectedRows` so shared tables can synchronize internal checkbox state after cancel, confirm, pagination, or mutation cleanup.
- Pure selection UI changes such as deselecting rows do not need toast feedback. Business mutations do.

### Semantic Status Action Buttons

Enterprise admin actions that mark a lifecycle state, review state, health state, or similar status must not repeat the same semantic color across every visual channel.

- Use `Button` plus shared status helpers such as `getStatusToneInteractiveOutlineClass` and `getStatusToneTextClass` before composing page-local status button classes.
- When a button already has a semantic icon and/or semantic border, keep the readable label on `text-foreground`. Do not make labels such as "Mark Reviewed", "Mark Pending", "Approve", or "Archive" green, gray, red, or blue only because the target state has that tone.
- Do not add status-tinted default or hover fills such as `bg-success/10`, `hover:bg-success/10`, `hover:bg-error/10`, or equivalent local utility bundles when the icon, border, badge, or adjacent state indicator already communicates the state.
- Use hover and pressed states as interaction feedback, not as a second state badge. For outline status actions, prefer transparent backgrounds with subtle border emphasis, or a shared neutral treatment when the surrounding control family already uses neutral feedback.
- Keep status action icons consistent with the status indicators used in nearby tables, headers, cards, and metrics. If "reviewed" uses `CheckCircle2` with the success tone in the table, the matching batch action and header action should use the same semantic icon and tone.
- Apply the same rule in light and dark themes. Do not add dark-only colored fills or palette-specific borders to compensate for weak contrast; improve the shared status helper or theme token instead.
- Do not use this restrained status-action pattern for destructive, irreversible, or high-risk commands. Delete, revoke, disable, block, and similarly risky actions should use the shared destructive/danger action pattern and confirmation model appropriate to their risk.

### Async Status Action Feedback

Reversible semantic status actions, such as marking selected rows as reviewed, pending, enabled, disabled-by-policy, archived, or acknowledged, should acknowledge the click quickly without making the user stare at a still-enabled toolbar.

- For selected table or batch-action surfaces, close the transient selection toolbar immediately after the command starts when the implementation can preserve or restore the selection context. This prevents duplicate submission and makes the click feel accepted.
- Show async progress through one stable toast lifecycle using `loadingToast` or an equivalent shared feedback owner. Prefer `loading -> success` or `loading -> error` under the same operation identity over stacking a "working" toast and a separate completion toast.
- Success feedback should report the concrete result count when available. Error feedback should keep or restore the user's context when practical, especially if the toolbar was dismissed optimistically.
- Do not use this optimistic toolbar-dismiss pattern for destructive, irreversible, or high-risk commands unless the flow has a reviewed confirmation, undo, or recovery model.
- If the operation is local and completes instantly, a loading toast may be skipped; the command still needs visible success, error, or state-change feedback.

### Hardcoding Rules

- Do not hand-code primary production buttons with `button` plus utility classes. Use `Button`.
- Do not combine `size="icon"` with ad-hoc `h-7 w-7` or `h-8 w-8`; use `icon-sm` or another approved shared size.
- Decorative icons, skeletons, charts, badges, and non-button layout dimensions are not governed by this button rule.

### Contract Tests

Button rules are enforced by:

- `frontend/components/ui/__tests__/button.contract.test.ts`
- `frontend/components/__tests__/button-standardization.contract.test.ts`

If a new exception is truly needed, document the reason near the component and update the relevant contract test instead of silently adding one-off button styling.
