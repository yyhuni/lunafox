import { api } from '@/lib/api-client'
import {
  agentName,
  organizationName,
	parseScanWorkflowName,
  scheduledScanName,
  scanWorkflowName,
	targetName,
} from '@/lib/resource-name'
import { parseWorkflowConfigurationStrict } from '@/lib/workflow-config'
import type {
  GetScheduledScansResponse,
	ScheduledScan,
	CreateScheduledScanRequest,
	UpdateScheduledScanRequest,
	ScheduledScanOverviewSummary,
	ScheduledScanOverviewUpcoming,
	ScanMode,
	BatchUpdateScheduledScanStatusInput,
	BatchUpdateScheduledScanStatusRequest,
	BatchUpdateScheduledScanStatusResponse,
} from '@/types/scheduled-scan.types'
import { isScanInputSource, type ScanInputSource } from '@/types/scan.types'

type ScheduledScanAipDto = Omit<ScheduledScan, "successfulHandoffCount" | "failedHandoffCount" | "inputSource"> & {
  successfulHandoffCount?: unknown
  failedHandoffCount?: unknown
  inputSource?: unknown
}

type GetScheduledScansAipResponse = Omit<GetScheduledScansResponse, "scheduledScans"> & {
  scheduledScans: ScheduledScanAipDto[]
}

function parseNonNegativeInteger(value: unknown, field: string): number {
  if (typeof value !== "number" || !Number.isSafeInteger(value) || value < 0) {
    throw new Error(`Scheduled scan response has an invalid ${field}`)
  }
  return value
}

function parseInputSource(value: unknown, context: string): ScanInputSource {
  if (value === undefined) {
    throw new Error(`${context} is missing inputSource`)
  }
  if (!isScanInputSource(value)) {
    throw new Error(`${context} has an invalid inputSource`)
  }
  return value
}

function overviewResponseError(field: string): Error {
	return new Error(`Scheduled scan overview response has an invalid ${field}`)
}

function requiredOverviewString(record: Record<string, unknown>, field: string): string {
	const value = record[field]
	if (typeof value !== "string" || !value.trim()) throw overviewResponseError(field)
	return value
}

function requiredOverviewTimestamp(record: Record<string, unknown>, field: string): string {
	const value = requiredOverviewString(record, field)
	if (Number.isNaN(Date.parse(value))) throw overviewResponseError(field)
	return value
}

function nullableOverviewString(record: Record<string, unknown>, field: string): string | null {
	const value = record[field]
	if (value === null) return null
	if (typeof value !== "string" || !value.trim()) throw overviewResponseError(field)
	return value
}

function parseScheduledScanOverviewUpcoming(value: unknown, index: number): ScheduledScanOverviewUpcoming {
	if (typeof value !== "object" || value === null || Array.isArray(value)) {
		throw overviewResponseError(`upcomingScheduledScans[${index}]`)
	}
	const record = value as Record<string, unknown>
	const resourceName = requiredOverviewString(record, "name")
	const match = /^scheduledScans\/([1-9]\d*)$/.exec(resourceName)
	if (!match) throw overviewResponseError(`upcomingScheduledScans[${index}].name`)
	const id = Number(match[1])
	if (!Number.isSafeInteger(id)) throw overviewResponseError(`upcomingScheduledScans[${index}].name`)

	const organization = nullableOverviewString(record, "organization")
	const organizationName = nullableOverviewString(record, "organizationDisplayName")
	const target = nullableOverviewString(record, "target")
	const targetName = nullableOverviewString(record, "targetDisplayName")
	let scanMode: ScanMode
	if (target !== null && targetName === null) {
		throw overviewResponseError(`upcomingScheduledScans[${index}].targetDisplayName`)
	}
	if (organization !== null && organizationName === null) {
		throw overviewResponseError(`upcomingScheduledScans[${index}].organizationDisplayName`)
	}
	if (target === null && targetName !== null) {
		throw overviewResponseError(`upcomingScheduledScans[${index}].target`)
	}
	if (organization === null && organizationName !== null) {
		throw overviewResponseError(`upcomingScheduledScans[${index}].organization`)
	}
	if (target !== null && organization === null) {
		scanMode = "target"
	} else if (organization !== null && target === null) {
		scanMode = "organization"
	} else {
		throw overviewResponseError(`upcomingScheduledScans[${index}].scope`)
	}

	return {
		id,
		resourceName,
		displayName: requiredOverviewString(record, "displayName"),
		scanMode,
		organizationName,
		targetName,
		nextRunTime: requiredOverviewTimestamp(record, "nextRunTime"),
	}
}

