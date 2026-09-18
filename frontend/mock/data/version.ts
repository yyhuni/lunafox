import type { ReleaseManifestSummary, UpdateCheckResult, UpgradeLogEntry, UpgradeOperation, UpgradeOperationStatus, VersionInfo } from '@/types/version.types'
import { getMockScenario } from "../scenarios"

export const mockVersionInfo: VersionInfo = {
  version: 'mock-2026.04.26',
  githubRepo: 'https://github.com/lunafox/lunafox',
}

export const mockCandidateManifest: ReleaseManifestSummary = {
  name: 'releases/1.2.3',
  manifestId: 'lunafox-1.2.3',
  manifestDigest: 'sha256:' + 'a'.repeat(64),
  releaseVersion: 'mock-2026.05.01',
  deploymentMode: 'single-node-compose',
  compatibilityRange: '>=1.0.0 <2.0.0',
  maintenanceWindowMinutes: 15,
  requiresAdminConfirmation: true,
  databaseMigration: {
    hasDatabaseMigration: false,
    migrationType: 'none',
    policyVersion: 1,
  },
  runtimeImageDigests: {
    server: 'sha256:' + 'b'.repeat(64),
    frontend: 'sha256:' + 'b'.repeat(64),
    nginx: 'sha256:' + 'b'.repeat(64),
    agent: 'sha256:' + 'b'.repeat(64),
    bootstrap: 'sha256:' + 'b'.repeat(64),
  },
  engineDigests: ['sha256:' + 'c'.repeat(64)],
}

export const mockUpdateCheckResult: UpdateCheckResult = {
  currentVersion: mockVersionInfo.version,
  hasUpdate: true,
  eligible: true,
  candidate: mockCandidateManifest,
}

const MOCK_UPGRADE_STATE_KEY = "lunafox.mock.upgrade.state.v1"

type PersistedMockUpgradeState = {
  operation: UpgradeOperation
  pollCount: number
}

let mockUpgradeOperation: UpgradeOperation | null = null
let mockUpgradePollCount = 0

function mockLogForStage(status: UpgradeOperationStatus, timestamp: string, message?: string): UpgradeLogEntry {
  const messages: Record<UpgradeOperationStatus, { messageKey: string; message: string }> = {
    queued: { messageKey: "requestAccepted", message: "Upgrade request accepted" },
    stopping: { messageKey: "stoppingWork", message: "Stopping active work" },
    preflight: { messageKey: "preflight", message: "Preflight checks started" },
    updating: { messageKey: "updatingServices", message: "Updating system services" },
    migrating: { messageKey: "migratingDatabase", message: "Database migration started" },
    restarting: { messageKey: "restartingServices", message: "Restarting services" },
    agent_verifying: { messageKey: "verifyingAgents", message: "Verifying Agents" },
    verifying: { messageKey: "verifyingSystem", message: "Verifying system" },
    succeeded: { messageKey: "completed", message: "Upgrade verification completed" },
    failed: { messageKey: "failed", message: "Upgrade failed" },
    needs_recovery: { messageKey: "needsRecovery", message: "Manual recovery is required" },
    needs_attention: { messageKey: "needsAttention", message: "Operator attention is required" },
  }
  const entry = messages[status]
  return { timestamp, level: status === "failed" || status === "needs_recovery" ? "error" : status === "needs_attention" ? "warn" : "info", stage: status, messageKey: entry.messageKey, message: message ?? entry.message }
}

