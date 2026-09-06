import * as React from "react"
import { act, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import SupportPageContent from "@/components/settings/support/support-page-content"
import { SupportPageLoadingState } from "@/components/settings/support/support-page-loading-state"

vi.mock("canvas-confetti", () => ({
  default: vi.fn(),
}))

vi.mock("framer-motion", async () => {
  const React = await import("react")
  const elements = new Map<string, React.ForwardRefExoticComponent<Record<string, unknown>>>()
  const motion = new Proxy(
    {},
    {
      get: (_, element: string) => {
        const cached = elements.get(element)
        if (cached) return cached

        const component = React.forwardRef<HTMLElement, Record<string, unknown>>(({ children, ...props }, ref) => {
          const { animate, exit, initial, transition, ...nativeProps } = props
          void animate
          void exit
          void initial
          void transition
          return React.createElement(element, { ...nativeProps, ref }, children as React.ReactNode)
        })
        component.displayName = `motion.${element}`
        elements.set(element, component)
        return component
      },
    },
  )

  return {
    AnimatePresence: ({ children }: React.PropsWithChildren) => <>{children}</>,
    motion,
    useReducedMotion: () => true,
  }
})

vi.mock("@/components/ui/dialog", async () => {
  const React = await import("react")

  return {
    Dialog: ({ children, open }: React.PropsWithChildren<{ open?: boolean }>) =>
      open ? <div role="dialog">{children}</div> : null,
    DialogContent: ({ children }: React.PropsWithChildren) => <div>{children}</div>,
    DialogHeader: ({ children }: React.PropsWithChildren) => <div>{children}</div>,
    DialogTitle: ({ children }: React.PropsWithChildren) => <h2>{children}</h2>,
  }
})

vi.mock("@/components/settings/support/support-flying-birds", () => ({
  SupportFlyingBirds: () => null,
  SupportGrowingBranch: ({ flowerScale, reveal, trigger }: { flowerScale: number; reveal: number; trigger: number }) => (
    <output
      data-testid="support-tree-stage"
      data-flower-scale={flowerScale}
      data-reveal={reveal}
      data-trigger={trigger}
    />
  ),
}))

describe("SupportPageContent", () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.runOnlyPendingTimers()
    vi.useRealTimers()
  })

  it("shows value first, reveals amounts after one contribution action, then opens the selected amount dialog", () => {
    render(<SupportPageContent />)

    act(() => {
      vi.runAllTimers()
    })

    expect(screen.getByTestId("support-value-first")).toBeInTheDocument()
    expect(screen.getByTestId("support-tier-options")).toHaveAttribute("aria-hidden", "true")

    const contributionButton = screen.getByRole("button", { name: "unlocked.primaryCta" })
    expect(contributionButton).toHaveAttribute("aria-expanded", "false")

    act(() => {
      fireEvent.click(contributionButton)
    })

    expect(contributionButton).toHaveAttribute("aria-expanded", "true")
    expect(screen.getByTestId("support-tier-options")).toHaveAttribute("aria-hidden", "false")
    expect(screen.getByTestId("support-tier-custom")).toBeInTheDocument()

    act(() => {
      fireEvent.click(screen.getByTestId("support-tier-growth"))
    })

    expect(screen.getByRole("dialog")).toBeInTheDocument()
    expect(screen.getByTestId("support-dialog-title")).toHaveTextContent("cards.wechat.title · ¥68")
    expect(screen.getByTestId("support-dialog-amount")).toHaveTextContent(
      'cards.wechat.description:{"amount":"¥68"}'
    )
    expect(screen.getByTestId("payment-method-wechat")).toHaveAttribute("aria-pressed", "true")
    expect(screen.getByTestId("support-tier-growth")).toHaveAttribute("aria-pressed", "true")
    expect(screen.queryByText("dialog.completionNote")).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "dialog.doneButton" })).not.toBeInTheDocument()

    act(() => {
      fireEvent.click(screen.getByTestId("payment-method-contact"))
    })

    expect(screen.getByTestId("payment-method-contact")).toHaveAttribute("aria-pressed", "true")
    expect(screen.getByTestId("support-dialog-title")).toHaveTextContent("cards.contact.title")
    expect(screen.getByTestId("support-dialog-amount")).toHaveTextContent("cards.contact.description")
    expect(screen.getByRole("heading", { name: "dialog.title" })).toBeInTheDocument()
  })

  it("grows the ¥10 tree beyond the unselected support state", () => {
    render(<SupportPageContent />)

    act(() => {
      vi.runAllTimers()
    })

    const tree = screen.getByTestId("support-tree-stage")
    expect(tree).toHaveAttribute("data-reveal", "0.12")
    expect(tree).toHaveAttribute("data-flower-scale", "0.5")

    act(() => {
      fireEvent.click(screen.getByRole("button", { name: "unlocked.primaryCta" }))
    })
    act(() => {
      fireEvent.click(screen.getByTestId("support-tier-starter"))
    })

    expect(tree).toHaveAttribute("data-reveal", "0.34")
    expect(tree).toHaveAttribute("data-flower-scale", "0.8")
  })

  it("renders the same geometry-critical loading slots in loading and resolved states", () => {
    const slots = ["support-page-header", "support-page-value-band", "support-page-actions", "support-page-tier-options"]
    const loading = render(<SupportPageLoadingState />)

    for (const slot of slots) {
      expect(loading.container.querySelectorAll(`[data-loading-slot="${slot}"]`)).toHaveLength(1)
    }

    loading.unmount()

    const resolved = render(<SupportPageContent />)
    act(() => {
      vi.runAllTimers()
    })

    for (const slot of slots) {
      expect(resolved.container.querySelectorAll(`[data-loading-slot="${slot}"]`)).toHaveLength(1)
    }
  })
})
