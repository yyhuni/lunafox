/**
 * Target type definitions
 */

/**
 * Target type
 */
export type TargetType = 'domain' | 'ip' | 'cidr'

/**
 * Target basic info (for list display)
 */
export interface Target {
  id: number
  name: string
  type: TargetType
  description?: string
  createdAt: string
  lastScannedAt?: string
  organizations?: Array<{
    id: number
    name: string
  }>
}

/**
 * Target detail info (includes statistics data)
 */
export interface TargetDetail extends Target {
  summary: {
    subdomains: number
    websites: number
    endpoints: number
    ips: number
    directories: number
    screenshots: number
    vulnerabilities: {
      total: number
      critical: number
      high: number
      medium: number
      low: number
    }
  }
}

/**
 * Target list response type
 */
export interface TargetsResponse {
  results: Target[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

/**
 * Create target request parameters
 */
export interface CreateTargetRequest {
  name: string
  description?: string
}

/**
 * Update target request parameters
 */
export interface UpdateTargetRequest {
  name?: string
  description?: string
}

/**
 * Batch delete targets request parameters
 */
export interface BatchDeleteTargetsRequest {
  ids: number[]
}

/**
 * Batch delete targets response
 */
export interface BatchDeleteTargetsResponse {
  deletedCount: number
  failedIds?: number[]
}

/**
 * Batch create targets request parameters
 */
export interface BatchCreateTargetsRequest {
  targets: Array<{
    name: string
    description?: string
  }>
  organizationId?: number
}

/**
 * Batch create targets response
 */
export interface BatchCreateTargetsResponse {
  createdCount: number
  reusedCount: number
  failedCount: number
  failedTargets: Array<{
    name: string
    reason: string
  }>
  message: string
}
