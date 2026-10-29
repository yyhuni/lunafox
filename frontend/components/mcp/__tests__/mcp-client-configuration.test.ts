import { describe, expect, it } from "vitest"
import { getMcpClientConfiguration } from "../mcp-client-configuration"

describe("MCP native client configuration", () => {
  const endpoint = "https://lunafox.example/mcp"
  const key = 'lf_mcp_once_"\\secret'

  it.each(["claudeCode", "cursor", "vscode"] as const)("keeps %s credentials in the header and preserves its native schema", (client) => {
    const config = JSON.parse(getMcpClientConfiguration(client, endpoint, key))
    const server = (client === "vscode" ? config.servers : config.mcpServers).lunafox
    expect(server.url).toBe(endpoint)
    expect(server.headers.Authorization).toBe(`Bearer ${key}`)
    expect(server.type).toBe(client === "cursor" ? "streamable-http" : "http")
  })

  it("escapes Codex TOML literals without putting the secret in the URL", () => {
    expect(getMcpClientConfiguration("codex", endpoint, key)).toBe([
      "[mcp_servers.lunafox]",
      'url = "https://lunafox.example/mcp"',
      'http_headers = { Authorization = "Bearer lf_mcp_once_\\"\\\\secret" }',
    ].join("\n"))
  })
})
