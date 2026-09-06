import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useOrganizationListState } from "@/components/organization/organization-list-state"

const organizationHooks = vi.hoisted(() => ({
  useOrganizations: vi.fn(),
  useDeleteOrganization: vi.fn(),
  useBatchDeleteOrganizations: vi.fn(),
}))

vi.mock("next-intl", () => ({
  useLocale: () => "zh",
  useTranslations: () => (key: string) => key,
}))

vi.mock("@/hooks/use-organizations", () => organizationHooks)

vi.mock("@/components/organization/organization-columns", () => ({
  createOrganizationColumns: vi.fn(() => []),
}))

function createMutation() {
  return { mutate: vi.fn(), mutateAsync: vi.fn() }
}

describe("useOrganizationListState cursor pagination", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    organizationHooks.useDeleteOrganization.mockReturnValue(createMutation())
    organizationHooks.useBatchDeleteOrganizations.mockReturnValue(createMutation())
    organizationHooks.useOrganizations.mockImplementation((request) => ({
      data: request.pageToken
        ? { organizations: [], totalSize: 40 }
        : { organizations: [], totalSize: 40, nextPageToken: "organizations-page-two" },
      isLoading: false,
      isFetching: false,
      isPlaceholderData: false,
      error: null,
      refetch: vi.fn(),
    }))
  })

  it("uses the active continuation for next, cached tokens for previous, and never derives a terminal next page", () => {
    const { result } = renderHook(() => useOrganizationListState())

    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })

    expect(organizationHooks.useOrganizations).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageToken: "organizations-page-two", pageIndex: 2 }),
      { enabled: true },
    )
    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: true,
      canPreviousPage: true,
      canNextPage: false,
    })

    const callsBeforeTerminalAttempt = organizationHooks.useOrganizations.mock.calls.length
    act(() => {
      result.current.handlePaginationChange({ pageIndex: 2, pageSize: 10 })
    })

    expect(organizationHooks.useOrganizations).toHaveBeenCalledTimes(callsBeforeTerminalAttempt)

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 0, pageSize: 10 })
    })

    expect(organizationHooks.useOrganizations).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageToken: undefined, pageIndex: 1 }),
      { enabled: true },
    )
  })

  it("does not let placeholder data authorize a continuation and clears the old cache when search changes", () => {
    organizationHooks.useOrganizations.mockReturnValue({
      data: { organizations: [], totalSize: 40, nextPageToken: "stale-page-two" },
      isLoading: false,
      isFetching: true,
      isPlaceholderData: true,
      error: null,
      refetch: vi.fn(),
    })

    const { result, rerender } = renderHook(() => useOrganizationListState())

    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: false,
    })

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })

    expect(organizationHooks.useOrganizations).toHaveBeenCalledTimes(1)

    organizationHooks.useOrganizations.mockImplementation((request) => ({
      data: request.pageToken
        ? { organizations: [], totalSize: 40 }
        : { organizations: [], totalSize: 40, nextPageToken: "fresh-page-two" },
      isLoading: false,
      isFetching: false,
      isPlaceholderData: false,
      error: null,
      refetch: vi.fn(),
    }))
    rerender()

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 1, pageSize: 10 })
    })
    expect(result.current.paginationNavigation.canPreviousPage).toBe(true)

    act(() => {
      result.current.commitSearch("acme")
    })

    expect(organizationHooks.useOrganizations).toHaveBeenLastCalledWith(
      expect.objectContaining({
        pageToken: undefined,
        pageIndex: 1,
        filter: 'displayName="acme"',
      }),
      { enabled: true },
    )
    expect(result.current.paginationNavigation.canPreviousPage).toBe(false)
  })
})
