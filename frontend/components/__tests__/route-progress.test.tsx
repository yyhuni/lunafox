import { act, cleanup, fireEvent, render } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { RouteProgress } from "@/components/route-progress"

const mockRouteState = vi.hoisted(() => ({
  pathname: "/from",
  search: "",
}))

const mockNProgress = vi.hoisted(() => ({
  configure: vi.fn(),
  done: vi.fn(),
  isRendered: vi.fn(() => false),
  isStarted: vi.fn(() => false),
  remove: vi.fn(),
  start: vi.fn(),
  status: null as number | null,
}))

vi.mock("next/navigation", () => ({
  usePathname: () => mockRouteState.pathname,
  useSearchParams: () => ({ toString: () => mockRouteState.search }),
}))

vi.mock("nprogress", () => ({
  default: mockNProgress,
}))

function visibleRect(): DOMRect {
  return {
    bottom: 100,
    height: 100,
    left: 0,
    right: 100,
    top: 0,
    width: 100,
    x: 0,
    y: 0,
    toJSON() {
      return this
    },
  } as DOMRect
}

function emptyRect(): DOMRect {
  return {
    bottom: 0,
    height: 0,
    left: 0,
    right: 0,
    top: 0,
    width: 0,
    x: 0,
    y: 0,
    toJSON() {
      return this
    },
  } as DOMRect
}

function appendVisibleLoadingOwner({ phase = "loading" }: { phase?: string } = {}) {
  const owner = document.createElement("div")
  owner.setAttribute("data-loading-owner", "route-loading-owner")
  owner.setAttribute("data-loading-layer", "route")
  owner.setAttribute("data-loading-phase", phase)
  owner.textContent = "Loading"
  document.body.append(owner)
  return owner
}

function clickSameOriginRoute(href = "/to") {
  const anchor = document.createElement("a")
  anchor.href = href
  anchor.textContent = "Navigate"
  anchor.addEventListener("click", (event) => event.preventDefault())
  document.body.append(anchor)
  fireEvent.click(anchor)
}

describe("RouteProgress", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockRouteState.pathname = "/from"
    mockRouteState.search = ""
    window.history.replaceState(null, "", "/from")
    mockNProgress.status = null
    mockNProgress.configure.mockClear()
    mockNProgress.done.mockClear()
    mockNProgress.isRendered.mockReset().mockReturnValue(false)
    mockNProgress.isStarted.mockReset().mockImplementation(() => typeof mockNProgress.status === "number")
    mockNProgress.remove.mockClear()
    mockNProgress.start.mockReset().mockImplementation(() => {
      mockNProgress.status = 0.08
      return mockNProgress
    })

    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function getBoundingClientRect(this: HTMLElement) {
      return this.hasAttribute("data-loading-owner") ? visibleRect() : emptyRect()
    })
  })

  afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
    document.body.innerHTML = ""
  })

  it("does not start the top bar when a route loading owner appears before the delay", async () => {
    render(<RouteProgress />)

    await act(async () => {})
    expect(mockNProgress.configure).toHaveBeenCalled()

    clickSameOriginRoute()
    appendVisibleLoadingOwner()

    act(() => {
      vi.advanceTimersByTime(241)
    })

    expect(mockNProgress.start).not.toHaveBeenCalled()
    expect(mockNProgress.remove).toHaveBeenCalled()
  })

  it("keeps the top bar hidden during the short navigation handoff grace window", async () => {
    render(<RouteProgress />)

    await act(async () => {})
    expect(mockNProgress.configure).toHaveBeenCalled()

    clickSameOriginRoute()

    act(() => {
      vi.advanceTimersByTime(239)
    })

    expect(mockNProgress.start).not.toHaveBeenCalled()
  })

  it("cancels an already visible top bar when a route loading owner takes over", async () => {
    const { rerender } = render(<RouteProgress />)

    await act(async () => {})
    expect(mockNProgress.configure).toHaveBeenCalled()

    clickSameOriginRoute()

    act(() => {
      vi.advanceTimersByTime(241)
    })

    expect(mockNProgress.start).toHaveBeenCalledTimes(1)
    expect(mockNProgress.status).toBe(0.08)

    appendVisibleLoadingOwner()
    mockRouteState.pathname = "/to"
    window.history.replaceState(null, "", "/to")
    rerender(<RouteProgress />)

    expect(mockNProgress.remove).toHaveBeenCalled()
    expect(mockNProgress.status).toBeNull()
  })
})
