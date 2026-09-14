import axios, { type AxiosError } from "axios"

import { api } from '@/lib/api-client'
import type {
  CreateUpgradeOperationInput,
  DatabaseMigrationInfo,
  ReleaseManifestSummary,
  UpgradeAgentSummary,
  UpgradeDiagnostic,
  UpgradeMigrationStatus,
  UpgradeOperation,
  UpgradeOperationStatus,
  UpdateCheckResult,
} from "@/types/version.types"

const CHECK_FOR_UPDATES_PATH = "/system:checkForUpdates"
const UPGRADE_OPERATIONS_PATH = "/upgradeOperations"

const OPERATION_STATUSES = new Set<UpgradeOperationStatus>([
  "queued", "stopping", "preflight", "updating", "migrating", "restarting",
  "agent_verifying", "verifying", "succeeded", "failed", "needs_recovery", "needs_attention",
])
const MIGRATION_STATUSES = new Set<UpgradeMigrationStatus>([
  "not_started", "running", "succeeded", "failed", "unknown",
])

export class UpgradeApiError extends Error {
  readonly code: string
  readonly status?: number
  readonly retryable: boolean

  constructor(message: string, code = "UPGRADE_REQUEST_FAILED", status?: number) {
    super(message)
    this.name = "UpgradeApiError"
    this.code = code
    this.status = status
    this.retryable = status === undefined || status >= 500
  }
}

export class VersionService {
  static async checkForUpdates(): Promise<UpdateCheckResult> {
    const response = await api.post<unknown>(CHECK_FOR_UPDATES_PATH, {})
    return parseUpdateCheckResult(response.data)
  }

  static async createUpgradeOperation(input: CreateUpgradeOperationInput): Promise<UpgradeOperation> {
    assertCanonicalUUID(input.requestId, "requestId")
    requireNonEmpty(input.manifestId, "manifestId")
    assertSha256Digest(input.manifestDigest, "manifestDigest")
    if (input.confirmed !== true) throw new Error("Upgrade operation requires explicit confirmation")
    const response = await api.post<unknown>(UPGRADE_OPERATIONS_PATH, {
      requestId: input.requestId,
      manifestId: input.manifestId,
      manifestDigest: input.manifestDigest,
      confirmed: true,
    })
    return parseUpgradeOperation(response.data)
  }

  static async getUpgradeOperation(operationId: string): Promise<UpgradeOperation> {
    assertCanonicalUUID(operationId, "operationId")
    const response = await api.get<unknown>(`${UPGRADE_OPERATIONS_PATH}/${encodeURIComponent(operationId)}`)
    return parseUpgradeOperation(response.data)
  }

  static async retryUpgradeOperation(operationId: string): Promise<UpgradeOperation> {
    assertCanonicalUUID(operationId, "operationId")
    const response = await api.post<unknown>(`${UPGRADE_OPERATIONS_PATH}/${encodeURIComponent(operationId)}:retry`, { confirmed: true })
    return parseUpgradeOperation(response.data)
  }
}

export function toUpgradeApiError(error: unknown): UpgradeApiError {
  if (error instanceof UpgradeApiError) return error
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<unknown>
    const body = asRecord(axiosError.response?.data)
    const detail = asRecord(body?.error)
    const message = typeof detail?.message === "string" && detail.message.trim()
      ? detail.message
      : axiosError.message || "Upgrade request failed"
    const code = typeof detail?.code === "string" ? detail.code : "UPGRADE_REQUEST_FAILED"
    return new UpgradeApiError(message, code, axiosError.response?.status)
  }
  if (error instanceof Error) return new UpgradeApiError(error.message)
  return new UpgradeApiError("Upgrade request failed")
}

export function isUpgradeNetworkError(error: unknown): boolean {
  if (error instanceof UpgradeApiError) return error.status === undefined
  if (axios.isAxiosError(error)) return !error.response
  return error instanceof TypeError || (error instanceof Error && /network|fetch|timeout|connection/i.test(error.message))
}

