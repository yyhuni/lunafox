export type NucleiPocSourceType = "git" | "gitee" | "custom"

export type NucleiPocSyncState =
  | "VALIDATING_SOURCE"
  | "CLONING"
  | "SCANNING_FILES"
  | "VALIDATING_TEMPLATES"
  | "COMMITTING"
  | "CLEANING"
  | "SUCCEEDED"
  | "FAILED"

export interface NucleiPocSource {
  name: "nucleiPocSources/current"
  sourceType: NucleiPocSourceType
  repoUrl: string
  commitSha: string
  syncedAt: string | null
  createdAt: string
  updatedAt: string
}

export interface NucleiPocDiagnosticsSample {
  category: string
  relativePath?: string
  reasonCode: string
}

export interface NucleiPocDiagnostics {
  samples: NucleiPocDiagnosticsSample[]
  total: number
  truncated: boolean
}

export interface NucleiPocSyncCounters {
  filesSeen: number | null
  yamlFilesSeen: number | null
  templatesValidated: number | null
  bytesRead: number | null
}

export interface NucleiPocSyncTask {
  name: `nucleiPocSyncTasks/${string}`
  requestId: string
  sourceType: NucleiPocSourceType
  state: NucleiPocSyncState
  phase: NucleiPocSyncState
  counters: NucleiPocSyncCounters
  commitSha?: string
  committedPocCount?: number
  failureCode?: string
  failureSummary?: string
  diagnostics: NucleiPocDiagnostics
  cleanupStatus: "pending" | "clean" | "residual"
  createdAt: string
  startedAt: string | null
  completedAt: string | null
  updatedAt: string
}

export interface NucleiPocListItem {
  name: `nucleiPocs/${string}`
  templateId: string
  displayName: string
  severity: "critical" | "high" | "medium" | "low" | "info"
  tags: string[]
  author: string
  description: string
  cve: string[]
  cwe: string[]
  references: string[]
  remediation: string
  relativePath: string
  contentSha256: string
  isEnabled: boolean
  createdAt: string
  updatedAt: string
}

export interface NucleiPocDetail extends NucleiPocListItem {
  content: string
}

export interface NucleiPocListResponse {
  results: NucleiPocListItem[]
  nextPageToken?: string
  totalSize: number
}

export interface NucleiPocListQuery {
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
}

export type NucleiPocFilterOptionField = "tags"

export interface NucleiPocFilterOption {
  value: string
  label: string
  count: number
}

export interface NucleiPocFilterOptionsResponse {
  results: NucleiPocFilterOption[]
}

export interface SyncNucleiPocSourceRequest {
  requestId: string
  sourceType: NucleiPocSourceType
  repoUrl: string
}

export interface UpdateNucleiPocRequest {
  name: `nucleiPocs/${string}`
  isEnabled: boolean
  updateMask: ["isEnabled"]
}

export interface SetNucleiPocActivationRequest {
  enabled: boolean
  names?: `nucleiPocs/${string}`[]
}

export interface SetNucleiPocActivationResponse {
  enabled: boolean
  affectedCount: number
}

export interface NucleiPocRpcErrorBody {
  error?: {
    code?: number | string
    status?: string
    message?: string
    details?: Array<Record<string, unknown>>
  }
}
