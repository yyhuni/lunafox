import * as yaml from 'js-yaml'
import type { ScanWorkflow, ScanWorkflowConfiguration, ScanWorkflowProfile } from '@/types/scan-workflow.types'
import type { ScanWorkflowWithEngines, WorkflowStepWithEngine } from '@/types/engine-config.types'

export {
  CAPABILITY_CONFIG,
  getScanWorkflowIcon,
} from '@/lib/scan-workflow-config'

export function normalizeWorkflowConfiguration(
  input: unknown
): ScanWorkflowConfiguration {
  if (!input) return {}
  if (typeof input === 'string') {
    try {
      const parsed = yaml.load(input)
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        return parsed as ScanWorkflowConfiguration
      }
    } catch {
      return {}
    }
    return {}
  }
  if (typeof input === 'object' && !Array.isArray(input)) {
    return input as ScanWorkflowConfiguration
  }
  return {}
}

export function parseWorkflowConfiguration(input: unknown): ScanWorkflowConfiguration {
  if (!input) return {}
  if (typeof input === 'string') {
    try {
      const parsed = yaml.load(input)
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        return parsed as ScanWorkflowConfiguration
      }
    } catch {
      return {}
    }
    return {}
  }
  return input as ScanWorkflowConfiguration
}

export function serializeWorkflowConfiguration(
  configuration: unknown
): string {
  const normalized = parseWorkflowConfiguration(configuration)
  if (Object.keys(normalized).length === 0) return ''
  return yaml.dump(normalized, { lineWidth: -1, noRefs: true }).trim()
}

/**
 * Parse an editable Profile/YAML draft without applying the request rule that
 * at least one Step must be enabled. A disabled Step may retain its complete
 * Profile engineConfig for same-session re-enable; request serialization drops
 * that retained object before submission.
 */
export function parseWorkflowConfigurationDraftStrict(input: unknown): Record<string, unknown> {
  const parsed = parseYamlObject(input)
  if (Object.keys(parsed).some((key) => key !== "steps")) {
    throw new WorkflowConfigurationDraftError("configuration", "configuration must contain only steps")
  }
  if (!isRecord(parsed.steps)) {
    throw new WorkflowConfigurationDraftError("configuration.steps", "steps must be an object")
  }
  validateDraftStepEnvelope(parsed.steps)
  return cloneRecord(parsed)
}

export function extractWorkflowIds(
  configuration: unknown
): string[] {
  const normalized = parseWorkflowConfiguration(configuration)
  const steps = normalized.steps
  if (steps && typeof steps === 'object' && !Array.isArray(steps)) {
    return Object.keys(steps as Record<string, unknown>)
  }
  return Object.keys(normalized).filter(
    (key) => key !== 'steps' && typeof normalized[key] === 'object'
  )
}

export function parseWorkflowCapabilities(
  configuration: unknown
): string[] {
  const normalized = parseWorkflowConfiguration(configuration)
  const steps = normalized.steps
  if (steps && typeof steps === 'object' && !Array.isArray(steps)) {
    return Object.keys(steps as Record<string, unknown>)
  }
  return Object.keys(normalized).filter(
    (key) => key !== 'steps' && typeof normalized[key] === 'object'
  )
}

function getWorkflowStepEntries(input: unknown): Array<Record<string, unknown>> {
  const steps = parseWorkflowConfiguration(input).steps
  if (!steps || typeof steps !== 'object' || Array.isArray(steps)) return []

  return Object.values(steps as Record<string, unknown>).filter(
    (step): step is Record<string, unknown> =>
      typeof step === 'object' && step !== null && !Array.isArray(step)
  )
}

/** Returns true only for a complete-looking set of explicit disabled Step branches. */
export function hasNoEnabledWorkflowSteps(input: unknown): boolean {
  const steps = getWorkflowStepEntries(input)
  if (steps.length === 0) return false
  return steps.every((step) => step.enabled === false)
}

export function countEnabledWorkflowSteps(input: unknown): number {
  return getWorkflowStepEntries(input).filter((step) => step.enabled === true).length
}

