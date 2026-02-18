/**
 * Scheduled scan type definitions
 */

// Scheduled scan status
export type ScheduledScanStatus = "active" | "paused" | "expired"

// Scan mode
export type ScanMode = 'organization' | 'target'

// Scheduled scan interface
export interface ScheduledScan {
  id: number
  name: string
  engineIds: number[] // Associated scan engine ID list
  engineNames: string[] // Associated scan engine name list
  organizationId: number | null // Organization ID (organization scan mode)
  organizationName: string | null // Organization name
  targetId: number | null // Target ID (target scan mode)
  targetName: string | null // Target name (target scan mode)
  scanMode: ScanMode // Scan mode
  cronExpression: string // Cron expression
  isEnabled: boolean // Whether enabled
  nextRunTime?: string // Next run time
  lastRunTime?: string // Last run time
  runCount: number // Execution count
  createdAt: string
  updatedAt: string
}

// Create scheduled scan request (organizationId and targetId are mutually exclusive)
export interface CreateScheduledScanRequest {
  name: string
  configuration: string // YAML configuration string (required)
  engineIds: number[] // Engine ID list (required)
  engineNames: string[] // Engine name list (required)
  organizationId?: number // Organization scan mode
  targetId?: number // Target scan mode
  cronExpression: string // Cron expression, format: minute hour day month weekday
  isEnabled?: boolean
}

// Update scheduled scan request (organizationId and targetId are mutually exclusive)
export interface UpdateScheduledScanRequest {
  name?: string
  configuration?: string // YAML configuration string
  engineIds?: number[] // Engine ID list (optional, for reference)
  engineNames?: string[] // Engine name list (optional, for reference)
  organizationId?: number // Organization scan mode (clears targetId when set)
  targetId?: number // Target scan mode (clears organizationId when set)
  cronExpression?: string
  isEnabled?: boolean
}

// API response
export interface GetScheduledScansResponse {
  results: ScheduledScan[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}
