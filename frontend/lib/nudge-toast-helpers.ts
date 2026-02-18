import type { NudgeToastAction } from "@/components/nudges/nudge-toast-card"

const isObjectRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null

export const isNudgeSuppressed = (storageKey: string, cooldownMs?: number): boolean => {
  try {
    const raw = localStorage.getItem(storageKey)
    if (!raw) return false

    if (!cooldownMs) return true

    const nextAllowedAt = Number(raw)
    if (!Number.isFinite(nextAllowedAt)) return true

    if (nextAllowedAt > Date.now()) return true

    localStorage.removeItem(storageKey)
    return false
  } catch {
    return true
  }
}

export const suppressNudge = (storageKey: string, cooldownMs?: number) => {
  try {
    if (!cooldownMs) {
      localStorage.setItem(storageKey, "true")
      return
    }

    localStorage.setItem(storageKey, String(Date.now() + cooldownMs))
  } catch {
    // ignore
  }
}

export const withNudgeDismiss = (
  action: NudgeToastAction | undefined,
  onDismiss: () => void
): NudgeToastAction | undefined => {
  if (!action) return undefined

  const original = action.onClick
  return {
    ...action,
    onClick: () => {
      original?.()
      onDismiss()
    },
  }
}

export const isLocalStorageAvailable = (): boolean => {
  if (typeof window === "undefined") return false
  if (!isObjectRecord(window)) return false
  return "localStorage" in window
}
