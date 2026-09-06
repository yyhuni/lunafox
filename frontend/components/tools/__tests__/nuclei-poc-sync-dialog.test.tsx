import * as React from "react"
import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"

import {
  NucleiPocBatchActivationControls,
  NucleiPocSyncDialog,
  NucleiPocSyncSourceStatus,
} from "../nuclei-poc-catalog-page"
import type { NucleiPocSyncTask } from "@/types/nuclei-poc.types"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string, params?: Record<string, string | number>) => params
    ? `${key} ${Object.entries(params).map(([name, value]) => `${name}=${value}`).join(" ")}`
    : key,
  useLocale: () => "zh-CN",
}))

afterEach(() => cleanup())

function SyncDialogHarness({ onSubmit }: { onSubmit: () => void }) {
  const [sourceKind, setSourceKind] = React.useState<"git" | "gitee" | "custom">("git")
  const [sourceUrl, setSourceUrl] = React.useState("https://github.com/projectdiscovery/nuclei-templates.git")
  const [submitted, setSubmitted] = React.useState(false)
  return <NucleiPocSyncDialog
    open
    onOpenChange={() => undefined}
    sourceKind={sourceKind}
    sourceUrl={sourceUrl}
    onSourceKindChange={(value) => {
      if (value === "git" || value === "gitee" || value === "custom") {
        setSourceKind(value)
        setSourceUrl(value === "git" ? "https://github.com/projectdiscovery/nuclei-templates.git" : "")
        setSubmitted(false)
      }
    }}
    onSourceUrlChange={setSourceUrl}
    submitted={submitted}
    isSubmitting={false}
    task={null}
    taskError={null}
    taskExpired={false}
    onSubmit={() => { setSubmitted(true); onSubmit() }}
    onStartNew={() => undefined}
  />
}

function BatchActivationHarness({
  disabled = false,
  isPending = false,
  onConfirm,
}: {
  disabled?: boolean
  isPending?: boolean
  onConfirm: () => void
}) {
  const [target, setTarget] = React.useState<boolean | null>(null)
  return (
    <NucleiPocBatchActivationControls
      activationTarget={target}
      onActivationTargetChange={setTarget}
      disabled={disabled}
      isPending={isPending}
      onConfirm={onConfirm}
    />
  )
}

const completedTask: NucleiPocSyncTask = {
  name: "nucleiPocSyncTasks/00000000-0000-4000-8000-000000000001",
  requestId: "00000000-0000-4000-8000-000000000002",
  sourceType: "git",
  state: "SUCCEEDED",
  phase: "SUCCEEDED",
  counters: { filesSeen: 10, yamlFilesSeen: 8, templatesValidated: 8, bytesRead: 1024 },
  commitSha: "a1b2c3d4e5f6",
  committedPocCount: 8,
  diagnostics: { samples: [], total: 2, truncated: false },
  cleanupStatus: "clean",
  createdAt: "2026-08-18T08:00:00.000Z",
  startedAt: "2026-08-18T08:00:00.000Z",
  completedAt: "2026-08-18T08:00:00.000Z",
  updatedAt: "2026-08-18T08:00:00.000Z",
}

const runningTask: NucleiPocSyncTask = {
  ...completedTask,
  state: "SCANNING_FILES",
  phase: "SCANNING_FILES",
  diagnostics: { samples: [], total: 0, truncated: false },
  completedAt: null,
  cleanupStatus: "pending",
}

const failedTask: NucleiPocSyncTask = {
  ...completedTask,
  state: "FAILED",
  phase: "FAILED",
  committedPocCount: undefined,
  diagnostics: {
    samples: [{ category: "yaml", relativePath: "http/example.yaml", reasonCode: "TEMPLATE_INVALID" }],
    total: 1,
    truncated: false,
  },
  failureCode: "TEMPLATE_INVALID",
}

