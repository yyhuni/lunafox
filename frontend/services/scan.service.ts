import { api } from '@/lib/api-client'
import { agentName, organizationName, parseAgentName, parseScanWorkflowName, scanName, scanWorkflowName, targetName } from '@/lib/resource-name'
import { parseWorkflowConfigurationStrict } from '@/lib/workflow-config'
import { isScanInputSource } from '@/types/scan.types'
import type {
	EngineDiagnosticAvailability,
	EngineDiagnosticErrorType,
	EngineDiagnosticFailedStage,
	EngineDiagnosticResultState,
	EngineExecutionDiagnostics,
	GetScanLogsParams,
	GetScanLogsResponse,
	GetScansParams,
  GetScansResponse,
	BulkInitiateScanRequest,
	BulkInitiateScanResponse,
	BatchStopScansResponse,
  InitiateScanRequest,
  InitiateScanResponse,
  QuickScanRequest,
  QuickScanResponse,
	ScanRecord,
	ScanInputSource,
	ScanTriggerType,
	ResultTypeWatermark,
	RuntimeTask,
} from '@/types/scan.types'

const engineDiagnosticAvailability = new Set<EngineDiagnosticAvailability>(['available', 'unavailable'])
const engineExecutionDiagnosticsCompatibilityRevision = 'engine-execution-diagnostics-r1'
const engineDiagnosticResultStates = new Set<EngineDiagnosticResultState>(['complete', 'partial', 'none', 'unknown'])
const engineDiagnosticFailedStages = new Set<EngineDiagnosticFailedStage>([
	'unknown',
	'plan_validation',
	'input_materialization',
	'resource_materialization',
	'context_build',
	'image_prepare',
	'container_create',
	'container_start',
	'container_wait',
	'container_cleanup',
	'protocol_bootstrap',
	'execution_projection',
	'handler',
	'progress_report',
	'result_encode',
	'result_submit',
])
const engineDiagnosticErrorTypes = new Set<EngineDiagnosticErrorType>([
	'execution_failed',
	'plan_validation_failed',
	'input_materialization_failed',
	'resource_materialization_failed',
	'context_build_failed',
	'image_prepare_failed',
	'container_create_failed',
	'container_start_failed',
	'container_wait_failed',
	'container_cleanup_failed',
	'protocol_bootstrap_failed',
	'handler_failed',
	'progress_report_failed',
	'result_encode_failed',
	'result_submit_failed',
	'timeout',
	'cancelled',
	'oom_killed',
	'agent_session_failed',
	'invalid_outcome',
])
const engineDiagnosticResultTypePattern = /^[a-z0-9](?:[a-z0-9_-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9_-]*[a-z0-9])?)*$/

type ScanAipDto = Omit<ScanRecord, 'plannedEngineIds' | 'triggerType' | 'inputSource' | 'runtimeTasks'> & {
	scanWorkflow?: string
	plannedEngineIds?: unknown
	triggerType?: unknown
	inputSource?: unknown
	runtimeTasks?: unknown
}

type GetScansAipResponse = Partial<Omit<GetScansResponse, 'results'>> & {
  results: ScanAipDto[]
}

type ScanWorkflowInput = {
  scanWorkflow: string
}

type BatchCreateScanAipResponse = Omit<BulkInitiateScanResponse, 'message' | 'scans'> & {
  scans: ScanAipDto[]
}

type QuickScanAipResponse = Omit<QuickScanResponse, 'scans'> & {
	scans: ScanAipDto[]
}

function asRecord(value: unknown, context: string): Record<string, unknown> {
	if (typeof value !== 'object' || value === null || Array.isArray(value)) {
		throw new Error(`${context} must be an object`)
	}
	return value as Record<string, unknown>
}

function assertOnlyFields(value: Record<string, unknown>, allowed: readonly string[], context: string) {
	for (const key of Object.keys(value)) {
		if (!allowed.includes(key)) {
			throw new Error(`${context} has an unsupported field: ${key}`)
		}
	}
}

function requiredString(value: Record<string, unknown>, field: string, context: string): string {
	const candidate = value[field]
	if (typeof candidate !== 'string' || candidate.trim() === '') {
		throw new Error(`${context} has an invalid ${field}`)
	}
	return candidate
}

function optionalString(value: Record<string, unknown>, field: string, context: string): string | undefined {
	if (value[field] === undefined) {
		return undefined
	}
	return requiredString(value, field, context)
}

