import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { AgentActivationPrototype } from "../agent-activation-prototype"

vi.mock("next-intl", () => ({ useTranslations: () => (key: string) => key }))
vi.mock("../agent-overview-section", () => ({ AgentOverviewSection: () => null }))

describe("Agent activation prototype", () => {
  it("submits the add dialog without clobbering DOM form properties or navigating", async () => {
    render(<AgentActivationPrototype />)
    fireEvent.click(screen.getByRole("button", { name: "已激活" }))
    fireEvent.click(screen.getByRole("button", { name: "添加 Agent" }))
    const input = screen.getByRole("textbox", { name: "节点名称" })
    fireEvent.change(input, { target: { value: "第四个演示节点" } })

    // Dialog focus management reads this native property before form submission.
    expect(input.closest("form")?.nodeName).toBe("FORM")
    fireEvent.click(screen.getByRole("button", { name: "添加节点" }))

    await waitFor(() => expect(screen.getByRole("button", { name: "删除 第四个演示节点" })).toBeInTheDocument())
    expect(screen.getByText("4/10 个节点 · 还可添加 6 个")).toBeInTheDocument()
    expect(screen.queryByRole("textbox", { name: "节点名称" })).not.toBeInTheDocument()
  })
})
