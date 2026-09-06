import { api } from "@/lib/api-client"
import type {
  GetCommandsRequest,
  GetCommandsResponse,
  CreateCommandRequest,
  UpdateCommandRequest,
  CommandResponseData,
  BatchDeleteCommandsResponseData,
} from "@/types/command.types"

/**
 * Command service
 */
export class CommandService {
  /**
   * Get command list
   */
  static async getCommands(
    params: GetCommandsRequest = {}
  ): Promise<GetCommandsResponse> {
    const response = await api.get<GetCommandsResponse>(
      "/commands/",
      { params }
    )
    return response.data
  }

  /**
   * Get single command
   */
  static async getCommandById(id: number): Promise<CommandResponseData> {
    const response = await api.get<CommandResponseData>(
      `/commands/${id}/`
    )
    return response.data
  }

  /**
   * Create command
   */
  static async createCommand(
    data: CreateCommandRequest
  ): Promise<CommandResponseData> {
    const response = await api.post<CommandResponseData>(
      "/commands/create/",
      data
    )
    return response.data
  }

  /**
   * Update command
   */
  static async updateCommand(
    id: number,
    data: UpdateCommandRequest
  ): Promise<CommandResponseData> {
    const response = await api.put<CommandResponseData>(
      `/commands/${id}/`,
      data
    )
    return response.data
  }

  /**
   * Delete command
   */
  static async deleteCommand(
    id: number
  ): Promise<void> {
    await api.delete(
      `/commands/${id}/`
    )
  }

  /**
   * Batch delete commands
   */
  static async batchDeleteCommands(
    ids: number[]
  ): Promise<BatchDeleteCommandsResponseData> {
    const response = await api.post<BatchDeleteCommandsResponseData>(
      "/commands/batch-delete/",
      { ids }
    )
    return response.data
  }
}