function requiredNonNegativeSafeInteger(value: Record<string, unknown>, field: string, context: string): number {
	const candidate = value[field]
	if (typeof candidate !== 'number' || !Number.isSafeInteger(candidate) || candidate < 0) {
		throw new Error(`${context} has an invalid ${field}`)
	}
	return candidate
}

function parseResultTypeWatermarks(value: unknown): ResultTypeWatermark[] {
	if (!Array.isArray(value) || value.length > 16) {
		throw new Error('Scan detail response has invalid diagnostic resultTypeWatermarks')
	}

	const resultTypes = new Set<string>()
	return value.map((item, index) => {
		const context = `Scan detail response diagnostic watermark ${index}`
		const watermark = asRecord(item, context)
		assertOnlyFields(
			watermark,
			['resultType', 'receivedItems', 'encodedItems', 'submittedItems', 'acknowledgedItems', 'submittedBatches', 'acknowledgedBatches'],
			context,
		)
		const resultType = requiredString(watermark, 'resultType', context)
		if (resultType.length > 128 || !engineDiagnosticResultTypePattern.test(resultType) || resultTypes.has(resultType)) {
			throw new Error(`${context} has an invalid resultType`)
		}
		resultTypes.add(resultType)
		const receivedItems = requiredNonNegativeSafeInteger(watermark, 'receivedItems', context)
		const encodedItems = requiredNonNegativeSafeInteger(watermark, 'encodedItems', context)
		const submittedItems = requiredNonNegativeSafeInteger(watermark, 'submittedItems', context)
		const acknowledgedItems = requiredNonNegativeSafeInteger(watermark, 'acknowledgedItems', context)
		const submittedBatches = requiredNonNegativeSafeInteger(watermark, 'submittedBatches', context)
		const acknowledgedBatches = requiredNonNegativeSafeInteger(watermark, 'acknowledgedBatches', context)
		if (
			receivedItems < encodedItems ||
			encodedItems < submittedItems ||
			submittedItems < acknowledgedItems ||
			submittedBatches < acknowledgedBatches
		) {
			throw new Error(`${context} has out-of-order watermarks`)
		}
		return {
			resultType,
			receivedItems,
			encodedItems,
			submittedItems,
			acknowledgedItems,
			submittedBatches,
			acknowledgedBatches,
		}
	})
}

function parseEngineExecutionDiagnostics(value: unknown): EngineExecutionDiagnostics {
	const context = 'Scan detail response diagnostics'
	const diagnostics = asRecord(value, context)
	assertOnlyFields(diagnostics, ['compatibilityRevision', 'availability', 'resultState', 'failedStage', 'errorType', 'resultTypeWatermarks'], context)
	const compatibilityRevision = requiredString(diagnostics, 'compatibilityRevision', context)
	if (compatibilityRevision !== engineExecutionDiagnosticsCompatibilityRevision) {
		throw new Error(`${context} has an invalid compatibilityRevision`)
	}
	const availability = requiredString(diagnostics, 'availability', context)
	const resultState = requiredString(diagnostics, 'resultState', context)
	if (!engineDiagnosticAvailability.has(availability as EngineDiagnosticAvailability)) {
		throw new Error(`${context} has an invalid availability`)
	}
	if (!engineDiagnosticResultStates.has(resultState as EngineDiagnosticResultState)) {
		throw new Error(`${context} has an invalid resultState`)
	}
	const failedStage = optionalString(diagnostics, 'failedStage', context)
	const errorType = optionalString(diagnostics, 'errorType', context)
	if ((failedStage === undefined) !== (errorType === undefined)) {
		throw new Error(`${context} must set failedStage and errorType together`)
	}
	if (failedStage !== undefined && !engineDiagnosticFailedStages.has(failedStage as EngineDiagnosticFailedStage)) {
		throw new Error(`${context} has an invalid failedStage`)
	}
	if (errorType !== undefined && !engineDiagnosticErrorTypes.has(errorType as EngineDiagnosticErrorType)) {
		throw new Error(`${context} has an invalid errorType`)
	}
	const resultTypeWatermarks = diagnostics.resultTypeWatermarks === undefined
		? undefined
		: parseResultTypeWatermarks(diagnostics.resultTypeWatermarks)
	if (availability === 'unavailable' && (resultState !== 'unknown' || failedStage || errorType || resultTypeWatermarks?.length)) {
		throw new Error(`${context} contains Engine evidence while unavailable`)
	}
	return {
		compatibilityRevision: engineExecutionDiagnosticsCompatibilityRevision,
		availability: availability as EngineDiagnosticAvailability,
		resultState: resultState as EngineDiagnosticResultState,
		...(failedStage ? { failedStage: failedStage as EngineDiagnosticFailedStage } : {}),
		...(errorType ? { errorType: errorType as EngineDiagnosticErrorType } : {}),
		...(resultTypeWatermarks !== undefined ? { resultTypeWatermarks } : {}),
	}
}

