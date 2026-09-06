# EdgePanelHeader

`EdgePanelHeader` is the shared layout foundation for visible right-side panel headers. It standardizes the common slots and responsive constraints while keeping scenario-specific density explicit.

## Variants

- `workbench`: multi-step or high-context tasks. Supports a visible title/description pair, a `size-10` leading icon container, right-side status/actions, and a progress or step slot.
- `form`: create, edit, link, and configuration forms. Supports title/description, an optional `size-10` leading icon container with a `size-7` concept glyph, and a compact action area. Its title uses `textRole.sectionTitle` and its description uses `textRole.helperText`, matching the quick-scan hierarchy. With a description, its icon, content, and actions align to the top; without one, they align vertically to the title row. It does not own workbench progress.
- `detail`: read-only record and diagnostic details. Keeps a single title row with optional `titleMeta` and `headerMeta`; it does not require a large icon or visible description.
- `compact`: lightweight feedback such as notifications. Keeps a one-line title/action rhythm and does not require description, leading, or progress.

## Slot ownership

The owning Sheet or Drawer supplies semantic `SheetTitle`, `DrawerTitle`, `SheetDescription`, or `DrawerDescription` nodes through `title` and `description`. `EdgePanelHeader` only arranges those nodes and MUST NOT create a second accessible title or description.

`leading`, `titleMeta`, `headerMeta`, `actions`, and `progress` are optional layout slots. `actions` owns the close control or other header action group; callers must not render a second close owner in the same panel.

## Use and boundaries

- Use the matching variant through `FormDrawer`, `DetailDrawer`, or a named workbench/feedback owner before composing route-local header geometry.
- Keep `min-w-0` on title, description, and metadata columns so localized content wraps or truncates inside the panel instead of pushing actions or widening the page.
- Use existing `textRole`, theme tokens, shared button sizes, and semantic icon components. The leading container supplies the muted semantic background; callers supply only the icon node.
- Do not use this component for the application navigation sidebar, centered dialogs, page-level headers, or professional log/editor/media surfaces whose toolbar rhythm is part of the product interaction.
- A specialized owner must document why a shared variant does not fit and may reuse only the foundation tokens that do not compromise its interaction model.