export function hasEnabledWorkflowStep(input: unknown): boolean {
  return countEnabledWorkflowSteps(input) > 0
}

export type WorkflowProfileDraftStep = {
  enabled: boolean
  engineConfig: Record<string, unknown>
}

export type WorkflowProfileDraft = {
  scanWorkflow: string
  steps: Record<string, WorkflowProfileDraftStep>
}

export type CanonicalWorkflowStep =
  | { enabled: false }
  | { enabled: true; engineConfig: Record<string, unknown> }

export type CanonicalWorkflowConfiguration = {
  steps: Record<string, CanonicalWorkflowStep>
}

export class WorkflowConfigurationDraftError extends Error {
  readonly field: string

  constructor(field: string, message: string) {
    super(`${field}: ${message}`)
    this.name = "WorkflowConfigurationDraftError"
    this.field = field
  }
}

/**
 * Converts the Server Profile into the only accepted initial editing draft.
 * Topology/catalog data is used for identity and ordering; it never fills a
 * missing enablement flag or Engine configuration.
 */
export function adaptWorkflowProfile(
  profile: ScanWorkflowProfile,
  workflow: ScanWorkflow,
  workflowWithEngines?: ScanWorkflowWithEngines,
): WorkflowProfileDraft {
  if (!isRecord(profile) || workflowIdentity(profile.scanWorkflow) !== workflowIdentity(workflow.name)) {
    throw new WorkflowConfigurationDraftError("scanWorkflow", "Profile parent does not match the selected Workflow")
  }
  const configuration = profile.configuration
  if (!isRecord(configuration) || Object.keys(configuration).some((key) => key !== "steps")) {
    throw new WorkflowConfigurationDraftError("configuration", "Profile configuration must contain only steps")
  }
  const rawSteps = configuration.steps
  if (!isRecord(rawSteps)) {
    throw new WorkflowConfigurationDraftError("configuration.steps", "Profile steps must be an object")
  }

  const stepIds = workflow.steps.map((step) => step.stepId)
  const expected = new Set(stepIds)
  for (const stepId of Object.keys(rawSteps)) {
    if (!expected.has(stepId)) {
      throw new WorkflowConfigurationDraftError(`configuration.steps["${stepId}"]`, "unknown Workflow Step")
    }
  }
  for (const stepId of stepIds) {
    if (!(stepId in rawSteps)) {
      throw new WorkflowConfigurationDraftError(`configuration.steps["${stepId}"]`, "Workflow Step is required")
    }
  }

  const steps: Record<string, WorkflowProfileDraftStep> = {}
  for (const stepId of stepIds) {
    const path = `configuration.steps["${stepId}"]`
    const rawStep = rawSteps[stepId]
    if (!isRecord(rawStep)) {
      throw new WorkflowConfigurationDraftError(path, "Profile Step must be an object")
    }
    const keys = Object.keys(rawStep).sort()
    if (keys.length !== 2 || keys[0] !== "enabled" || keys[1] !== "engineConfig") {
      throw new WorkflowConfigurationDraftError(path, "Profile Step must contain enabled and engineConfig")
    }
    if (typeof rawStep.enabled !== "boolean") {
      throw new WorkflowConfigurationDraftError(`${path}.enabled`, "Profile Step enabled must be an explicit boolean")
    }
    if (!isRecord(rawStep.engineConfig)) {
      throw new WorkflowConfigurationDraftError(`${path}.engineConfig`, "Profile engineConfig must be an object")
    }
    if (Object.keys(rawStep.engineConfig).length === 0) {
      throw new WorkflowConfigurationDraftError(`${path}.engineConfig`, "Profile engineConfig must be complete")
    }
    if (workflowWithEngines) {
      const engineStep = findWorkflowEngineStep(workflowWithEngines, stepId)
      if (!engineStep) {
        throw new WorkflowConfigurationDraftError(path, "Workflow Step engine metadata is unavailable")
      }
      validateCompleteProfileEngineConfig(rawStep.engineConfig, engineStep, path)
    }
    steps[stepId] = {
      enabled: rawStep.enabled,
      engineConfig: cloneRecord(rawStep.engineConfig),
    }
  }
  return { scanWorkflow: workflow.name, steps }
}

