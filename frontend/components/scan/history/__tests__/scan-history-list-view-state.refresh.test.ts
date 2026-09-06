import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useScanHistoryListViewState } from "@/components/scan/history/scan-history-list-view-state"

const scanHooksMocks = vi.hoisted(() => ({
  refetch: vi.fn(),
  useScans: vi.fn(),
}))

const executedEngineDisplayMocks = vi.hoisted(() => ({
  useScanExecutedEngineDisplay: vi.fn(),
}))

vi.mock("next-intl", () => ({
  useLocale: () => "zh",
  useTranslations: () => (key: string) => key,
}))

vi.mock("@/hooks/use-scans", () => ({
  useScans: scanHooksMocks.useScans,
}))

vi.mock("@/hooks/use-scan-executed-engine-display", () => ({
  useScanExecutedEngineDisplay: executedEngineDisplayMocks.useScanExecutedEngineDisplay,
}))

vi.mock("@/components/scan/history/scan-history-columns", () => ({
  createScanHistoryColumns: vi.fn(() => []),
}))

vi.mock("@/components/scan/history/scan-history-list-state", () => ({
  useScanHistoryActions: ({ selectedScans }: { selectedScans: unknown[] }) => ({
    selectedScans,
    deleteDialogOpen: false,
    setDeleteDialogOpen: vi.fn(),
    scanToDelete: null,
    bulkDeleteDialogOpen: false,
    setBulkDeleteDialogOpen: vi.fn(),
    stopDialogOpen: false,
    setStopDialogOpen: vi.fn(),
    scanToStop: null,
    progressDialogOpen: false,
    setProgressDialogOpen: vi.fn(),
    progressData: null,
    runtimeDetailOpen: false,
    setRuntimeDetailOpen: vi.fn(),
    scanForRuntimeDetail: null,
    handleDeleteScan: vi.fn(),
    confirmDelete: vi.fn(),
    handleBulkDelete: vi.fn(),
    confirmBulkDelete: vi.fn(),
    handleStopScan: vi.fn(),
    confirmStop: vi.fn(),
    handleViewProgress: vi.fn(),
    handleViewRuntimeDetail: vi.fn(),
  }),
}))

describe("useScanHistoryListViewState query state", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    scanHooksMocks.refetch.mockResolvedValue({ data: undefined })
    executedEngineDisplayMocks.useScanExecutedEngineDisplay.mockReturnValue({
      engineNamesByScanId: new Map(),
      engineDescriptionsByScanId: new Map(),
      engineNamesById: new Map(),
      engineDescriptionsById: new Map(),
      isLoading: false,
      error: null,
    })
    scanHooksMocks.useScans.mockReturnValue({
      data: {
        results: [],
        total: 0,
        page: 1,
        pageSize: 10,
        totalPages: 0,
      },
      isLoading: false,
      isFetching: true,
      error: null,
      refetch: scanHooksMocks.refetch,
    })
  })

  it("exposes query fetching as search pending state", () => {
    const { result } = renderHook(() => useScanHistoryListViewState({}))

    expect(result.current.isSearching).toBe(true)
    expect(scanHooksMocks.useScans).toHaveBeenCalledWith({
      pageSize: 10,
      pageToken: undefined,
      filter: undefined,
      orderBy: "createdAt desc",
      target: undefined,
    })
  })

  it("compiles target-name search and multi-status filters into the canonical filter", () => {
    const { result } = renderHook(() => useScanHistoryListViewState({ targetId: 7 }))

    act(() => {
      result.current.commitSearch(" acme ")
    })

    expect(scanHooksMocks.useScans).toHaveBeenLastCalledWith({
      pageSize: 10,
      pageToken: undefined,
      filter: 'targetName="acme"',
      orderBy: "createdAt desc",
      target: 7,
    })

    act(() => {
      result.current.handleStatusFilterChange(["running", "failed"])
    })

    expect(scanHooksMocks.useScans).toHaveBeenLastCalledWith({
      pageSize: 10,
      pageToken: undefined,
      filter: '(status=="running" || status=="failed") && targetName="acme"',
      orderBy: "createdAt desc",
      target: 7,
    })
  })

  it("preserves controls while paging and resets paging when sorting changes", () => {
    scanHooksMocks.useScans.mockImplementation((params) => ({
      data: {
        results: [],
        totalSize: 25,
        nextPageToken: params.filter ? "status-page-two" : "page-two",
      },
      isLoading: false,
      isFetching: false,
      error: null,
      refetch: scanHooksMocks.refetch,
    }))

    const { result } = renderHook(() => useScanHistoryListViewState({ pageSize: 10 }))

    act(() => {
      result.current.handleStatusFilterChange(["running"])
    })
    act(() => {
      result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })

    expect(scanHooksMocks.useScans).toHaveBeenLastCalledWith({
      pageSize: 10,
      pageToken: "status-page-two",
      filter: 'status=="running"',
      orderBy: "createdAt desc",
      target: undefined,
    })

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 0, pageSize: 10 })
    })

    expect(scanHooksMocks.useScans).toHaveBeenLastCalledWith({
      pageSize: 10,
      pageToken: undefined,
      filter: 'status=="running"',
      orderBy: "createdAt desc",
      target: undefined,
    })
    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })

    act(() => {
      result.current.handleSortingChange([{ id: "createdAt", desc: false }])
    })

    expect(scanHooksMocks.useScans).toHaveBeenLastCalledWith({
      pageSize: 10,
      pageToken: undefined,
      filter: 'status=="running"',
      orderBy: "createdAt",
      target: undefined,
    })
  })

  it("uses active continuations for next, cached tokens for previous, and never lets placeholder data advance", () => {
    scanHooksMocks.useScans.mockImplementation((params) => ({
      data: params.pageToken
        ? { results: [], totalSize: 40 }
        : { results: [], totalSize: 40, nextPageToken: "scan-page-two" },
      isLoading: false,
      isFetching: false,
      isPlaceholderData: false,
      error: null,
      refetch: scanHooksMocks.refetch,
    }))

    const { result, rerender } = renderHook(() => useScanHistoryListViewState({ pageSize: 10 }))

    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })

    expect(scanHooksMocks.useScans).toHaveBeenLastCalledWith(expect.objectContaining({
      pageToken: "scan-page-two",
      pageSize: 10,
    }))
    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: true,
      canPreviousPage: true,
      canNextPage: false,
    })

    const callsBeforeTerminalAttempt = scanHooksMocks.useScans.mock.calls.length
    act(() => {
      result.current.handlePaginationChange({ pageIndex: 2, pageSize: 10 })
    })
    expect(scanHooksMocks.useScans).toHaveBeenCalledTimes(callsBeforeTerminalAttempt)

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 0, pageSize: 10 })
    })
    expect(scanHooksMocks.useScans).toHaveBeenLastCalledWith(expect.objectContaining({ pageToken: undefined }))

    scanHooksMocks.useScans.mockReturnValue({
      data: { results: [], totalSize: 40, nextPageToken: "stale-scan-page-two" },
      isLoading: false,
      isFetching: true,
      isPlaceholderData: true,
      error: null,
      refetch: scanHooksMocks.refetch,
    })

    rerender()

    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: false,
    })

    const callsBeforePlaceholderAttempt = scanHooksMocks.useScans.mock.calls.length
    act(() => {
      result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })
    expect(scanHooksMocks.useScans).toHaveBeenCalledTimes(callsBeforePlaceholderAttempt)
  })
})
