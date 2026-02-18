import { describe, expect, it, vi } from "vitest"
import { handleScheduledScanMutationSuccess } from "@/hooks/_shared/scheduled-scan-mutation-helpers"

describe("scheduled-scan-mutation-helpers", () => {
  it("calls parse and success handler", () => {
    const parse = vi.fn().mockReturnValue({ ok: true })
    const onSuccess = vi.fn()

    handleScheduledScanMutationSuccess({
      response: { ok: true },
      parse,
      onSuccess,
    })

    expect(parse).toHaveBeenCalledWith({ ok: true })
    expect(onSuccess).toHaveBeenCalled()
  })

  it("uses default parse when not provided", () => {
    const onSuccess = vi.fn()

    handleScheduledScanMutationSuccess({
      response: { message: "ok" },
      onSuccess,
    })

    expect(onSuccess).toHaveBeenCalled()
  })
})