function readPersistedUpgradeState(): PersistedMockUpgradeState | null {
  if (typeof globalThis === "undefined" || !("localStorage" in globalThis)) return null
  try {
    const raw = globalThis.localStorage.getItem(MOCK_UPGRADE_STATE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<PersistedMockUpgradeState>
    if (!parsed.operation || typeof parsed.operation !== "object" || typeof parsed.pollCount !== "number") return null
    return {
      operation: parsed.operation as UpgradeOperation,
      pollCount: Number.isSafeInteger(parsed.pollCount) && parsed.pollCount >= 0 ? parsed.pollCount : 0,
    }
  } catch {
    return null
  }
}

function hydrateUpgradeState(): void {
  if (mockUpgradeOperation) return
  const persisted = readPersistedUpgradeState()
  if (!persisted) return
  mockUpgradeOperation = {
    ...persisted.operation,
    currentVersion: persisted.operation.currentVersion || mockVersionInfo.version,
    logs: Array.isArray(persisted.operation.logs) && persisted.operation.logs.length > 0
      ? persisted.operation.logs
      : [mockLogForStage(persisted.operation.status, persisted.operation.updatedAt || now())],
  }
  mockUpgradePollCount = persisted.pollCount
}

function persistUpgradeState(): void {
  if (!mockUpgradeOperation || typeof globalThis === "undefined" || !("localStorage" in globalThis)) return
  try {
    globalThis.localStorage.setItem(MOCK_UPGRADE_STATE_KEY, JSON.stringify({
      operation: mockUpgradeOperation,
      pollCount: mockUpgradePollCount,
    } satisfies PersistedMockUpgradeState))
  } catch {
    // The in-memory fixture remains usable when browser storage is unavailable.
  }
}

function clearPersistedUpgradeState(): void {
  if (typeof globalThis === "undefined" || !("localStorage" in globalThis)) return
  try {
    globalThis.localStorage.removeItem(MOCK_UPGRADE_STATE_KEY)
  } catch {
    // The in-memory fixture remains reset even when browser storage is unavailable.
  }
}

function createMockOperationId(): string {
  try {
    if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") return crypto.randomUUID()
  } catch {
    // Fall through to the deterministic fixture ID in runtimes without Web Crypto.
  }
  return "11111111-1111-4111-8111-111111111111"
}

function now() {
  return new Date().toISOString()
}

export function createMockUpgradeOperation(input: { requestId: string; manifestId: string; manifestDigest: string }): UpgradeOperation {
  hydrateUpgradeState()
  const timestamp = now()
  if (mockUpgradeOperation && !["succeeded", "failed", "needs_recovery", "needs_attention"].includes(mockUpgradeOperation.status)) {
    return getMockUpgradeOperation() as UpgradeOperation
  }
  const operationId = createMockOperationId()
  mockUpgradeOperation = {
    name: `upgradeOperations/${operationId}`,
    operationId,
    requestId: input.requestId,
    operatorId: 1,
    manifestId: input.manifestId,
    manifestDigest: input.manifestDigest,
    currentVersion: mockVersionInfo.version,
    releaseVersion: mockCandidateManifest.releaseVersion,
    compatibilityRange: mockCandidateManifest.compatibilityRange,
    maintenanceWindowMinutes: mockCandidateManifest.maintenanceWindowMinutes,
    status: 'queued',
    migrationStatus: 'not_started',
    migrationType: 'none',
    cancelledScanCount: 0,
    cancelledTaskCount: 0,
    agentSummary: { expected: 0, ready: 0, missing: 0, unhealthy: 0 },
    observedDigests: {},
    logs: [mockLogForStage("queued", timestamp)],
    stageTimes: { queued: timestamp },
    createdAt: timestamp,
    updatedAt: timestamp,
    completedAt: null,
  }
  mockUpgradePollCount = 0
  persistUpgradeState()
  return getMockUpgradeOperation() as UpgradeOperation
}

export function getMockUpgradeOperation(): UpgradeOperation | null {
  hydrateUpgradeState()
  if (!mockUpgradeOperation) return null
  return {
    ...mockUpgradeOperation,
    stageTimes: { ...mockUpgradeOperation.stageTimes },
    observedDigests: { ...mockUpgradeOperation.observedDigests },
    logs: mockUpgradeOperation.logs.map((entry) => ({ ...entry, metadata: entry.metadata ? { ...entry.metadata } : undefined })),
  }
}

export function retryMockUpgradeOperation(): UpgradeOperation | null {
  hydrateUpgradeState()
  if (!mockUpgradeOperation) return null
  const timestamp = now()
  mockUpgradeOperation = {
    ...mockUpgradeOperation,
    status: 'queued',
    diagnostic: undefined,
    completedAt: null,
    updatedAt: timestamp,
    stageTimes: { ...mockUpgradeOperation.stageTimes, queued: timestamp },
    logs: [...mockUpgradeOperation.logs, mockLogForStage("queued", timestamp)],
  }
  mockUpgradePollCount = 0
  persistUpgradeState()
  return getMockUpgradeOperation()
}

/** Advance only when the status resource is observed, matching server polling. */
export function observeMockUpgradeOperation(): UpgradeOperation | null {
  hydrateUpgradeState()
  if (!mockUpgradeOperation) return null
  if (["succeeded", "failed", "needs_recovery", "needs_attention"].includes(mockUpgradeOperation.status)) {
    return getMockUpgradeOperation()
  }
  mockUpgradePollCount += 1
  const scenario = getMockScenario()
  const transitions = scenario === "edge"
    ? ["queued", "stopping", "updating", "needs_attention"] as const
    : scenario === "stress"
      ? ["queued", "stopping", "updating", "restarting", "failed"] as const
      : ["queued", "stopping", "updating", "migrating", "restarting", "verifying", "succeeded"] as const
  const nextStatus = transitions[Math.min(mockUpgradePollCount - 1, transitions.length - 1)]
  if (mockUpgradeOperation.status !== nextStatus) {
    const timestamp = now()
    mockUpgradeOperation = {
      ...mockUpgradeOperation,
      status: nextStatus,
      currentVersion: nextStatus === "succeeded" ? mockUpgradeOperation.releaseVersion : mockUpgradeOperation.currentVersion,
      updatedAt: timestamp,
      completedAt: nextStatus === "succeeded" || nextStatus === "failed" || nextStatus === "needs_attention" ? timestamp : null,
      diagnostic: nextStatus === "failed"
        ? "The mock verifier rejected one service digest."
        : nextStatus === "needs_attention"
          ? "One Agent did not report a healthy version."
          : undefined,
      agentSummary: nextStatus === "needs_attention"
        ? { expected: 3, ready: 2, missing: 1, unhealthy: 0 }
        : mockUpgradeOperation.agentSummary,
      stageTimes: { ...mockUpgradeOperation.stageTimes, [nextStatus]: timestamp },
      logs: [...mockUpgradeOperation.logs, mockLogForStage(nextStatus, timestamp)],
    }
  }
  persistUpgradeState()
  return getMockUpgradeOperation()
}

export function stopMockUpgradeOperation(): UpgradeOperation | null {
  hydrateUpgradeState()
  if (!mockUpgradeOperation || ["succeeded", "failed", "needs_recovery", "needs_attention"].includes(mockUpgradeOperation.status)) return getMockUpgradeOperation()
  const timestamp = now()
  const postMigration = mockUpgradeOperation.migrationStatus !== "not_started" || ["migrating", "restarting", "agent_verifying", "verifying"].includes(mockUpgradeOperation.status)
  const status: UpgradeOperationStatus = postMigration ? "needs_recovery" : "needs_attention"
  const diagnostic = postMigration
    ? "Upgrade stopped after the migration boundary; manual recovery is required."
    : "Upgrade stopped by operator."
  mockUpgradeOperation = {
    ...mockUpgradeOperation,
    status,
    diagnostic,
    completedAt: timestamp,
    updatedAt: timestamp,
    stageTimes: { ...mockUpgradeOperation.stageTimes, [status]: timestamp },
    logs: [...mockUpgradeOperation.logs, mockLogForStage(status, timestamp, diagnostic)],
  }
  persistUpgradeState()
  return getMockUpgradeOperation()
}

export function resetMockUpgradeOperation() {
  mockUpgradeOperation = null
  mockUpgradePollCount = 0
  clearPersistedUpgradeState()
}

export function getMockVersionInfo(): VersionInfo {
  return { ...mockVersionInfo }
}

export function getMockUpdateCheckResult(): UpdateCheckResult {
  const completedOperation = getMockUpgradeOperation()
  const hasCompletedUpgrade = completedOperation?.status === "succeeded"
  const currentVersion = hasCompletedUpgrade ? completedOperation.releaseVersion : mockVersionInfo.version
  return {
    ...mockUpdateCheckResult,
    currentVersion,
    hasUpdate: !hasCompletedUpgrade,
    ...(hasCompletedUpgrade ? { candidate: undefined } : {
      candidate: mockUpdateCheckResult.candidate
      ? {
          ...mockUpdateCheckResult.candidate,
          databaseMigration: { ...mockUpdateCheckResult.candidate.databaseMigration },
          runtimeImageDigests: { ...mockUpdateCheckResult.candidate.runtimeImageDigests },
          engineDigests: [...mockUpdateCheckResult.candidate.engineDigests],
        }
      : undefined,
    }),
  }
}
