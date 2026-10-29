import { fireEvent, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { McpConnectionWorkspace } from "../mcp-connection-workspace"
import { renderWithProviders } from "@/test/utils/render-with-providers"
import { McpKeyService } from "@/services/mcp-key.service"

describe("MCP client selection", () => {
  it("reuses the revealed key when switching between client configuration formats", async () => {
    const generate = vi.spyOn(McpKeyService, "generate")
    try {
      renderWithProviders(<McpConnectionWorkspace />)
      fireEvent.click(await screen.findByRole("button", { name: "key.generate" }))
      await screen.findByDisplayValue("lf_mcp_mock_once_1")

      fireEvent.click(screen.getByRole("tab", { name: "configuration.clients.codex" }))
      expect(screen.getByDisplayValue("lf_mcp_mock_once_1")).toBeInTheDocument()
      expect(screen.getByText(/\[mcp_servers.lunafox\]/)).toHaveTextContent('http_headers = { Authorization = "Bearer lf_mcp_mock_once_1" }')
      expect(screen.getByRole("button", { name: "configuration.copy" })).toBeEnabled()

      fireEvent.click(screen.getByRole("tab", { name: "configuration.clients.vscode" }))
      expect(JSON.parse(screen.getByLabelText("configuration.label", { selector: "pre" }).textContent ?? "")).toEqual({
        servers: { lunafox: { type: "http", url: `${window.location.origin}/mcp`, headers: { Authorization: "Bearer lf_mcp_mock_once_1" } } },
      })
      expect(generate).toHaveBeenCalledTimes(1)
    } finally {
      generate.mockRestore()
    }
  })
})