function parseRuntimeTasks(value: unknown): RuntimeTask[] | undefined {
	if (value === undefined) {
		return undefined
	}
	if (!Array.isArray(value)) {
		throw new Error('Scan detail response has invalid runtimeTasks')
	}
	return value.map((item, index) => {
    const runtimeTask = asRecord(item, `Scan detail response runtime task ${index}`)
    if (runtimeTask.diagnostics === undefined) {
      return runtimeTask as unknown as RuntimeTask
		}
		if (!['succeeded', 'failed', 'cancelled', 'skipped'].includes(runtimeTask.status as string)) {
			throw new Error('Scan detail response has diagnostics on a non-terminal runtime task')
		}
		return {
			...runtimeTask,
			diagnostics: parseEngineExecutionDiagnostics(runtimeTask.diagnostics),
		} as RuntimeTask
	})
}

function normalizeScanWorkflowId(name: string): string {
  return name.startsWith('scanWorkflows/') ? parseScanWorkflowName(name) : name
}

function resolveSingleScanWorkflow(data: ScanWorkflowInput): string {
  if (!data.scanWorkflow) {
    throw new Error('scanWorkflow is required by the backend AIP endpoint')
  }
  return data.scanWorkflow
}

function validateScanAgentProjection(dto: ScanAipDto, requireDetail: boolean) {
  if (!requireDetail) {
    return
  }
  if (dto.assignmentMode !== 'automatic' && dto.assignmentMode !== 'pinned') {
    throw new Error('Scan detail response has an invalid assignmentMode')
  }

  if (dto.agent === undefined) {
    if (
      dto.agentId !== undefined ||
      dto.agentName !== undefined ||
      dto.agentStatus !== undefined ||
      dto.agentHealthState !== undefined ||
      dto.agentDeleted !== undefined
    ) {
      throw new Error('Scan detail response has Agent fields without a canonical Agent reference')
    }
    return
  }

  let resourceId: number
  try {
    resourceId = parseAgentName(dto.agent)
  } catch {
    throw new Error('Scan detail response has an invalid Agent resource name')
  }
  if (!Number.isSafeInteger(dto.agentId) || dto.agentId !== resourceId) {
    throw new Error('Scan detail response Agent identity does not match its resource name')
  }
  if (typeof dto.agentDeleted !== 'boolean') {
    throw new Error('Scan detail response is missing agentDeleted')
  }
  if (dto.agentStatus !== undefined && (typeof dto.agentStatus !== 'string' || dto.agentStatus.trim() === '')) {
    throw new Error('Scan detail response has an invalid agentStatus')
  }
  if (
    dto.agentHealthState !== undefined &&
    (typeof dto.agentHealthState !== 'string' || dto.agentHealthState.trim() === '')
  ) {
    throw new Error('Scan detail response has an invalid agentHealthState')
  }
  if (dto.agentDeleted) {
    if (dto.agentName !== undefined || dto.agentStatus !== undefined || dto.agentHealthState !== undefined) {
      throw new Error('Deleted Scan Agent projection must omit current Agent fields')
    }
    return
  }
  if (typeof dto.agentName !== 'string' || dto.agentName.trim() === '') {
    throw new Error('Existing Scan Agent projection is missing agentName')
  }
}

function parseScanTriggerType(value: unknown): ScanTriggerType {
  if (value === undefined) {
    throw new Error('Scan response is missing triggerType')
  }
  if (value !== 'manual' && value !== 'scheduled' && value !== 'ai') {
    throw new Error('Scan response has an invalid triggerType')
  }
  return value
}

function parseScanInputSource(value: unknown, context: string): ScanInputSource {
  if (value === undefined) {
    throw new Error(`${context} is missing inputSource`)
  }
  if (!isScanInputSource(value)) {
    throw new Error(`${context} has an invalid inputSource`)
  }
  return value
}