/** Serialize only an already validated Profile draft for the editing surface. */
export function serializeWorkflowProfileDraft(draft: WorkflowProfileDraft): string {
  return yaml.dump({ steps: draft.steps }, { lineWidth: -1, noRefs: true }).trim()
}

/**
 * Verifies a Profile Engine draft against the current catalog schema. A
 * Profile may start with either explicit Workflow Step state, but its retained
 * Engine object must already be complete; the form never supplies missing values.
 */
export function validateCompleteProfileEngineConfig(
  engineConfig: Record<string, unknown>,
  step: WorkflowStepWithEngine,
  path = `configuration.steps["${step.stepId}"].engineConfig`,
): void {
  const sections = step.engine.execution.configSections
  const expected = new Set(sections.map((section) => section.id))
  for (const sectionId of Object.keys(engineConfig)) {
    if (!expected.has(sectionId)) {
      throw new WorkflowConfigurationDraftError(`${path}.${sectionId}`, "unknown Engine config section")
    }
  }

  let enabledSectionCount = 0
  for (const section of sections) {
    const sectionPath = `${path}.${section.id}`
    const rawSection = engineConfig[section.id]
    if (!isRecord(rawSection)) {
      throw new WorkflowConfigurationDraftError(sectionPath, "Engine config section is required")
    }
    if (typeof rawSection.enabled !== "boolean") {
      throw new WorkflowConfigurationDraftError(`${sectionPath}.enabled`, "Engine config section enabled must be boolean")
    }
    if (!rawSection.enabled) {
	  if (section.requiredEnabled) {
		throw new WorkflowConfigurationDraftError(`${sectionPath}.enabled`, "required Engine config section must be enabled")
	  }
      if (Object.keys(rawSection).length !== 1) {
        throw new WorkflowConfigurationDraftError(sectionPath, "disabled Engine config section must contain only enabled")
      }
      continue
    }
    enabledSectionCount += 1
    const params = new Set(section.params.map((param) => param.key))
    for (const key of Object.keys(rawSection)) {
      if (key !== "enabled" && !params.has(key)) {
        throw new WorkflowConfigurationDraftError(`${sectionPath}.${key}`, "unknown Engine config parameter")
      }
    }
    for (const param of section.params) {
      if (!(param.key in rawSection)) {
        throw new WorkflowConfigurationDraftError(`${sectionPath}.${param.key}`, "Engine config parameter is required")
      }
      if (!isValidEngineParameterValue(rawSection[param.key], param.type)) {
        throw new WorkflowConfigurationDraftError(`${sectionPath}.${param.key}`, "Engine config parameter has an invalid type")
      }
      validateEngineParameterConstraints(rawSection[param.key], param, `${sectionPath}.${param.key}`)
    }
  }
  if (enabledSectionCount === 0) {
    throw new WorkflowConfigurationDraftError(path, "at least one Engine config section must be enabled")
  }
}

function validateEngineParameterConstraints(value: unknown, param: { type: string; enum?: string[]; minItems?: number; maxItems?: number }, path: string): void {
  if (param.type !== "stringArray") return
  if (!Array.isArray(value)) return
  if (param.minItems !== undefined && value.length < param.minItems) {
    throw new WorkflowConfigurationDraftError(path, `Engine config parameter must contain at least ${param.minItems} items`)
  }
  if (param.maxItems !== undefined && value.length > param.maxItems) {
    throw new WorkflowConfigurationDraftError(path, `Engine config parameter must contain at most ${param.maxItems} items`)
  }
  const seen = new Set<string>()
  for (const item of value) {
    if (typeof item !== "string") continue
    if (seen.has(item)) {
      throw new WorkflowConfigurationDraftError(path, "Engine config parameter must not contain duplicate values")
    }
    seen.add(item)
    if (param.enum && !param.enum.includes(item)) {
      throw new WorkflowConfigurationDraftError(path, "Engine config parameter contains a value outside the enum")
    }
  }
}