describe("NucleiPocSyncDialog", () => {
  it("shows the committed source outside the POC table", () => {
    render(<NucleiPocSyncSourceStatus source={{ sourceType: "git", repoUrl: "https://github.com/projectdiscovery/nuclei-templates.git", commitSha: "a1b2c3d4e5f6", syncedAt: "2026-08-18T08:00:00.000Z" }} loading={false} error={false} locale="zh-CN" />)
    const status = screen.getByRole("status", { name: "sync.currentSource" })
    expect(status).toHaveTextContent("sync.sourceGit")
    expect(status).toHaveTextContent("https://github.com/projectdiscovery/nuclei-templates.git")
    expect(status).toHaveTextContent("sync.committedLabel")
  })

  it("keeps the committed-source line budget while initial source metadata loads", () => {
    const { container } = render(
      <NucleiPocSyncSourceStatus source={undefined} loading error={false} locale="zh-CN" />
    )

    expect(container.querySelectorAll('[data-slot="nuclei-poc-source-status-loading-text"]')).toHaveLength(3)
    expect(screen.queryByText("sync.noSourceDescription")).not.toBeInTheDocument()
    expect(screen.queryByText("sync.notSynced")).not.toBeInTheDocument()
  })

  it("requires public HTTPS for a custom source before submitting", () => {
    const onSubmit = vi.fn()
    render(<SyncDialogHarness onSubmit={onSubmit} />)
    fireEvent.click(screen.getByRole("radio", { name: /sync\.sourceCustom/ }))
    const sourceInput = screen.getByLabelText("sync.customUrl")
    fireEvent.change(sourceInput, { target: { value: "http://example.test/nuclei-templates.git" } })
    fireEvent.click(screen.getByRole("button", { name: "sync.submit" }))
    expect(screen.getByText("sync.invalidUrl")).toBeInTheDocument()
    expect(onSubmit).toHaveBeenCalledTimes(1)
  })

  it("requires an HTTPS gitee.com URL for the Gitee source", () => {
    render(<SyncDialogHarness onSubmit={vi.fn()} />)
    fireEvent.click(screen.getByRole("radio", { name: /sync\.sourceGitee/ }))
    const sourceInput = screen.getByLabelText("sync.giteeUrl")
    fireEvent.change(sourceInput, { target: { value: "https://github.com/projectdiscovery/nuclei-templates.git" } })
    fireEvent.click(screen.getByRole("button", { name: "sync.submit" }))
    expect(screen.getByText("sync.invalidGiteeUrl")).toBeInTheDocument()
  })

  it("renders terminal task progress and leaves a new sync as an explicit action", () => {
    const onStartNew = vi.fn()
    render(<NucleiPocSyncDialog open onOpenChange={() => undefined} sourceKind="git" sourceUrl="https://github.com/projectdiscovery/nuclei-templates.git" onSourceKindChange={() => undefined} onSourceUrlChange={() => undefined} submitted={false} isSubmitting={false} task={completedTask} taskError={null} taskExpired={false} onSubmit={() => undefined} onStartNew={onStartNew} />)
    expect(screen.getByRole("status")).toHaveTextContent("sync.successSummary")
    expect(screen.getByRole("status")).toHaveTextContent("skipped=2")
    expect(screen.getByRole("status")).toHaveTextContent("commit=a1b2c3d4e5f6")
    expect(screen.getByRole("list", { name: "sync.phaseTimeline" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "sync.newSync" })).toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "sync.newSync" }))
    expect(onStartNew).toHaveBeenCalledTimes(1)
  })

  it("shows the active phase without manufacturing percentage or ETA", () => {
    render(<NucleiPocSyncDialog open onOpenChange={() => undefined} sourceKind="git" sourceUrl="https://github.com/projectdiscovery/nuclei-templates.git" onSourceKindChange={() => undefined} onSourceUrlChange={() => undefined} submitted={false} isSubmitting={false} task={runningTask} taskError={null} taskExpired={false} onSubmit={() => undefined} onStartNew={() => undefined} />)
    const status = screen.getByRole("status")
    expect(status).toHaveTextContent("sync.phases.SCANNING_FILES")
    expect(status).toHaveTextContent("sync.counters.filesSeen")
    expect(status).not.toHaveTextContent("%")
    expect(status).not.toHaveTextContent("ETA")
  })

  it("localizes failure codes, keeps the previous catalog active, and hides raw errors", () => {
    render(<NucleiPocSyncDialog open onOpenChange={() => undefined} sourceKind="git" sourceUrl="https://github.com/projectdiscovery/nuclei-templates.git" onSourceKindChange={() => undefined} onSourceUrlChange={() => undefined} submitted={false} isSubmitting={false} task={failedTask} taskError={null} taskExpired={false} onSubmit={() => undefined} onStartNew={() => undefined} />)
    const status = screen.getByRole("status")
    expect(status).toHaveTextContent("sync.failureCodes.TEMPLATE_INVALID")
    expect(status).toHaveTextContent("sync.failedSummary")
    expect(status).not.toHaveTextContent("One or more templates could not be validated")
  })

  it("uses localized recovery copy for a generic create error", () => {
    const rawError = new Error("Another Nuclei POC sync is already running.")
    render(<NucleiPocSyncDialog open onOpenChange={() => undefined} sourceKind="git" sourceUrl="https://github.com/projectdiscovery/nuclei-templates.git" onSourceKindChange={() => undefined} onSourceUrlChange={() => undefined} submitted={false} isSubmitting={false} task={null} taskError={rawError} errorKind="create" taskExpired={false} onSubmit={() => undefined} onStartNew={() => undefined} />)
    const status = screen.getByRole("status")
    expect(status).toHaveTextContent("sync.createError")
    expect(status).not.toHaveTextContent(rawError.message)
  })
})

describe("NucleiPocBatchActivationControls", () => {
  it("offers exactly enable/disable commands and submits only after confirmation", async () => {
    const onConfirm = vi.fn()
    render(<BatchActivationHarness onConfirm={onConfirm} />)

    fireEvent.click(screen.getByRole("button", { name: "actions.batch" }))
    expect(await screen.findByRole("menuitem", { name: "actions.enableAll" })).toBeInTheDocument()
    expect(screen.getByRole("menuitem", { name: "actions.disableAll" })).toBeInTheDocument()
    expect(screen.getAllByRole("menuitem")).toHaveLength(2)

    fireEvent.click(screen.getByRole("menuitem", { name: "actions.enableAll" }))
    expect(screen.getByRole("alertdialog")).toHaveTextContent("batch.enableDescription")
    expect(onConfirm).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole("button", { name: "batch.confirm" }))
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it("cancels without submitting and disables the menu while a mutation is pending", async () => {
    const onConfirm = vi.fn()
    const { unmount } = render(<BatchActivationHarness onConfirm={onConfirm} />)
    fireEvent.click(screen.getByRole("button", { name: "actions.batch" }))
    fireEvent.click(await screen.findByRole("menuitem", { name: "actions.disableAll" }))
    fireEvent.click(screen.getByRole("button", { name: "cancel" }))
    expect(onConfirm).not.toHaveBeenCalled()
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument()
    unmount()

    render(<BatchActivationHarness onConfirm={onConfirm} disabled />)
    expect(screen.getByRole("button", { name: "actions.batch" })).toBeDisabled()
  })
})
