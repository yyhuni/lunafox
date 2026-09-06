/**
 * Engine Definition-backed config type definitions.
 *
 * These types separate engine.json's machine-readable config contract from the
 * localized render model consumed by the form renderer.
 */

/** Allowed value types for engine config parameters. */
export type EngineParamType = "integer" | "string" | "stringArray" | "boolean"

export type EngineParamValue = number | string | boolean | string[]

/** Dynamic LunaFox resource collection referenced by a string parameter. */
export interface EngineParamResourceBinding {
  kind: "wordlist"
}

/** A single configurable parameter as declared by engine.json. */
export interface EngineParamManifestDefinition {
  /** Parameter key used in engineConfig, e.g. "timeout", "threads". */
  key: string
  /** Value type determining the form control to render. */
  type: EngineParamType
  /** Default value pre-filled when the section is enabled. */
  default?: EngineParamValue
  /** Minimum value constraint (integer params only). */
  minimum?: number
  /** Maximum value constraint (integer params only). */
  maximum?: number
  /** Minimum string length (string params only). */
  minLength?: number
  /** Maximum string length (string params only). */
  maxLength?: number
  /** Minimum number of values (stringArray params only). */
  minItems?: number
  /** Maximum number of values (stringArray params only). */
  maxItems?: number
  /** Enumerated allowed values; renders a Select instead of free text. */
  enum?: string[]
  /** Regex pattern for string validation. */
  pattern?: string
  /** Dynamic resource binding; platform UI decides the picker implementation. */
  resource?: EngineParamResourceBinding
}

/** A logical grouping of engine config parameters as declared by engine.json. */
export interface EngineConfigSectionManifestDefinition {
  /** Section identifier, e.g. "recon", "bruteforce". */
  id: string
  /** Default enabled state supplied by engine manifest. */
  defaultEnabled?: boolean
  /** This individual section stays enabled whenever its Workflow Step is enabled. */
  requiredEnabled?: boolean
  /** Parameters belonging to this section. */
  params: EngineParamManifestDefinition[]
}

/** The execution-level metadata from engine.json. */
export interface EngineManifestExecutionDefinition {
  /** Engine API compatibility selected by engine.v5. */
  engineApiMajor: number
  supportedTargetTypes: string[]
  /** engine.v5 closed execution resource IDs, classified by the Server Registry. */
  executionResources?: string[]
  configSections: EngineConfigSectionManifestDefinition[]
}

/** Top-level engine manifest structure (subset relevant to config rendering). */
export interface EngineManifest {
  manifestVersion: "engine.v5"
  engineId: string
  publisher: string
  execution: EngineManifestExecutionDefinition
}

/** A single localized configurable parameter rendered by the frontend. */
export interface EngineParamDefinition extends EngineParamManifestDefinition {
  /** Human-readable description shown as helper text. */
  description?: string
}

/** A localized config section rendered by the frontend. */
export interface EngineConfigSectionDefinition extends Omit<EngineConfigSectionManifestDefinition, "params"> {
  /** Display name shown in the section header. */
  name: string
  /** Description shown below the section name. */
  description?: string
  /** Parameters belonging to this section. */
  params: EngineParamDefinition[]
}

/** Execution metadata after localization resolution at the render boundary. */
export interface LocalizedEngineExecutionDefinition extends Omit<EngineManifestExecutionDefinition, "configSections"> {
  configSections: EngineConfigSectionDefinition[]
}

/** Render-only localized engine manifest. Never persist these display strings. */
export interface LocalizedEngineManifest extends Omit<EngineManifest, "execution"> {
  displayName: string
  description: string
  execution: LocalizedEngineExecutionDefinition
}

/**
 * A step within a scan workflow stage, enriched with its engine manifest
 * for form rendering. This is the composite type the form component consumes.
 */
export interface WorkflowStepWithEngine {
  stepId: string
  engineId: string
  profileDefaultEnabled: boolean
  /** Localized engine manifest providing configSections for this step. */
  engine: LocalizedEngineManifest
}

/** A stage in the scan workflow, containing ordered steps. */
export interface WorkflowStageWithEngines {
  stageId: string
  steps: WorkflowStepWithEngine[]
}

/**
 * A complete scan workflow enriched with engine manifests,
 * ready for the form renderer to consume.
 */
export interface ScanWorkflowWithEngines {
  scanWorkflowId: string
  displayName: string
  description: string
  stages: WorkflowStageWithEngines[]
}

/**
 * A section's session-local draft. Parameters are intentionally retained while
 * the parent Step is disabled, but disabled sections clear this object before
 * they can be re-enabled.
 */
export interface EngineConfigSectionFormValue {
  enabled: boolean
  params: Record<string, EngineParamValue>
}

/** Structured form values keyed by Workflow Step ID. */
export interface EngineConfigStepFormValue {
  enabled: boolean
  sections: Record<string, EngineConfigSectionFormValue>
}

export type EngineConfigFormValues = Record<string, EngineConfigStepFormValue>
