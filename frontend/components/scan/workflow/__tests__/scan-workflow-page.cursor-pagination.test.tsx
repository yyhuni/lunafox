import * as React from "react"
import { act, render, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import ScanWorkflowPage from "@/components/scan/workflow/scan-workflow-page"

const workflowHooks = vi.hoisted(() => ({
  useCreateScanWorkflow: vi.fn(),
  useScanWorkflowList: vi.fn(),
  useUpdateScanWorkflow: vi.fn(),
}))

const engineCatalogHooks = vi.hoisted(() => ({
  useEngineCatalog: vi.fn(),
}))

const tableMocks = vi.hoisted(() => ({
  latestProps: undefined as undefined | {
    state?: {
      pagination?: { pageIndex: number; pageSize: number }
      paginationNavigation?: {
        mode: "cursor"
        canPreviousPage: boolean
        canNextPage: boolean
      }
      onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
    }
  },
}))

const columnMocks = vi.hoisted(() => ({
  latestOptions: undefined as undefined | {
    onEdit?: (row: Record<string, unknown>) => void
  },
}))

const canvasMocks = vi.hoisted(() => ({
  latestProps: undefined as undefined | {
    onSave?: (stages: Array<{ stageId: string; steps: Array<{ stageId: string; stepId: string; engineId: string; profileDefaultEnabled: boolean }> }>) => Promise<void> | void
  },
}))

const sonnerMocks = vi.hoisted(() => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    loading: vi.fn(),
    dismiss: vi.fn(),
    custom: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: sonnerMocks.toast,
  Toaster: () => null,
}))

vi.mock("next-intl", () => ({
  useLocale: () => "zh",
  useTranslations: () => (key: string) => key,
}))

vi.mock("@/hooks/use-scan-workflows", () => workflowHooks)
vi.mock("@/hooks/use-engine-catalog", () => engineCatalogHooks)

vi.mock("@/components/common/page-header", () => ({
  PageHeader: () => null,
}))

vi.mock("@/components/shared/data-table", () => ({
  BusinessListDataTable: (props: typeof tableMocks.latestProps) => {
    tableMocks.latestProps = props
    return null
  },
}))

vi.mock("@/components/shared/loading/content-handoff", () => ({
  ContentHandoff: ({ children }: { children: React.ReactNode }) => children,
}))

vi.mock("@/components/scan/workflow/scan-workflow-management-columns", () => ({
  createWorkflowManagementColumns: (options: { onEdit?: (row: Record<string, unknown>) => void }) => {
    columnMocks.latestOptions = options
    return []
  },
}))

vi.mock("@/components/scan/workflow/workflow-composition-canvas", async () => {
  const React = await import("react")
  return {
    WorkflowCompositionCanvas: (props: typeof canvasMocks.latestProps) => {
      canvasMocks.latestProps = props
      return React.createElement("div", { "data-testid": "workflow-builder" })
    },
  }
})

function getPaginationState() {
  const state = tableMocks.latestProps?.state
  if (!state?.pagination || !state.paginationNavigation || !state.onPaginationChange) {
    throw new Error("ScanWorkflowPage did not supply controlled cursor pagination state.")
  }
  return state as Required<typeof state>
}

