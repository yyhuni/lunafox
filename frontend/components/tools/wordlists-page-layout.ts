import {
  COMPACT_CONTENT_GUTTER_CLASS,
  COMPACT_FULL_PAGE_SHELL_CLASS,
} from "@/components/shared/layout/page-shell-density"

export const WORDLISTS_PAGE_SHELL_CLASS =
  COMPACT_FULL_PAGE_SHELL_CLASS

export const WORDLISTS_CONTENT_SHELL_CLASS =
  `flex min-h-0 flex-1 ${COMPACT_CONTENT_GUTTER_CLASS}`

export const WORDLISTS_WORKSPACE_HANDOFF_CLASS = "flex h-full min-h-0 flex-1 flex-col"

export const WORDLISTS_WORKSPACE_SURFACE_CLASS =
  "flex h-full min-h-0 flex-1 flex-col overflow-hidden"

export const WORDLISTS_CATALOG_CONTROLS_CLASS =
  "flex flex-wrap items-start justify-between gap-2 border-b py-3"

export const WORDLISTS_CATALOG_GRID_CLASS =
  "grid grid-cols-1 gap-3 py-3 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4"

export const WORDLISTS_CATALOG_FOOTER_CLASS = "shrink-0 border-t py-2.5"
