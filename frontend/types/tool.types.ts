// Tool type enum
export type ToolType = 'opensource' | 'custom'

// Tool type definition
export interface Tool {
  id: number
  name: string
  type: ToolType
  repoUrl: string
  version: string
  description: string
  categoryNames: string[]
  directory: string
  installCommand: string
  updateCommand: string
  versionCommand: string
  createdAt: string
  updatedAt: string
}

// Tool category name to display name mapping
// All categories reference backend model design document
export const CategoryNameMap: Record<string, string> = {
  subdomain: 'Subdomain Scan',
  vulnerability: 'Vulnerability Scan',
  port: 'Port Scan',
  directory: 'Directory Scan',
  dns: 'DNS Resolution',
  http: 'HTTP Probe',
  crawler: 'Web Crawler',
  recon: 'Reconnaissance',
  fuzzer: 'Fuzzing',
  wordlist: 'Wordlist Generation',
  screenshot: 'Screenshot Tool',
  exploit: 'Exploitation',
  network: 'Network Scan',
  other: 'Other',
}

// Tool list response type
export interface GetToolsResponse {
  tools: Tool[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

// Create tool request type
export interface CreateToolRequest {
  name: string
  type: ToolType
  repoUrl?: string
  version?: string
  description?: string
  categoryNames?: string[]
  directory?: string
  installCommand?: string
  updateCommand?: string
  versionCommand?: string
}

// Update tool request type
export interface UpdateToolRequest {
  name?: string
  type?: ToolType
  repoUrl?: string
  version?: string
  description?: string
  categoryNames?: string[]
  directory?: string
  installCommand?: string
  updateCommand?: string
  versionCommand?: string
}

// Tool query parameters
// Backend sorts by update time descending, custom sorting not supported
export interface GetToolsParams {
  page?: number
  pageSize?: number
}

// Tool filter type
export type ToolFilter = 'all' | 'default' | 'custom'
