import type { ReleaseManifestSummary, UpdateCheckResult, UpgradeOperation, VersionInfo } from '@/types/version.types'

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

let mockUpgradeOperation: UpgradeOperation | null = null

function now() {
  return new Date().toISOString()
}

export function createMockUpgradeOperation(input: { requestId: string; manifestId: string; manifestDigest: string }): UpgradeOperation {
  const timestamp = now()
  if (mockUpgradeOperation) return getMockUpgradeOperation() as UpgradeOperation
  mockUpgradeOperation = {
    name: 'upgradeOperations/11111111-1111-4111-8111-111111111111',
    operationId: '11111111-1111-4111-8111-111111111111',
    requestId: input.requestId,
    operatorId: 1,
    manifestId: input.manifestId,
    manifestDigest: input.manifestDigest,
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
    stageTimes: { queued: timestamp },
    createdAt: timestamp,
    updatedAt: timestamp,
    completedAt: null,
  }
  return getMockUpgradeOperation() as UpgradeOperation
}

export function getMockUpgradeOperation(): UpgradeOperation | null {
  if (!mockUpgradeOperation) return null
  return {
    ...mockUpgradeOperation,
    stageTimes: { ...mockUpgradeOperation.stageTimes },
    observedDigests: { ...mockUpgradeOperation.observedDigests },
  }
}

export function retryMockUpgradeOperation(): UpgradeOperation | null {
  if (!mockUpgradeOperation) return null
  const timestamp = now()
  mockUpgradeOperation = {
    ...mockUpgradeOperation,
    status: 'queued',
    diagnostic: undefined,
    completedAt: null,
    updatedAt: timestamp,
    stageTimes: { ...mockUpgradeOperation.stageTimes, queued: timestamp },
  }
  return getMockUpgradeOperation()
}

export function resetMockUpgradeOperation() {
  mockUpgradeOperation = null
}

export function getMockVersionInfo(): VersionInfo {
  return { ...mockVersionInfo }
}

export function getMockUpdateCheckResult(): UpdateCheckResult {
  return {
    ...mockUpdateCheckResult,
    candidate: mockUpdateCheckResult.candidate
      ? {
          ...mockUpdateCheckResult.candidate,
          databaseMigration: { ...mockUpdateCheckResult.candidate.databaseMigration },
          runtimeImageDigests: { ...mockUpdateCheckResult.candidate.runtimeImageDigests },
          engineDigests: [...mockUpdateCheckResult.candidate.engineDigests],
        }
      : undefined,
  }
}