/**
 * Serializes a draft or YAML object into the canonical request envelope. The
 * disabled branch intentionally drops the retained Profile engineConfig.
 */
export function serializeCanonicalWorkflowConfiguration(
  input: unknown,
  workflow: ScanWorkflow | { name: string; steps: Array<{ stepId: string }> },
  workflowWithEngines?: ScanWorkflowWithEngines,
): CanonicalWorkflowConfiguration {
  const parsed = parseWorkflowConfigurationStrict(input)
  if (Object.keys(parsed).some((key) => key !== "steps")) {
    throw new WorkflowConfigurationDraftError("configuration", "configuration must contain only steps")
  }
  if (!isRecord(parsed.steps)) {
    throw new WorkflowConfigurationDraftError("configuration.steps", "steps must be an object")
  }
  const stepIds = workflow.steps.map((step) => step.stepId)
  const expected = new Set(stepIds)
  for (const stepId of Object.keys(parsed.steps)) {
    if (!expected.has(stepId)) {
      throw new WorkflowConfigurationDraftError(`configuration.steps["${stepId}"]`, "unknown Workflow Step")
    }
  }
  for (const stepId of stepIds) {
    if (!(stepId in parsed.steps)) {
      throw new WorkflowConfigurationDraftError(`configuration.steps["${stepId}"]`, "Workflow Step is required")
    }
  }

  validateCanonicalStepEnvelope(parsed.steps)

  const steps: Record<string, CanonicalWorkflowStep> = {}
  for (const stepId of stepIds) {
    const rawStep = parsed.steps[stepId] as Record<string, unknown>
    if (rawStep.enabled === false) {
      steps[stepId] = { enabled: false }
      continue
    }
    if (workflowWithEngines) {
      if (workflowIdentity(workflowWithEngines.scanWorkflowId) !== workflowIdentity(workflow.name)) {
        throw new WorkflowConfigurationDraftError("scanWorkflow", "Engine schema parent does not match the selected Workflow")
      }
      const engineStep = findWorkflowEngineStep(workflowWithEngines, stepId)
      if (!engineStep) {
        throw new WorkflowConfigurationDraftError(`configuration.steps["${stepId}"]`, "Workflow Step engine metadata is unavailable")
      }
      validateCompleteProfileEngineConfig(
        rawStep.engineConfig as Record<string, unknown>,
        engineStep,
        `configuration.steps["${stepId}"].engineConfig`,
      )
    }
    steps[stepId] = { enabled: true, engineConfig: cloneRecord(rawStep.engineConfig as Record<string, unknown>) }
  }
  return { steps }
}

export function parseWorkflowConfigurationStrict(input: unknown): Record<string, unknown> {
  const parsed = parseWorkflowConfigurationDraftStrict(input)
  validateCanonicalStepEnvelope(parsed.steps as Record<string, unknown>)
  return parsed
}

function parseYamlObject(input: unknown): Record<string, unknown> {
  let parsed: unknown
  if (typeof input === "string") {
    try {
      parsed = yaml.load(input)
    } catch (error) {
      throw new WorkflowConfigurationDraftError(
        "configuration",
        `configuration YAML is invalid${error instanceof Error && error.message ? `: ${error.message}` : ""}`,
      )
    }
  } else {
    parsed = input
  }
  if (!isRecord(parsed)) {
    throw new WorkflowConfigurationDraftError("configuration", "configuration must be an object")
  }
  return parsed
}