function parseUpdateCheckResult(value: unknown): UpdateCheckResult {
  const record = requireObject(value, "checkForUpdates")
  assertKnownFields(record, ["currentVersion", "hasUpdate", "candidate", "eligible", "diagnostic"], "checkForUpdates")
  const candidate = record.candidate === undefined || record.candidate === null ? undefined : parseManifestSummary(record.candidate)
  const diagnostic = record.diagnostic === undefined || record.diagnostic === null ? undefined : parseDiagnostic(record.diagnostic)
  return {
    currentVersion: requireNonEmpty(record.currentVersion, "checkForUpdates.currentVersion"),
    hasUpdate: requireBoolean(record.hasUpdate, "checkForUpdates.hasUpdate"),
    eligible: requireBoolean(record.eligible, "checkForUpdates.eligible"),
    ...(candidate ? { candidate } : {}),
    ...(diagnostic ? { diagnostic } : {}),
  }
}

function parseManifestSummary(value: unknown): ReleaseManifestSummary {
  const record = requireObject(value, "candidate")
  assertKnownFields(record, [
    "name", "manifestId", "manifestDigest", "releaseVersion", "deploymentMode", "compatibilityRange",
    "maintenanceWindowMinutes", "requiresAdminConfirmation", "databaseMigration", "runtimeImageDigests", "engineDigests",
  ], "candidate")
  const migration = parseMigration(record.databaseMigration)
  const images = requireObject(record.runtimeImageDigests, "candidate.runtimeImageDigests")
  const runtimeImageDigests: Record<string, string> = {}
  for (const [name, digest] of Object.entries(images)) runtimeImageDigests[name] = assertSha256Digest(digest, `candidate.runtimeImageDigests.${name}`)
  if (!Array.isArray(record.engineDigests)) throw invalidResponse("candidate.engineDigests")
  const engineDigests = record.engineDigests.map((digest, index) => assertSha256Digest(digest, `candidate.engineDigests[${index}]`))
  return {
    name: requireNonEmpty(record.name, "candidate.name"),
    manifestId: requireNonEmpty(record.manifestId, "candidate.manifestId"),
    manifestDigest: assertSha256Digest(record.manifestDigest, "candidate.manifestDigest"),
    releaseVersion: requireNonEmpty(record.releaseVersion, "candidate.releaseVersion"),
    deploymentMode: requireNonEmpty(record.deploymentMode, "candidate.deploymentMode"),
    compatibilityRange: requireNonEmpty(record.compatibilityRange, "candidate.compatibilityRange"),
    maintenanceWindowMinutes: requireSafeInteger(record.maintenanceWindowMinutes, "candidate.maintenanceWindowMinutes", 1),
    requiresAdminConfirmation: requireBoolean(record.requiresAdminConfirmation, "candidate.requiresAdminConfirmation"),
    databaseMigration: migration,
    runtimeImageDigests,
    engineDigests,
  }
}

function parseMigration(value: unknown): DatabaseMigrationInfo {
  const record = requireObject(value, "candidate.databaseMigration")
  assertKnownFields(record, ["hasDatabaseMigration", "migrationType", "migrationId", "checksum", "policyVersion"], "candidate.databaseMigration")
  const migrationId = record.migrationId === undefined ? undefined : optionalString(record.migrationId, "candidate.databaseMigration.migrationId")
  const checksum = record.checksum === undefined ? undefined : optionalDigest(record.checksum, "candidate.databaseMigration.checksum")
  return {
    hasDatabaseMigration: requireBoolean(record.hasDatabaseMigration, "candidate.databaseMigration.hasDatabaseMigration"),
    migrationType: requireNonEmpty(record.migrationType, "candidate.databaseMigration.migrationType"),
    policyVersion: requireSafeInteger(record.policyVersion, "candidate.databaseMigration.policyVersion", 1),
    ...(migrationId ? { migrationId } : {}),
    ...(checksum ? { checksum } : {}),
  }
}

