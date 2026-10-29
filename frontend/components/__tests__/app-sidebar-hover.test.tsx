import { act, cleanup, fireEvent, render, screen } from "@testing-library/react"
import type { ReactNode } from "react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { AppSidebar } from "../app-sidebar"
import { SidebarProvider } from "../ui/sidebar"
import { sidebarHoverTiming } from "@/lib/ui/overlay-styles"

vi.mock("next/navigation", () => ({ usePathname: () => "/settings/mcp/" }))
vi.mock("next/link", () => ({
  default: ({ children, href, ...props }: { children: ReactNode; href: string }) => <a href={href} {...props}>{children}</a>,
  useLinkStatus: () => ({ pending: false }),
}))
vi.mock("@/components/nav-user", () => ({ NavUser: () => null, NavUserSkeleton: () => null }))
vi.mock("@/components/about-dialog", () => ({ AboutDialog: ({ children }: { children: ReactNode }) => children }))
vi.mock("@/hooks/use-login-visual", () => ({ useLoginVisualDiscoverability: () => ({ unlocked: false }) }))

function openSettings() {
  const view = render(<SidebarProvider defaultOpen={false}><AppSidebar /></SidebarProvider>)
  const trigger = screen.getByRole("button", { name: "systemSettings" })
  fireEvent.mouseEnter(trigger)
  const panel = view.container.querySelector('[data-sidebar-collapsed-submenu-panel="true"]')!
  return { ...view, trigger, panel }
}

function advance(ms: number) {
  act(() => vi.advanceTimersByTime(ms))
}

describe("collapsed sidebar hover lifecycle", () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => {
    cleanup()
    vi.useRealTimers()
  })

  it("waits for the shared leave delay before exiting, then unmounts after the exit animation", () => {
    const { trigger, panel } = openSettings()
    fireEvent.mouseLeave(trigger)
    expect(panel).not.toHaveClass("animate-out")
    advance(sidebarHoverTiming.closeDelay - 1)
    expect(panel).not.toHaveClass("animate-out")
    advance(1)
    expect(panel).toHaveClass("animate-out")
    advance(sidebarHoverTiming.unmountDelay - 1)
    expect(panel).toBeInTheDocument()
    advance(1)
    expect(panel).not.toBeInTheDocument()
  })

  it("keeps the same panel open when the pointer crosses the hover bridge or reenters during exit", () => {
    const { container, trigger, panel } = openSettings()
    const bridge = container.querySelector('[data-sidebar-collapsed-submenu-bridge="true"]')!
    fireEvent.mouseLeave(trigger)
    advance(sidebarHoverTiming.closeDelay - 1)
    fireEvent.mouseEnter(bridge)
    advance(sidebarHoverTiming.closeDelay + sidebarHoverTiming.unmountDelay)
    expect(panel).toBeInTheDocument()
    expect(panel).not.toHaveClass("animate-out")

    fireEvent.mouseLeave(panel.parentElement!)
    advance(sidebarHoverTiming.closeDelay)
    expect(panel).toHaveClass("animate-out")
    fireEvent.mouseEnter(panel)
    advance(sidebarHoverTiming.unmountDelay)
    expect(container.querySelector('[data-sidebar-collapsed-submenu-panel="true"]')).toBe(panel)
    expect(panel).not.toHaveClass("animate-out")
  })
})
