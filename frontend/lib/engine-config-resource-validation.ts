import type {
  EngineConfigFormValues,
  ScanWorkflowWithEngines,
} from "@/types/engine-config.types"
import type { Wordlist } from "@/types/wordlist.types"

export type ConfigResourceFieldLocation = {
  stepId: string
  sectionId: string
  paramKey: string
}

export type ConfigResourceFieldError = ConfigResourceFieldLocation & {
  key: string
  fieldId: string
}

export function configResourceFieldKey(location: ConfigResourceFieldLocation): string {
  return JSON.stringify([location.stepId, location.sectionId, location.paramKey])
}

export function configResourceFieldId(location: ConfigResourceFieldLocation): string {
  return `engine-config-${location.stepId}-${location.sectionId}-${location.paramKey}`
}

// Resource selections cross the persistence boundary only in this canonical
// form. Engine Definition defaults are plain fileName candidates and are
// intentionally rejected until the complete Catalog resolves them.
export function isCanonicalWordlistResource(value: string): boolean {
  return /^wordlists\/[1-9]\d*$/.test(value.trim()) && value === value.trim()
}

export function validateRequiredConfigResources(
  workflow: ScanWorkflowWithEngines,
  values: EngineConfigFormValues,
): ConfigResourceFieldError[] {
  const errors: ConfigResourceFieldError[] = []

  for (const stage of workflow.stages) {
    for (const step of stage.steps) {
      const stepValue = values[step.stepId]
      if (!stepValue?.enabled) continue

      for (const section of step.engine.execution.configSections) {
        const sectionValue = stepValue.sections[section.id]
        if (!sectionValue?.enabled) continue

        for (const param of section.params) {
          if (!param.resource) continue
          const value = sectionValue.params[param.key]
          if (param.resource.kind === "wordlist" && typeof value === "string" && isCanonicalWordlistResource(value)) continue
          if (param.resource.kind !== "wordlist" && typeof value === "string" && value.trim() !== "") continue

          const location = {
            stepId: step.stepId,
            sectionId: section.id,
            paramKey: param.key,
          }
          errors.push({
            ...location,
            key: configResourceFieldKey(location),
            fieldId: configResourceFieldId(location),
          })
        }
      }
    }
  }

  return errors
}

export function reconcileConfirmedWordlistCatalog(
  workflow: ScanWorkflowWithEngines,
  values: EngineConfigFormValues,
  availableWordlists: readonly Pick<Wordlist, "name" | "fileName">[],
): EngineConfigFormValues {
  const byResourceName = new Set(availableWordlists.map((wordlist) => wordlist.name))
  const byFileName = new Map<string, Array<Pick<Wordlist, "name" | "fileName">>>()
  for (const wordlist of availableWordlists) {
    const matches = byFileName.get(wordlist.fileName) ?? []
    matches.push(wordlist)
    byFileName.set(wordlist.fileName, matches)
  }
  let nextValues = values

  for (const stage of workflow.stages) {
    for (const step of stage.steps) {
      const stepValue = nextValues[step.stepId]
      if (!stepValue) continue

      for (const section of step.engine.execution.configSections) {
        const sectionValue = stepValue.sections[section.id]
        if (!sectionValue) continue

        for (const param of section.params) {
          if (param.resource?.kind !== "wordlist") continue
          const currentValue = sectionValue.params[param.key]
          if (typeof currentValue !== "string" || currentValue === "") {
            continue
          }

          const canonicalMatch = byResourceName.has(currentValue)
          const fileNameMatches = byFileName.get(currentValue) ?? []
          const resolvedValue = canonicalMatch
            ? currentValue
            : fileNameMatches.length === 1
              ? fileNameMatches[0].name
              : fileNameMatches.length === 0
                ? ""
                : currentValue
          if (resolvedValue === currentValue) continue

          if (nextValues === values) {
            nextValues = { ...values }
          }
          const currentStep = nextValues[step.stepId]
          const currentSection = currentStep.sections[section.id]
          nextValues[step.stepId] = {
            ...currentStep,
            sections: {
              ...currentStep.sections,
              [section.id]: {
                ...currentSection,
                params: {
                  ...currentSection.params,
                  [param.key]: resolvedValue,
                },
              },
            },
          }
        }
      }
    }
  }

  return nextValues
}