function parseUpgradeOperation(value: unknown): UpgradeOperation {
  const record = requireObject(value, "upgradeOperation")
  assertKnownFields(record, [
    "name", "operationId", "requestId", "operatorId", "manifestId", "manifestDigest", "releaseVersion",
    "compatibilityRange", "maintenanceWindowMinutes", "status", "migrationStatus", "migrationType", "migrationId",
    "migrationChecksum", "cancelledScanCount", "cancelledTaskCount", "agentDesiredVersion", "agentTargetDigest",
    "agentSummary", "observedDigests", "diagnostic", "stageTimes", "createdAt", "updatedAt", "completedAt",
  ], "upgradeOperation")
  const status = requireEnum(record.status, OPERATION_STATUSES, "upgradeOperation.status")
  const migrationStatus = requireEnum(record.migrationStatus, MIGRATION_STATUSES, "upgradeOperation.migrationStatus")
  const observed = requireObject(record.observedDigests, "upgradeOperation.observedDigests")
  const observedDigests: Record<string, string> = {}
  for (const [key, digest] of Object.entries(observed)) observedDigests[key] = assertSha256Digest(digest, `upgradeOperation.observedDigests.${key}`)
  const stageTimesRecord = requireObject(record.stageTimes, "upgradeOperation.stageTimes")
  const stageTimes: Record<string, string> = {}
  for (const [stage, timestamp] of Object.entries(stageTimesRecord)) {
    if (typeof timestamp !== "string" || !Number.isFinite(Date.parse(timestamp))) throw invalidResponse(`upgradeOperation.stageTimes.${stage}`)
    stageTimes[stage] = timestamp
  }
  return {
    name: requireNonEmpty(record.name, "upgradeOperation.name"),
    operationId: assertCanonicalUUID(record.operationId, "upgradeOperation.operationId"),
    requestId: assertCanonicalUUID(record.requestId, "upgradeOperation.requestId"),
    operatorId: requireSafeInteger(record.operatorId, "upgradeOperation.operatorId", 1),
    manifestId: requireNonEmpty(record.manifestId, "upgradeOperation.manifestId"),
    manifestDigest: assertSha256Digest(record.manifestDigest, "upgradeOperation.manifestDigest"),
    releaseVersion: requireNonEmpty(record.releaseVersion, "upgradeOperation.releaseVersion"),
    compatibilityRange: requireNonEmpty(record.compatibilityRange, "upgradeOperation.compatibilityRange"),
    maintenanceWindowMinutes: requireSafeInteger(record.maintenanceWindowMinutes, "upgradeOperation.maintenanceWindowMinutes", 1),
    status,
    migrationStatus,
    migrationType: requireNonEmpty(record.migrationType, "upgradeOperation.migrationType"),
    migrationId: optionalString(record.migrationId, "upgradeOperation.migrationId"),
    migrationChecksum: optionalDigest(record.migrationChecksum, "upgradeOperation.migrationChecksum"),
    cancelledScanCount: requireSafeInteger(record.cancelledScanCount, "upgradeOperation.cancelledScanCount"),
    cancelledTaskCount: requireSafeInteger(record.cancelledTaskCount, "upgradeOperation.cancelledTaskCount"),
    agentDesiredVersion: optionalString(record.agentDesiredVersion, "upgradeOperation.agentDesiredVersion"),
    agentTargetDigest: optionalDigest(record.agentTargetDigest, "upgradeOperation.agentTargetDigest"),
    agentSummary: parseAgentSummary(record.agentSummary),
    observedDigests,
    diagnostic: optionalString(record.diagnostic, "upgradeOperation.diagnostic"),
    stageTimes,
    createdAt: requireTimestamp(record.createdAt, "upgradeOperation.createdAt"),
    updatedAt: requireTimestamp(record.updatedAt, "upgradeOperation.updatedAt"),
    completedAt: record.completedAt === undefined || record.completedAt === null ? null : requireTimestamp(record.completedAt, "upgradeOperation.completedAt"),
  }
}