function toScanRecord(dto: ScanAipDto, requireDetail = false): ScanRecord {
  const scanWorkflow = dto.scanWorkflow ? normalizeScanWorkflowId(dto.scanWorkflow) : undefined
  const triggerType = parseScanTriggerType(dto.triggerType)
  const inputSource = parseScanInputSource(dto.inputSource, 'Scan response')
  if (!Array.isArray(dto.plannedEngineIds) || !dto.plannedEngineIds.every((engineId) => typeof engineId === "string")) {
    throw new Error("Scan response is missing plannedEngineIds")
  }
  validateScanAgentProjection(dto, requireDetail)

  return {
    id: dto.id,
    name: dto.name,
    targetId: dto.targetId,
    target: dto.target,
    cachedStats: dto.cachedStats,
    scanWorkflow,
    plannedEngineIds: dto.plannedEngineIds,
    triggerType,
    inputSource,
    createdAt: dto.createdAt,
    stoppedAt: dto.stoppedAt,
    status: dto.status,
    errorMessage: dto.errorMessage,
    failure: dto.failure,
    progress: dto.progress,
    currentStage: dto.currentStage,
    configuration: dto.configuration,
    resultsDir: dto.resultsDir,
    agentId: dto.agentId,
    agent: dto.agent,
    agentName: dto.agentName,
    agentStatus: dto.agentStatus,
    agentHealthState: dto.agentHealthState,
    assignmentMode: dto.assignmentMode,
    agentDeleted: dto.agentDeleted,
		runtimeTasks: parseRuntimeTasks(dto.runtimeTasks),
  }
}

function toScanTask(scan: ScanRecord) {
  return {
    id: scan.id,
    target: scan.targetId,
    scanWorkflow: scan.scanWorkflow,
    status: scan.status,
    inputSource: scan.inputSource,
    createdAt: scan.createdAt,
    updatedAt: scan.stoppedAt || scan.createdAt,
  }
}

export async function getScans(params?: GetScansParams): Promise<GetScansResponse> {
  const { pageSize: requestedPageSize } = params ?? {}
  const res = await api.get<GetScansAipResponse>('/scans', {
    params,
  })
  const total = res.data.total ?? res.data.totalSize ?? 0
  const page = res.data.page ?? 1
  const pageSize = res.data.pageSize ?? requestedPageSize ?? 20
  const totalPages = res.data.totalPages ?? Math.ceil(total / pageSize)
  return {
    ...res.data,
    total,
    page,
    pageSize,
    totalPages,
    results: res.data.results.map((scan) => toScanRecord(scan)),
  }
}

export async function getScan(id: number): Promise<ScanRecord> {
  const res = await api.get<ScanAipDto>(`/scans/${id}`)
  return toScanRecord(res.data, true)
}

export async function initiateScan(data: InitiateScanRequest): Promise<InitiateScanResponse> {
  const targetIds = Number.isFinite(data.targetId) ? [data.targetId as number] : []
  const organizationIds = Number.isFinite(data.organizationId) ? [data.organizationId as number] : []

  if (targetIds.length === 0 && organizationIds.length === 0) {
    throw new Error('targetId or organizationId is required')
  }

  return bulkInitiateScan({
    targetIds,
    organizationIds,
    configuration: data.configuration,
    scanWorkflow: data.scanWorkflow,
    agentId: data.agentId,
    inputSource: data.inputSource,
  })
}

export async function bulkInitiateScan(data: BulkInitiateScanRequest): Promise<BulkInitiateScanResponse> {
  const targetIds = data.targetIds?.filter((id) => Number.isFinite(id)) ?? []
  const organizationIds = data.organizationIds?.filter((id) => Number.isFinite(id)) ?? []

  if (targetIds.length === 0 && organizationIds.length === 0) {
    throw new Error('targetIds or organizationIds is required')
  }

  const workflow = resolveSingleScanWorkflow(data)
  const inputSource = parseScanInputSource(data.inputSource, 'Scan request')
  const res = await api.post<BatchCreateScanAipResponse>('/scans:batchCreate', {
    requests: [
      ...targetIds.map((targetId) => ({ target: targetName(targetId) })),
      ...organizationIds.map((organizationId) => ({ organization: organizationName(organizationId) })),
    ],
    scanWorkflow: scanWorkflowName(workflow),
    inputSource,
    configuration: parseWorkflowConfigurationStrict(data.configuration),
    ...(Number.isFinite(data.agentId) ? { agent: agentName(data.agentId as number) } : {}),
  })
  const scans = res.data.scans.map((scan) => toScanRecord(scan))
  return {
    message: 'Scan initiated successfully',
    count: res.data.createdCount ?? res.data.count ?? scans.length,
    createdCount: res.data.createdCount,
    scans: scans.map(toScanTask),
    skipped: res.data.skipped ?? [],
    failed: res.data.failed ?? [],
  }
}