function parseScheduledScanOverviewSummary(value: unknown): ScheduledScanOverviewSummary {
	if (typeof value !== "object" || value === null || Array.isArray(value)) {
		throw overviewResponseError("body")
	}
	const record = value as Record<string, unknown>
	const upcoming = record.upcomingScheduledScans
	if (!Array.isArray(upcoming) || upcoming.length > 5) {
		throw overviewResponseError("upcomingScheduledScans")
	}
	return {
		asOfTime: requiredOverviewTimestamp(record, "asOfTime"),
		enabledScheduledScanCount: parseNonNegativeInteger(record.enabledScheduledScanCount, "enabledScheduledScanCount"),
		pausedScheduledScanCount: parseNonNegativeInteger(record.pausedScheduledScanCount, "pausedScheduledScanCount"),
		todayScheduledScanCount: parseNonNegativeInteger(record.todayScheduledScanCount, "todayScheduledScanCount"),
		next24HoursScheduledScanCount: parseNonNegativeInteger(record.next24HoursScheduledScanCount, "next24HoursScheduledScanCount"),
		upcomingScheduledScans: upcoming.map(parseScheduledScanOverviewUpcoming),
	}
}

function toBatchUpdateScheduledScanStatusPayload(
  input: BatchUpdateScheduledScanStatusInput
): BatchUpdateScheduledScanStatusRequest {
  if (typeof input.isEnabled !== 'boolean') {
    throw new Error('Batch scheduled scan status update requires an explicit isEnabled value')
  }
  if (!Array.isArray(input.ids) || input.ids.length < 1 || input.ids.length > 100) {
    throw new Error('Batch scheduled scan status update requires between 1 and 100 IDs')
  }

  const uniqueIds = new Set<number>()
  for (const id of input.ids) {
    if (!Number.isSafeInteger(id) || id <= 0) {
      throw new Error(`Batch scheduled scan status update has an invalid ID: ${id}`)
    }
    if (uniqueIds.has(id)) {
      throw new Error(`Batch scheduled scan status update contains a duplicate ID: ${id}`)
    }
    uniqueIds.add(id)
  }

  return {
    requests: input.ids.map((id) => ({
      name: scheduledScanName(id),
      isEnabled: input.isEnabled,
      updateMask: 'isEnabled',
    })),
  }
}

function parseBatchUpdateScheduledScanStatusResponse(
  value: unknown,
  requestedCount: number
): BatchUpdateScheduledScanStatusResponse {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    throw new Error('Scheduled scan batch status response has an invalid body')
  }

  const updatedCount = parseNonNegativeInteger(
    (value as Record<string, unknown>).updatedCount,
    'batch updatedCount'
  )
  if (updatedCount !== requestedCount) {
    throw new Error('Scheduled scan batch status response has an unexpected updatedCount')
  }

  return { updatedCount }
}

function normalizeScheduledScan(scan: ScheduledScanAipDto): ScheduledScan {
  return {
    ...scan,
    inputSource: parseInputSource(scan.inputSource, 'Scheduled scan response'),
    successfulHandoffCount: parseNonNegativeInteger(scan.successfulHandoffCount, "successfulHandoffCount"),
    failedHandoffCount: parseNonNegativeInteger(scan.failedHandoffCount, "failedHandoffCount"),
    name: scan.displayName,
    resourceName: scan.name,
    scanWorkflow: scan.scanWorkflow.startsWith('scanWorkflows/')
      ? parseScanWorkflowName(scan.scanWorkflow)
      : scan.scanWorkflow,
  }
}

function toCreateScheduledScanPayload(data: CreateScheduledScanRequest) {
  return {
    displayName: data.displayName,
    scanWorkflow: scanWorkflowName(data.scanWorkflow),
    inputSource: parseInputSource(data.inputSource, 'Scheduled scan request'),
    configuration: parseWorkflowConfigurationStrict(data.configuration),
    ...(data.organizationId ? { organization: organizationName(data.organizationId) } : {}),
    ...(data.targetId ? { target: targetName(data.targetId) } : {}),
    ...(Number.isFinite(data.agentId) ? { agent: agentName(data.agentId as number) } : {}),
    cronExpression: data.cronExpression,
    ...(data.isEnabled !== undefined ? { isEnabled: data.isEnabled } : {}),
  }
}

