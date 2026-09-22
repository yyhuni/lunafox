import { act, fireEvent, render, screen, waitFor } from "@testing-library/react"
import React from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import {
  ScanConfigViewToggle,
  type ScanConfigValidationHandle,
} from "@/components/scan/scan-config-view-toggle"
import type { CompleteWordlistCatalogState } from "@/hooks/use-wordlists"
import { serializeFormValuesToConfig } from "@/components/scan/engine-config-form"
import type { WorkflowProfileDraft } from "@/lib/workflow-config"
import type { EngineConfigFormValues, ScanWorkflowWithEngines } from "@/types/engine-config.types"

const wordlistMocks = vi.hoisted(() => ({
  state: undefined as CompleteWordlistCatalogState | undefined,
  useCatalog: vi.fn(),
}))

vi.mock("@/hooks/use-wordlists", () => ({
  useCompleteWordlistCatalogState: () => {
    wordlistMocks.useCatalog()
    return wordlistMocks.state
  },
}))

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

vi.mock("@/components/scan/scan-config-editor", () => ({
  ScanConfigEditor: ({ configuration, onChange }: { configuration: string; onChange: (value: string) => void }) => (
    <textarea aria-label="yaml-editor" value={configuration} onChange={(event) => onChange(event.target.value)} />
  ),
}))

