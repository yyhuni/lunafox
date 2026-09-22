import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import React from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import {
  EngineConfigForm,
  initFormValuesFromWorkflow,
  serializeFormValuesToConfig,
} from "@/components/scan/engine-config-form"
import {
  configResourceFieldKey,
  type ConfigResourceFieldError,
  type ConfigResourceFieldLocation,
} from "@/lib/engine-config-resource-validation"
import type { CompleteWordlistCatalogState } from "@/hooks/use-wordlists"
import type {
  EngineConfigFormValues,
  ScanWorkflowWithEngines,
} from "@/types/engine-config.types"
import type { WorkflowProfileDraft } from "@/lib/workflow-config"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

describe("EngineConfigForm resources", () => {
  beforeEach(() => vi.clearAllMocks())

  it("renders a disabled selector and retry action instead of free text on Catalog failure", () => {
    const retry = vi.fn()
    render(
      <EngineConfigForm
        workflow={workflow}
        values={formValues()}
        expandedStepIds={new Set(["discovery"])}
        wordlistCatalog={{ status: "failed", wordlists: [], retry }}
        onChange={vi.fn()}
      />
    )

    expect(screen.getByRole("combobox", { name: "wordlist" })).toBeDisabled()
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument()
    fireEvent.click(screen.getAllByRole("button", { name: "engineConfigForm.retryWordlists" })[0])
    expect(retry).toHaveBeenCalledTimes(1)
  })

  it("associates every field error and clears only the repaired selector", async () => {
    render(<ResourceFormProbe />)

    const wordlist = screen.getByRole("combobox", { name: "wordlist" })
    const exclude = screen.getByRole("combobox", { name: "exclude" })
    expect(wordlist).toHaveAttribute("aria-invalid", "true")
    expect(exclude).toHaveAttribute("aria-invalid", "true")
    expect(wordlist).toHaveAttribute(
      "aria-describedby",
      "engine-config-discovery-recon-wordlist-error"
    )
    expect(exclude).toHaveAttribute(
      "aria-describedby",
      "engine-config-discovery-recon-exclude-error"
    )
    expect(screen.getAllByText("engineConfigForm.resourceRequired")).toHaveLength(2)

    fireEvent.click(wordlist)
    const option = await screen.findByRole("option", { name: "dns.txt" })
    fireEvent.mouseMove(option)
    fireEvent.click(option)

    await waitFor(() => {
      expect(screen.getAllByText("engineConfigForm.resourceRequired")).toHaveLength(1)
    })
    expect(screen.getByRole("combobox", { name: "exclude" })).toHaveAttribute("aria-invalid", "true")
  })

  it("disables an uninitialized Workflow Step instead of emitting a partial configuration", () => {
    const onChange = vi.fn()
    render(<EngineConfigForm workflow={workflow} values={{}} onChange={onChange} />)

    const stepSwitch = screen.getByRole("switch", { name: "Discovery" })
    expect(stepSwitch).toBeDisabled()
    fireEvent.click(stepSwitch)
    expect(onChange).not.toHaveBeenCalled()
  })

  it("renders boolean Engine parameters as joined sliding controls and preserves their config value", () => {
    const onChange = vi.fn()
    render(<EngineConfigForm workflow={workflow} values={formValues()} expandedStepIds={new Set(["discovery"])} onChange={onChange} />)

    fireEvent.click(screen.getByRole("button", { name: "engineConfigForm.off" }))
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({
      discovery: expect.objectContaining({
        sections: expect.objectContaining({
          recon: expect.objectContaining({
            params: expect.objectContaining({ "auto-calibration": false }),
          }),
        }),
      }),
    }))
  })
})

