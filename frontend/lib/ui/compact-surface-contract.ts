/**
 * Shared geometry for ordinary compact business surfaces.
 *
 * These classes intentionally live outside route components so read-only
 * values and short notices cannot drift into a collection of near-identical
 * border, radius, and inset combinations. Content-driven surfaces (editors,
 * logs, code, charts, and long copy) must keep their owning geometry instead.
 */
export const compactSurfaceClassNames = {
  value:
    "radius-control flex h-8 min-h-8 min-w-0 items-center border border-border bg-muted/30 px-3 py-1",
  info: "radius-control border border-border px-3 py-2",
  mutedInfo: "radius-control border border-border bg-muted/30 px-3 py-2",
} as const

export type CompactSurfaceKind = keyof typeof compactSurfaceClassNames

export const compactSingleLineValueHeight = "h-8"
export const compactSurfaceBorder = "border"
export const compactSurfaceInset = "px-3 py-2"