vi.mock("@/components/ui/scroll-area", () => ({
  ScrollArea: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

describe("ScanConfigViewToggle resources", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    wordlistMocks.state = catalogState("loading", [])
  })

  it("resolves a preferred fileName only after one complete Catalog generation", async () => {
    const onSync = vi.fn()
    const props = {
      workflow,
      configuration: configuration("stale.txt", "keep.txt"),
      onChange: vi.fn(),
      onSync,
    }
    const renderToggle = () => <ScanConfigViewToggle {...props} />
    const view = render(renderToggle())
    expect(onSync).not.toHaveBeenCalled()

    wordlistMocks.state = catalogState("complete", [
      wordlistFixture(1, "stale.txt"),
      wordlistFixture(2, "keep.txt"),
    ])
    view.rerender(renderToggle())
    await waitFor(() => expect(onSync).toHaveBeenCalledTimes(1))
    expect(onSync.mock.calls[0][0]).toContain("wordlist: wordlists/1")
    expect(onSync.mock.calls[0][0]).toContain("exclude: wordlists/2")

    wordlistMocks.state = catalogState("incomplete", [wordlistFixture(1, "replacement.txt")])
    view.rerender(renderToggle())
    expect(onSync).toHaveBeenCalledTimes(1)

    wordlistMocks.state = catalogState("failed", [])
    view.rerender(renderToggle())
    expect(onSync).toHaveBeenCalledTimes(1)

    wordlistMocks.state = catalogState("complete", [
      wordlistFixture(3, "replacement.txt"),
      wordlistFixture(2, "keep.txt"),
    ])
    view.rerender(renderToggle())

    await waitFor(() => expect(onSync).toHaveBeenCalledTimes(2))
    const normalized = onSync.mock.calls[1][0] as string
    expect(normalized).toContain("wordlist: ''")
    expect(normalized).toContain("exclude: wordlists/2")
    expect(normalized).not.toContain("wordlist: replacement.txt")

    view.rerender(renderToggle())
    await waitFor(() => expect(onSync).toHaveBeenCalledTimes(2))
  })

  it("syncs normalized wordlists after the component commit", async () => {
    const onConfigSync = vi.fn()
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => undefined)

    function ParentConfigProbe() {
      const [value, setValue] = React.useState(() => configuration("stale.txt", "keep.txt"))
      const handleSync = React.useCallback((nextValue: string) => {
        setValue(nextValue)
        onConfigSync(nextValue)
      }, [])

      return <ScanConfigViewToggle workflow={workflow} configuration={value} onChange={setValue} onSync={handleSync} />
    }

    try {
      const renderProbe = () => (
        <React.StrictMode>
          <ParentConfigProbe />
        </React.StrictMode>
      )
      const view = render(renderProbe())
      wordlistMocks.state = catalogState("complete", [
        wordlistFixture(1, "replacement.txt"),
        wordlistFixture(2, "keep.txt"),
      ])
      view.rerender(renderProbe())

      await waitFor(() => expect(onConfigSync).toHaveBeenCalledTimes(1))
      expect(consoleError.mock.calls.flat().join(" ")).not.toContain("Cannot update a component")
    } finally {
      consoleError.mockRestore()
    }
  })

  it("reveals all YAML-mode errors, focuses the first field, and succeeds after repair", async () => {
    wordlistMocks.state = catalogState("complete", [
      wordlistFixture(1, "dns.txt"),
      wordlistFixture(2, "exclude.txt"),
    ])
    const ref = React.createRef<ScanConfigValidationHandle>()
    render(<ControlledConfigProbe ref={ref} />)

    fireEvent.click(screen.getByRole("switch", { name: "advancedYamlTitle" }))
    expect(screen.getByRole("textbox", { name: "yaml-editor" })).toBeInTheDocument()

    let valid = true
    act(() => {
      valid = ref.current?.validateAndReveal() ?? true
    })
    expect(valid).toBe(false)

    const wordlist = await screen.findByRole("combobox", { name: "wordlist" })
    const exclude = screen.getByRole("combobox", { name: "exclude" })
    expect(wordlist).toHaveAttribute("aria-invalid", "true")
    expect(exclude).toHaveAttribute("aria-invalid", "true")
    await waitFor(() => expect(document.activeElement).toBe(wordlist))

    await chooseOption(wordlist, "dns.txt")
    expect(screen.getAllByText("engineConfigForm.resourceRequired")).toHaveLength(1)
    await chooseOption(screen.getByRole("combobox", { name: "exclude" }), "exclude.txt")

    act(() => {
      valid = ref.current?.validateAndReveal() ?? false
    })
    expect(valid).toBe(true)
    expect(screen.queryByText("engineConfigForm.resourceRequired")).not.toBeInTheDocument()
  })

  it("caches an edited disabled-Step draft across a real editor remount without leaking it into canonical YAML", async () => {
    wordlistMocks.state = catalogState("complete", [
      wordlistFixture(1, "dns.txt"),
      wordlistFixture(2, "exclude.txt"),
      wordlistFixture(3, "session.txt"),
    ])
    const onChange = vi.fn()
    const onValidationChange = vi.fn()
    const formValuesCacheRef = { current: {} } as React.MutableRefObject<EngineConfigFormValues>
    let latestConfiguration = configuration("dns.txt", "exclude.txt")

    function SessionConfigProbe() {
      const [value, setValue] = React.useState(latestConfiguration)
      const handleChange = React.useCallback((nextValue: string) => {
        latestConfiguration = nextValue
        onChange(nextValue)
        setValue(nextValue)
      }, [])
      return (
        <ScanConfigViewToggle
          workflow={workflow}
          configuration={value}
          onChange={handleChange}
          onValidationChange={onValidationChange}
          formValuesCacheRef={formValuesCacheRef}
        />
      )
    }

    const firstView = render(<SessionConfigProbe />)
    fireEvent.click(screen.getByRole("button", { name: "engineConfigForm.expand" }))
    await chooseOption(await screen.findByRole("combobox", { name: "wordlist" }), "session.txt")
    fireEvent.click(screen.getByRole("switch", { name: "Discovery" }))

    await waitFor(() => expect(latestConfiguration).toContain("enabled: false"))
    expect(formValuesCacheRef.current.discovery.enabled).toBe(false)
    expect(formValuesCacheRef.current.discovery.sections.recon.params.wordlist).toBe("wordlists/3")

    firstView.unmount()
    render(
      <ScanConfigViewToggle
        workflow={workflow}
        configuration={latestConfiguration}
        onChange={onChange}
        onValidationChange={onValidationChange}
        formValuesCacheRef={formValuesCacheRef}
      />
    )

    await waitFor(() => expect(onValidationChange).toHaveBeenCalledWith(true))
    expect(onValidationChange.mock.calls.every(([isValid]) => isValid === true)).toBe(true)
    expect(formValuesCacheRef.current.discovery.sections.recon.params.wordlist).toBe("wordlists/3")
    expect(serializeFormValuesToConfig(formValuesCacheRef.current)).toEqual({
      steps: { discovery: { enabled: false } },
    })
  })

  it("recovers a missing disabled-Step draft from a strict Profile and rejects invalid fallback paths", async () => {
    const validCache = { current: {} } as React.MutableRefObject<EngineConfigFormValues>
    const validValidation = vi.fn()
    const validView = render(
      <ScanConfigViewToggle
        workflow={workflow}
        configuration={disabledConfiguration()}
        onChange={vi.fn()}
        onValidationChange={validValidation}
        formValuesCacheRef={validCache}
        workflowProfileDraft={workflowProfileDraft("profile.txt")}
      />
    )
    await waitFor(() => expect(validValidation).toHaveBeenCalledWith(true))
    expect(validCache.current.discovery.sections.recon.params.wordlist).toBe("profile.txt")
    validView.unmount()

    const invalidValidation = vi.fn()
    render(
      <ScanConfigViewToggle
        workflow={workflow}
        configuration={disabledConfiguration()}
        onChange={vi.fn()}
        onValidationChange={invalidValidation}
        formValuesCacheRef={{ current: {} }}
        workflowProfileDraft={workflowProfileDraft("profile.txt", "scanWorkflows/other")}
      />
    )
    await waitFor(() => expect(invalidValidation).toHaveBeenCalledWith(false))
  })

  it("keeps an incomplete enabled Step invalid even when cache and Profile are available", async () => {
    const onValidationChange = vi.fn()
    render(
      <ScanConfigViewToggle
        workflow={workflow}
        configuration="steps:\n  discovery:\n    enabled: true"
        onChange={vi.fn()}
        onValidationChange={onValidationChange}
        formValuesCacheRef={{ current: disabledSessionValues("session.txt") }}
        workflowProfileDraft={workflowProfileDraft("profile.txt")}
      />
    )

    await waitFor(() => expect(onValidationChange).toHaveBeenCalledWith(false))
  })
})

