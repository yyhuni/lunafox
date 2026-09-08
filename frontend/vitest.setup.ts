import "@testing-library/jest-dom/vitest"
import * as React from "react"
import { afterAll, afterEach, beforeAll, vi } from "vitest"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (!params) return key
    return `${key}:${JSON.stringify(params)}`
  },
  useLocale: () => "en",
  NextIntlClientProvider: ({ children }: { children: React.ReactNode }) => children,
}))

if (typeof window !== "undefined") {
  if (!window.matchMedia) {
    Object.defineProperty(window, "matchMedia", {
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    })
  }

  class ResizeObserverMock {
    observe() {}
    unobserve() {}
    disconnect() {}
  }

  class IntersectionObserverMock {
    root = null
    rootMargin = ""
    thresholds = []
    observe() {}
    unobserve() {}
    disconnect() {}
    takeRecords() {
      return []
    }
  }

  Object.defineProperty(globalThis, "ResizeObserver", {
    value: ResizeObserverMock,
    writable: true,
    configurable: true,
  })

  Object.defineProperty(globalThis, "IntersectionObserver", {
    value: IntersectionObserverMock,
    writable: true,
    configurable: true,
  })

  if (typeof HTMLElement !== "undefined" && !HTMLElement.prototype.scrollIntoView) {
    HTMLElement.prototype.scrollIntoView = vi.fn()
  }
}

let mockServerInstance: (typeof import("./mock/server"))["mockServer"] | null = null
let mockResetFns: {
  resetMockScenario: () => unknown
  resetMockScheduledScans: () => unknown
  resetMockMcpKey: () => unknown
  resetMockLoginVisualDiscoverability: () => unknown
} | null = null

async function getMockPlatform() {
  if (!mockServerInstance) {
    const serverMod = await import("./mock/server")
    mockServerInstance = serverMod.mockServer
    const [scenariosMod, scheduledScansMod, mcpKeyMod, loginVisualMod] = await Promise.all([
      import("./mock/scenarios"),
      import("./mock/data/scheduled-scans"),
      import("./mock/data/mcp-key"),
      import("./mock/data/login-visual"),
    ])
    mockResetFns = {
      resetMockScenario: scenariosMod.resetMockScenario,
      resetMockScheduledScans: scheduledScansMod.resetMockScheduledScans,
      resetMockMcpKey: mcpKeyMod.resetMockMcpKey,
      resetMockLoginVisualDiscoverability: loginVisualMod.resetMockLoginVisualDiscoverability,
    }
  }
  return { mockServer: mockServerInstance, resetFns: mockResetFns }
}

beforeAll(async () => {
  if (typeof window !== "undefined") {
    const { mockServer } = await getMockPlatform()
    mockServer.listen({ onUnhandledRequest: "bypass" })
  }
})

afterEach(async () => {
  if (typeof window !== "undefined") {
    const { mockServer, resetFns } = await getMockPlatform()
    mockServer.resetHandlers()
    resetFns?.resetMockScenario()
    resetFns?.resetMockScheduledScans()
    resetFns?.resetMockMcpKey()
    resetFns?.resetMockLoginVisualDiscoverability()
  }
})

afterAll(async () => {
  if (mockServerInstance) {
    mockServerInstance.close()
  }
})
