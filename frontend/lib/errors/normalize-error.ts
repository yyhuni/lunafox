import axios from "axios"

import {
  createAppError,
  isAppError,
  type AppError,
  type AppErrorKind,
} from "@/lib/errors/app-error"

type NormalizeErrorOptions = {
  notFoundKind?: Extract<AppErrorKind, "resource-not-found" | "unexpected-error">
}

export function normalizeError(error: unknown, options: NormalizeErrorOptions = {}): AppError {
  if (isAppError(error)) {
    return error
  }

  if (axios.isAxiosError(error)) {
    const status = error.response?.status
    const code = typeof error.code === "string" ? error.code : undefined

    if (status === 404) {
      return createAppError(options.notFoundKind ?? "resource-not-found", {
        status,
        code,
        cause: error,
      })
    }

    if (status === 403) {
      return createAppError("permission-denied", { status, code, cause: error, retryable: false })
    }

    if (status === 429) {
      return createAppError("rate-limited", { status, code, cause: error })
    }

    if (typeof status === "number" && status >= 500) {
      return createAppError("service-unavailable", { status, code, cause: error })
    }

    if (code === "ERR_NETWORK" || !error.response) {
      return createAppError("network-error", { status, code, cause: error })
    }

    return createAppError("unexpected-error", { status, code, cause: error })
  }

  return createAppError("unexpected-error", { cause: error })
}
