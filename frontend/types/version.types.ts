export interface VersionInfo {
  version: string
  githubRepo: string
}

export type UpgradeOperationStatus =
  | "queued"
  | "stopping"
  | "preflight"
  | "updating"
  | "migrating"
  | "restarting"
  | "agent_verifying"
  | "verifying"
  | "succeeded"
  | "failed"
  | "needs_recovery"
  | "needs_attention"

export type UpgradeUserStage =
  | "preparing"
  | "stopping"
  | "updating"
  | "restarting"
  | "verifying"
  | "finished"

export const UPGRADE_USER_STAGES: readonly UpgradeUserStage[] = [
  "preparing",
  "stopping",
  "updating",
  "restarting",
  "verifying",
  "finished",
]

export type UpgradeMigrationStatus =
  | "not_started"
  | "running"
  | "succeeded"
  | "failed"
  | "unknown"

export type UpgradeExecutionMode = "full" | "frontend_only"

export type UpgradeWorkDisposition =
  | "not_required"
  | "cancelled"
  | "cancellation_failed"
  | "legacy_unknown"

export interface UpgradeDiagnostic {
  code: string
  stage?: string
  field?: string
  reason: string
}

export interface DatabaseMigrationInfo {
  hasDatabaseMigration: boolean
  migrationType: string
  migrationId?: string
  checksum?: string
  policyVersion: number
}

export interface ReleaseNotes {
  body: string
  sha256: string
}

export interface ReleaseManifestSummary {
  name: string
  manifestId: string
  manifestDigest: string
  releaseVersion: string
  deploymentMode: string
  compatibilityRange: string
  maintenanceWindowMinutes: number
  requiresAdminConfirmation: boolean
  databaseMigration: DatabaseMigrationInfo
  runtimeImageDigests: Record<string, string>
  engineDigests: string[]
  releaseNotes?: ReleaseNotes
}

export interface UpdateCheckResult {
  currentVersion: string
  hasUpdate: boolean
  candidate?: ReleaseManifestSummary
  eligible: boolean
  diagnostic?: UpgradeDiagnostic | null
}

export interface UpgradeAgentSummary {
  expected: number
  ready: number
  missing: number
  unhealthy: number
}

export type UpgradeAgentDiagnosticSource = "agent_heartbeat" | "server_observation" | "historical"

export type UpgradeAgentDiagnosticReasonCode =
  | "not_registered"
  | "heartbeat_missing_or_stale"
  | "paused"
  | "health_reported"
  | "health_unhealthy"
  | "claim_not_ready"
  | "version_mismatch"
  | "legacy_unknown"

export interface UpgradeAgentDiagnostic {
  agentId: number
  name: string
  displayNameSnapshot?: string
  reasonCode: UpgradeAgentDiagnosticReasonCode
  detail: string
  source: UpgradeAgentDiagnosticSource
}

export type UpgradeHostAction =
  | "preflight"
  | "pull_images"
  | "update_services"
  | "database_migration"
  | "update_resident_agent"
  | "wait_for_service_health"
  | "verify_runtime_images"
  | "verify_frontend_container"
  | "verify_frontend_edge"

/** A bounded current host-side wait fact from the enhanced FULL projection. */
export interface UpgradeHostActivity {
  action: UpgradeHostAction
  startedAt: string
  lastHeartbeatAt: string
}

export interface UpgradeOperation {
  name: string
  operationId: string
  requestId: string
  operatorId: number
  manifestId: string
  manifestDigest: string
  releaseVersion: string
  currentVersion: string
  compatibilityRange: string
  maintenanceWindowMinutes: number
  status: UpgradeOperationStatus
  migrationStatus: UpgradeMigrationStatus
  migrationType: string
  migrationId?: string
  migrationChecksum?: string
  cancelledScanCount: number
  cancelledTaskCount: number
  agentDesiredVersion?: string
  agentTargetDigest?: string
  agentSummary: UpgradeAgentSummary
  observedDigests: Record<string, string>
  diagnostic?: string
  logs: UpgradeLogEntry[]
  stageTimes: Record<string, string>
  createdAt: string
  updatedAt: string
  completedAt?: string | null
}

/**
 * The BASIC Operation projection deliberately remains separate from this
 * representation so cached clients can keep decoding the old field set.
 * Upgrade pages must request and consume this FULL projection before making
 * scope-dependent presentation or route-lock decisions.
 */
export interface UpgradeOperationFull extends UpgradeOperation {
  executionMode: UpgradeExecutionMode
  workDisposition: UpgradeWorkDisposition
  planSummary: UpgradePlanSummary
  confirmedDeploymentVersion: string
  agentDiagnostics: UpgradeAgentDiagnostic[]
  // Undefined means an older Server ignored the enhanced FULL opt-in. Null
  // means a supporting Server currently has no in-flight host action.
  hostActivity?: UpgradeHostActivity | null
}

export interface UpgradePlanSummary {
  touchedServices: string[]
}

export type UpgradeLogLevel = "info" | "warn" | "error"

export interface UpgradeLogEntry {
  timestamp: string
  level: UpgradeLogLevel
  stage: string
  messageKey: string
  message: string
  metadata?: Record<string, string>
}

export interface CreateUpgradeOperationInput {
  requestId: string
  manifestId: string
  manifestDigest: string
  // The transport accepts a boolean so runtime callers can be rejected when
  // confirmation is absent; VersionService enforces that it is exactly true.
  confirmed: boolean
}