function toUpdateScheduledScanPayload(id: number, data: UpdateScheduledScanRequest) {
  const payload: Record<string, unknown> = {
    name: scheduledScanName(id),
  }
  const updateMask: string[] = []

  if (data.displayName !== undefined) {
    payload.displayName = data.displayName
    updateMask.push('displayName')
  }
  if (data.scanWorkflow !== undefined) {
    payload.scanWorkflow = scanWorkflowName(data.scanWorkflow)
    updateMask.push('scanWorkflow')
  }
  if (data.inputSource !== undefined) {
    payload.inputSource = parseInputSource(data.inputSource, 'Scheduled scan request')
    updateMask.push('inputSource')
  }
  if (data.configuration !== undefined) {
    payload.configuration = parseWorkflowConfigurationStrict(data.configuration)
    updateMask.push('configuration')
  }
  if (data.organizationId !== undefined) {
    payload.organization = organizationName(data.organizationId)
    updateMask.push('organization')
  }
  if (data.targetId !== undefined) {
    payload.target = targetName(data.targetId)
    updateMask.push('target')
  }
  if (data.agentId !== undefined) {
    payload.agent = data.agentId === null ? '' : agentName(data.agentId)
    updateMask.push('agent')
  }
  if (data.cronExpression !== undefined) {
    payload.cronExpression = data.cronExpression
    updateMask.push('cronExpression')
  }
  if (data.isEnabled !== undefined) {
    payload.isEnabled = data.isEnabled
    updateMask.push('isEnabled')
  }

  if (updateMask.length === 0) {
    throw new Error('updateScheduledScan requires at least one field')
  }

  return {
    ...payload,
    updateMask: updateMask.join(','),
  }
}

/**
 * Get scheduled scan list
 */
export async function getScheduledScans(params?: {
  /** Legacy callers still using numbered pagination are kept compatible until their surfaces migrate. */
  page?: number
  pageSize?: number
  pageToken?: string
  search?: string
  targetId?: number
  organizationId?: number
}): Promise<GetScheduledScansResponse> {
  const apiParams: Record<string, unknown> = {}
  if (params?.page) apiParams.page = params.page
  if (params?.pageSize) apiParams.pageSize = params.pageSize
  if (params?.pageToken) apiParams.pageToken = params.pageToken
  if (params?.search) apiParams.filter = params.search
  if (params?.targetId) apiParams.targetId = params.targetId
  if (params?.organizationId) apiParams.organizationId = params.organizationId

  const res = await api.get<GetScheduledScansAipResponse>('/scheduledScans', { params: apiParams })
  return {
    ...res.data,
    scheduledScans: res.data.scheduledScans.map(normalizeScheduledScan),
  }
}

export async function getScheduledScanOverviewSummary(): Promise<ScheduledScanOverviewSummary> {
	const res = await api.get<unknown>("/scheduledScans:summarize", { params: {} })
	return parseScheduledScanOverviewSummary(res.data)
}

/**
 * Get scheduled scan details
 */
export async function getScheduledScan(id: number): Promise<ScheduledScan> {
  const res = await api.get<ScheduledScanAipDto>(`/scheduledScans/${id}`)
  return normalizeScheduledScan(res.data)
}

/**
 * Create scheduled scan
 */
export async function createScheduledScan(data: CreateScheduledScanRequest): Promise<ScheduledScan> {
  const res = await api.post<ScheduledScanAipDto>('/scheduledScans', toCreateScheduledScanPayload(data))
  return normalizeScheduledScan(res.data)
}

/**
 * Update scheduled scan
 */
export async function updateScheduledScan(id: number, data: UpdateScheduledScanRequest): Promise<ScheduledScan> {
  const res = await api.patch<ScheduledScanAipDto>(
    `/scheduledScans/${id}`,
    toUpdateScheduledScanPayload(id, data)
  )
  return normalizeScheduledScan(res.data)
}

/**
 * Delete scheduled scan
 */
export async function deleteScheduledScan(id: number): Promise<{ message: string; id: number }> {
  await api.delete(`/scheduledScans/${id}`)
  return { message: 'ok', id }
}

/**
 * Delete scheduled scans in batches by reusing the RESTful item delete endpoint.
 */
export async function batchDeleteScheduledScans(ids: number[]): Promise<{ message: string; deletedCount: number }> {
  await Promise.all(ids.map((id) => deleteScheduledScan(id)))
  return {
    message: 'ok',
    deletedCount: ids.length,
  }
}

/**
 * Assign one explicit enabled state to a bounded set of Scheduled Scans.
 */
export async function batchUpdateScheduledScanStatus(
  input: BatchUpdateScheduledScanStatusInput
): Promise<BatchUpdateScheduledScanStatusResponse> {
  const payload = toBatchUpdateScheduledScanStatusPayload(input)
  const res = await api.post<unknown>('/scheduledScans:batchUpdate', payload)
  return parseBatchUpdateScheduledScanStatusResponse(res.data, payload.requests.length)
}

/**
 * Toggle scheduled scan enabled status
 */
export async function toggleScheduledScan(id: number, isEnabled: boolean): Promise<{
  message: string
  scheduledScan: ScheduledScan
}> {
  const scheduledScan = await updateScheduledScan(id, { isEnabled })
  return { message: 'ok', scheduledScan }
}