describe("ScanWorkflowPage cursor pagination", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    tableMocks.latestProps = undefined
    columnMocks.latestOptions = undefined
    canvasMocks.latestProps = undefined
    workflowHooks.useCreateScanWorkflow.mockReturnValue({ mutateAsync: vi.fn(), isPending: false })
    workflowHooks.useUpdateScanWorkflow.mockReturnValue({ mutateAsync: vi.fn(), isPending: false })
    engineCatalogHooks.useEngineCatalog.mockReturnValue({ data: [], isError: false })
  })

  it("uses the active continuation, cached predecessor, and never a total-derived terminal page", () => {
    let placeholder = false
    workflowHooks.useScanWorkflowList.mockImplementation((request: { pageToken?: string }) => ({
      data: request.pageToken
        ? { scanWorkflows: [], totalSize: 60 }
        : { scanWorkflows: [], totalSize: 60, nextPageToken: "workflow-page-two" },
      isLoading: false,
      isPlaceholderData: placeholder,
    }))

    const view = render(<ScanWorkflowPage />)

    expect(workflowHooks.useScanWorkflowList).toHaveBeenLastCalledWith({
      pageSize: 20,
      pageToken: undefined,
      filter: undefined,
    })
    expect(getPaginationState().paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })

    act(() => {
      getPaginationState().onPaginationChange({ pageIndex: 1, pageSize: 20 })
    })

    expect(workflowHooks.useScanWorkflowList).toHaveBeenLastCalledWith({
      pageSize: 20,
      pageToken: "workflow-page-two",
      filter: undefined,
    })
    expect(getPaginationState().paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: true,
      canPreviousPage: true,
      canNextPage: false,
    })

    const callsBeforeTerminalAttempt = workflowHooks.useScanWorkflowList.mock.calls.length
    act(() => {
      getPaginationState().onPaginationChange({ pageIndex: 2, pageSize: 20 })
    })
    expect(workflowHooks.useScanWorkflowList).toHaveBeenCalledTimes(callsBeforeTerminalAttempt)

    act(() => {
      getPaginationState().onPaginationChange({ pageIndex: 0, pageSize: 20 })
    })
    expect(workflowHooks.useScanWorkflowList).toHaveBeenLastCalledWith({
      pageSize: 20,
      pageToken: undefined,
      filter: undefined,
    })

    placeholder = true
    view.rerender(<ScanWorkflowPage />)

    expect(getPaginationState().paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: false,
    })
    const callsBeforePlaceholderAttempt = workflowHooks.useScanWorkflowList.mock.calls.length
    act(() => {
      getPaginationState().onPaginationChange({ pageIndex: 1, pageSize: 20 })
    })
    expect(workflowHooks.useScanWorkflowList).toHaveBeenCalledTimes(callsBeforePlaceholderAttempt)
  })

  it("clears the cursor cache and returns to the first response when page size changes", () => {
    workflowHooks.useScanWorkflowList.mockImplementation((request: { pageToken?: string }) => ({
      data: request.pageToken
        ? { scanWorkflows: [], totalSize: 60 }
        : { scanWorkflows: [], totalSize: 60, nextPageToken: "workflow-page-two" },
      isLoading: false,
      isPlaceholderData: false,
    }))

    render(<ScanWorkflowPage />)

    act(() => {
      getPaginationState().onPaginationChange({ pageIndex: 1, pageSize: 20 })
    })
    expect(workflowHooks.useScanWorkflowList).toHaveBeenLastCalledWith(expect.objectContaining({
      pageToken: "workflow-page-two",
      pageSize: 20,
    }))

    act(() => {
      getPaginationState().onPaginationChange({ pageIndex: 0, pageSize: 50 })
    })
    expect(workflowHooks.useScanWorkflowList).toHaveBeenLastCalledWith({
      pageSize: 50,
      pageToken: undefined,
      filter: undefined,
    })
    expect(getPaginationState().pagination).toEqual({ pageIndex: 0, pageSize: 50 })
  })

  const workflowName = "scanWorkflows/wf-11111111-1111-4111-8111-111111111111"
  const workflowRow = {
    key: workflowName,
    name: workflowName,
    title: "Discovery",
    description: "",
    engineCount: 1,
    isBuiltin: false,
    isExecutable: true,
    item: {
      name: workflowName,
      displayName: "Discovery",
      description: "",
      stages: [{
        stageId: "stage-1",
        steps: [{ stageId: "stage-1", stepId: "step-local-100", engineId: "engine.example.custom", profileDefaultEnabled: true }],
      }],
      steps: [],
      isBuiltin: false,
      isExecutable: true,
      etag: "v1.etag",
      createTime: "",
      updateTime: "",
    },
  }
  const submittedStages = [{
    stageId: "stage-1",
    steps: [{ stageId: "stage-1", stepId: "step-local-100", engineId: "engine.example.custom", profileDefaultEnabled: true }],
  }]

  it("keeps the builder and its unsaved draft mounted when a save is rejected", async () => {
    const mutateAsync = vi.fn().mockRejectedValue(new Error("Invalid request body"))
    workflowHooks.useUpdateScanWorkflow.mockReturnValue({ mutateAsync, isPending: false })
    workflowHooks.useScanWorkflowList.mockReturnValue({
      data: { scanWorkflows: [], totalSize: 0 },
      isLoading: false,
      isPlaceholderData: false,
    })

    render(<ScanWorkflowPage />)
    act(() => columnMocks.latestOptions?.onEdit?.(workflowRow))
    expect(screen.getByTestId("workflow-builder")).toBeInTheDocument()

    await act(async () => {
      await canvasMocks.latestProps?.onSave?.(submittedStages)
    })

    expect(mutateAsync).toHaveBeenCalledTimes(1)
    expect(sonnerMocks.toast.success).not.toHaveBeenCalled()
    expect(sonnerMocks.toast.error).toHaveBeenCalledWith("workflow.update.error")
    expect(screen.getByTestId("workflow-builder")).toBeInTheDocument()
  })

  it("returns to management and reports success after a saved update", async () => {
    const mutateAsync = vi.fn().mockResolvedValue(undefined)
    workflowHooks.useUpdateScanWorkflow.mockReturnValue({ mutateAsync, isPending: false })
    workflowHooks.useScanWorkflowList.mockReturnValue({
      data: { scanWorkflows: [], totalSize: 0 },
      isLoading: false,
      isPlaceholderData: false,
    })

    render(<ScanWorkflowPage />)
    act(() => columnMocks.latestOptions?.onEdit?.(workflowRow))

    await act(async () => {
      await canvasMocks.latestProps?.onSave?.(submittedStages)
    })

    expect(mutateAsync).toHaveBeenCalledTimes(1)
    expect(sonnerMocks.toast.success).toHaveBeenCalledWith("workflow.update.success")
    expect(sonnerMocks.toast.error).not.toHaveBeenCalled()
    expect(screen.queryByTestId("workflow-builder")).not.toBeInTheDocument()
  })
})
