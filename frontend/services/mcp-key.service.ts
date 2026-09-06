import { api } from "@/lib/api-client"
import type { McpKeyGeneration, McpKeyStatus } from "@/types/mcp-key.types"

const currentMcpKeyPath = "/users/me/mcpKey"

export class McpKeyService {
  static async getStatus(): Promise<McpKeyStatus> {
    const response = await api.get<McpKeyStatus>(currentMcpKeyPath)
    return response.data
  }

  static async generate(): Promise<McpKeyGeneration> {
    const response = await api.post<McpKeyGeneration>("/users/me:generateMcpKey")
    return response.data
  }
}
