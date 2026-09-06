import { render } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import * as SidebarModule from "@/components/ui/sidebar"

const linkStatus = vi.hoisted(() => ({ pending: false }))

vi.mock("next/link", () => ({
  useLinkStatus: () => linkStatus,
}))

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

vi.mock("@/hooks/use-mobile", () => ({
  useIsMobile: () => true,
}))

describe("SidebarNavigationPendingIndicator", () => {
  afterEach(() => {
    linkStatus.pending = false
  })

  it("shows compact navigation feedback only while its link is pending", () => {
    const PendingIndicator = SidebarModule.SidebarNavigationPendingIndicator

    expect(PendingIndicator).toBeTypeOf("function")
    if (!PendingIndicator) return

    const view = render(<PendingIndicator />)
    expect(view.container.querySelector("[data-sidebar-navigation-pending]")).not.toBeInTheDocument()

    linkStatus.pending = true
    view.rerender(<PendingIndicator />)

    const pendingSurface = view.container.querySelector("[data-sidebar-navigation-pending]")
    expect(pendingSurface).toHaveAttribute(
      "data-sidebar-navigation-pending",
      "true"
    )
    expect(pendingSurface).toHaveAttribute("aria-hidden", "true")
    expect(pendingSurface).toHaveClass("absolute!", "inset-0", "z-0!")
    expect(pendingSurface).not.toHaveClass("bg-sidebar-accent/70")
    expect(pendingSurface?.querySelector("[data-slot='spinner']")).not.toBeInTheDocument()

    linkStatus.pending = false
    view.rerender(<PendingIndicator />)

    expect(view.container.querySelector("[data-sidebar-navigation-pending]")).not.toBeInTheDocument()
  })

})
