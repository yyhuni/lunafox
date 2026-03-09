import type { WorkflowConfigurationValue } from '@/types/workflow.types'

export type ScanStatus = "pending" | "running" | "completed" | "failed" | "cancelled"
export type ScanStage = string
export type StageStatus = "pending" | "running" | "completed" | "failed" | "cancelled"

export interface StageProgressItem { status: StageStatus; order: number; startedAt?: string; duration?: number; detail?: string; error?: string; reason?: string }
export type StageProgress = Record<string, StageProgressItem>
export interface ScanTargetBrief { id: number; name: string; type: string }
export interface ScanCachedStats { subdomainsCount: number; websitesCount: number; endpointsCount: number; ipsCount: number; directoriesCount: number; screenshotsCount: number; vulnsTotal: number; vulnsCritical: number; vulnsHigh: number; vulnsMedium: number; vulnsLow: number }
export interface FailureDetail { kind: string; message: string }

export interface ScanRecord {
  id: number
  targetId: number
  target?: ScanTargetBrief
  workerName?: string | null
  cachedStats?: ScanCachedStats
  workflowNames: string[]
  scanMode: string
  createdAt: string
  stoppedAt?: string
  status: ScanStatus
  errorMessage?: string
  failure?: FailureDetail
  progress: number
  currentStage?: ScanStage
  stageProgress?: StageProgress
  configuration?: WorkflowConfigurationValue
  resultsDir?: string
  workerId?: number
}

export interface GetScansParams { page?: number; pageSize?: number; status?: ScanStatus; search?: string; target?: number }
export interface GetScansResponse { results: ScanRecord[]; total: number; page: number; pageSize: number; totalPages: number }
export interface InitiateScanRequest { organizationId?: number; targetId?: number; configuration: WorkflowConfigurationValue; workflowNames: string[] }
export interface QuickScanRequest { targets: { name: string }[]; configuration: WorkflowConfigurationValue; workflowNames: string[] }
export interface QuickScanResponse { count: number; targetStats: { created: number; skipped: number; failed: number }; assetStats: { websites: number; endpoints: number }; errors: Array<{ input: string; error: string }>; scans: ScanTask[] }
export interface ScanTask { id: number; target: number; workflowNames: string[]; status: ScanStatus; createdAt: string; updatedAt: string }
export interface InitiateScanResponse { message: string; count: number; scans: ScanTask[] }
