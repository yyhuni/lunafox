export const mcpClients = ["claudeCode", "cursor", "codex", "vscode"] as const
export type McpClient = (typeof mcpClients)[number]

export const mcpClientFiles: Record<McpClient, string> = {
  claudeCode: ".mcp.json",
  cursor: ".cursor/mcp.json",
  codex: "~/.codex/config.toml",
  vscode: ".vscode/mcp.json",
}

function escapeTomlString(value: string) {
  return value.replaceAll("\\", "\\\\").replaceAll('"', '\\"').replaceAll("\n", "\\n").replaceAll("\r", "\\r")
}

/** Keep secrets in the client-native Authorization header, never in the MCP URL. */
export function getMcpClientConfiguration(client: McpClient, endpoint: string, key: string) {
  const authorization = `Bearer ${key}`
  if (client === "codex") {
    return [
      "[mcp_servers.lunafox]",
      `url = "${escapeTomlString(endpoint)}"`,
      `http_headers = { Authorization = "${escapeTomlString(authorization)}" }`,
    ].join("\n")
  }
  return JSON.stringify({
    [client === "vscode" ? "servers" : "mcpServers"]: {
      lunafox: {
        type: client === "cursor" ? "streamable-http" : "http",
        url: endpoint,
        headers: { Authorization: authorization },
      },
    },
  }, null, 2)
}
