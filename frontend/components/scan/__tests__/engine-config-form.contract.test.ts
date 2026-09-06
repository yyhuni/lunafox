import { fireEvent, render, screen } from "@testing-library/react"
import React from "react"
import { describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

import {
  EngineConfigForm,
  initFormValuesFromWorkflow,
  serializeFormValuesToConfig,
} from "@/components/scan/engine-config-form"
import type { ScanWorkflowWithEngines } from "@/types/engine-config.types"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string, values?: Record<string, string | number>) => {
    if (!values) return key

    return Object.entries(values).reduce(
      (message, [name, value]) => message.replace(`{${name}}`, String(value)),
      key
    )
  },
}))

const source = readFileSync(path.resolve(process.cwd(), "components/scan/engine-config-form.tsx"), "utf8")
const typeSource = readFileSync(path.resolve(process.cwd(), "types/engine-config.types.ts"), "utf8")
const paramHelpTextSource = source.slice(
  source.indexOf("function ParamHelpText"),
  source.indexOf("function ParamField")
)

const workflowWithMultiSectionEngine: ScanWorkflowWithEngines = {
  scanWorkflowId: "subdomain_discovery",
  displayName: "Subdomain Discovery",
  description: "",
  stages: [
    {
      stageId: "discovery",
      steps: [
        {
          stepId: "subdomain_discovery",
          engineId: "engine.lunafox.subdomain_discovery",
          profileDefaultEnabled: true,
          engine: {
            manifestVersion: "engine.v5",
            engineId: "engine.lunafox.subdomain_discovery",
            displayName: "Subdomain Discovery",
            description: "",
            publisher: "lunafox",
            execution: {
              engineApiMajor: 2,
              supportedTargetTypes: ["domain"],
              configSections: [
                {
                  id: "recon",
                  name: "Reconnaissance",
                  description: "Collect subdomains",
                  defaultEnabled: true,
                  params: [],
                },
                {
                  id: "bruteforce",
                  name: "Dictionary Bruteforce",
                  description: "Bruteforce subdomains",
                  defaultEnabled: true,
                  params: [],
                },
                {
                  id: "resolve",
                  name: "Resolution",
                  description: "Resolve subdomains",
                  defaultEnabled: true,
                  requiredEnabled: true,
                  params: [],
                },
              ],
            },
          },
        },
      ],
    },
  ],
}

function profileConfigurationFor(workflow: ScanWorkflowWithEngines) {
  return {
    steps: Object.fromEntries(workflow.stages.flatMap((stage) => stage.steps).map((step) => [
      step.stepId,
      {
        enabled: false,
        engineConfig: Object.fromEntries(step.engine.execution.configSections.map((section) => [
          section.id,
          section.defaultEnabled === false
            ? { enabled: false }
            : {
                enabled: true,
                ...Object.fromEntries(section.params.map((param) => [
                  param.key,
                  param.default ?? (param.type === "integer" ? 1 : param.type === "boolean" ? false : ""),
                ])),
              },
        ])),
      },
    ])),
  }
}

