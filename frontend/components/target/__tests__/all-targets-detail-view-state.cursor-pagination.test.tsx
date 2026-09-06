import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useAllTargetsDetailViewState } from "@/components/target/all-targets-detail-view-state"

const targetHooks = vi.hoisted(() => ({
  useTargets: vi.fn(),
  useDeleteTarget: vi.fn(),
  useBatchDeleteTargets: vi.fn(),
}))

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock("@/hooks/use-targets", () => targetHooks)

vi.mock("@/components/target/all-targets-columns", () => ({
  createAllTargetsColumns: vi.fn(() => []),
}))

vi.mock("@/components/route-progress", () => ({
  pushWithRouteProgress: vi.fn(),
}))

function createMutation() {
  return { mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }
}

describe("useAllTargetsDetailViewState cursor pagination", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    targetHooks.useDeleteTarget.mockReturnValue(createMutation())
    targetHooks.useBatchDeleteTargets.mockReturnValue(createMutation())
    targetHooks.useTargets.mockImplementation((request) => ({
      data: request.pageToken
        ? { results: [], totalSize: 99 }
        : { results: [], totalSize: 99, nextPageToken: "targets-page-two" },
      isLoading: false,
      isFetching: false,
      isPlaceholderData: false,
      error: null,
      refetch: vi.fn(),
    }))
  })

  it("uses only the returned continuation and cached predecessor for the global targets table", () => {
    const { result } = renderHook(() => useAllTargetsDetailViewState({}))

    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })

    expect(targetHooks.useTargets).toHaveBeenLastCalledWith(expect.objectContaining({
      pageToken: "targets-page-two",
      pageSize: 10,
    }))
    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: true,
      canPreviousPage: true,
      canNextPage: false,
    })

    const callsBeforeTerminalAttempt = targetHooks.useTargets.mock.calls.length
    act(() => {
      result.current.handlePaginationChange({ pageIndex: 2, pageSize: 10 })
    })
    expect(targetHooks.useTargets).toHaveBeenCalledTimes(callsBeforeTerminalAttempt)

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 0, pageSize: 10 })
    })
    expect(targetHooks.useTargets).toHaveBeenLastCalledWith(expect.objectContaining({ pageToken: undefined }))
  })

  it("does not expose next while the global targets response is placeholder data", () => {
    targetHooks.useTargets.mockReturnValue({
      data: { results: [], totalSize: 99, nextPageToken: "stale-targets-page-two" },
      isLoading: false,
      isFetching: true,
      isPlaceholderData: true,
      error: null,
      refetch: vi.fn(),
    })

    const { result } = renderHook(() => useAllTargetsDetailViewState({}))

    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: false,
    })
  })
})
