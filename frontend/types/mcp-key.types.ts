export interface McpKeyStatus {
  configured: boolean
  createdAt?: string
  updatedAt?: string
}

export interface McpKeyGeneration extends McpKeyStatus {
  key: string
  createdAt: string
  updatedAt: string
}