function validateDraftStepEnvelope(steps: Record<string, unknown>): void {
  if (Object.keys(steps).length === 0) {
    throw new WorkflowConfigurationDraftError("configuration.steps", "at least one Workflow Step is required")
  }

  for (const [stepId, rawStep] of Object.entries(steps)) {
    const path = `configuration.steps["${stepId}"]`
    if (!isRecord(rawStep)) {
      throw new WorkflowConfigurationDraftError(path, "Step must be an object")
    }
    if (typeof rawStep.enabled !== "boolean") {
      throw new WorkflowConfigurationDraftError(`${path}.enabled`, "enabled must be an explicit boolean")
    }
    if (rawStep.enabled === false) {
      if (Object.keys(rawStep).some((key) => key !== "enabled" && key !== "engineConfig")) {
        throw new WorkflowConfigurationDraftError(path, "disabled Step contains an unknown field")
      }
      if (
        "engineConfig" in rawStep &&
        (!isRecord(rawStep.engineConfig) || Object.keys(rawStep.engineConfig).length === 0)
      ) {
        throw new WorkflowConfigurationDraftError(
          `${path}.engineConfig`,
          "retained Engine configuration must be a non-empty object",
        )
      }
      continue
    }
    if (Object.keys(rawStep).some((key) => key !== "enabled" && key !== "engineConfig")) {
      throw new WorkflowConfigurationDraftError(path, "enabled Step contains an unknown field")
    }
    if (!isRecord(rawStep.engineConfig)) {
      throw new WorkflowConfigurationDraftError(`${path}.engineConfig`, "enabled Step requires a complete object")
    }
    if (Object.keys(rawStep.engineConfig).length === 0) {
      throw new WorkflowConfigurationDraftError(
        `${path}.engineConfig`,
        "enabled Step requires a non-empty complete object",
      )
    }
  }
}

function validateCanonicalStepEnvelope(steps: Record<string, unknown>): void {
  if (Object.keys(steps).length === 0) {
    throw new WorkflowConfigurationDraftError("configuration.steps", "at least one Workflow Step must be enabled")
  }

  let enabledCount = 0
  for (const [stepId, rawStep] of Object.entries(steps)) {
    const path = `configuration.steps["${stepId}"]`
    if (!isRecord(rawStep)) {
      throw new WorkflowConfigurationDraftError(path, "Step must be an object")
    }
    if (typeof rawStep.enabled !== "boolean") {
      throw new WorkflowConfigurationDraftError(`${path}.enabled`, "enabled must be an explicit boolean")
    }
    if (rawStep.enabled === false) {
      if (Object.keys(rawStep).some((key) => key !== "enabled")) {
        throw new WorkflowConfigurationDraftError(path, "disabled Step must contain only enabled")
      }
      continue
    }
    if (Object.keys(rawStep).some((key) => key !== "enabled" && key !== "engineConfig")) {
      throw new WorkflowConfigurationDraftError(path, "enabled Step contains an unknown field")
    }
    if (!isRecord(rawStep.engineConfig)) {
      throw new WorkflowConfigurationDraftError(`${path}.engineConfig`, "enabled Step requires a complete object")
    }
    if (Object.keys(rawStep.engineConfig).length === 0) {
      throw new WorkflowConfigurationDraftError(`${path}.engineConfig`, "enabled Step requires a non-empty complete object")
    }
    enabledCount += 1
  }
  if (enabledCount === 0) {
    throw new WorkflowConfigurationDraftError("configuration.steps", "at least one Workflow Step must be enabled")
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function findWorkflowEngineStep(workflow: ScanWorkflowWithEngines, stepId: string): WorkflowStepWithEngine | null {
  for (const stage of workflow.stages) {
    const step = stage.steps.find((candidate) => candidate.stepId === stepId)
    if (step) return step
  }
  return null
}

function isValidEngineParameterValue(value: unknown, type: string): boolean {
  if (type === "integer") return typeof value === "number" && Number.isInteger(value)
  if (type === "boolean") return typeof value === "boolean"
  if (type === "string") return typeof value === "string"
  if (type === "stringArray") return Array.isArray(value) && value.every((item) => typeof item === "string")
  return false
}

function cloneRecord(value: Record<string, unknown>): Record<string, unknown> {
  return Object.fromEntries(Object.entries(value).map(([key, nested]) => [key, cloneValue(nested)]))
}

function cloneValue(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(cloneValue)
  if (isRecord(value)) return cloneRecord(value)
  return value
}

function workflowIdentity(value: string): string {
  return value.startsWith("scanWorkflows/") ? value.slice("scanWorkflows/".length) : value
}
