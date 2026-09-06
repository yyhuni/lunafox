export const LOADING_LAYERS = [
  "boot",
  "auth-shell",
  "app-shell",
  "route",
  "workspace",
  "section",
  "interaction",
  "compact-pending",
  "domain-status",
] as const

export type LoadingLayer = (typeof LOADING_LAYERS)[number]

export const LOADING_INTENTS = [
  "boot",
  "auth",
  "app-shell",
  "route",
  "data",
  "navigation",
  "interaction",
  "pending",
  "status",
] as const

export type LoadingIntent = (typeof LOADING_INTENTS)[number]

export function isLoadingLayer(value: string | null | undefined): value is LoadingLayer {
  return LOADING_LAYERS.includes(value as LoadingLayer)
}

export function getLoadingOwnerAttributes({
  owner,
  layer,
  intent,
}: {
  owner: string
  layer: LoadingLayer
  intent?: LoadingIntent
}) {
  if (!owner.trim()) {
    throw new Error("Loading owner requires a non-empty owner.")
  }

  return {
    "data-loading-owner": owner,
    "data-loading-layer": layer,
    ...(intent ? { "data-loading-intent": intent } : {}),
  } as const
}

export function getLoadingStructureSlotAttributes(slot: string) {
  if (!slot.trim()) {
    throw new Error("Loading structure slot requires a non-empty slot.")
  }

  return {
    "data-loading-slot": slot,
  } as const
}
