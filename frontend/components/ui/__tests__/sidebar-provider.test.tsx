import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { SidebarProvider, useSidebar } from "@/components/ui/sidebar"

type MediaQueryController = MediaQueryList & {
  setMatches: (matches: boolean) => void
}

function createMediaQueryController(initialMatches: boolean): MediaQueryController {
  let matches = initialMatches
  const media = "(min-width: 768px) and (max-width: 1279px)"
  const listeners = new Set<(event: MediaQueryListEvent) => void>()

  return {
    get matches() {
      return matches
    },
    media,
    onchange: null,
    addEventListener: (_type: string, listener: (event: MediaQueryListEvent) => void) => listeners.add(listener),
    removeEventListener: (_type: string, listener: (event: MediaQueryListEvent) => void) => listeners.delete(listener),
    addListener: (listener: (event: MediaQueryListEvent) => void) => listeners.add(listener),
    removeListener: (listener: (event: MediaQueryListEvent) => void) => listeners.delete(listener),
    dispatchEvent: () => true,
    setMatches(nextMatches: boolean) {
      matches = nextMatches
      const event = { matches, media } as MediaQueryListEvent
      listeners.forEach((listener) => listener(event))
    },
  } as unknown as MediaQueryController
}

function SidebarStateProbe() {
  const { state, toggleSidebar } = useSidebar()

  return (
    <button type="button" data-testid="sidebar-state" onClick={toggleSidebar}>
      {state}
    </button>
  )
}

describe("SidebarProvider responsive defaults", () => {
  let compactDesktopQuery: MediaQueryController

  beforeEach(() => {
    document.cookie = "sidebar_state=; Max-Age=0; Path=/"
    compactDesktopQuery = createMediaQueryController(true)

    vi.spyOn(window, "matchMedia").mockImplementation((query) => {
      if (query === "(min-width: 768px) and (max-width: 1279px)") {
        return compactDesktopQuery
      }

      return createMediaQueryController(false)
    })
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    document.cookie = "sidebar_state=; Max-Age=0; Path=/"
  })

  it("defaults to icon mode on compact desktop viewports", async () => {
    render(
      <SidebarProvider>
        <SidebarStateProbe />
      </SidebarProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId("sidebar-state")).toHaveTextContent("collapsed")
    })
  })

  it("keeps the expanded desktop width mode on large viewports", async () => {
    compactDesktopQuery.setMatches(false)

    render(
      <SidebarProvider>
        <SidebarStateProbe />
      </SidebarProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId("sidebar-state")).toHaveTextContent("expanded")
    })
  })

  it("respects an explicit defaultOpen value", async () => {
    compactDesktopQuery.setMatches(false)

    render(
      <SidebarProvider defaultOpen={false}>
        <SidebarStateProbe />
      </SidebarProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId("sidebar-state")).toHaveTextContent("collapsed")
    })
  })

  it("ignores a legacy stored sidebar preference", async () => {
    document.cookie = "sidebar_state=true; Path=/"

    render(
      <SidebarProvider>
        <SidebarStateProbe />
      </SidebarProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId("sidebar-state")).toHaveTextContent("collapsed")
    })
  })

  it("returns to the responsive state after a manual toggle", async () => {
    compactDesktopQuery.setMatches(false)

    render(
      <SidebarProvider>
        <SidebarStateProbe />
      </SidebarProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId("sidebar-state")).toHaveTextContent("expanded")
    })

    fireEvent.click(screen.getByTestId("sidebar-state"))
    expect(screen.getByTestId("sidebar-state")).toHaveTextContent("collapsed")

    act(() => compactDesktopQuery.setMatches(true))
    expect(screen.getByTestId("sidebar-state")).toHaveTextContent("collapsed")

    act(() => compactDesktopQuery.setMatches(false))
    expect(screen.getByTestId("sidebar-state")).toHaveTextContent("expanded")
  })

  it("does not persist a manual sidebar toggle", async () => {
    render(
      <SidebarProvider>
        <SidebarStateProbe />
      </SidebarProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId("sidebar-state")).toHaveTextContent("collapsed")
    })

    fireEvent.click(screen.getByTestId("sidebar-state"))
    expect(document.cookie).not.toContain("sidebar_state=")
  })
})
