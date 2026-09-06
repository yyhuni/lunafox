import apiClient from "@/lib/api-client"
import type { EngineCatalogDetail, EngineCatalogSummary } from "@/types/engine-catalog.types"

export type EngineInstallRequest = { artifactRef: string; allowReplacement: boolean }

export async function installEngine(request: EngineInstallRequest): Promise<EngineCatalogDetail> {
  if (!request.artifactRef || request.artifactRef !== request.artifactRef.trim()) throw new Error("A canonical OCI digest reference is required")
  const response = await apiClient.post("/engines:install", request)
  const summary = normalizeEngineCatalogSummary(response.data)
  const configSections = response.data?.execution?.configSections
  if (!Array.isArray(configSections)) throw new Error("Installed engine detail is missing configuration metadata")
  return { ...summary, execution: { ...summary.execution, configSections } }
}

export async function getEngineCatalog(): Promise<EngineCatalogSummary[]> {
  const response = await apiClient.get("/engines")
  const payload = response.data?.results ?? response.data
  if (!Array.isArray(payload)) {
    throw new Error("Engine catalog response must be an array")
  }
  return payload.map(normalizeEngineCatalogSummary)
}

export async function getEngineCatalogDetail(engineId: string): Promise<EngineCatalogDetail> {
  const normalizedEngineId = engineId.trim()
  if (!normalizedEngineId) {
    throw new Error("Engine ID is required")
  }
  const response = await apiClient.get(`/engines/${encodeURIComponent(normalizedEngineId)}`)
  const summary = normalizeEngineCatalogSummary(response.data)
  const configSections = response.data?.execution?.configSections
  if (!Array.isArray(configSections)) {
    throw new Error(`Engine ${normalizedEngineId} detail is missing configuration metadata`)
  }
  return {
    ...summary,
    execution: {
      ...summary.execution,
      configSections,
    },
  }
}

function normalizeEngineCatalogSummary(payload: unknown): EngineCatalogSummary {
  if (!isRecord(payload) || !isRecord(payload.execution) || !isRecord(payload.localeResources)) {
    throw new Error("Engine catalog item is invalid")
  }
  const { name, engineId, manifestVersion, publisher, packageVersion, artifactRef, packageDigest } = payload
  const { engineApiMajor, supportedTargetTypes, executionResources } = payload.execution
  if (
    typeof name !== "string" ||
    typeof engineId !== "string" ||
    typeof manifestVersion !== "string" ||
    typeof publisher !== "string" ||
    typeof packageVersion !== "string" ||
    typeof artifactRef !== "string" ||
    typeof packageDigest !== "string" ||
    !Array.isArray(supportedTargetTypes) ||
    !supportedTargetTypes.every((item) => typeof item === "string")
  ) {
    throw new Error("Engine catalog item is missing required fields")
  }
  if (manifestVersion !== "engine.v5") {
    throw new Error(`Unsupported engine manifest version: ${manifestVersion}`)
  }

  if (hasAnyOwnProperty(payload.execution, ["runtimeRef", "inputProfile", "inputs", "platformResources", "runtimeArtifacts"])) {
    throw new Error("Engine catalog engine.v5 execution must not include retired fields")
  }
  if (
    !Number.isInteger(engineApiMajor) ||
    (engineApiMajor as number) <= 0 ||
    (executionResources !== undefined && (!Array.isArray(executionResources) || !executionResources.every((item) => typeof item === "string")))
  ) {
    throw new Error("Engine catalog item is missing required engine.v5 execution fields")
  }

  const execution = {
    engineApiMajor: engineApiMajor as number,
    supportedTargetTypes,
    ...(executionResources === undefined ? {} : { executionResources: executionResources as string[] }),
  }
  return {
    name,
    engineId,
    manifestVersion,
    publisher,
    packageVersion,
    artifactRef,
    packageDigest,
    execution,
    localeResources: payload.localeResources as Record<string, Record<string, unknown>>,
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function hasAnyOwnProperty(record: Record<string, unknown>, keys: string[]): boolean {
  return keys.some((key) => Object.prototype.hasOwnProperty.call(record, key))
}