describe("engine-config-form contract", () => {
  it("aligns section defaults with engine manifest metadata", () => {
    expect(typeSource).toContain("defaultEnabled?: boolean")
    expect(typeSource).toContain("requiredEnabled?: boolean")
    expect(source).toContain("validateCompleteProfileEngineConfig")
    expect(source).not.toContain("enabled: section.defaultEnabled ?? true")
    expect(source).not.toContain("stepValues.sections[section.id] ?? { enabled: true, params: {} }")
    expect(source).not.toContain("sectionData.params[param.key] ?? param.default")
  })

  it("initializes from Engine defaults without reading workflow configuration", () => {
    expect(source).not.toContain("applyStepEngineConfig")
    expect(source).not.toContain("step.engineConfig")
  })

  it("renders manifest descriptions at section and parameter levels", () => {
    expect(source).toContain("section.description")
    expect(source).toContain("param.description")
    expect(source).toContain("section.params.map((param)")
    expect(source).not.toContain("section.tools.map")
    expect(source).not.toContain("getToolParamCount")
  })

  it("shows clamped parameter descriptions through a quick tooltip", () => {
    expect(source).toContain('from "@/components/ui/tooltip"')
    expect(source).toContain("PARAM_HELP_HOVER_OPEN_DELAY_MS = 200")
    expect(source).toContain("function ParamHelpText")
    expect(source).toContain("<TooltipProvider delay={PARAM_HELP_HOVER_OPEN_DELAY_MS}>")
    expect(source).toContain("<TooltipTrigger")
    expect(source).toContain("<TooltipContent")
    expect(source).not.toContain('from "@/components/ui/hover-card"')
    expect(paramHelpTextSource).not.toContain("title={param.description}")
  })

  it("keeps section descriptions available through native title", () => {
    expect(source).toContain("title={section.description}")
  })

  it("renders integer constraints beside the parameter label", () => {
    expect(source).toContain("constraintHint")
    expect(source).toContain('tScanInitiate("engineConfigForm.minimum", { value: param.minimum })')
    expect(source).toContain('tScanInitiate("engineConfigForm.maximum", { value: param.maximum })')
    expect(source).toContain("justify-between")
    expect(source).not.toContain("paramHelpText")
  })

  it("localizes engine configuration form chrome", () => {
    expect(source).toContain('useTranslations("scan.initiate")')
    expect(source).not.toContain('useTranslations("scan.initiate.engineConfigForm")')
    expect(source).toContain('tScanInitiate("engineConfigForm.stageLabel", { index: stageIndex + 1, stageId: stage.stageId })')
    expect(source).toContain('tScanInitiate("engineConfigForm.enabledSummary", { enabled: enabledEngineCount, total: stage.steps.length })')
    expect(source).not.toContain("个配置区")
    expect(source).not.toContain("启用")
  })

  it("does not expose section-level notes as scan form warnings", () => {
    expect(typeSource).not.toContain("notes?:")
    expect(source).not.toContain("section.notes")
  })

  it("uses narrow requiredEnabled metadata without reintroducing generic section rules", () => {
    expect(typeSource).not.toContain("required?: boolean")
    expect(source).toContain("section.requiredEnabled")
    expect(source).not.toContain("section.group")
    expect(source).not.toContain("section.condition")
  })

  it("keeps engine cards collapsed before the user expands detailed parameters", () => {
    expect(source).toContain("React.useState<Set<string>>(() => new Set())")
    expect(source).toContain("expandedStepIds?: ReadonlySet<string>")
    expect(source).toContain("open={expandedStepIds.has(step.stepId)}")
  })

  it("renders integer engine parameters with the shared number stepper", () => {
    expect(source).toContain('from "@/components/ui/number-stepper-input"')
    expect(source).toContain("<NumberStepperInput")
    expect(source).toContain("getIntegerStep(param)")
    expect(source).toContain("param.type === \"integer\"")
  })

  it("renders resource-backed wordlist parameters through the wordlist resource selector", () => {
    expect(typeSource).toContain("resource?: EngineParamResourceBinding")
    expect(typeSource).toContain('kind: "wordlist"')
    expect(source).toContain('param.resource?.kind === "wordlist"')
    expect(source).toContain("<WordlistResourceSelect")
    expect(source).not.toContain('param.key === "wordlist"')
    expect(source).not.toContain("useCompleteWordlistCatalogState")
    expect(source).not.toContain("useWordlists(")
  })

  it("uses the engine-level toggle as the Workflow Step switch", () => {
    const values = initFormValuesFromWorkflow(workflowWithMultiSectionEngine, profileConfigurationFor(workflowWithMultiSectionEngine))
    const handleChange = vi.fn()

    render(React.createElement(EngineConfigForm, {
      workflow: workflowWithMultiSectionEngine,
      values,
      onChange: handleChange,
    }))

    fireEvent.click(screen.getByRole("switch", { name: "Subdomain Discovery" }))

    expect(handleChange).toHaveBeenCalledTimes(1)
    expect(handleChange).toHaveBeenCalledWith(expect.objectContaining({
      subdomain_discovery: expect.objectContaining({
        enabled: true,
        sections: expect.objectContaining({
          recon: expect.objectContaining({ enabled: true }),
          bruteforce: expect.objectContaining({ enabled: true }),
          resolve: expect.objectContaining({ enabled: true }),
        }),
      }),
    }))
  })

  it("rejects disabling the final enabled config section without changing the Step", () => {
    const workflow = {
      ...workflowWithMultiSectionEngine,
      stages: [{
        ...workflowWithMultiSectionEngine.stages[0],
        steps: [{
          ...workflowWithMultiSectionEngine.stages[0].steps[0],
          engine: {
            ...workflowWithMultiSectionEngine.stages[0].steps[0].engine,
            execution: {
              ...workflowWithMultiSectionEngine.stages[0].steps[0].engine.execution,
              configSections: [workflowWithMultiSectionEngine.stages[0].steps[0].engine.execution.configSections[0]],
            },
          },
        }],
      }],
    }
    const values = initFormValuesFromWorkflow(workflow, profileConfigurationFor(workflow))
    const handleChange = vi.fn()

    render(React.createElement(EngineConfigForm, { workflow, values, onChange: handleChange }))
    fireEvent.click(screen.getByRole("button", { name: "engineConfigForm.expand" }))
    fireEvent.click(screen.getAllByRole("switch")[1])

    expect(handleChange).not.toHaveBeenCalled()
  })

  it("serializes disabled Steps without engine configuration", () => {
    const values = initFormValuesFromWorkflow(workflowWithMultiSectionEngine, profileConfigurationFor(workflowWithMultiSectionEngine))
    values.subdomain_discovery.enabled = false

    expect(serializeFormValuesToConfig(values)).toEqual({
      steps: { subdomain_discovery: { enabled: false } },
    })
  })

  it("locks a required inner section without taking control of the outer Workflow Step", () => {
    const values = initFormValuesFromWorkflow(workflowWithMultiSectionEngine, profileConfigurationFor(workflowWithMultiSectionEngine))
    values.subdomain_discovery.enabled = true
    const handleChange = vi.fn()

    render(React.createElement(EngineConfigForm, {
      workflow: workflowWithMultiSectionEngine,
      values,
      onChange: handleChange,
    }))

    fireEvent.click(screen.getByRole("button", { name: "engineConfigForm.expand" }))
    const resolveToggle = screen.getByRole("switch", { name: "Resolution" })
    expect(resolveToggle).toBeChecked()
    expect(resolveToggle).toBeDisabled()
    fireEvent.click(resolveToggle)
    expect(handleChange).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole("switch", { name: "Subdomain Discovery" }))
    expect(handleChange).toHaveBeenCalledWith(expect.objectContaining({
      subdomain_discovery: expect.objectContaining({ enabled: false }),
    }))
  })

  it("keeps Bruteforce and Resolve resolver Wordlists independent", () => {
    const workflow = {
      ...workflowWithMultiSectionEngine,
      stages: [{
        ...workflowWithMultiSectionEngine.stages[0],
        steps: [{
          ...workflowWithMultiSectionEngine.stages[0].steps[0],
          engine: {
            ...workflowWithMultiSectionEngine.stages[0].steps[0].engine,
            execution: {
              ...workflowWithMultiSectionEngine.stages[0].steps[0].engine.execution,
              configSections: [
                {
                  id: "bruteforce",
                  name: "Dictionary Bruteforce",
                  defaultEnabled: true,
                  params: [
                    { key: "wordlist", type: "string" as const, default: "subdomains.txt", resource: { kind: "wordlist" as const } },
                    { key: "resolvers", type: "string" as const, default: "bruteforce-resolvers.txt", resource: { kind: "wordlist" as const } },
                  ],
                },
                {
                  id: "resolve",
                  name: "Resolution",
                  defaultEnabled: true,
                  requiredEnabled: true,
                  params: [{ key: "resolvers", type: "string" as const, default: "resolve-resolvers.txt", resource: { kind: "wordlist" as const } }],
                },
              ],
            },
          },
        }],
      }],
    }
    const values = initFormValuesFromWorkflow(workflow, {
      steps: {
        subdomain_discovery: {
          enabled: true,
          engineConfig: {
            bruteforce: { enabled: true, wordlist: "subdomains.txt", resolvers: "bruteforce-resolvers.txt" },
            resolve: { enabled: true, resolvers: "resolve-resolvers.txt" },
          },
        },
      },
    })

    expect(serializeFormValuesToConfig(values)).toEqual({
      steps: {
        subdomain_discovery: {
          enabled: true,
          engineConfig: {
            bruteforce: { enabled: true, wordlist: "subdomains.txt", resolvers: "bruteforce-resolvers.txt" },
            resolve: { enabled: true, resolvers: "resolve-resolvers.txt" },
          },
        },
      },
    })
  })

  it("restores the retained Profile engineConfig when a disabled Step is re-enabled", () => {
    const profile = profileConfigurationFor(workflowWithMultiSectionEngine)
    const values = initFormValuesFromWorkflow(workflowWithMultiSectionEngine, profile)
    const retained = values.subdomain_discovery.sections.recon.params
    values.subdomain_discovery.enabled = true

    const serialized = serializeFormValuesToConfig(values) as { steps: Record<string, unknown> }
    expect(serialized.steps.subdomain_discovery).toMatchObject({
      enabled: true,
      engineConfig: {
        recon: expect.objectContaining({ enabled: true, ...retained }),
      },
    })
  })

  it("preserves string-array Engine defaults through Profile initialization and form serialization", () => {
    const workflow = {
      ...workflowWithMultiSectionEngine,
      stages: [{
        ...workflowWithMultiSectionEngine.stages[0],
        steps: [{
          ...workflowWithMultiSectionEngine.stages[0].steps[0],
          engine: {
            ...workflowWithMultiSectionEngine.stages[0].steps[0].engine,
            execution: {
              ...workflowWithMultiSectionEngine.stages[0].steps[0].engine.execution,
              configSections: [{
                id: "filters",
                name: "Filters",
                defaultEnabled: true,
                params: [{ key: "whitelist", type: "stringArray" as const, default: ["safe"] }],
              }],
            },
          },
        }],
      }],
    }
    const values = initFormValuesFromWorkflow(workflow, {
      steps: {
        subdomain_discovery: {
          enabled: false,
          engineConfig: { filters: { enabled: true, whitelist: ["safe", "internal"] } },
        },
      },
    })

    expect(values.subdomain_discovery.sections.filters.params.whitelist).toEqual(["safe", "internal"])
    values.subdomain_discovery.enabled = true
    expect(serializeFormValuesToConfig(values)).toEqual({
      steps: {
        subdomain_discovery: {
          enabled: true,
          engineConfig: {
            filters: { enabled: true, whitelist: ["safe", "internal"] },
          },
        },
      },
    })
  })

  it("renders Nuclei scan targets and severity as multi-select popovers", () => {
    const workflow = {
      ...workflowWithMultiSectionEngine,
      stages: [{
        ...workflowWithMultiSectionEngine.stages[0],
        steps: [{
          ...workflowWithMultiSectionEngine.stages[0].steps[0],
          engine: {
            ...workflowWithMultiSectionEngine.stages[0].steps[0].engine,
            execution: {
              ...workflowWithMultiSectionEngine.stages[0].steps[0].engine.execution,
              configSections: [{
                id: "nuclei",
                name: "Nuclei",
                defaultEnabled: true,
              params: [
                { key: "scan-targets", type: "stringArray" as const, default: ["website"], enum: ["website", "endpoint"], minItems: 1, maxItems: 2 },
                { key: "severity", type: "stringArray" as const, default: ["medium", "high", "critical"], enum: ["info", "low", "medium", "high", "critical"] },
              ],
              }],
            },
          },
        }],
      }],
    }
    const values = initFormValuesFromWorkflow(workflow, {
      steps: { subdomain_discovery: { enabled: true, engineConfig: { nuclei: { enabled: true, "scan-targets": ["website"], severity: ["medium", "high", "critical"] } } } },
    })
    const handleChange = vi.fn()
    render(React.createElement(EngineConfigForm, { workflow, values, onChange: handleChange }))
    fireEvent.click(screen.getByRole("button", { name: "engineConfigForm.expand" }))
    expect(screen.queryByRole("checkbox", { name: "website" })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "scan-targets" }))
    const website = screen.getByRole("checkbox", { name: "website" })
    const endpoint = screen.getByRole("checkbox", { name: "endpoint" })
    expect(website).toBeChecked()
    expect(endpoint).not.toBeChecked()
    fireEvent.click(website)
    expect(handleChange).not.toHaveBeenCalled()
    fireEvent.click(endpoint)
    expect(screen.getByRole("checkbox", { name: "website" })).toBeInTheDocument()
    expect(handleChange).toHaveBeenCalledWith(expect.objectContaining({
      subdomain_discovery: expect.objectContaining({
        sections: expect.objectContaining({ nuclei: expect.objectContaining({ params: expect.objectContaining({ "scan-targets": ["website", "endpoint"] }) }) }),
      }),
    }))

    fireEvent.click(screen.getByRole("button", { name: "scan-targets" }))
    expect(screen.queryByRole("checkbox", { name: "medium" })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "severity" }))
    expect(screen.getByRole("checkbox", { name: "medium" })).toBeChecked()
    expect(screen.getByRole("checkbox", { name: "high" })).toBeChecked()
    expect(screen.getByRole("checkbox", { name: "critical" })).toBeChecked()
    handleChange.mockClear()
    fireEvent.click(screen.getByRole("checkbox", { name: "info" }))
    expect(handleChange).toHaveBeenCalledWith(expect.objectContaining({
      subdomain_discovery: expect.objectContaining({
        sections: expect.objectContaining({ nuclei: expect.objectContaining({ params: expect.objectContaining({ severity: ["info", "medium", "high", "critical"] }) }) }),
      }),
    }))
  })

  it("keeps scalar enum parameters on the Select path", () => {
    expect(source).toContain('param.type === "string" && param.enum')
    expect(source).toContain('param.type === "stringArray" && param.enum')
  })
})
