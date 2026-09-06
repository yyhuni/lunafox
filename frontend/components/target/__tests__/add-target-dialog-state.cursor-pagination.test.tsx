import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useAddTargetDialogState } from "@/components/target/add-target-dialog-state"

const organizationHooks = vi.hoisted(() => ({
  useOrganizations: vi.fn(),
}))

const targetHooks = vi.hoisted(() => ({
  useBatchCreateTargets: vi.fn(),
}))

vi.mock("@/hooks/use-organizations", () => organizationHooks)
vi.mock("@/hooks/use-targets", () => targetHooks)

function createMutation() {
  return { mutate: vi.fn(), isPending: false }
}

function renderState() {
  return renderHook(() => useAddTargetDialogState({
    externalOpen: true,
    t: (key) => key,
  }))
}

describe("useAddTargetDialogState cursor pagination", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    targetHooks.useBatchCreateTargets.mockReturnValue(createMutation())
    organizationHooks.useOrganizations.mockImplementation((request) => ({
      data: request.pageToken
        ? { organizations: [], totalSize: 40 }
        : { organizations: [], totalSize: 40, nextPageToken: "organization-page-two" },
      isLoading: false,
      isPlaceholderData: false,
    }))
  })

  it("uses the active organization continuation, cached predecessor, and no total-derived terminal page", () => {
    const { result } = renderState()

    expect(result.current.organizationPaginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })

    act(() => {
      result.current.onNextOrgPage()
    })

    expect(organizationHooks.useOrganizations).toHaveBeenLastCalledWith(
      expect.objectContaining({
        pageToken: "organization-page-two",
        pageIndex: 2,
        pageSize: 10,
      }),
      { enabled: true },
    )
    expect(result.current.organizationPaginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: true,
      canPreviousPage: true,
      canNextPage: false,
    })

    const callsBeforeTerminalAttempt = organizationHooks.useOrganizations.mock.calls.length
    act(() => {
      result.current.onNextOrgPage()
    })
    expect(organizationHooks.useOrganizations).toHaveBeenCalledTimes(callsBeforeTerminalAttempt)

    act(() => {
      result.current.onPreviousOrgPage()
    })
    expect(organizationHooks.useOrganizations).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageToken: undefined, pageIndex: 1 }),
      { enabled: true },
    )
  })

  it("clears cursor state for search and page-size changes, and rejects placeholder continuations", () => {
    const { result, rerender } = renderState()

    act(() => {
      result.current.onNextOrgPage()
    })
    expect(result.current.organizationPaginationNavigation.canPreviousPage).toBe(true)

    act(() => {
      result.current.setOrgSearchQuery("acme")
    })
    expect(organizationHooks.useOrganizations).toHaveBeenLastCalledWith(
      expect.objectContaining({
        pageToken: undefined,
        pageIndex: 1,
        filter: 'displayName="acme"',
      }),
      { enabled: true },
    )
    expect(result.current.organizationPaginationNavigation.canPreviousPage).toBe(false)

    act(() => {
      result.current.onNextOrgPage()
    })
    act(() => {
      result.current.setOrgPageSize(20)
    })
    expect(organizationHooks.useOrganizations).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageToken: undefined, pageIndex: 1, pageSize: 20 }),
      { enabled: true },
    )

    organizationHooks.useOrganizations.mockReturnValue({
      data: { organizations: [], totalSize: 40, nextPageToken: "stale-page-two" },
      isLoading: false,
      isPlaceholderData: true,
    })
    rerender()

    expect(result.current.organizationPaginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: false,
    })
    const callsBeforePlaceholderAttempt = organizationHooks.useOrganizations.mock.calls.length
    act(() => {
      result.current.onNextOrgPage()
    })
    expect(organizationHooks.useOrganizations).toHaveBeenCalledTimes(callsBeforePlaceholderAttempt)
  })
})
