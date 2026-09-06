import type { CSSProperties } from "react"

export const protectedAppShellStyle: CSSProperties = {
  "--sidebar-top": "0px",
  "--sidebar-height": "100svh",
  "--header-height": "calc(var(--spacing) * 10)",
} as CSSProperties

export const protectedAppShellScrollAreaClassName =
  "flex min-h-0 flex-1 flex-col overflow-x-hidden overflow-y-auto [scrollbar-gutter:stable_both-edges]"

// Keep the scroll area and route frame full width so wide displays can use the
// available canvas; individual readable surfaces own any intentional max-width.
export const protectedAppShellContentFrameClassName =
  "@container/main mx-auto flex min-h-0 min-w-0 w-full max-w-none flex-1 flex-col gap-2"
