import { api } from '@/lib/api-client'
import { normalizeWorkflowConfiguration } from '@/lib/workflow-config'
import type {
  GetScansParams,
  GetScansResponse,
  InitiateScanRequest,
  InitiateScanResponse,
  QuickScanRequest,
  QuickScanResponse,
  ScanRecord,
} from '@/types/scan.types'
import { USE_MOCK, mockDelay, getMockScans, getMockScanById, mockScanStatistics } from '@/mock'

export async function getScans(params?: GetScansParams): Promise<GetScansResponse> {
  if (USE_MOCK) {
    await mockDelay()
    return getMockScans(params)
  }
  const res = await api.get<GetScansResponse>('/scans/', { params })
  return res.data
}

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

export async function initiateScan(data: InitiateScanRequest): Promise<InitiateScanResponse> {
  if (!data.targetId) {
    throw new Error('targetId is required')
  }

  const createScanRequest = {
    targetId: data.targetId,
    workflowIds: data.workflowNames,
    configuration: normalizeWorkflowConfiguration(data.configuration),
  }

  const res = await api.post<ScanRecord>('/scans/normal', createScanRequest)
  const scan = res.data

  return {
    message: 'Scan initiated successfully',
    count: 1,
    scans: [{
      id: scan.id,
      target: scan.targetId,
      workflowNames: scan.workflowNames,
      status: scan.status,
      createdAt: scan.createdAt,
      updatedAt: scan.stoppedAt || scan.createdAt,
    }],
  }
}

export async function quickScan(data: QuickScanRequest): Promise<QuickScanResponse> {
  const createScanRequest = {
    targets: data.targets.map(t => t.name),
    workflowIds: data.workflowNames,
    configuration: normalizeWorkflowConfiguration(data.configuration),
  }

  const res = await api.post<QuickScanResponse>('/scans/quick', createScanRequest)
  return res.data
}

export async function deleteScan(id: number): Promise<void> {
  await api.delete(`/scans/${id}/`)
}

export async function bulkDeleteScans(ids: number[]): Promise<{ message: string; deletedCount: number }> {
  const res = await api.post<{ message: string; deletedCount: number }>('/scans/deletions/', { ids })
  return res.data
}

export async function stopScan(id: number): Promise<{ message: string; revokedTaskCount: number }> {
  const res = await api.post<{ message: string; revokedTaskCount: number }>(`/scans/${id}/stoppages/`)
  return res.data
}

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

export async function getScanStatistics(): Promise<ScanStatistics> {
  if (USE_MOCK) {
    await mockDelay()
    return mockScanStatistics
  }
  const res = await api.get<ScanStatistics>('/scans/stats/')
  return res.data
}

export interface ScanLog {
  id: number
  level: 'info' | 'warning' | 'error'
  content: string
  createdAt: string
}

export interface GetScanLogsResponse {
  results: ScanLog[]
  hasMore: boolean
}

export interface GetScanLogsParams {
  afterId?: number
  limit?: number
}

export async function getScanLogs(scanId: number, params?: GetScanLogsParams): Promise<GetScanLogsResponse> {
  const res = await api.get<GetScanLogsResponse>(`/scans/${scanId}/logs/`, { params })
  return res.data
}
