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

  // Base UI checks active viewport animations while ScrollArea unmounts. jsdom
  // does not implement this browser API, but an empty animation list matches
  // the test environment's non-visual behavior.
  if (typeof Element !== "undefined" && !Element.prototype.getAnimations) {
    Object.defineProperty(Element.prototype, "getAnimations", {
      value: () => [],
      writable: true,
      configurable: true,
    })
  }

  // Keep Base UI dialog teardown synchronous in jsdom. The environment has no
  // visual animation timeline, while ScrollArea still needs the getAnimations
  // shim above to inspect its viewport during setup and cleanup.
  Object.defineProperty(globalThis, "BASE_UI_ANIMATIONS_DISABLED", {
    value: true,
    writable: true,
    configurable: true,
  })
}

let mockServerInstance: (typeof import("./mock/server"))["mockServer"] | null = null
let mockResetFns: {
  resetMockScenario: () => unknown
  resetMockScheduledScans: () => unknown
  resetMockMcpKey: () => unknown
  resetMockLoginVisualDiscoverability: () => unknown
  resetMockUpgradeOperation: () => unknown
} | null = null

async function getMockPlatform() {
  if (!mockServerInstance) {
    const serverMod = await import("./mock/server")
    mockServerInstance = serverMod.mockServer
    const [scenariosMod, scheduledScansMod, mcpKeyMod, loginVisualMod, versionMod] = await Promise.all([
      import("./mock/scenarios"),
      import("./mock/data/scheduled-scans"),
      import("./mock/data/mcp-key"),
      import("./mock/data/login-visual"),
      import("./mock/data/version"),
    ])
    mockResetFns = {
      resetMockScenario: scenariosMod.resetMockScenario,
      resetMockScheduledScans: scheduledScansMod.resetMockScheduledScans,
      resetMockMcpKey: mcpKeyMod.resetMockMcpKey,
      resetMockLoginVisualDiscoverability: loginVisualMod.resetMockLoginVisualDiscoverability,
      resetMockUpgradeOperation: versionMod.resetMockUpgradeOperation,
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
    resetFns?.resetMockUpgradeOperation()
  }
})

afterAll(async () => {
  if (mockServerInstance) {
    mockServerInstance.close()
  }
})
