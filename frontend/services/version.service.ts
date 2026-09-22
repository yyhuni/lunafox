import axios, { type AxiosError } from "axios"

import { api } from '@/lib/api-client'
import type {
	CreateUpgradeOperationInput,
	DatabaseMigrationInfo,
	ReleaseManifestSummary,
	ReleaseNotes,
  UpgradeAgentSummary,
	UpgradeDiagnostic,
	UpgradeExecutionMode,
	UpgradeLogEntry,
  UpgradeLogLevel,
  UpgradeMigrationStatus,
  UpgradeOperation,
	UpgradeOperationFull,
  UpgradeOperationStatus,
	UpgradePlanSummary,
	UpgradeWorkDisposition,
  UpdateCheckResult,
} from "@/types/version.types"

const CHECK_FOR_UPDATES_PATH = "/system:checkForUpdates"
const UPGRADE_OPERATIONS_PATH = "/upgradeOperations"
const ACTIVE_UPGRADE_OPERATION_PATH = "/upgradeOperations:active"

const OPERATION_STATUSES = new Set<UpgradeOperationStatus>([
  "queued", "stopping", "preflight", "updating", "migrating", "restarting",
  "agent_verifying", "verifying", "succeeded", "failed", "needs_recovery", "needs_attention",
])
const MIGRATION_STATUSES = new Set<UpgradeMigrationStatus>([
  "not_started", "running", "succeeded", "failed", "unknown",
])
const EXECUTION_MODES = new Set<UpgradeExecutionMode>(["full", "frontend_only"])
const WORK_DISPOSITIONS = new Set<UpgradeWorkDisposition>([
  "not_required", "cancelled", "cancellation_failed", "legacy_unknown",
])
const FULL_UPGRADE_VIEW = "FULL"
const MAX_PLAN_SERVICES = 9
const MAX_PLAN_SERVICE_NAME_LENGTH = 32
const PLAN_SERVICES = new Set([
  "server", "frontend", "nginx", "agent", "bootstrap", "engine", "engine_runtime", "engine_package", "migration",
])
const FULL_PLAN_SERVICES = [
  "agent", "bootstrap", "engine", "engine_package", "engine_runtime", "frontend", "migration", "nginx", "server",
] as const
const BASIC_UPGRADE_OPERATION_FIELDS = [
  "name", "operationId", "requestId", "operatorId", "manifestId", "manifestDigest", "releaseVersion",
  "currentVersion", "compatibilityRange", "maintenanceWindowMinutes", "status", "migrationStatus", "migrationType", "migrationId",
  "migrationChecksum", "cancelledScanCount", "cancelledTaskCount", "agentDesiredVersion", "agentTargetDigest",
  "agentSummary", "observedDigests", "diagnostic", "logs", "stageTimes", "createdAt", "updatedAt", "completedAt",
] as const
const FULL_UPGRADE_OPERATION_FIELDS = [
  ...BASIC_UPGRADE_OPERATION_FIELDS,
  "executionMode", "workDisposition", "planSummary", "confirmedDeploymentVersion",
] as const
const MAX_UPGRADE_LOG_ENTRIES = 32
const MAX_UPGRADE_LOG_MESSAGE_LENGTH = 512
const MAX_UPGRADE_LOG_STAGE_LENGTH = 64
const MAX_UPGRADE_LOG_MESSAGE_KEY_LENGTH = 64
const MAX_UPGRADE_LOG_METADATA_ENTRIES = 8
const MAX_UPGRADE_LOG_METADATA_KEY_LENGTH = 64
const MAX_UPGRADE_LOG_METADATA_VALUE_LENGTH = 128
const MAX_RELEASE_NOTES_BYTES = 64 * 1024

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

	static async getUpgradeOperationFull(operationId: string): Promise<UpgradeOperationFull> {
		assertCanonicalUUID(operationId, "operationId")
		const response = await api.get<unknown>(`${UPGRADE_OPERATIONS_PATH}/${encodeURIComponent(operationId)}`, {
			params: { view: FULL_UPGRADE_VIEW },
		})
		return parseUpgradeOperationFull(response.data)
	}

	static async getActiveUpgradeOperation(): Promise<UpgradeOperation | null> {
		try {
			const response = await api.get<unknown>(ACTIVE_UPGRADE_OPERATION_PATH)
			return parseUpgradeOperation(response.data)
		} catch (error) {
			const parsed = toUpgradeApiError(error)
			if (parsed.status === 404) return null
			throw parsed
		}
	}

	static async getActiveUpgradeOperationFull(): Promise<UpgradeOperationFull | null> {
		try {
			const response = await api.get<unknown>(ACTIVE_UPGRADE_OPERATION_PATH, {
				params: { view: FULL_UPGRADE_VIEW },
			})
			return parseUpgradeOperationFull(response.data)
		} catch (error) {
			const parsed = toUpgradeApiError(error)
			if (parsed.status === 404) return null
			throw parsed
		}
	}

	static async retryUpgradeOperation(operationId: string): Promise<UpgradeOperation> {
    assertCanonicalUUID(operationId, "operationId")
    const response = await api.post<unknown>(`${UPGRADE_OPERATIONS_PATH}/${encodeURIComponent(operationId)}:retry`, { confirmed: true })
    return parseUpgradeOperation(response.data)
	}

	static async stopUpgradeOperation(operationId: string): Promise<UpgradeOperation> {
		assertCanonicalUUID(operationId, "operationId")
		const response = await api.post<unknown>(`${UPGRADE_OPERATIONS_PATH}/${encodeURIComponent(operationId)}:stop`, { confirmed: true })
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

async function parseUpdateCheckResult(value: unknown): Promise<UpdateCheckResult> {
  const record = requireObject(value, "checkForUpdates")
  assertKnownFields(record, ["currentVersion", "hasUpdate", "candidate", "eligible", "diagnostic"], "checkForUpdates")
  const candidate = record.candidate === undefined || record.candidate === null ? undefined : await parseManifestSummary(record.candidate)
  const diagnostic = record.diagnostic === undefined || record.diagnostic === null ? undefined : parseDiagnostic(record.diagnostic)
  return {
    currentVersion: requireNonEmpty(record.currentVersion, "checkForUpdates.currentVersion"),
    hasUpdate: requireBoolean(record.hasUpdate, "checkForUpdates.hasUpdate"),
    eligible: requireBoolean(record.eligible, "checkForUpdates.eligible"),
    ...(candidate ? { candidate } : {}),
    ...(diagnostic ? { diagnostic } : {}),
  }
}

async function parseManifestSummary(value: unknown): Promise<ReleaseManifestSummary> {
  const record = requireObject(value, "candidate")
  assertKnownFields(record, [
    "name", "manifestId", "manifestDigest", "releaseVersion", "deploymentMode", "compatibilityRange",
    "maintenanceWindowMinutes", "requiresAdminConfirmation", "databaseMigration", "runtimeImageDigests", "engineDigests", "releaseNotes",
  ], "candidate")
  const migration = parseMigration(record.databaseMigration)
  const images = requireObject(record.runtimeImageDigests, "candidate.runtimeImageDigests")
  const runtimeImageDigests: Record<string, string> = {}
  for (const [name, digest] of Object.entries(images)) runtimeImageDigests[name] = assertSha256Digest(digest, `candidate.runtimeImageDigests.${name}`)
  if (!Array.isArray(record.engineDigests)) throw invalidResponse("candidate.engineDigests")
  const engineDigests = record.engineDigests.map((digest, index) => assertSha256Digest(digest, `candidate.engineDigests[${index}]`))
  const releaseVersion = requireNonEmpty(record.releaseVersion, "candidate.releaseVersion")
  const releaseNotes = record.releaseNotes === undefined ? undefined : await parseReleaseNotes(record.releaseNotes)
  if (!releaseNotes && releaseVersion !== "0.0.0-dev") throw invalidResponse("candidate.releaseNotes")
  return {
    name: requireNonEmpty(record.name, "candidate.name"),
    manifestId: requireNonEmpty(record.manifestId, "candidate.manifestId"),
    manifestDigest: assertSha256Digest(record.manifestDigest, "candidate.manifestDigest"),
    releaseVersion,
    deploymentMode: requireNonEmpty(record.deploymentMode, "candidate.deploymentMode"),
    compatibilityRange: requireNonEmpty(record.compatibilityRange, "candidate.compatibilityRange"),
    maintenanceWindowMinutes: requireSafeInteger(record.maintenanceWindowMinutes, "candidate.maintenanceWindowMinutes", 1),
    requiresAdminConfirmation: requireBoolean(record.requiresAdminConfirmation, "candidate.requiresAdminConfirmation"),
    databaseMigration: migration,
    runtimeImageDigests,
    engineDigests,
    ...(releaseNotes ? { releaseNotes } : {}),
  }
}

async function parseReleaseNotes(value: unknown): Promise<ReleaseNotes> {
  const record = requireObject(value, "candidate.releaseNotes")
  assertKnownFields(record, ["body", "sha256"], "candidate.releaseNotes")
  const body = requireString(record.body, "candidate.releaseNotes.body")
  const bytes = new TextEncoder().encode(body)
  if (!body.trim() || bytes.length > MAX_RELEASE_NOTES_BYTES || !body.endsWith("\n")) {
    throw invalidResponse("candidate.releaseNotes.body")
  }
  if (body.includes("\r") || body.includes("\u0000") || body.includes("\uFFFD")) {
    throw invalidResponse("candidate.releaseNotes.body")
  }
  const lines = body.split("\n")
  const sections = new Set<string>()
  for (const [index, line] of lines.entries()) {
    const heading = line.match(/^##[ \t]+(English|简体中文)[ \t]*$/u)
    if (!heading) continue
    const name = heading[1]
    if (sections.has(name)) throw invalidResponse("candidate.releaseNotes.body")
    let end = lines.length - 1
    for (let next = index + 1; next < lines.length; next += 1) {
      if (/^##[ \t]+[^\n]+$/u.test(lines[next])) {
        end = next
        break
      }
    }
    if (!lines.slice(index + 1, end).join("\n").trim()) throw invalidResponse("candidate.releaseNotes.body")
    sections.add(name)
  }
  if (sections.size !== 2) throw invalidResponse("candidate.releaseNotes.body")
  const sha256 = assertSha256Digest(record.sha256, "candidate.releaseNotes.sha256")
  const actual = `sha256:${await sha256Hex(body)}`
  if (actual !== sha256) throw invalidResponse("candidate.releaseNotes.sha256")
  return { body, sha256 }
}

async function sha256Hex(value: string): Promise<string> {
  if (!globalThis.crypto?.subtle) throw invalidResponse("candidate.releaseNotes.sha256")
  const digest = await globalThis.crypto.subtle.digest("SHA-256", new TextEncoder().encode(value))
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("")
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
  assertKnownFields(record, BASIC_UPGRADE_OPERATION_FIELDS, "upgradeOperation")
  return parseUpgradeOperationFields(record)
}

function parseUpgradeOperationFull(value: unknown): UpgradeOperationFull {
  const record = requireObject(value, "upgradeOperation")
  assertKnownFields(record, FULL_UPGRADE_OPERATION_FIELDS, "upgradeOperation")
  const operation = parseUpgradeOperationFields(record)
	const executionMode = requireEnum(record.executionMode, EXECUTION_MODES, "upgradeOperation.executionMode")
	const workDisposition = requireEnum(record.workDisposition, WORK_DISPOSITIONS, "upgradeOperation.workDisposition")
	const isLegacyFull = executionMode === "full" && workDisposition === "legacy_unknown"
	if (executionMode === "full" && workDisposition === "not_required") {
		throw invalidResponse("upgradeOperation.workDisposition")
	}
	const planSummary = parseUpgradePlanSummary(record.planSummary, isLegacyFull)

  if (executionMode === "frontend_only") {
    if (workDisposition !== "not_required") throw invalidResponse("upgradeOperation.workDisposition")
    if (planSummary.touchedServices.length !== 1 || planSummary.touchedServices[0] !== "frontend") {
      throw invalidResponse("upgradeOperation.planSummary.touchedServices")
    }
    for (const stage of ["stopping", "migrating", "agent_verifying"]) {
      if (operation.status === stage || operation.stageTimes[stage] !== undefined) {
        throw invalidResponse(`upgradeOperation.${operation.status === stage ? "status" : `stageTimes.${stage}`}`)
		}
		}
	} else {
		const completePlan = planSummary.touchedServices.length === FULL_PLAN_SERVICES.length
		  && planSummary.touchedServices.every((service, index) => service === FULL_PLAN_SERVICES[index])
		if ((!isLegacyFull && !completePlan) || (planSummary.touchedServices.length !== 0 && !completePlan)) {
			throw invalidResponse("upgradeOperation.planSummary.touchedServices")
		}
	}

  return {
    ...operation,
    executionMode,
    workDisposition,
    planSummary,
		confirmedDeploymentVersion: requireBoundedString(record.confirmedDeploymentVersion, "upgradeOperation.confirmedDeploymentVersion", 64, executionMode === "full"),
  }
}

function parseUpgradeOperationFields(record: Record<string, unknown>): UpgradeOperation {
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
		currentVersion: requireNonEmpty(record.currentVersion, "upgradeOperation.currentVersion"),
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
		logs: parseUpgradeLogs(record.logs),
    stageTimes,
    createdAt: requireTimestamp(record.createdAt, "upgradeOperation.createdAt"),
    updatedAt: requireTimestamp(record.updatedAt, "upgradeOperation.updatedAt"),
    completedAt: record.completedAt === undefined || record.completedAt === null ? null : requireTimestamp(record.completedAt, "upgradeOperation.completedAt"),
  }
}

function parseUpgradePlanSummary(value: unknown, allowEmpty: boolean): UpgradePlanSummary {
  const record = requireObject(value, "upgradeOperation.planSummary")
  assertKnownFields(record, ["touchedServices"], "upgradeOperation.planSummary")
  if (!Array.isArray(record.touchedServices) || record.touchedServices.length > MAX_PLAN_SERVICES || (!allowEmpty && record.touchedServices.length === 0)) {
    throw invalidResponse("upgradeOperation.planSummary.touchedServices")
  }
  let previous = ""
  const touchedServices = record.touchedServices.map((service, index) => {
    const path = `upgradeOperation.planSummary.touchedServices[${index}]`
    const parsed = requireBoundedToken(service, path, MAX_PLAN_SERVICE_NAME_LENGTH)
    if (!PLAN_SERVICES.has(parsed) || (previous && parsed <= previous)) throw invalidResponse(path)
    previous = parsed
    return parsed
  })
  return { touchedServices }
}

function parseUpgradeLogs(value: unknown): UpgradeLogEntry[] {
	if (!Array.isArray(value) || value.length > MAX_UPGRADE_LOG_ENTRIES) throw invalidResponse("upgradeOperation.logs")
	return value.map((entry, index) => {
		const record = requireObject(entry, `upgradeOperation.logs[${index}]`)
		assertKnownFields(record, ["timestamp", "level", "stage", "messageKey", "message", "metadata"], `upgradeOperation.logs[${index}]`)
		const level = requireEnum(record.level, new Set<UpgradeLogLevel>(["info", "warn", "error"]), `upgradeOperation.logs[${index}].level`)
		const metadataRecord = record.metadata === undefined || record.metadata === null ? {} : requireObject(record.metadata, `upgradeOperation.logs[${index}].metadata`)
		if (Object.keys(metadataRecord).length > MAX_UPGRADE_LOG_METADATA_ENTRIES) throw invalidResponse(`upgradeOperation.logs[${index}].metadata`)
		const metadata: Record<string, string> = {}
		for (const [key, item] of Object.entries(metadataRecord)) {
			const metadataPath = `upgradeOperation.logs[${index}].metadata.${key}`
			const safeKey = requireBoundedToken(key, metadataPath, MAX_UPGRADE_LOG_METADATA_KEY_LENGTH)
			if (hasUnsafeLogMarker(safeKey)) throw invalidResponse(metadataPath)
			const value = requireBoundedSafeLogText(item, metadataPath, MAX_UPGRADE_LOG_METADATA_VALUE_LENGTH)
			metadata[key] = value
		}
		const stage = requireBoundedToken(record.stage, `upgradeOperation.logs[${index}].stage`, MAX_UPGRADE_LOG_STAGE_LENGTH)
		const messageKey = requireBoundedToken(record.messageKey, `upgradeOperation.logs[${index}].messageKey`, MAX_UPGRADE_LOG_MESSAGE_KEY_LENGTH)
		const message = requireBoundedSafeLogMessage(record.message, `upgradeOperation.logs[${index}].message`)
		return {
			timestamp: requireTimestamp(record.timestamp, `upgradeOperation.logs[${index}].timestamp`),
			level,
			stage,
			messageKey,
			message,
			metadata,
		}
	})
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

function assertKnownFields(record: Record<string, unknown>, fields: readonly string[], path: string): void {
  const allowed = new Set(fields)
  for (const field of Object.keys(record)) if (!allowed.has(field)) throw invalidResponse(`${path}.${field}`)
}

function requireString(value: unknown, path: string): string {
  if (typeof value !== "string") throw invalidResponse(path)
  return value
}

function requireBoundedString(value: unknown, path: string, maximumLength: number, allowEmpty = false): string {
	const result = requireString(value, path)
	if ((!allowEmpty && !result.trim()) || (allowEmpty && result !== "" && !result.trim()) || result.length > maximumLength || /[\u0000-\u001f\u007f]/.test(result)) throw invalidResponse(path)
	return result
}

function requireBoundedToken(value: unknown, path: string, maximumLength: number): string {
	const result = requireBoundedString(value, path, maximumLength)
	if (!/^[A-Za-z0-9_.-]+$/.test(result)) throw invalidResponse(path)
	return result
}

function requireBoundedSafeLogMessage(value: unknown, path: string): string {
	return requireBoundedSafeLogText(value, path, MAX_UPGRADE_LOG_MESSAGE_LENGTH)
}

function requireBoundedSafeLogText(value: unknown, path: string, maximumLength: number): string {
	const result = requireBoundedString(value, path, maximumLength)
	if (/[\\/`$]/.test(result) || hasUnsafeLogMarker(result)) {
		throw invalidResponse(path)
	}
	return result
}

function hasUnsafeLogMarker(value: string): boolean {
	const lower = value.toLowerCase()
	return ["authorization", "bearer ", "jwt", "password", "passwd", "secret", "token", "private key", "docker compose", "command:", "stderr:"].some((marker) => lower.includes(marker))
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
