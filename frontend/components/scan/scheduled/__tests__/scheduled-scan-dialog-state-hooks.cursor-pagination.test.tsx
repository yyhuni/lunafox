import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useScheduledScanSearch } from "@/components/scan/scheduled/scheduled-scan-dialog-state-hooks"

const organizationHooks = vi.hoisted(() => ({
  useOrganizations: vi.fn(),
}))

const targetHooks = vi.hoisted(() => ({
  useTargets: vi.fn(),
}))

vi.mock("@/hooks/use-organizations", () => organizationHooks)
vi.mock("@/hooks/use-targets", () => targetHooks)

function installContinuationResponses() {
  organizationHooks.useOrganizations.mockImplementation((request) => ({
    data: request.pageToken
      ? { organizations: [], totalSize: 32 }
      : { organizations: [], totalSize: 32, nextPageToken: "organization-page-two" },
    isFetching: false,
    isPlaceholderData: false,
  }))
  targetHooks.useTargets.mockImplementation((request) => ({
    data: request.pageToken
      ? { targets: [], totalSize: 48 }
      : { targets: [], totalSize: 48, nextPageToken: "target-page-two" },
    isFetching: false,
    isPlaceholderData: false,
  }))
}

describe("useScheduledScanSearch cursor pagination", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    installContinuationResponses()
  })

  it("uses active continuations and cached predecessors independently for organization and target pickers", () => {
    const { result } = renderHook(() => useScheduledScanSearch({ open: true }))

    expect(result.current.organizationPaginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })
    expect(result.current.targetPaginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })

    act(() => {
      result.current.onNextOrgPage()
      result.current.onNextTargetPage()
    })

    expect(organizationHooks.useOrganizations).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageToken: "organization-page-two", pageIndex: 2, pageSize: 8 }),
      { enabled: true },
    )
    expect(targetHooks.useTargets).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageToken: "target-page-two", pageSize: 10 }),
      { enabled: true },
    )
    expect(result.current.organizationPaginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: true,
      canPreviousPage: true,
      canNextPage: false,
    })
    expect(result.current.targetPaginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: true,
      canPreviousPage: true,
      canNextPage: false,
    })

    const organizationCallsBeforeTerminalAttempt = organizationHooks.useOrganizations.mock.calls.length
    const targetCallsBeforeTerminalAttempt = targetHooks.useTargets.mock.calls.length
    act(() => {
      result.current.onNextOrgPage()
      result.current.onNextTargetPage()
    })
    expect(organizationHooks.useOrganizations).toHaveBeenCalledTimes(organizationCallsBeforeTerminalAttempt)
    expect(targetHooks.useTargets).toHaveBeenCalledTimes(targetCallsBeforeTerminalAttempt)

    act(() => {
      result.current.onPreviousOrgPage()
      result.current.onPreviousTargetPage()
    })
    expect(organizationHooks.useOrganizations).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageToken: undefined, pageIndex: 1 }),
      { enabled: true },
    )
    expect(targetHooks.useTargets).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageToken: undefined }),
      { enabled: true },
    )
  })

  it("clears each picker cache on query-shape changes and ignores placeholder responses", () => {
    const { result, rerender } = renderHook(() => useScheduledScanSearch({ open: true }))

    act(() => {
      result.current.onNextOrgPage()
      result.current.onNextTargetPage()
      result.current.setOrgSearchInput("acme")
      result.current.setTargetPageSize(20)
    })

    expect(organizationHooks.useOrganizations).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageToken: undefined, pageIndex: 1, filter: 'displayName="acme"' }),
      { enabled: true },
    )
    expect(targetHooks.useTargets).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageToken: undefined, pageSize: 20 }),
      { enabled: true },
    )
    expect(result.current.organizationPaginationNavigation.canPreviousPage).toBe(false)
    expect(result.current.targetPaginationNavigation.canPreviousPage).toBe(false)

    organizationHooks.useOrganizations.mockReturnValue({
      data: { organizations: [], totalSize: 32, nextPageToken: "stale-organization-page-two" },
      isFetching: true,
      isPlaceholderData: true,
    })
    targetHooks.useTargets.mockReturnValue({
      data: { targets: [], totalSize: 48, nextPageToken: "stale-target-page-two" },
      isFetching: true,
      isPlaceholderData: true,
    })
    rerender()

    expect(result.current.organizationPaginationNavigation.canNextPage).toBe(false)
    expect(result.current.targetPaginationNavigation.canNextPage).toBe(false)
    const organizationCallsBeforePlaceholderAttempt = organizationHooks.useOrganizations.mock.calls.length
    const targetCallsBeforePlaceholderAttempt = targetHooks.useTargets.mock.calls.length
    act(() => {
      result.current.onNextOrgPage()
      result.current.onNextTargetPage()
    })
    expect(organizationHooks.useOrganizations).toHaveBeenCalledTimes(organizationCallsBeforePlaceholderAttempt)
    expect(targetHooks.useTargets).toHaveBeenCalledTimes(targetCallsBeforePlaceholderAttempt)
  })
})
