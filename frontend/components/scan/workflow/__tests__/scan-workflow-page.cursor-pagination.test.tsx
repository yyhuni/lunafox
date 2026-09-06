import * as React from "react"
import { act, render } from "@testing-library/react"
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
  createWorkflowManagementColumns: vi.fn(() => []),
}))

vi.mock("@/components/scan/workflow/workflow-composition-canvas", () => ({
  WorkflowCompositionCanvas: () => null,
}))

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
})
