import { api } from "@/lib/api-client"
import type { Tool, GetToolsResponse, CreateToolRequest, UpdateToolRequest, GetToolsParams } from "@/types/tool.types"

export class ToolService {
  /**
   * Get tool list
   * @param params - Query parameter object
   * @param params.page - Current page number, 1-based
   * @param params.pageSize - Page size
   * @returns Promise<GetToolsResponse>
   * @description Backend is fixed to sort by update time in descending order, does not support custom sorting
   */
  static async getTools(params?: GetToolsParams): Promise<GetToolsResponse> {
    const response = await api.get<GetToolsResponse>(
      '/tools/',
      { params }
    )
    return response.data
  }

  /**
   * Create new tool
   * @param data - Tool information object
   * @param data.name - Tool name
   * @param data.repoUrl - Repository URL
   * @param data.version - Version number
   * @param data.description - Tool description
   * @returns Promise<{ tool: Tool }>
   */
  static async createTool(data: CreateToolRequest): Promise<{ tool: Tool }> {
    const response = await api.post<{ tool: Tool }>('/tools/create/', data)
    return response.data
  }

  /**
   * Update tool
   * @param id - Tool ID
   * @param data - Updated tool information (all fields optional)
   * @returns Promise<{ tool: Tool }>
   */
  static async updateTool(id: number, data: UpdateToolRequest): Promise<{ tool: Tool }> {
    const response = await api.put<{ tool: Tool }>(`/tools/${id}/`, data)
    return response.data
  }

  /**
   * Delete tool
   * @param id - Tool ID
   * @returns Promise<void>
   */
  static async deleteTool(id: number): Promise<void> {
    await api.delete(`/tools/${id}/`)
  }
}
