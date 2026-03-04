import { api } from '@/lib/api-client'
import type {
  GetScansParams,
  GetScansResponse,
  InitiateScanRequest,
  InitiateScanResponse,
  QuickScanRequest,
  QuickScanResponse,
  ScanRecord
} from '@/types/scan.types'
import { USE_MOCK, mockDelay, getMockScans, getMockScanById, mockScanStatistics } from '@/mock'

/**
 * Get scan list
 */
export async function getScans(params?: GetScansParams): Promise<GetScansResponse> {
  if (USE_MOCK) {
    await mockDelay()
    return getMockScans(params)
  }
  const res = await api.get<GetScansResponse>('/scans/', { params })
  return res.data
}

/**
 * Get single scan details
 * @param id - Scan ID
 * @returns Scan details
 */
export async function getScan(id: number): Promise<ScanRecord> {
  if (USE_MOCK) {
    await mockDelay()
    const scan = getMockScanById(id)
    if (!scan) throw new Error('Scan not found')
    return scan
  }
  const res = await api.get<ScanRecord>(`/scans/${id}/`)
  return res.data
}

/**
 * Initiate scan task (for existing targets/organizations)
 * @param data - Scan request parameters
 * @returns Scan task information
 */
export async function initiateScan(data: InitiateScanRequest): Promise<InitiateScanResponse> {
  // Backend endpoint: POST /api/scans/normal
  if (!data.targetId) {
    throw new Error('targetId is required')
  }

  const createScanRequest = {
    targetId: data.targetId,
    workflowIds: data.workflowIds,
    workflowNames: data.workflowNames,
    configuration: data.configuration,
  }

  // Backend returns ScanDetailResponse (wrapped in dto.Success), convert to InitiateScanResponse
  const res = await api.post<ScanRecord>('/scans/normal', createScanRequest)

  // Convert response format: ScanRecord -> InitiateScanResponse
  // Backend returns the scan record directly (via dto.Created)
  const scan = res.data

  return {
    message: 'Scan initiated successfully',
    count: 1,
    scans: [{
      id: scan.id,
      target: scan.targetId,
      workflowIds: scan.workflowIds,
      workflowNames: scan.workflowNames,
      status: scan.status,
      createdAt: scan.createdAt,
      updatedAt: scan.stoppedAt || scan.createdAt, // Use stoppedAt if available, otherwise createdAt
    }],
  }
}

/**
 * Quick scan (automatically create target and scan immediately)
 * @param data - Quick scan request parameters
 * @returns Scan task information
 */
export async function quickScan(data: QuickScanRequest): Promise<QuickScanResponse> {
  // Backend endpoint: POST /api/scans/quick
  const createScanRequest = {
    targets: data.targets.map(t => t.name),
    workflowIds: data.workflowIds,
    workflowNames: data.workflowNames,
    configuration: data.configuration,
  }

  const res = await api.post<QuickScanResponse>('/scans/quick', createScanRequest)
  return res.data
}

/**
 * Delete single scan record
 * @param id - Scan ID
 */
export async function deleteScan(id: number): Promise<void> {
  await api.delete(`/scans/${id}/`)
}

/**
 * Bulk delete scan records
 * @param ids - Array of scan IDs
 * @returns Deletion result
 */
export async function bulkDeleteScans(ids: number[]): Promise<{ message: string; deletedCount: number }> {
  const res = await api.post<{ message: string; deletedCount: number }>('/scans/deletions/', { ids })
  return res.data
}

/**
 * Stop scan task
 * @param id - Scan ID
 * @returns Operation result
 */
export async function stopScan(id: number): Promise<{ message: string; revokedTaskCount: number }> {
  const res = await api.post<{ message: string; revokedTaskCount: number }>(`/scans/${id}/stoppages/`)
  return res.data
}

/**
 * Scan statistics data type
 */
export interface ScanStatistics {
  total: number
  pending: number
  running: number
  completed: number
  failed: number
  cancelled: number
  totalVulns: number
  totalSubdomains: number
  totalEndpoints: number
  totalWebsites: number
  totalAssets: number
}

/**
 * Get scan statistics data
 * @returns Statistics data
 */
export async function getScanStatistics(): Promise<ScanStatistics> {
  if (USE_MOCK) {
    await mockDelay()
    return mockScanStatistics
  }
  const res = await api.get<ScanStatistics>('/scans/stats/')
  return res.data
}

/**
 * Scan log entry type
 */
export interface ScanLog {
  id: number
  level: 'info' | 'warning' | 'error'
  content: string
  createdAt: string
}

/**
 * Get scan logs response type
 */
export interface GetScanLogsResponse {
  results: ScanLog[]
  hasMore: boolean
}

/**
 * Get scan logs params type
 */
export interface GetScanLogsParams {
  afterId?: number
  limit?: number
}

/**
 * Get scan logs
 * @param scanId - Scan ID
 * @param params - Query parameters (afterId for pagination, limit for max results)
 * @returns Scan logs with hasMore indicator
 */
export async function getScanLogs(scanId: number, params?: GetScanLogsParams): Promise<GetScanLogsResponse> {
  const res = await api.get<GetScanLogsResponse>(`/scans/${scanId}/logs/`, { params })
  return res.data
}
