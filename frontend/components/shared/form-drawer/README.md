# FormDrawer

`FormDrawer` is the shared right-side drawer for create and edit workflows.

## Use When

- The user is creating, editing, linking, or configuring one business object.
- The task needs a persistent submit/cancel footer.
- The user should keep page context while completing a form.
- A centered dialog would be too cramped or would hide important table context.

## Do Not Use When

- The surface is read-only detail inspection; use `DetailDrawer`.
- The task is a short confirmation; use the shared confirm dialog.
- The content is a complex diagnostic workspace with logs, tables, and multi-panel navigation; start with `DetailDrawer` and add a workbench owner only if the pattern repeats.

## Layout Contract

- Shell: `Sheet` + shared `formDrawerContentClassName`.
- Backdrop: inherits the shared shadcn base-nova overlay from `SheetContent`; callers must not pass local backdrop variants or bypass blur.
- Motion: `FormDrawer` always uses the shared edge-panel transform transition, while its backdrop transitions only opacity. The shared Sheet primitive stages a dynamically loaded controlled drawer's first `open=true` frame so the transition is not skipped; users requesting reduced motion open without that frame staging. `formDrawerContentClassName` owns layout and width only; forms must mount their visible loading or content shell with the panel instead of delaying body rendering.
- Width: shared form drawer tier, currently aligned with existing form dialog width.
- Header: `px-5 py-4 sm:px-6`, bottom border, and the shared `EdgePanelHeader` `form` variant. It retains the shared `min-h-8` title row and follows the quick-scan hierarchy with a `size-10` semantic icon container, `textRole.sectionTitle` title, `textRole.helperText` description, and one close action.
- Body: scrolls independently with `px-6 py-4` and a `gap-4` form rhythm.
- Footer: fixed at the bottom with `border-t px-6 py-4`.
- Close: uses `SheetClose` with a shared `Button size="icon-sm"` and translated close label.

## Embedded Panel

Use `FormDrawerPanel` when a form is part of a wider split detail workflow and should sit beside the current detail context instead of opening another overlay. It reuses the same header, body, footer, close button, and form rhythm as `FormDrawer`, but the parent workbench owns placement, width, borders, and responsive behavior.

- Desktop split detail workflows SHOULD place the read-only detail context and `FormDrawerPanel` side by side inside one outer drawer or workbench. The panel form fills the parent column width; parent workbenches own only the column size and borders.
- Narrow screens MAY show only the panel with a close/back action that returns to the detail context.
- Do not use `FormDrawerPanel` as a free-form local card; it is for embedded create, edit, link, or configure tasks that still need the shared form drawer chrome.

## Field Ordering

Form drawers should keep the primary task field visible and stable. The primary task field is the field or group that directly completes the drawer's job, such as a bulk target input, object name, or configuration editor.

- Put required primary task fields before optional association fields, metadata, tags, owners, or secondary settings.
- Place optional association fields after the primary task field unless they change validation, permissions, available options, or defaults for fields that follow.
- Searchable optional association controls should use a stable trigger with `Popover` and `Command` content, or another shared overlay owner, instead of expanding inline above the primary task field.
- Inline expanding optional controls must not push the primary task field, validation summary, or active editor. If inline expansion is unavoidable, place it after the primary task field, constrain its height, and let the drawer body or panel scroll.
- AI agents should select field order by dependency first, task importance second, and layout stability third.

## Styling Contract

- Use theme tokens through shared primitives. Do not add page-local raw colors.
- The drawer surface is `bg-card text-card-foreground` through the shared sheet shell.
- Field controls should use shared `Input`, `Select`, `Button`, `Textarea`, and form presentation components.
- Do not nest cards just to create sections; use labels, spacing, and shared form components first.

## Route Usage

Route modules should pass domain copy, icon, fields, and footer actions into `FormDrawer`. They should not rebuild drawer header, close affordance, body scroll region, or footer geometry locally.
