import { Tool } from "./tool.types"

/**
 * Command model
 */
export interface Command {
  id: number
  createdAt: string
  updatedAt: string
  toolId: number
  tool?: Tool
  name: string
  displayName: string
  description: string
  commandTemplate: string
}

/**
 * Get commands list request parameters
 */
export interface GetCommandsRequest {
  page?: number
  pageSize?: number
  toolId?: number
}

/**
 * Get commands list response
 */
export interface GetCommandsResponse {
  commands: Command[]
  page: number
  pageSize: number      // Backend returns camelCase format
  total: number         // Unified total field
  totalPages: number    // Backend returns camelCase format
  // Compatibility fields (backward compatible)
  page_size?: number
  total_count?: number
  total_pages?: number
}

/**
 * Create command request
 */
export interface CreateCommandRequest {
  toolId: number
  name: string
  displayName?: string
  description?: string
  commandTemplate: string
}

/**
 * Update command request
 */
export interface UpdateCommandRequest {
  name?: string
  displayName?: string
  description?: string
  commandTemplate?: string
}

/**
 * Command response data
 */
export interface CommandResponseData {
  command: Command
}

/**
 * Batch delete commands response data
 */
export interface BatchDeleteCommandsResponseData {
  deletedCount: number
}
