import { render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const startMockWorker = vi.fn(async () => undefined)
const cleanupLegacyMockWorker = vi.fn(async () => undefined)

vi.mock("@/mock/config", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/mock/config")>()
  return {
    ...actual,
    USE_MOCK: true,
  }
})

vi.mock("@/mock/browser", () => ({
  startMockWorker,
}))

vi.mock("@/mock/legacy-worker-cleanup", () => ({
  cleanupLegacyMockWorker,
}))

describe("MockProvider", () => {
  beforeEach(() => {
    startMockWorker.mockClear()
    cleanupLegacyMockWorker.mockClear()
  })

  it("以服务端传入的 enabled=false 为准，避免客户端 mock 开关漂移导致首屏树形不一致", async () => {
    const { MockProvider } = await import("@/components/providers/mock-provider")

    render(
      <MockProvider enabled={false}>
        <div data-testid="provider-child">ready</div>
      </MockProvider>
    )

    expect(screen.getByTestId("provider-child")).toBeInTheDocument()
    expect(startMockWorker).not.toHaveBeenCalled()
    await waitFor(() => {
      expect(cleanupLegacyMockWorker).toHaveBeenCalledTimes(1)
    })
  })

  it("启用 mock 时仍立即保留首屏树形，由 network layer 接管 worker 就绪等待", async () => {
    startMockWorker.mockImplementationOnce(() => new Promise(() => undefined))

    const { MockProvider } = await import("@/components/providers/mock-provider")

    render(
      <MockProvider enabled>
        <div data-testid="provider-child">ready</div>
      </MockProvider>
    )

    expect(screen.getByTestId("provider-child")).toBeInTheDocument()
    await waitFor(() => {
      expect(startMockWorker).toHaveBeenCalledTimes(1)
    })
    expect(cleanupLegacyMockWorker).not.toHaveBeenCalled()
  })
})
