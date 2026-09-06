import { AxiosError } from "axios"
import { describe, expect, it } from "vitest"

import { normalizeError } from "@/lib/errors/normalize-error"

function createHttpError(status: number, message = `HTTP ${status}`) {
  return new AxiosError(
    message,
    "ERR_BAD_RESPONSE",
    undefined,
    undefined,
    {
      status,
      statusText: String(status),
      config: {} as never,
      headers: {},
      data: {},
    }
  )
}

describe("normalizeError", () => {
  it("maps known HTTP statuses into shared business error kinds", () => {
    expect(normalizeError(createHttpError(404)).kind).toBe("resource-not-found")
    expect(normalizeError(createHttpError(403)).kind).toBe("permission-denied")
    expect(normalizeError(createHttpError(429)).kind).toBe("rate-limited")
    expect(normalizeError(createHttpError(503)).kind).toBe("service-unavailable")
  })

  it("maps transport failures into network or unexpected errors", () => {
    expect(normalizeError(new AxiosError("Network Error", "ERR_NETWORK")).kind).toBe("network-error")
    expect(normalizeError(new Error("boom")).kind).toBe("unexpected-error")
  })

  it("lets collection owners classify transport 404 as a retryable query failure", () => {
    const normalized = normalizeError(createHttpError(404), {
      notFoundKind: "unexpected-error",
    })

    expect(normalized.kind).toBe("unexpected-error")
    expect(normalized.retryable).toBe(true)
    expect(normalized.status).toBe(404)
  })
})
