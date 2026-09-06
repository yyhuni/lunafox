import { act, renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { useScreenshotsGalleryState } from "@/components/screenshots/screenshots-gallery-state"

const screenshotHooks = vi.hoisted(() => ({
  useBulkDeleteScreenshots: vi.fn(),
  useScanScreenshotFilterOptions: vi.fn(),
  useScreenshotImageUrlResolver: vi.fn(),
  useScanScreenshots: vi.fn(),
  useTargetScreenshotFilterOptions: vi.fn(),
  useTargetScreenshots: vi.fn(),
}))

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

vi.mock("@/hooks/use-screenshots", () => screenshotHooks)

function terminalResponse() {
  return {
    data: { results: [], totalSize: 36 },
    isLoading: false,
    isFetching: false,
    isPlaceholderData: false,
    error: null,
    refetch: vi.fn(),
  }
}

function continuationResponse(nextPageToken: string, isPlaceholderData = false) {
  return {
    data: { results: [], totalSize: 36, nextPageToken },
    isLoading: false,
    isFetching: isPlaceholderData,
    isPlaceholderData,
    error: null,
    refetch: vi.fn(),
  }
}

function installTargetCursorResponse() {
  screenshotHooks.useTargetScreenshots.mockImplementation((_targetId, params: { pageToken?: string }) => (
    params.pageToken
      ? terminalResponse()
      : continuationResponse("target-screenshots-page-two")
  ))
  screenshotHooks.useScanScreenshots.mockReturnValue(terminalResponse())
}

function installScanCursorResponse() {
  screenshotHooks.useTargetScreenshots.mockReturnValue(terminalResponse())
  screenshotHooks.useScanScreenshots.mockImplementation((_scanId, params: { pageToken?: string }) => (
    params.pageToken
      ? terminalResponse()
      : continuationResponse("scan-screenshots-page-two")
  ))
}

describe("useScreenshotsGalleryState cursor pagination", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    screenshotHooks.useBulkDeleteScreenshots.mockReturnValue({ mutateAsync: vi.fn() })
    screenshotHooks.useScreenshotImageUrlResolver.mockReturnValue((id: number) => `/screenshots/${id}`)
    screenshotHooks.useTargetScreenshotFilterOptions.mockReturnValue({ data: { results: [] } })
    screenshotHooks.useScanScreenshotFilterOptions.mockReturnValue({ data: { results: [] } })
  })

  it("uses the target response continuation, a cached predecessor, and no derived terminal page", () => {
    installTargetCursorResponse()

    const { result } = renderHook(() => useScreenshotsGalleryState({ targetId: 7 }))

    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 1, pageSize: 12 })
    })

    expect(screenshotHooks.useTargetScreenshots).toHaveBeenLastCalledWith(
      7,
      expect.objectContaining({ pageToken: "target-screenshots-page-two", pageSize: 12 }),
      { enabled: true },
    )
    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: true,
      canPreviousPage: true,
      canNextPage: false,
    })

    const callsBeforeTerminalAttempt = screenshotHooks.useTargetScreenshots.mock.calls.length
    act(() => {
      result.current.handlePaginationChange({ pageIndex: 2, pageSize: 12 })
    })
    expect(screenshotHooks.useTargetScreenshots).toHaveBeenCalledTimes(callsBeforeTerminalAttempt)

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 0, pageSize: 12 })
    })
    expect(screenshotHooks.useTargetScreenshots).toHaveBeenLastCalledWith(
      7,
      expect.objectContaining({ pageToken: undefined, pageSize: 12 }),
      { enabled: true },
    )
  })

  it("applies the same adjacent-token behavior to scan screenshots", () => {
    installScanCursorResponse()

    const { result } = renderHook(() => useScreenshotsGalleryState({ scanId: 11 }))

    act(() => {
      result.current.handlePaginationChange({ pageIndex: 1, pageSize: 12 })
    })

    expect(screenshotHooks.useScanScreenshots).toHaveBeenLastCalledWith(
      11,
      expect.objectContaining({ pageToken: "scan-screenshots-page-two", pageSize: 12 }),
      { enabled: true },
    )
    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: true,
      canPreviousPage: true,
      canNextPage: false,
    })
  })

  it("does not let placeholder screenshot data authorize a continuation", () => {
    screenshotHooks.useTargetScreenshots.mockReturnValue(
      continuationResponse("stale-target-screenshots-page-two", true),
    )
    screenshotHooks.useScanScreenshots.mockReturnValue(terminalResponse())

    const { result } = renderHook(() => useScreenshotsGalleryState({ targetId: 7 }))

    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: false,
    })

    const callsBeforeAttempt = screenshotHooks.useTargetScreenshots.mock.calls.length
    act(() => {
      result.current.handlePaginationChange({ pageIndex: 1, pageSize: 12 })
    })
    expect(screenshotHooks.useTargetScreenshots).toHaveBeenCalledTimes(callsBeforeAttempt)
  })
})