const ControlledConfigProbe = React.forwardRef<ScanConfigValidationHandle>(function ControlledConfigProbe(_, ref) {
  const [value, setValue] = React.useState(() => configuration("", ""))
  return (
    <ScanConfigViewToggle
      ref={ref}
      workflow={workflow}
      configuration={value}
      onChange={setValue}
    />
  )
})

async function chooseOption(trigger: HTMLElement, name: string) {
  fireEvent.click(trigger)
  const option = await screen.findByRole("option", { name })
  fireEvent.mouseMove(option)
  fireEvent.click(option)
  await waitFor(() => expect(trigger).toHaveTextContent(name))
}

function catalogState(
  status: CompleteWordlistCatalogState["status"],
  wordlists: CompleteWordlistCatalogState["wordlists"],
): CompleteWordlistCatalogState {
  return { status, wordlists, retry: vi.fn() }
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

function configuration(wordlist: string, exclude: string): string {
  return `steps:\n  discovery:\n    enabled: true\n    engineConfig:\n      recon:\n        enabled: true\n        wordlist: '${wordlist}'\n        exclude: '${exclude}'`
}

function disabledConfiguration(): string {
  return "steps:\n  discovery:\n    enabled: false"
}

function disabledSessionValues(wordlist: string): EngineConfigFormValues {
  return {
    discovery: {
      enabled: false,
      sections: {
        recon: {
          enabled: true,
          params: { wordlist, exclude: "exclude.txt" },
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
          recon: { enabled: true, wordlist, exclude: "exclude.txt" },
        },
      },
    },
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
            ],
          }],
        },
      },
    }],
  }],
}