describe("EngineConfigForm disabled-Step recovery", () => {
  const canonicalDisabledConfiguration = {
    steps: {
      discovery: { enabled: false },
    },
  }

  it("uses the same-dialog draft before Profile recovery and still serializes only the disabled branch", () => {
    const values = initFormValuesFromWorkflow(
      workflow,
      canonicalDisabledConfiguration,
      disabledSessionValues("session.txt"),
      workflowProfileDraft("profile.txt"),
    )

    expect(values.discovery.enabled).toBe(false)
    expect(values.discovery.sections.recon.params.wordlist).toBe("session.txt")
    expect(serializeFormValuesToConfig(values)).toEqual({
      steps: {
        discovery: { enabled: false },
      },
    })
  })

  it("uses a complete strictly shaped Profile draft only when a disabled Step has no session draft", () => {
    const values = initFormValuesFromWorkflow(
      workflow,
      canonicalDisabledConfiguration,
      {},
      workflowProfileDraft("profile.txt"),
    )

    expect(values.discovery.enabled).toBe(false)
    expect(values.discovery.sections.recon.params.wordlist).toBe("profile.txt")
  })

  it("rejects an unavailable or mismatched Profile instead of synthesizing a disabled-Step draft", () => {
    expect(() => initFormValuesFromWorkflow(workflow, canonicalDisabledConfiguration)).toThrow(
      "validated Workflow Profile draft is required",
    )
    expect(() => initFormValuesFromWorkflow(
      workflow,
      canonicalDisabledConfiguration,
      {},
      workflowProfileDraft("profile.txt", "scanWorkflows/other"),
    )).toThrow("Profile parent does not match the selected Workflow")
  })

  it("does not repair an incomplete enabled Step from a session draft or Profile", () => {
    expect(() => initFormValuesFromWorkflow(
      workflow,
      { steps: { discovery: { enabled: true } } },
      disabledSessionValues("session.txt"),
      workflowProfileDraft("profile.txt"),
    )).toThrow("enabled Step requires a complete object")
  })
})

function ResourceFormProbe() {
  const [values, setValues] = React.useState(() => {
    const initial = formValues()
    initial.discovery.sections.recon.params = { wordlist: "", exclude: "" }
    return initial
  })
  const [errors, setErrors] = React.useState(() => new Map([
    fieldError({ stepId: "discovery", sectionId: "recon", paramKey: "wordlist" }),
    fieldError({ stepId: "discovery", sectionId: "recon", paramKey: "exclude" }),
  ]))
  const catalog: CompleteWordlistCatalogState = {
    status: "complete",
    wordlists: [wordlistFixture(1, "dns.txt"), wordlistFixture(2, "exclude.txt")],
    retry: vi.fn(),
  }
  return (
    <EngineConfigForm
      workflow={workflow}
      values={values}
      expandedStepIds={new Set(["discovery"])}
      wordlistCatalog={catalog}
      fieldErrors={errors}
      onFieldRepaired={(location) => {
        setErrors((current) => {
          const next = new Map(current)
          next.delete(configResourceFieldKey(location))
          return next
        })
      }}
      onChange={setValues}
    />
  )
}

function fieldError(location: ConfigResourceFieldLocation): [string, ConfigResourceFieldError] {
  const key = configResourceFieldKey(location)
  return [key, { ...location, key, fieldId: `engine-config-${location.stepId}-${location.sectionId}-${location.paramKey}` }]
}

function wordlistFixture(id: number, fileName: string) {
  return {
    id,
    name: `wordlists/${id}`,
    fileName,
    tags: [],
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  }
}

const workflow: ScanWorkflowWithEngines = {
  scanWorkflowId: "default",
  displayName: "Default",
  description: "",
  stages: [{
    stageId: "discovery",
    steps: [{
      stepId: "discovery",
      engineId: "engine.lunafox.discovery",
      profileDefaultEnabled: true,
      engine: {
        manifestVersion: "engine.v5",
        engineId: "engine.lunafox.discovery",
        publisher: "lunafox",
        displayName: "Discovery",
        description: "",
        execution: {
          engineApiMajor: 2,
          supportedTargetTypes: ["domain"],
          configSections: [{
            id: "recon",
            name: "Recon",
            defaultEnabled: true,
            params: [
              { key: "wordlist", type: "string", default: "dns.txt", resource: { kind: "wordlist" } },
              { key: "exclude", type: "string", default: "exclude.txt", resource: { kind: "wordlist" } },
              { key: "auto-calibration", type: "boolean", default: true, description: "Enable FFUF auto calibration." },
            ],
          }],
        },
      },
    }],
  }],
}

function formValues(): EngineConfigFormValues {
  return {
    discovery: {
      enabled: true,
      sections: {
        recon: {
          enabled: true,
          params: { wordlist: "stale.txt", exclude: "exclude.txt", "auto-calibration": true },
        },
      },
    },
  }
}

function disabledSessionValues(wordlist: string): EngineConfigFormValues {
  return {
    discovery: {
      enabled: false,
      sections: {
        recon: {
          enabled: true,
          params: { wordlist, exclude: "exclude.txt", "auto-calibration": true },
        },
      },
    },
  }
}

function workflowProfileDraft(
  wordlist: string,
  scanWorkflow = "scanWorkflows/default",
): WorkflowProfileDraft {
  return {
    scanWorkflow,
    steps: {
      discovery: {
        enabled: true,
        engineConfig: {
          recon: {
            enabled: true,
            wordlist,
            exclude: "exclude.txt",
            "auto-calibration": true,
          },
        },
      },
    },
  }
}
