import { fireEvent, screen, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { useAgentClusterSummary, useSelectedAgentDetail } from "@/hooks/use-agents"
import { renderWithProviders } from "@/test/utils/render-with-providers"
import { AgentLicensePanel } from "../agent-license-panel"

function LicenseFlow({ nodeCount }: { nodeCount?: number } = {}) {
  const summary = useAgentClusterSummary()
  const selected = useSelectedAgentDetail("agents/4")
  if (!summary.data?.license) return null
  const count = nodeCount ?? summary.data.totalNodes
  return <>
    <output data-testid="eligibility">{selected.data?.executionEligibility}</output>
    <output data-testid="quota">{count}/{summary.data.agentLimit}</output>
    <AgentLicensePanel license={summary.data.license} nodeCount={count} visible={count >= summary.data.agentLimit || summary.data.license.state === "expired"} />
  </>
}

describe("Agent license flow", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["Date"] })
    vi.setSystemTime(new Date("2026-10-19T00:00:00.000Z"))
  })
  afterEach(() => vi.useRealTimers())

  it("explains a future Key without replacing the quota or clearing its input", async () => {
    renderWithProviders(<LicenseFlow />)
    fireEvent.click(await screen.findByRole("button", { name: "open" }))
    const input = screen.getByRole("textbox", { name: "keyLabel" })
    fireEvent.change(input, { target: { value: "LF-DEMO-NOT-YET-VALID" } })
    fireEvent.click(screen.getByRole("button", { name: "activate" }))
    await screen.findByText("notYetValidError")
    expect(input).toHaveValue("LF-DEMO-NOT-YET-VALID")
    expect(input).toBeEnabled()
    expect(screen.getByTestId("quota")).toHaveTextContent("10/3")
  })
  it("keeps success feedback open when the refreshed quota hides the banner", async () => {
    renderWithProviders(<LicenseFlow nodeCount={3} />)
    fireEvent.click(await screen.findByRole("button", { name: "open" }))
    fireEvent.change(screen.getByRole("textbox", { name: "keyLabel" }), { target: { value: "LF-DEMO-1M" } })
    fireEvent.click(screen.getByRole("button", { name: "activate" }))
    await screen.findByText("successTitle", {}, { timeout: 3000 })
    expect(screen.getByTestId("quota")).toHaveTextContent("3/10")
    expect(screen.queryByRole("region", { name: "sectionLabel" })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "return" }))
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(screen.queryByRole("button", { name: "manage" })).not.toBeInTheDocument()
  })

  it("retains invalid input, blocks duplicate submission and refreshes quota before showing success", async () => {
    renderWithProviders(<LicenseFlow />)
    fireEvent.click(await screen.findByRole("button", { name: "open" }))
    const input = screen.getByRole("textbox", { name: "keyLabel" })
    fireEvent.change(input, { target: { value: "wrong" } })
    fireEvent.click(screen.getByRole("button", { name: "activate" }))
    await waitFor(() => expect(input).toBeDisabled())
    expect(screen.getByRole("button", { name: "cancel" })).toBeDisabled()
    await screen.findByRole("alert")
    expect(input).toHaveValue("wrong")
    expect(screen.getByTestId("quota")).toHaveTextContent("10/3")
    expect(screen.getByTestId("eligibility")).toHaveTextContent("license_restricted")
    fireEvent.change(input, { target: { value: "LF-DEMO-2M" } })
    fireEvent.click(screen.getByRole("button", { name: "activate" }))
    await screen.findByText("successTitle", {}, { timeout: 3000 })
    expect(screen.getByTestId("quota")).toHaveTextContent("10/10")
    expect(screen.getByTestId("eligibility")).toHaveTextContent("eligible")
    expect(screen.getByText("successLimit:{\"limit\":10}")).toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "return" }))
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    const manage = screen.getByRole("button", { name: "manage" })
    fireEvent.click(manage)
    expect(screen.getByRole("textbox", { name: "keyLabel" })).toHaveValue("")
  })
})
