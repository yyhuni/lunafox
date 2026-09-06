import "@testing-library/jest-dom/vitest"
import * as React from "react"
import { afterAll, afterEach, beforeAll, vi } from "vitest"
import { mockServer } from "./mock/server"
import { resetMockScenario } from "./mock/scenarios"
import { resetMockScheduledScans } from "./mock/data/scheduled-scans"
import { resetMockMcpKey } from "./mock/data/mcp-key"
import { resetMockLoginVisualDiscoverability } from "./mock/data/login-visual"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (!params) return key
    return `${key}:${JSON.stringify(params)}`
  },
  useLocale: () => "en",
  NextIntlClientProvider: ({ children }: { children: React.ReactNode }) => children,
}))

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

if (!HTMLElement.prototype.scrollIntoView) {
  HTMLElement.prototype.scrollIntoView = vi.fn()
}

beforeAll(() => {
  mockServer.listen({ onUnhandledRequest: "bypass" })
})

afterEach(() => {
  mockServer.resetHandlers()
  resetMockScenario()
  resetMockScheduledScans()
  resetMockMcpKey()
  resetMockLoginVisualDiscoverability()
})

afterAll(() => {
  mockServer.close()
})
