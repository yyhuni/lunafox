/**
 * Scheduled scan type definitions
 */

import type { ScanWorkflowConfigurationValue } from '@/types/scan-workflow.types'
import type { ScanInputSource } from '@/types/scan.types'

// Scheduled scan status
export type ScheduledScanStatus = "active" | "paused" | "expired"

// Scan mode
export type ScanMode = 'organization' | 'target'

// Scheduled scan interface
export interface ScheduledScan {
  id: number
  name: string
  resourceName?: string
  displayName: string
  scanWorkflow: string
  inputSource: ScanInputSource
  engineNames?: string[]
  configuration?: ScanWorkflowConfigurationValue
  organization?: string | null
  organizationId: number | null // Organization ID (organization scan mode)
  organizationName: string | null // Organization name
  target?: string | null
  targetId: number | null // Target ID (target scan mode)
  targetName: string | null // Target name (target scan mode)
  agent?: string | null
  agentId?: number | null
  scanMode: ScanMode // Scan mode
  cronExpression: string // Cron expression
  isEnabled: boolean // Whether enabled
  nextRunTime: string | null // Persisted next trigger time; null while disabled
  lastRunTime: string | null // Most recent committed trigger-attempt time
  runCount: number // Committed trigger-attempt count
  successfulHandoffCount: number // Completed schedule-to-Scan handoff count
  failedHandoffCount: number // Observed non-complete schedule-to-Scan handoff count
  createdAt: string
  updatedAt: string
}

// Create scheduled scan request (organizationId and targetId are mutually exclusive)
export interface CreateScheduledScanRequest {
  displayName: string
  configuration: ScanWorkflowConfigurationValue
  scanWorkflow: string
  inputSource: ScanInputSource
  organizationId?: number // Organization scan mode
  targetId?: number // Target scan mode
  agentId?: number
  cronExpression: string // Cron expression, format: minute hour day month weekday
  isEnabled?: boolean
}

// Update scheduled scan request (organizationId and targetId are mutually exclusive)
export interface UpdateScheduledScanRequest {
  displayName?: string
  configuration?: ScanWorkflowConfigurationValue
  scanWorkflow?: string
  inputSource?: ScanInputSource
  organizationId?: number // Organization scan mode (clears targetId when set)
  targetId?: number // Target scan mode (clears organizationId when set)
  // `null` explicitly clears the persisted Agent selection for future triggers.
  agentId?: number | null
  cronExpression?: string
  isEnabled?: boolean
}

export interface BatchUpdateScheduledScanStatusInput {
  ids: number[]
  isEnabled: boolean
}

export interface BatchUpdateScheduledScanStatusRequestItem {
  name: string
  isEnabled: boolean
  updateMask: 'isEnabled'
}

export interface BatchUpdateScheduledScanStatusRequest {
  requests: BatchUpdateScheduledScanStatusRequestItem[]
}

export interface BatchUpdateScheduledScanStatusResponse {
  updatedCount: number
}

// API response
export interface GetScheduledScansResponse {
	scheduledScans: ScheduledScan[]
	nextPageToken?: string
	totalSize?: number
}

export interface ScheduledScanOverviewUpcoming {
	id: number
	resourceName: string
	displayName: string
	scanMode: ScanMode
	organizationName: string | null
	targetName: string | null
	nextRunTime: string
}

export interface ScheduledScanOverviewSummary {
	asOfTime: string
	enabledScheduledScanCount: number
	pausedScheduledScanCount: number
	todayScheduledScanCount: number
	next24HoursScheduledScanCount: number
	upcomingScheduledScans: ScheduledScanOverviewUpcoming[]
}
