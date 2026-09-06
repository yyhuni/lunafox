import type { Locale } from "@/i18n/config"
import type { EngineCatalogDetail, EngineCatalogSummary, EngineLocaleResource } from "@/types/engine-catalog.types"
import type { LocalizedEngineManifest, ScanWorkflowWithEngines } from "@/types/engine-config.types"
import type { ScanWorkflow } from "@/types/scan-workflow.types"

const engineDisplayNameKey = "engine.displayName"
const engineDescriptionKey = "engine.description"

export type LocalizedExecutedEngineDisplay = {
  engineId: string
  displayName: string
  description: string
}

export type LocalizedEngineSummaryDisplay = LocalizedExecutedEngineDisplay

export type WorkflowEngineLibraryItem = LocalizedEngineSummaryDisplay

export type EngineReplacementConflict = {
  engineId: string
  currentPackageDigest: string
  proposedPackageDigest: string
}

export function getEngineReplacementConflict(error: unknown): EngineReplacementConflict | null {
  if (typeof error !== "object" || error === null || !("response" in error)) return null
  const response = error.response
  if (typeof response !== "object" || response === null || !("status" in response) || response.status !== 409 || !("data" in response)) return null
  const data = response.data
  if (typeof data !== "object" || data === null || !("error" in data)) return null
  const body = data.error
  if (typeof body !== "object" || body === null || !("details" in body) || !Array.isArray(body.details)) return null
  const fields = new Map(body.details.flatMap((detail) =>
    typeof detail === "object" && detail !== null && typeof detail.field === "string" && typeof detail.message === "string"
      ? [[detail.field, detail.message] as const]
      : []
  ))
  const engineId = fields.get("engineId")
  const currentPackageDigest = fields.get("currentPackageDigest")
  const proposedPackageDigest = fields.get("proposedPackageDigest")
  return engineId && currentPackageDigest && proposedPackageDigest
    ? { engineId, currentPackageDigest, proposedPackageDigest }
    : null
}

export function localizeEngineCatalogDetail(detail: EngineCatalogDetail, locale: Locale): LocalizedEngineManifest {
  const resource = detail.localeResources[locale]
  if (!resource) {
    throw new Error(`Engine ${detail.engineId} locale ${locale} is unavailable`)
  }
  return {
    manifestVersion: detail.manifestVersion,
    engineId: detail.engineId,
    publisher: detail.publisher,
    displayName: resolveLocaleString(detail.engineId, locale, resource, engineDisplayNameKey),
    description: resolveLocaleString(detail.engineId, locale, resource, engineDescriptionKey),
    execution: {
      ...detail.execution,
      configSections: detail.execution.configSections.map((section) => ({
        ...section,
        name: resolveLocaleString(detail.engineId, locale, resource, `sections.${section.id}.name`),
        description: resolveLocaleString(detail.engineId, locale, resource, `sections.${section.id}.description`),
        params: section.params.map((param) => ({
          ...param,
          description: resolveLocaleString(detail.engineId, locale, resource, `sections.${section.id}.params.${param.key}.description`),
        })),
      })),
    },
  }
}

export function getLocalizedEngineSummaryName(summary: EngineCatalogSummary, locale: Locale): string {
  const resource = summary.localeResources[locale]
  if (!resource) {
    throw new Error(`Engine ${summary.engineId} locale ${locale} is unavailable`)
  }
  return resolveLocaleString(summary.engineId, locale, resource, engineDisplayNameKey)
}

export function getLocalizedEngineSummaryDisplay(
  summary: EngineCatalogSummary,
  locale: Locale,
): LocalizedEngineSummaryDisplay {
  const resource = summary.localeResources[locale]
  if (!resource) {
    throw new Error(`Engine ${summary.engineId} locale ${locale} is unavailable`)
  }
  return {
    engineId: summary.engineId,
    displayName: resolveLocaleString(summary.engineId, locale, resource, engineDisplayNameKey),
    description: resolveLocaleString(summary.engineId, locale, resource, engineDescriptionKey),
  }
}

export function buildWorkflowEngineLibrary(
  catalog: EngineCatalogSummary[],
  locale: Locale,
): WorkflowEngineLibraryItem[] {
  return catalog.map((summary) => getLocalizedEngineSummaryDisplay(summary, locale))
}

export function getLocalizedExecutedEngineNames(
  engineIds: string[],
  catalog: EngineCatalogSummary[],
  locale: Locale,
): string[] {
  return getLocalizedExecutedEngineDisplays(engineIds, catalog, locale)
    .map((engine) => engine.displayName)
}

export function getLocalizedExecutedEngineDisplays(
  engineIds: string[],
  catalog: EngineCatalogSummary[],
  locale: Locale,
): LocalizedExecutedEngineDisplay[] {
  const summariesByID = new Map(catalog.map((summary) => [summary.engineId, summary]))
  const seen = new Set<string>()
  const displays: LocalizedExecutedEngineDisplay[] = []

  for (const engineID of engineIds) {
    if (seen.has(engineID)) continue
    seen.add(engineID)
    const summary = summariesByID.get(engineID)
    if (!summary) {
      throw new Error(`Executed engine ${engineID} is unavailable`)
    }
    displays.push(getLocalizedEngineSummaryDisplay(summary, locale))
  }

  return displays
}

export function buildWorkflowWithEngineCatalog(
  workflow: ScanWorkflow,
  details: EngineCatalogDetail[],
  locale: Locale,
): ScanWorkflowWithEngines {
  const detailsById = new Map(details.map((detail) => [detail.engineId, detail]))
  const stagesById = new Map<string, ScanWorkflowWithEngines["stages"][number]>()
  for (const step of workflow.steps ?? []) {
    const detail = detailsById.get(step.engineId)
    if (!detail) {
      throw new Error(`Workflow ${workflow.name} references unavailable engine ${step.engineId}`)
    }
    const stageId = step.stageId || "workflow"
    const stage = stagesById.get(stageId) ?? { stageId, steps: [] }
    stage.steps.push({
      stepId: step.stepId,
      engineId: step.engineId,
      profileDefaultEnabled: step.profileDefaultEnabled,
      engine: localizeEngineCatalogDetail(detail, locale),
    })
    stagesById.set(stageId, stage)
  }
  return {
    scanWorkflowId: workflow.name,
    displayName: workflow.displayName || workflow.name,
    description: workflow.description || "",
    stages: Array.from(stagesById.values()),
  }
}

function resolveLocaleString(engineId: string, locale: Locale, resource: EngineLocaleResource, key: string): string {
  let current: unknown = resource
  for (const segment of key.split(".")) {
    if (typeof current !== "object" || current === null || Array.isArray(current)) {
      throw new Error(`Engine ${engineId} locale ${locale} is missing localization key: ${key}`)
    }
    current = (current as Record<string, unknown>)[segment]
  }
  if (typeof current !== "string" || !current.trim()) {
    throw new Error(`Engine ${engineId} locale ${locale} is missing localization key: ${key}`)
  }
  return current
}
