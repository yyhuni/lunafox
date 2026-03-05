/**
 * Scan task status enum
 * Consistent with backend ScanStatus
 */
export type ScanStatus = "pending" | "running" | "completed" | "failed" | "cancelled"

/**
 * Scan stage (dynamic, from workflow config key)
 */
export type ScanStage = string

/**
 * Stage progress status
 */
export type StageStatus = "pending" | "running" | "completed" | "failed" | "cancelled"

/**
 * Single stage progress info
 */
export interface StageProgressItem {
  status: StageStatus
  order: number          // Execution order (starting from 0)
  startedAt?: string     // ISO time string
  duration?: number      // Execution duration (seconds)
  detail?: string        // Completion details
  error?: string         // Error message
  reason?: string        // Skip reason
}

/**
 * Stage progress dictionary (dynamic keys)
 */
export type StageProgress = Record<string, StageProgressItem>

/**
 * Target brief info in scan response
 */
export interface ScanTargetBrief {
  id: number
  name: string
  type: string
}

/**
 * Cached statistics in scan response
 */
export interface ScanCachedStats {
  subdomainsCount: number
  websitesCount: number
  endpointsCount: number
  ipsCount: number
  directoriesCount: number
  screenshotsCount: number
  vulnsTotal: number
  vulnsCritical: number
  vulnsHigh: number
  vulnsMedium: number
  vulnsLow: number
}

export interface ScanRecord {
  id: number
  targetId: number             // Target ID
  target?: ScanTargetBrief     // Target info (nested object)
  workerName?: string | null   // Worker node name
  cachedStats?: ScanCachedStats // Cached statistics
  workflowIds?: number[]         // Legacy field
  workflowNames: string[]        // Workflow name list
  scanMode: string             // Scan mode
  createdAt: string            // Creation time
  stoppedAt?: string           // Stop time
  status: ScanStatus
  errorMessage?: string        // Error message (has value when failed)
  progress: number             // 0-100
  currentStage?: ScanStage     // Current scan stage (only has value in running status)
  stageProgress?: StageProgress // Stage progress details
  yamlConfiguration?: string   // YAML configuration string
  resultsDir?: string          // Results directory
  workerId?: number            // Worker ID
}

export interface GetScansParams {
  page?: number
  pageSize?: number
  status?: ScanStatus
  search?: string
  target?: number  // Filter by target ID
}

export interface GetScansResponse {
  results: ScanRecord[]        // Corresponds to backend results field
  total: number
  page: number
  pageSize: number
  totalPages: number
}

/**
 * Initiate scan request parameters (for existing target/organization)
 */
export interface InitiateScanRequest {
  organizationId?: number  // Organization ID (choose one)
  targetId?: number        // Target ID (choose one)
  configuration: string    // YAML configuration string (required)
  workflowNames: string[]    // Workflow name list (required)
}

/**
 * Quick scan request parameters (auto-create target and scan)
 */
export interface QuickScanRequest {
  targets: { name: string }[]  // Target list
  configuration: string        // YAML configuration string (required)
  workflowNames: string[]        // Workflow name list (required)
}

/**
 * Quick scan response
 */
export interface QuickScanResponse {
  count: number             // Number of scan tasks created
  targetStats: {
    created: number
    skipped: number
    failed: number
  }
  assetStats: {
    websites: number
    endpoints: number
  }
  errors: Array<{ input: string; error: string }>
  scans: ScanTask[]
}

/**
 * Single scan task info
 */
export interface ScanTask {
  id: number
  target: number           // Target ID
  workflowIds?: number[]     // Legacy field
  workflowNames: string[]    // Workflow name list
  status: ScanStatus
  createdAt: string
  updatedAt: string
}

/**
 * Initiate scan response
 */
export interface InitiateScanResponse {
  message: string          // Success message
  count: number            // Number of scan tasks created
  scans: ScanTask[]        // Scan task list
}
