import { beforeEach, describe, expect, it, vi } from "vitest"

const cleanupLegacyMockWorker = vi.fn(async () => undefined)

vi.mock("@/mock/config", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/mock/config")>()),
  USE_MOCK: false,
}))
vi.mock("@/mock/legacy-worker-cleanup", () => ({ cleanupLegacyMockWorker }))

describe("real API readiness", () => {
  beforeEach(() => {
    cleanupLegacyMockWorker.mockClear()
  })

  it("waits for legacy worker cleanup before real requests proceed", async () => {
    const { ensureMockInterceptionReady } = await import("@/lib/api-client")

    await ensureMockInterceptionReady()

    expect(cleanupLegacyMockWorker).toHaveBeenCalledOnce()
  })
})