export async function quickScan(data: QuickScanRequest): Promise<QuickScanResponse> {
  const workflow = resolveSingleScanWorkflow(data)
  const inputSource = parseScanInputSource(data.inputSource, 'Scan request')
  const createScanRequest = {
    targets: data.targets.map(t => t.name),
    scanWorkflow: scanWorkflowName(workflow),
    inputSource,
    configuration: parseWorkflowConfigurationStrict(data.configuration),
    ...(Number.isFinite(data.agentId) ? { agent: agentName(data.agentId as number) } : {}),
  }

  const res = await api.post<QuickScanAipResponse>('/scans:quickCreate', createScanRequest)
  return {
    ...res.data,
    scans: res.data.scans.map((scan) => toScanRecord(scan)).map(toScanTask),
  }
}

export async function deleteScan(id: number): Promise<void> {
  await api.delete(`/scans/${id}`)
}

export async function bulkDeleteScans(ids: number[]): Promise<{ message: string; deletedCount: number }> {
  const res = await api.post<{ message: string; deletedCount: number }>('/scans:batchDelete', {
    names: ids.map(scanName),
  })
  return res.data
}

function validateBatchStopScanIds(ids: number[]) {
	if (!Array.isArray(ids) || ids.length < 1 || ids.length > 100) {
		throw new Error('batchStop requires between 1 and 100 scan IDs')
	}

	const seen = new Set<number>()
	for (const id of ids) {
		if (!Number.isSafeInteger(id) || id <= 0) {
			throw new Error(`batchStop received an invalid scan ID: ${id}`)
		}
		if (seen.has(id)) {
			throw new Error(`batchStop received a duplicate scan ID: ${id}`)
		}
		seen.add(id)
	}
}

function parseBatchStopScansResponse(value: unknown): BatchStopScansResponse {
	const payload = asRecord(value, 'Batch stop response')
	assertOnlyFields(payload, ['stoppedCount', 'skippedCount', 'revokedTaskCount'], 'Batch stop response')
	return {
		stoppedCount: requiredNonNegativeSafeInteger(payload, 'stoppedCount', 'Batch stop response'),
		skippedCount: requiredNonNegativeSafeInteger(payload, 'skippedCount', 'Batch stop response'),
		revokedTaskCount: requiredNonNegativeSafeInteger(payload, 'revokedTaskCount', 'Batch stop response'),
	}
}

export async function batchStopScans(ids: number[]): Promise<BatchStopScansResponse> {
	validateBatchStopScanIds(ids)
	const res = await api.post<unknown>('/scans:batchStop', {
		names: ids.map(scanName),
	})
	return parseBatchStopScansResponse(res.data)
}

export async function stopScan(id: number): Promise<{ message: string; revokedTaskCount: number }> {
  const res = await api.post<{ message: string; revokedTaskCount: number }>(`/scans/${id}:stop`)
  return res.data
}

export interface ScanStatistics {
  total: number
  pending: number
  running: number
  succeeded: number
  failed: number
  cancelled: number
  totalVulns: number
  totalSubdomains: number
  totalEndpoints: number
  totalWebsites: number
  totalAssets: number
  retentionPolicy: ScanHistoryRetentionPolicy
}

export interface ScanHistoryRetentionPolicy {
  minimumRetentionSeconds: number
  automaticCleanupEnabled: boolean
}

type ScanStatisticsDto = Omit<ScanStatistics, 'succeeded'> & {
  completed?: number
  succeeded?: number
}

export async function getScanStatistics(): Promise<ScanStatistics> {
  const res = await api.get<ScanStatisticsDto>('/scanStatistics')
  const { completed, succeeded, retentionPolicy, ...rest } = res.data
  return {
    ...rest,
    retentionPolicy,
    succeeded: succeeded ?? completed ?? 0,
  }
}

export async function getScanLogs(scanId: number, params?: GetScanLogsParams): Promise<GetScanLogsResponse> {
  const res = await api.get<GetScanLogsResponse>(`/scans/${scanId}/taskProgressLogs`, { params })
  return res.data
}
