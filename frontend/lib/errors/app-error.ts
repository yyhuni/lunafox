export const APP_ERROR_KINDS = [
  "resource-not-found",
  "permission-denied",
  "rate-limited",
  "service-unavailable",
  "network-error",
  "unexpected-error",
] as const

export type AppErrorKind = (typeof APP_ERROR_KINDS)[number]

export interface AppError {
  kind: AppErrorKind
  status?: number
  code?: string
  retryable: boolean
  cause?: unknown
}

export function createAppError(
  kind: AppErrorKind,
  overrides: Partial<Omit<AppError, "kind">> = {}
): AppError {
  return {
    kind,
    retryable:
      kind === "rate-limited" ||
      kind === "service-unavailable" ||
      kind === "network-error" ||
      kind === "unexpected-error",
    ...overrides,
  }
}

export function isAppError(error: unknown): error is AppError {
  if (!error || typeof error !== "object") return false

  const kind = (error as { kind?: unknown }).kind
  return typeof kind === "string" && APP_ERROR_KINDS.includes(kind as AppErrorKind)
}
