import * as React from "react"
import { act, renderHook } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { createBusinessListQuery } from "@/components/shared/data-table/business-list-query"
import type { FingerPrintHubFingerprint } from "@/types/fingerprint.types"

import { FINGERPRINT_LIBRARY_LIST_CONFIG } from "../fingerprint-library-list-config"
import { useFingerprintLibraryQueryState } from "../fingerprint-library-query-state"

const pageOne = {
  results: [] as FingerPrintHubFingerprint[],
  totalSize: 20,
  nextPageToken: "page-two-token",
}

describe("useFingerprintLibraryQueryState", () => {
  it("只允许服务端已返回 token 支持的下一页，并将其用于后续请求", () => {
    const config = FINGERPRINT_LIBRARY_LIST_CONFIG.fingerprinthub

    const { result } = renderHook(() => {
      const [query, setQuery] = React.useState(() => createBusinessListQuery({
        pageSize: 10,
        sorting: config.defaultSorting,
      }))

      return useFingerprintLibraryQueryState({
        query,
        setQuery,
        data: pageOne,
        filterConfig: config.filterConfig,
        sortableFields: config.sortableFields,
        defaultSorting: config.defaultSorting,
      })
    })

    expect(result.current.cursorPaginationSummary).toEqual({ total: 20 })
    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: true,
    })

    act(() => {
      result.current.onPaginationChange({ pageIndex: 1, pageSize: 10 })
    })

    expect(result.current.request).toEqual({
      pageSize: 10,
      pageToken: "page-two-token",
      orderBy: "createdAt desc",
    })
    expect(result.current.paginationNavigation.canPreviousPage).toBe(true)
  })

  it("不会让 placeholder 响应授权下一页或写入 continuation token", () => {
    const config = FINGERPRINT_LIBRARY_LIST_CONFIG.fingerprinthub

    const { result } = renderHook(() => {
      const [query, setQuery] = React.useState(() => createBusinessListQuery({
        pageSize: 10,
        sorting: config.defaultSorting,
      }))

      return useFingerprintLibraryQueryState({
        query,
        setQuery,
        data: pageOne,
        isPlaceholderData: true,
        filterConfig: config.filterConfig,
        sortableFields: config.sortableFields,
        defaultSorting: config.defaultSorting,
      })
    })

    expect(result.current.paginationNavigation).toEqual({
      mode: "cursor",
      canFirstPage: false,
      canPreviousPage: false,
      canNextPage: false,
    })

    act(() => {
      result.current.onPaginationChange({ pageIndex: 1, pageSize: 10 })
    })

    expect(result.current.request).not.toHaveProperty("pageToken")
    expect(result.current.pageTokens).toEqual({ 1: undefined })
  })
})
