import { fireEvent, screen, waitFor } from "@testing-library/react"
import { HttpResponse, http } from "msw"
import { afterEach, describe, expect, it, vi } from "vitest"
import { McpConnectionSettings } from "../mcp-connection-settings"
import { renderWithProviders } from "@/test/utils/render-with-providers"
import { mcpKeyKeys } from "@/hooks/use-mcp-key"
import { McpKeyService } from "@/services/mcp-key.service"
import { mockServer } from "@/mock/server"

afterEach(() => vi.restoreAllMocks())

describe("MCP page credential lifecycle", () => {
  it("only caches metadata and discards the revealed key when leaving the page", async () => {
    const first = renderWithProviders(<McpConnectionSettings client="claudeCode" onClientChange={() => {}} />)
    fireEvent.click(await screen.findByRole("button", { name: "key.generate" }))
    await screen.findByDisplayValue("lf_mcp_mock_once_1")
    expect(screen.getByText(/Bearer lf_mcp_mock_once_1/)).toBeInTheDocument()
    expect(screen.getByDisplayValue(`${window.location.origin}/mcp`)).toBeInTheDocument()
    expect(first.queryClient.getQueryData(mcpKeyKeys.status)).toEqual({ configured: true, createdAt: expect.any(String), updatedAt: expect.any(String) })
    first.unmount()

    renderWithProviders(<McpConnectionSettings client="claudeCode" onClientChange={() => {}} />, { queryClient: first.queryClient })
    await screen.findByRole("button", { name: "key.regenerate" })
    expect(screen.queryByDisplayValue("lf_mcp_mock_once_1")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "configuration.copy" })).toBeDisabled()
  })

  it("never rotates on cancel and only reveals a new key after explicit confirmation", async () => {
    const generate = vi.spyOn(McpKeyService, "generate")
    renderWithProviders(<McpConnectionSettings client="cursor" onClientChange={() => {}} />)
    fireEvent.click(await screen.findByRole("button", { name: "key.generate" }))
    await screen.findByDisplayValue("lf_mcp_mock_once_1")
    fireEvent.click(screen.getByRole("button", { name: "page.hideKey" }))
    fireEvent.click(screen.getByRole("button", { name: "key.regenerate" }))
    await screen.findByRole("alertdialog")
    fireEvent.click(screen.getByRole("button", { name: "key.confirmation.cancel" }))
    await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument())
    expect(generate).toHaveBeenCalledTimes(1)

    fireEvent.click(screen.getByRole("button", { name: "key.regenerate" }))
    fireEvent.click(await screen.findByRole("button", { name: "key.confirmation.confirm" }))
    await screen.findByDisplayValue("lf_mcp_mock_once_2")
    expect(generate).toHaveBeenCalledTimes(2)
    expect(screen.getByText("key.rotatedDescription")).toBeInTheDocument()
  })

  it("fails visibly and can retry if key status is unavailable", async () => {
    mockServer.use(http.get("*/v1/users/me/mcpKey", () => HttpResponse.json({}, { status: 500 }), { once: true }))
    renderWithProviders(<McpConnectionSettings client="vscode" onClientChange={() => {}} />)
    await screen.findByRole("alert")
    expect(screen.queryByRole("button", { name: "key.generate" })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "key.retry" }))
    expect(await screen.findByRole("button", { name: "key.generate" })).toBeEnabled()
  })
})
