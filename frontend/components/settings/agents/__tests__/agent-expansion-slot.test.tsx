import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { AgentExpansionSlot } from "../agent-expansion-slot"

describe("Agent expansion quota gate", () => {
  it("blocks installation while full and restores the action after capacity is released", () => {
    const onOpenInstall = vi.fn()
    const props = { title: "Add Agent", description: "Install a node", actionLabel: "Open installer", onOpenInstall }
    const { rerender } = render(<AgentExpansionSlot {...props} disabled />)
    const button = screen.getByRole("button", { name: "Add Agent" })
    expect(button).toBeDisabled()
    fireEvent.click(button)
    expect(onOpenInstall).not.toHaveBeenCalled()
    rerender(<AgentExpansionSlot {...props} disabled={false} />)
    expect(button).toBeEnabled()
    fireEvent.click(button)
    expect(onOpenInstall).toHaveBeenCalledOnce()
  })
})
