import { fireEvent, screen, waitFor } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { getMcpClientConfiguration, McpAccessPopover } from "@/components/mcp-access-popover"
import { renderWithProviders } from "@/test/utils/render-with-providers"

async function openPopover() {
  fireEvent.click(screen.getByRole("button", { name: "open" }))
  await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument())
}

describe("McpAccessPopover", () => {
  it("generates a one-time key, embeds it in client configuration, and clears it when closed", async () => {
    renderWithProviders(<McpAccessPopover />)

    await openPopover()
    await waitFor(() => expect(screen.getByRole("button", { name: "key.generate" })).toBeEnabled())
    fireEvent.click(screen.getByRole("button", { name: "key.generate" }))

    await waitFor(() => {
      expect(screen.getByDisplayValue("lf_mcp_mock_once_1")).toBeInTheDocument()
    })
    expect(screen.getByText(/Bearer lf_mcp_mock_once_1/)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "configuration.copy" })).toBeEnabled()

    fireEvent.click(screen.getByRole("button", { name: "open" }))
    await waitFor(() => {
      expect(screen.queryByDisplayValue("lf_mcp_mock_once_1")).not.toBeInTheDocument()
    })

    await openPopover()
    await waitFor(() => expect(screen.getByRole("button", { name: "key.regenerate" })).toBeEnabled())
    expect(screen.queryByDisplayValue("lf_mcp_mock_once_1")).not.toBeInTheDocument()
  })

  it("requires confirmation before rotation and only reveals the replacement after confirmation", async () => {
    renderWithProviders(<McpAccessPopover />)

    await openPopover()
    await waitFor(() => expect(screen.getByRole("button", { name: "key.generate" })).toBeEnabled())
    fireEvent.click(screen.getByRole("button", { name: "key.generate" }))
    await screen.findByDisplayValue("lf_mcp_mock_once_1")

    fireEvent.click(screen.getByRole("button", { name: "open" }))
    await openPopover()
    fireEvent.click(screen.getByRole("button", { name: "key.regenerate" }))

    await waitFor(() => expect(screen.getByRole("alertdialog")).toBeInTheDocument())
    expect(screen.getByText("key.confirmation.description")).toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "key.confirmation.cancel" }))

    await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument())
    expect(screen.queryByDisplayValue("lf_mcp_mock_once_2")).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "key.regenerate" }))
    await waitFor(() => expect(screen.getByRole("alertdialog")).toBeInTheDocument())
    fireEvent.click(screen.getByRole("button", { name: "key.confirmation.confirm" }))

    await waitFor(() => {
      expect(screen.getByDisplayValue("lf_mcp_mock_once_2")).toBeInTheDocument()
    })
    expect(screen.getByText("key.rotatedDescription")).toBeInTheDocument()
  })

  it("builds every client template with the real Bearer key and no environment placeholder", () => {
    const endpoint = "https://lunafox.example/mcp"
    const key = "lf_mcp_actual_once"

    for (const client of ["vscode", "cursor", "claudeCode", "codex"] as const) {
      const configuration = getMcpClientConfiguration(client, endpoint, key)

      expect(configuration).toContain(endpoint)
      expect(configuration).toContain(`Bearer ${key}`)
      expect(configuration).not.toContain("LUNAFOX_MCP_KEY")
      expect(configuration).not.toContain("?key=")
    }
  })
})
