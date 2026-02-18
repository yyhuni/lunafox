// Tool type enum
export type ToolType = 'opensource' | 'custom'

// Tool type definition (matches frontend camelCase converted format)
// Note: Backend returns snake_case, api-client.ts auto-converts to camelCase
export interface Tool {
  id: number
  name: string                 // Tool name
  type: ToolType              // Tool type: opensource/custom (backend: type)
  repoUrl: string             // Repository URL (backend: repo_url)
  version: string              // Version number
  description: string          // Tool description
  categoryNames: string[]      // Category tags array (backend: category_names)
  directory: string            // Tool path (backend: directory)
  installCommand: string       // Install command (backend: install_command)
  updateCommand: string        // Update command (backend: update_command)
  versionCommand: string       // Version query command (backend: version_command)
  createdAt: string           // Backend: created_at
  updatedAt: string           // Backend: updated_at
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

// Tool list response type (api-client.ts auto-converts to camelCase)
export interface GetToolsResponse {
  tools: Tool[]
  total: number
  page: number
  pageSize: number      // Backend returns camelCase format
  totalPages: number    // Backend returns camelCase format
  // Compatibility fields (backward compatible)
  page_size?: number
  total_pages?: number
}

// Create tool request type
export interface CreateToolRequest {
  name: string
  type: ToolType              // Tool type (required)
  repoUrl?: string
  version?: string
  description?: string
  categoryNames?: string[]    // Category tags array
  directory?: string          // Tool path (required for custom tools)
  installCommand?: string     // Install command (required for opensource tools)
  updateCommand?: string      // Update command (required for opensource tools)
  versionCommand?: string     // Version query command (required for opensource tools)
}

// Update tool request type
export interface UpdateToolRequest {
  name?: string
  type?: ToolType             // Tool type (for command field validation)
  repoUrl?: string
  version?: string
  description?: string
  categoryNames?: string[]    // Category tags array
  directory?: string          // Tool path
  installCommand?: string     // Install command
  updateCommand?: string      // Update command
  versionCommand?: string     // Version query command
}

// Tool query parameters
// Backend sorts by update time descending, custom sorting not supported
export interface GetToolsParams {
  page?: number
  pageSize?: number
}

// Tool filter type
export type ToolFilter = 'all' | 'default' | 'custom'