function parseAgentSummary(value: unknown): UpgradeAgentSummary {
  const record = requireObject(value, "upgradeOperation.agentSummary")
  assertKnownFields(record, ["expected", "ready", "missing", "unhealthy"], "upgradeOperation.agentSummary")
  return {
    expected: requireSafeInteger(record.expected, "upgradeOperation.agentSummary.expected"),
    ready: requireSafeInteger(record.ready, "upgradeOperation.agentSummary.ready"),
    missing: requireSafeInteger(record.missing, "upgradeOperation.agentSummary.missing"),
    unhealthy: requireSafeInteger(record.unhealthy, "upgradeOperation.agentSummary.unhealthy"),
  }
}

function parseDiagnostic(value: unknown): UpgradeDiagnostic {
  const record = requireObject(value, "checkForUpdates.diagnostic")
  assertKnownFields(record, ["code", "stage", "field", "reason"], "checkForUpdates.diagnostic")
  return {
    code: requireNonEmpty(record.code, "checkForUpdates.diagnostic.code"),
    stage: optionalString(record.stage, "checkForUpdates.diagnostic.stage"),
    field: optionalString(record.field, "checkForUpdates.diagnostic.field"),
    reason: requireNonEmpty(record.reason, "checkForUpdates.diagnostic.reason"),
  }
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : undefined
}

function requireObject(value: unknown, path: string): Record<string, unknown> {
  const record = asRecord(value)
  if (!record) throw invalidResponse(path)
  return record
}

function assertKnownFields(record: Record<string, unknown>, fields: string[], path: string): void {
  const allowed = new Set(fields)
  for (const field of Object.keys(record)) if (!allowed.has(field)) throw invalidResponse(`${path}.${field}`)
}

function requireString(value: unknown, path: string): string {
  if (typeof value !== "string") throw invalidResponse(path)
  return value
}

function requireNonEmpty(value: unknown, path: string): string {
  const result = requireString(value, path)
  if (!result.trim()) throw invalidResponse(path)
  return result
}

function optionalString(value: unknown, path: string): string | undefined {
  if (value === undefined || value === null || value === "") return undefined
  return requireString(value, path)
}

function requireBoolean(value: unknown, path: string): boolean {
  if (typeof value !== "boolean") throw invalidResponse(path)
  return value
}

function requireSafeInteger(value: unknown, path: string, minimum = 0): number {
  if (typeof value !== "number" || !Number.isSafeInteger(value) || value < minimum) throw invalidResponse(path)
  return value
}

function requireTimestamp(value: unknown, path: string): string {
  const timestamp = requireNonEmpty(value, path)
  if (!Number.isFinite(Date.parse(timestamp))) throw invalidResponse(path)
  return timestamp
}

function assertSha256Digest(value: unknown, path: string): string {
  const digest = requireString(value, path)
  if (!/^sha256:[0-9a-f]{64}$/.test(digest)) throw invalidResponse(path)
  return digest
}

function optionalDigest(value: unknown, path: string): string | undefined {
  if (value === undefined || value === null || value === "") return undefined
  return assertSha256Digest(value, path)
}

function assertCanonicalUUID(value: unknown, path: string): string {
  const uuid = requireString(value, path)
  if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(uuid) || uuid === "00000000-0000-0000-0000-000000000000") throw invalidResponse(path)
  return uuid
}

function requireEnum<T extends string>(value: unknown, allowed: ReadonlySet<T>, path: string): T {
  if (typeof value !== "string" || !allowed.has(value as T)) throw invalidResponse(path)
  return value as T
}

function invalidResponse(path: string): Error {
  return new Error(`Upgrade API response is invalid at ${path}`)
}
