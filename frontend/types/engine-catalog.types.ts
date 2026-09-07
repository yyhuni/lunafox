import type { EngineManifestExecutionDefinition } from "@/types/engine-config.types"

export type EngineLocaleResource = Record<string, unknown>

export interface EngineCatalogSummary {
  name: string
  engineId: string
  manifestVersion: "engine.v5"
  publisher: string
  packageVersion: string
  artifactRef: string
  packageDigest: string
  execution: Omit<EngineManifestExecutionDefinition, "configSections">
  localeResources: Record<string, EngineLocaleResource>
}

export interface EngineCatalogDetail extends EngineCatalogSummary {
  execution: EngineManifestExecutionDefinition
}
