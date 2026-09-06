"use client"

import * as React from "react"
import type { SortingState } from "@tanstack/react-table"

import {
  applyBusinessListControlChange,
  compileBusinessListFilter,
  compileBusinessListOrderBy,
  getCurrentCursorNextPageToken,
  getCursorPaginationNavigation,
  getCursorPageTransition,
  setBusinessListPage,
  type BusinessListFilterCompilerConfig,
  type BusinessListQuery,
  type BusinessListSortableFieldConfig,
  type BusinessListSorting,
} from "@/components/shared/data-table/business-list-query"
import type { FingerprintListResponse, FingerprintResource } from "@/types/fingerprint.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  PaginationState,
} from "@/types/data-table.types"

type QueryUpdater = (current: BusinessListQuery) => BusinessListQuery

type UseFingerprintLibraryQueryStateOptions<T extends FingerprintResource> = {
  query: BusinessListQuery
  setQuery: React.Dispatch<React.SetStateAction<BusinessListQuery>>
  data: FingerprintListResponse<T> | undefined
  isPlaceholderData?: boolean
  filterConfig: BusinessListFilterCompilerConfig
  sortableFields: Record<string, BusinessListSortableFieldConfig>
  defaultSorting: BusinessListSorting
}

function filterSignature(filters: BusinessListQuery["filters"]) {
  return Object.entries(filters)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([field, values]) => `${field}:${[...values].sort().join(",")}`)
    .join("|")
}

export function buildFingerprintLibraryListParams(
  query: BusinessListQuery,
  filterConfig: BusinessListFilterCompilerConfig,
  sortableFields: Record<string, BusinessListSortableFieldConfig>
) {
  const filter = compileBusinessListFilter({ search: query.search, filters: query.filters }, filterConfig)
  const orderBy = compileBusinessListOrderBy(query.sorting, sortableFields)

  return {
    pageSize: query.pageSize,
    ...(query.pageToken ? { pageToken: query.pageToken } : {}),
    ...(filter ? { filter } : {}),
    ...(orderBy ? { orderBy } : {}),
  }
}

/**
 * Keeps an AIP page-token list's controls in one state transition boundary.
 * Search, facets, sorting, and page-size changes always discard previous-page
 * tokens, because those tokens describe a different ordered result set.
 */
export function useFingerprintLibraryQueryState<T extends FingerprintResource>({
  query,
  setQuery,
  data,
  isPlaceholderData,
  filterConfig,
  sortableFields,
  defaultSorting,
}: UseFingerprintLibraryQueryStateOptions<T>) {
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const page = query.pageIndex ?? 1
  const pageSize = query.pageSize
  const controlSignature = `${query.search ?? ""}|${filterSignature(query.filters)}|${query.sorting?.field ?? ""}|${query.sorting?.direction ?? ""}|${pageSize}`
  const previousControlSignatureRef = React.useRef(controlSignature)
  const nextPageToken = getCurrentCursorNextPageToken(
    data?.nextPageToken,
    isPlaceholderData,
  )

  const resetPageTokens = React.useCallback(() => {
    setPageTokens({ 1: undefined })
  }, [])

  const resetPagination = React.useCallback(() => {
    // A cursor describes an earlier ordered result set and cannot survive a mutation.
    resetPageTokens()
    setQuery((current) => setBusinessListPage(current, {
      pageIndex: 1,
      pageToken: undefined,
    }))
  }, [resetPageTokens, setQuery])

  React.useEffect(() => {
    if (nextPageToken) {
      setPageTokens((current) => ({ ...current, [page + 1]: nextPageToken }))
    }
  }, [nextPageToken, page])

  React.useEffect(() => {
    if (previousControlSignatureRef.current !== controlSignature) {
      previousControlSignatureRef.current = controlSignature
      resetPageTokens()
    }
  }, [controlSignature, resetPageTokens])

  const commitSearch = React.useCallback((value: string) => {
    const currentSearch = query.search ?? ""
    const normalizedSearch = value.trim()
    if (normalizedSearch === currentSearch) {
      return
    }

    resetPageTokens()
    setQuery((current) => applyBusinessListControlChange(current, {
      search: normalizedSearch || undefined,
    }))
  }, [query.search, resetPageTokens, setQuery])

  const updateFacet = React.useCallback((field: string, values: string[]) => {
    resetPageTokens()
    setQuery((current) => applyBusinessListControlChange(current, {
      filters: {
        ...current.filters,
        [field]: Array.from(new Set(values.map((value) => value.trim()).filter(Boolean))),
      },
    }))
  }, [resetPageTokens, setQuery])

  const onQueryChange = React.useCallback((updater: QueryUpdater) => {
    setQuery((current) => updater(current))
  }, [setQuery])

  const pagination = React.useMemo<PaginationState>(() => ({
    pageIndex: page - 1,
    pageSize,
  }), [page, pageSize])

  const cursorPaginationSummary = React.useMemo<CursorPaginationSummary>(() => ({
    total: data?.totalSize ?? 0,
  }), [data?.totalSize])

  const paginationNavigation = React.useMemo<CursorPaginationNavigation>(() => getCursorPaginationNavigation({
    currentPage: page,
    pageTokens,
    nextPageToken,
  }), [nextPageToken, page, pageTokens])

  const sorting = React.useMemo<SortingState>(() => {
    if (!query.sorting) {
      return []
    }
    return [{ id: query.sorting.field, desc: query.sorting.direction === "desc" }]
  }, [query.sorting])

  const onPaginationChange = React.useCallback((nextPagination: PaginationState) => {
    const nextPage = nextPagination.pageIndex + 1
    if (nextPagination.pageSize !== pageSize) {
      resetPageTokens()
      setQuery((current) => applyBusinessListControlChange(current, { pageSize: nextPagination.pageSize }))
      return
    }

    if (nextPage === 1) {
      resetPagination()
      return
    }

    const transition = getCursorPageTransition({
      currentPage: page,
      pageTokens,
      nextPageToken,
      requestedPage: nextPage,
    })
    if (!transition.reachable) return

    setQuery((current) => setBusinessListPage(current, {
      pageIndex: nextPage,
      pageToken: transition.pageToken,
    }))
  }, [nextPageToken, page, pageSize, pageTokens, resetPageTokens, setQuery])

  const onSortingChange = React.useCallback((nextSorting: SortingState) => {
    const next = nextSorting[0]
    const nextField = next?.id
    const nextConfig = nextField ? sortableFields[nextField] : undefined

    resetPageTokens()
    setQuery((current) => applyBusinessListControlChange(current, {
      sorting: nextConfig && nextField
        ? { field: nextField, direction: next?.desc ? "desc" : "asc" }
        : defaultSorting,
    }))
  }, [defaultSorting, resetPageTokens, setQuery, sortableFields])

  return {
    query,
    page,
    pageSize,
    pageTokens,
    resetPagination,
    commitSearch,
    updateFacet,
    onQueryChange,
    pagination,
    cursorPaginationSummary,
    paginationNavigation,
    sorting,
    onPaginationChange,
    onSortingChange,
    request: buildFingerprintLibraryListParams(query, filterConfig, sortableFields),
  }
}
