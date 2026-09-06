import type { McpKeyGeneration, McpKeyStatus } from "@/types/mcp-key.types"

let configuredAt: string | undefined
let mockGeneration = 0

export function getMockMcpKeyStatus(): McpKeyStatus {
  return configuredAt
    ? { configured: true, createdAt: configuredAt, updatedAt: configuredAt }
    : { configured: false }
}

export function generateMockMcpKey(): McpKeyGeneration {
  const timestamp = new Date().toISOString()
  mockGeneration += 1
  configuredAt = timestamp

  return {
    configured: true,
    key: `lf_mcp_mock_once_${mockGeneration}`,
    createdAt: timestamp,
    updatedAt: timestamp,
  }
}

export function resetMockMcpKey() {
  configuredAt = undefined
  mockGeneration = 0
}
