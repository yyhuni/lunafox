/**
 * Shared page rhythm for authenticated production workspaces.
 *
 * The shell values are intentionally separate from control dimensions: page
 * whitespace can become denser without shrinking the controls people operate.
 */
export const COMPACT_PAGE_RHYTHM_CLASS = "gap-3 py-3"

/** Shared responsive gutter for page content regions below route chrome. */
export const COMPACT_CONTENT_GUTTER_CLASS = "px-3"

// 406 spacing units include 1600px of content and the existing two 12px
// route gutters. An active canvas opts out without adding route state to auth.
export const COMPACT_CONTENT_FRAME_CLASS =
  "mx-auto w-full max-w-406 has-[[data-workspace-width=full]]:max-w-none"

/** Shared inner rhythm for form-oriented right-side overlays. */
export const COMPACT_FORM_OVERLAY_HORIZONTAL_INSET_CLASS = "px-4"
export const COMPACT_FORM_OVERLAY_VERTICAL_INSET_CLASS = "py-3"
export const COMPACT_FORM_OVERLAY_INSET_CLASS = `${COMPACT_FORM_OVERLAY_HORIZONTAL_INSET_CLASS} ${COMPACT_FORM_OVERLAY_VERTICAL_INSET_CLASS}`
export const COMPACT_FORM_OVERLAY_SECTION_GAP_CLASS = "gap-3"

/** Centered form dialogs use the same readable insets and field rhythm as edge forms. */
export const COMPACT_FORM_DIALOG_CONTENT_CLASS = `${COMPACT_FORM_OVERLAY_SECTION_GAP_CLASS} ${COMPACT_FORM_OVERLAY_INSET_CLASS}`

// Page-owned scroll regions use the shared overlay primitive. A native scrollbar
// reserves horizontal width and turns the right 12px gutter into a wider gap.
export const COMPACT_PAGE_SCROLL_AREA_CLASS = "min-h-0 min-w-0 flex-1"
export const COMPACT_PAGE_SCROLL_AREA_CONTENT_CLASS = "!min-w-0 w-full"
export const COMPACT_PAGE_SCROLL_AREA_VIEWPORT_CLASS = "!overflow-x-hidden"

export const COMPACT_PAGE_SHELL_CLASS =
  `flex flex-col ${COMPACT_PAGE_RHYTHM_CLASS}`

export const COMPACT_FLEX_PAGE_SHELL_CLASS =
  `flex min-h-0 flex-1 flex-col ${COMPACT_PAGE_RHYTHM_CLASS}`

export const COMPACT_FULL_PAGE_SHELL_CLASS =
  `flex h-full min-h-0 flex-1 flex-col ${COMPACT_PAGE_RHYTHM_CLASS}`

export const COMPACT_CONTAINER_PAGE_SHELL_CLASS =
  `@container/main flex min-h-0 flex-1 flex-col ${COMPACT_PAGE_RHYTHM_CLASS}`

export const COMPACT_SECTION_GAP_CLASS = "gap-3"
export const COMPACT_SECTION_STACK_CLASS = "space-y-3"
export const COMPACT_PANEL_PADDING_CLASS = "p-4"
