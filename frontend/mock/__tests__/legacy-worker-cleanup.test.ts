import { describe, expect, it, vi } from "vitest"
import { cleanupLegacyMockWorker } from "@/mock/legacy-worker-cleanup"

function registration(scriptURL: string) {
  const postMessage = vi.fn()
  const unregister = vi.fn(async () => true)
  return {
    registration: {
      active: { scriptURL, postMessage },
      waiting: null,
      installing: null,
      unregister,
    },
    postMessage,
    unregister,
  }
}

describe("legacy mock worker cleanup", () => {
  it("closes and unregisters only the legacy mock worker", async () => {
    const legacy = registration("https://console.example/mockServiceWorker.js")
    const unrelated = registration("https://console.example/app-worker.js")
    const container = {
      getRegistrations: vi.fn(async () => [legacy.registration, unrelated.registration]),
    }

    await cleanupLegacyMockWorker(container)

    expect(legacy.postMessage).toHaveBeenCalledWith("CLIENT_CLOSED")
    expect(legacy.unregister).toHaveBeenCalledOnce()
    expect(unrelated.postMessage).not.toHaveBeenCalled()
    expect(unrelated.unregister).not.toHaveBeenCalled()
  })

  it("resolves when Service Worker APIs are unavailable or reject", async () => {
    await expect(cleanupLegacyMockWorker(null)).resolves.toBeUndefined()
    await expect(
      cleanupLegacyMockWorker({
        getRegistrations: vi.fn(async () => {
          throw new Error("unavailable")
        }),
      }),
    ).resolves.toBeUndefined()
  })

  it("continues cleanup when a worker close signal fails", async () => {
    const legacy = registration("https://console.example/mockServiceWorker.js")
    legacy.postMessage.mockImplementation(() => {
      throw new Error("worker stopped")
    })

    await cleanupLegacyMockWorker({
      getRegistrations: vi.fn(async () => [legacy.registration]),
    })

    expect(legacy.unregister).toHaveBeenCalledOnce()
  })

  it("finds a legacy waiting worker even when the active worker is unrelated", async () => {
    const active = { scriptURL: "https://console.example/app-worker.js", postMessage: vi.fn() }
    const waiting = { scriptURL: "https://console.example/mockServiceWorker.js", postMessage: vi.fn() }
    const unregister = vi.fn(async () => true)

    await cleanupLegacyMockWorker({
      getRegistrations: vi.fn(async () => [{ active, waiting, installing: null, unregister }]),
    })

    expect(active.postMessage).not.toHaveBeenCalled()
    expect(waiting.postMessage).toHaveBeenCalledWith("CLIENT_CLOSED")
    expect(unregister).toHaveBeenCalledOnce()
  })
})
