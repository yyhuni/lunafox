import * as React from "react"
import { act, fireEvent, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { EngineInstallationPage } from "@/components/tools/engines/engine-installation-page"
import { renderWithProviders } from "@/test/utils/render-with-providers"
import type { EngineCatalogDetail, EngineCatalogSummary } from "@/types/engine-catalog.types"

const hookMocks = vi.hoisted(() => ({
  useEngineCatalog: vi.fn(),
  useEngineCatalogDetail: vi.fn(),
  useInstallEngine: vi.fn(),
}))

vi.mock("@/hooks/use-engine-catalog", () => ({
  useEngineCatalog: hookMocks.useEngineCatalog,
  useEngineCatalogDetail: hookMocks.useEngineCatalogDetail,
  useInstallEngine: hookMocks.useInstallEngine,
}))

vi.mock("@/components/shared/form-drawer", () => ({
  FormDrawer: ({ open, onOpenChange, trigger, children, footer, formProps }: {
    open: boolean
    onOpenChange: (open: boolean) => void
    trigger: React.ReactNode
    children: React.ReactNode
    footer: React.ReactNode
    formProps?: React.ComponentProps<"form">
  }) => {
    const triggerElement = React.isValidElement<{ onClick?: () => void }>(trigger)
      ? React.cloneElement(trigger, { onClick: () => onOpenChange(true) })
      : trigger
    return <>{triggerElement}{open ? <form {...formProps}>{children}{footer}</form> : null}</>
  },
}))

vi.mock("@/components/ui/alert-dialog", () => ({
  AlertDialog: ({ open, children }: { open: boolean; children: React.ReactNode }) => open ? <div>{children}</div> : null,
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => <p>{children}</p>,
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => <h2>{children}</h2>,
}))

const artifactRef = "registry.example/team/engine@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

const engine: EngineCatalogSummary = {
  name: "engines/engine.example.scanner",
  engineId: "engine.example.scanner",
  manifestVersion: "engine.v5",
  publisher: "example",
  packageVersion: "1.0.0",
  artifactRef,
  packageDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  execution: {
    engineApiMajor: 2,
    supportedTargetTypes: ["domain"],
    executionResources: ["managedExampleCorpus"],
  },
  localeResources: {
    en: {
      engine: { displayName: "Example Scanner", description: "Example scanner" },
      sections: {
        scan: {
          name: "Scanner",
          description: "Configure the scanner.",
          params: { timeout: { description: "Execution timeout" } },
        },
      },
    },
  },
}

const engineDetail: EngineCatalogDetail = {
  ...engine,
  execution: {
    ...engine.execution,
    configSections: [{
      id: "scan",
      defaultEnabled: true,
      params: [{ key: "timeout", type: "integer", default: 3600, minimum: 1, maximum: 7200 }],
    }],
  },
}

describe("EngineInstallationPage", () => {
  const mutateAsync = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    hookMocks.useEngineCatalog.mockReturnValue({ data: [engine], isPending: false, isError: false, refetch: vi.fn() })
    hookMocks.useEngineCatalogDetail.mockReturnValue({
      data: engineDetail,
      isPending: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    })
    hookMocks.useInstallEngine.mockReturnValue({ mutateAsync, isPending: false })
  })

  it("renders backend summary fields and opens the exact engine detail", async () => {
    renderWithProviders(<EngineInstallationPage />)

    const card = screen.getByRole("button", { name: /Example Scanner/ })
    expect(card).toHaveTextContent("1.0.0")
    expect(card).toHaveTextContent("example")
    expect(card).toHaveTextContent("domain")
    expect(card).toHaveTextContent("1")
    expect(card).not.toHaveTextContent("inputTypes")

    await act(async () => {
      fireEvent.click(card)
    })

    expect(hookMocks.useEngineCatalogDetail).toHaveBeenLastCalledWith("engine.example.scanner", true)
    expect(await screen.findByText(artifactRef)).toBeInTheDocument()
    expect(screen.getByText("Example scanner", { selector: '[data-slot="engine-detail-description"]' })).toBeInTheDocument()
    expect(screen.getByText("engineApiVersion")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "configuration" })).toBeInTheDocument()
  })

  it("submits first with replacement disabled, leaves current catalog unchanged on cancel, then retries after explicit confirmation", async () => {
    const conflict = {
      response: {
        status: 409,
        data: {
          error: {
            details: [
              { field: "engineId", message: engine.engineId },
              { field: "currentPackageDigest", message: engine.packageDigest },
              { field: "proposedPackageDigest", message: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" },
            ],
          },
        },
      },
    }
    mutateAsync.mockRejectedValueOnce(conflict).mockRejectedValueOnce(conflict).mockResolvedValueOnce(engine)
    renderWithProviders(<EngineInstallationPage />)

    await openAndSubmitArtifact()
    expect(mutateAsync).toHaveBeenLastCalledWith({ artifactRef, allowReplacement: false })
    expect(await screen.findByText(engine.packageDigest)).toBeInTheDocument()

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "cancel" }))
    })
    expect(mutateAsync).toHaveBeenCalledTimes(1)
    expect(mutateAsync).not.toHaveBeenCalledWith({ artifactRef, allowReplacement: true })

    await act(async () => {
      fireEvent.click(screen.getAllByRole("button", { name: "install" }).at(-1)!)
      await Promise.resolve()
    })
    await screen.findByText(engine.packageDigest)
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "replace" }))
      await Promise.resolve()
    })

    expect(mutateAsync).toHaveBeenLastCalledWith({ artifactRef, allowReplacement: true })
    expect(mutateAsync).toHaveBeenCalledTimes(3)
  })
})

async function openAndSubmitArtifact() {
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "install" }))
  })
  await act(async () => {
    fireEvent.change(screen.getByLabelText("artifactRef"), { target: { value: artifactRef } })
    fireEvent.click(screen.getAllByRole("button", { name: "install" }).at(-1)!)
    await Promise.resolve()
  })
}
