import type { CSSProperties } from "react"
import { COMPACT_CONTENT_FRAME_CLASS } from "@/components/shared/layout/page-shell-density"

export const protectedAppShellStyle: CSSProperties = {
  "--sidebar-top": "0px",
  "--sidebar-height": "100svh",
  "--header-height": "calc(var(--spacing) * 9)",
} as CSSProperties

export const protectedAppShellScrollAreaClassName =
  "min-h-0 min-w-0 flex-1"

// Base UI owns the actual scroll viewport. Keep page-level horizontal clipping
// there so the overlay scrollbar cannot reserve space from the route canvas.
export const protectedAppShellScrollViewportClassName = "!overflow-x-hidden"

// Base UI's content part defaults to min-width: fit-content. The app shell
// needs its route frame to shrink with the available workspace instead.
export const protectedAppShellContentStyle: CSSProperties = {
  minWidth: 0,
}

// Only the route frame is bounded; the scroll viewport and shell chrome remain
// full width. Route owners already supply the gutters, including while loading.
export const protectedAppShellContentFrameClassName =
  `@container/main flex h-full min-h-0 min-w-0 flex-col gap-2 ${COMPACT_CONTENT_FRAME_CLASS}`
