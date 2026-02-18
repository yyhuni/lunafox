import apiClient from "@/lib/api-client"
import type { SystemLogResponse, LogFilesResponse } from "@/types/system-log.types"

const BASE_URL = "/system/logs"

export const systemLogService = {
  /**
   * Get log file list
   */
  async getLogFiles(): Promise<LogFilesResponse> {
    const response = await apiClient.get<LogFilesResponse>(`${BASE_URL}/files/`)
    return response.data
  },

  /**
   * Get log content
   */
  async getSystemLogs(params?: { file?: string; lines?: number }): Promise<SystemLogResponse> {
    const searchParams = new URLSearchParams()

    if (params?.file != null) {
      searchParams.set("file", params.file)
    }
    if (params?.lines != null) {
      searchParams.set("lines", String(params.lines))
    }

    const query = searchParams.toString()
    const url = query ? `${BASE_URL}/?${query}` : `${BASE_URL}/`

    const response = await apiClient.get<SystemLogResponse>(url)
    return response.data
  },
}
