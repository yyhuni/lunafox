/**
 * Scheduled scan type definitions
 */

import type { ScanWorkflowConfigurationValue } from '@/types/scan-workflow.types'
import type { ScanInputSource } from '@/types/scan.types'

// Scheduled scan status
export type ScheduledScanStatus = "active" | "paused" | "expired"

// Scan mode
export type ScanMode = 'organization' | 'target'

// Closed public cause enum for zero-created handoff failures; server contract
// is additive-only, so this union only grows.
export const SCHEDULED_SCAN_HANDOFF_FAILURE_CAUSES = [
  'WORKFLOW_UNAVAILABLE',
  'AGENT_NOT_FOUND',
  'CONFIG_RESOURCE_UNAVAILABLE',
  'ENGINE_UNAVAILABLE',
  'TARGET_UNAVAILABLE',
  'INTERNAL_UNAVAILABLE',
] as const

export type ScheduledScanHandoffFailureCause = (typeof SCHEDULED_SCAN_HANDOFF_FAILURE_CAUSES)[number]

export function isScheduledScanHandoffFailureCause(value: unknown): value is ScheduledScanHandoffFailureCause {
  return (
    typeof value === 'string' &&
    (SCHEDULED_SCAN_HANDOFF_FAILURE_CAUSES as readonly string[]).includes(value)
  )
}

// Closed public status enum for the occurrence history projection; the
// server contract is additive-only, so this union only grows.
export const SCHEDULED_SCAN_OCCURRENCE_STATUSES = [
  'PENDING',
  'DISPATCHING',
  'RETRYING',
  'SUCCEEDED',
  'FAILED',
] as const

export type ScheduledScanOccurrenceStatus = (typeof SCHEDULED_SCAN_OCCURRENCE_STATUSES)[number]

export function isScheduledScanOccurrenceStatus(value: unknown): value is ScheduledScanOccurrenceStatus {
  return (
    typeof value === 'string' &&
    (SCHEDULED_SCAN_OCCURRENCE_STATUSES as readonly string[]).includes(value)
  )
}

// One retention-bounded occurrence history row. The status is derived
// server-side; durationMs exists only on SUCCEEDED rows.
export interface ScheduledScanOccurrence {
  name: string // Canonical nested resource name: scheduledScans/{scanId}/occurrences/{id}
  id: number
  scheduledFor: string
  attemptedAt: string | null
  dispatchedAt: string | null
  status: ScheduledScanOccurrenceStatus
  failureKind: string | null
  failureCause: ScheduledScanHandoffFailureCause | null
  failureMessage: string | null
  retryCount: number
  nextRetryAt: string | null
  durationMs: number | null
}

export interface ScheduledScanOccurrenceStatusCounts {
  pending: number
  dispatching: number
  retrying: number
  succeeded: number
  failed: number
}

export interface GetScheduledScanOccurrencesResponse {
  occurrences: ScheduledScanOccurrence[]
  statusCounts: ScheduledScanOccurrenceStatusCounts
  totalSize?: number
  nextPageToken?: string
}

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
  timeZone: string // IANA time zone used to evaluate the Cron rule
  cronExpression: string // Cron expression
  isEnabled: boolean // Whether enabled
  nextRunTime: string | null // Persisted next trigger time; null while disabled
  lastRunTime: string | null // Most recent committed trigger-attempt time
  runCount: number // Committed trigger-attempt count
  successfulHandoffCount: number // Completed schedule-to-Scan handoff count
  failedHandoffCount: number // Observed non-complete schedule-to-Scan handoff count
  lastHandoffFailureCause: ScheduledScanHandoffFailureCause | null // Final-failure cause of the latest failed handoff
  lastHandoffFailureTime: string | null // Settlement time of the latest failed handoff; null together with the cause
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
  timeZone: string // IANA time zone used to evaluate the Cron rule
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
  timeZone?: string
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

// Horizon items share the upcoming item shape minus its cursor timestamp.
export interface ScheduledScanHorizonItem {
	id: number
	resourceName: string
	displayName: string
	scanMode: ScanMode
	organizationName: string | null
	targetName: string | null
}

export interface ScheduledScanHorizonWindow {
	start: string
	end: string
}

export interface ScheduledScanHorizonBucket {
	hourStart: string
	items: ScheduledScanHorizonItem[]
}

export interface ScheduledScanOverviewSummary {
	asOfTime: string
	enabledScheduledScanCount: number
	pausedScheduledScanCount: number
	todayScheduledScanCount: number
	next24HoursScheduledScanCount: number
	upcomingScheduledScans: ScheduledScanOverviewUpcoming[]
	horizonWindow: ScheduledScanHorizonWindow
	horizonBuckets: ScheduledScanHorizonBucket[]
}
