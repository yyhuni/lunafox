import type { ScanWorkflowConfigurationValue as WorkflowConfigurationValue } from '@/types/scan-workflow.types'

export type ScanStatus = "pending" | "running" | "succeeded" | "failed" | "cancelled"
export type ScanTriggerType = "manual" | "scheduled" | "ai"
export type ScanInputSource = "scanSnapshot" | "targetInventory"
export type ScanStage = string
export type StageStatus = "pending" | "running" | "succeeded" | "skipped" | "failed" | "cancelled"
export type EngineDiagnosticAvailability = "available" | "unavailable"
export type EngineDiagnosticResultState = "complete" | "partial" | "none" | "unknown"
export type EngineDiagnosticFailedStage =
  | "unknown"
  | "plan_validation"
  | "input_materialization"
  | "resource_materialization"
  | "context_build"
  | "image_prepare"
  | "container_create"
  | "container_start"
  | "container_wait"
  | "container_cleanup"
  | "protocol_bootstrap"
  | "execution_projection"
  | "handler"
  | "progress_report"
  | "result_encode"
  | "result_submit"
export type EngineDiagnosticErrorType =
  | "execution_failed"
  | "plan_validation_failed"
  | "input_materialization_failed"
  | "resource_materialization_failed"
  | "context_build_failed"
  | "image_prepare_failed"
  | "container_create_failed"
  | "container_start_failed"
  | "container_wait_failed"
  | "container_cleanup_failed"
  | "protocol_bootstrap_failed"
  | "handler_failed"
  | "progress_report_failed"
  | "result_encode_failed"
  | "result_submit_failed"
  | "timeout"
  | "cancelled"
  | "oom_killed"
  | "agent_session_failed"
  | "invalid_outcome"

export function isScanInputSource(value: unknown): value is ScanInputSource {
  return value === "scanSnapshot" || value === "targetInventory"
}

export interface ResultTypeWatermark {
  resultType: string
  receivedItems: number
  encodedItems: number
  submittedItems: number
  acknowledgedItems: number
  submittedBatches: number
  acknowledgedBatches: number
}

// EngineExecutionDiagnostics is the Server-projected terminal delivery
// evidence. It is deliberately distinct from scan coverage and raw logs.
export interface EngineExecutionDiagnostics {
	compatibilityRevision: 'engine-execution-diagnostics-r1'
  availability: EngineDiagnosticAvailability
  resultState: EngineDiagnosticResultState
  failedStage?: EngineDiagnosticFailedStage
  errorType?: EngineDiagnosticErrorType
  resultTypeWatermarks?: ResultTypeWatermark[]
}

export interface RuntimeTask {
  id: number
  name: string
  stepId: string
  stageId: string
  engineId: string
  status: StageStatus
  skipReason?: string
  order: number
  startedAt?: string
  completedAt?: string
  duration?: number
  error?: string
  failureKind?: string
  failureDetail?: string
  diagnostics?: EngineExecutionDiagnostics
}
export interface ScanTargetBrief { id: number; name: string; displayName: string; type: string }
export interface ScanCachedStats { subdomainsCount: number; websitesCount: number; endpointsCount: number; ipsCount: number; directoriesCount: number; screenshotsCount: number; vulnsTotal: number; vulnsCritical: number; vulnsHigh: number; vulnsMedium: number; vulnsLow: number }
export interface FailureDetail { kind: string; message: string }

export interface ScanRecord {
  id: number
  name?: string
  targetId: number
  target?: ScanTargetBrief
  cachedStats?: ScanCachedStats
  scanWorkflow?: string
  plannedEngineIds: string[]
  triggerType: ScanTriggerType
  inputSource: ScanInputSource
  createdAt: string
  stoppedAt?: string
  status: ScanStatus
  errorMessage?: string
  failure?: FailureDetail
  progress: number
  currentStage?: ScanStage
  configuration?: WorkflowConfigurationValue
  resultsDir?: string
  agentId?: number
  agent?: string
  agentName?: string | null
  agentStatus?: string
  agentHealthState?: string
  assignmentMode?: "automatic" | "pinned"
  agentDeleted?: boolean
  runtimeTasks?: RuntimeTask[]
}

export interface GetScansParams { page?: number; pageSize?: number; pageToken?: string; status?: ScanStatus; target?: number; filter?: string; orderBy?: string }
export type ScanListRecord = Omit<ScanRecord, "agentId" | "agentName" | "configuration" | "resultsDir" | "runtimeTasks">
export interface GetScansResponse { results: ScanListRecord[]; total: number; page: number; pageSize: number; totalPages: number; totalSize?: number; nextPageToken?: string }
export interface InitiateScanRequest { organizationId?: number; targetId?: number; agentId?: number; configuration: WorkflowConfigurationValue; scanWorkflow: string; inputSource: ScanInputSource }
export interface BulkInitiateScanRequest { organizationIds?: number[]; targetIds?: number[]; agentId?: number; configuration: WorkflowConfigurationValue; scanWorkflow: string; inputSource: ScanInputSource }
export interface QuickScanRequest { targets: { name: string }[]; agentId?: number; configuration: WorkflowConfigurationValue; scanWorkflow: string; inputSource: ScanInputSource }
export interface QuickScanResponse { count: number; targetStats: { created: number; skipped: number; failed: number }; assetStats: { websites: number; endpoints: number }; errors: Array<{ input: string; error: string }>; scans: ScanTask[] }
export interface ScanTask { id: number; target: number; scanWorkflow?: string; status: ScanStatus; inputSource: ScanInputSource; createdAt: string; updatedAt: string }
export interface InitiateScanResponse { message: string; count: number; scans: ScanTask[] }
export interface BulkScanItemOutcome { index: number; target?: string; organization?: string; reason: string; message?: string }
export interface BulkInitiateScanResponse { message: string; count: number; createdCount?: number; scans: ScanTask[]; skipped?: BulkScanItemOutcome[]; failed?: BulkScanItemOutcome[] }
export interface BatchStopScansResponse {
  stoppedCount: number
  skippedCount: number
  revokedTaskCount: number
}
export interface ScanLog { id: number; taskId: number; level: "info" | "warning" | "error"; content: string; createdAt: string }
export interface GetScanLogsResponse { results: ScanLog[]; nextPageToken?: string }
export interface GetScanLogsParams { pageSize?: number; pageToken?: string }
