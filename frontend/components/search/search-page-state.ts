"use client"

import * as React from "react"
import { useSearchParams } from "next/navigation"
import { useTranslations } from "next-intl"
import { readJsonStorage, writeJsonStorage } from "@/lib/browser-storage"
import {
  GLOBAL_ASSET_SEARCH_DEFAULT_PAGE_SIZE,
  getGlobalAssetSearchDiagnosticCode,
  getGlobalAssetSearchDiagnosticMessageKeys,
  parseGlobalAssetSearchQuery,
  isGlobalAssetSearchQueryValid,
  type GlobalAssetSearchDiagnosticCode,
} from "@/lib/global-asset-search-query"
import { useAssetSearch } from "@/hooks/use-search"
import type { AssetType, SearchParams, SearchState } from "@/types/search.types"

const RECENT_SEARCHES_KEY = "star_patrol_recent_searches"
const MAX_RECENT_SEARCHES = 5

function getRecentSearches(): string[] {
  if (typeof window === "undefined") return []
  return readJsonStorage<string[]>(RECENT_SEARCHES_KEY, [])
}

function saveRecentSearch(query: string) {
  if (typeof window === "undefined" || !query.trim()) return
  const searches = getRecentSearches().filter((search) => search !== query)
  searches.unshift(query)
  writeJsonStorage(RECENT_SEARCHES_KEY, searches.slice(0, MAX_RECENT_SEARCHES))
}

function removeRecentSearch(query: string) {
  if (typeof window === "undefined") return
  writeJsonStorage(RECENT_SEARCHES_KEY, getRecentSearches().filter((search) => search !== query))
}

function getDiagnosticText(
  t: (key: string) => string,
  code: GlobalAssetSearchDiagnosticCode
) {
  const keys = getGlobalAssetSearchDiagnosticMessageKeys(code)
  return `${t(keys.message)} ${t(keys.fix)}`
}

function getQueryDiagnosticCode(query: string): GlobalAssetSearchDiagnosticCode | null {
  try {
    parseGlobalAssetSearchQuery(query)
    return null
  } catch (error) {
    return getGlobalAssetSearchDiagnosticCode(error)
  }
}

export function useSearchPageState() {
  const t = useTranslations("search")
  const urlSearchParams = useSearchParams()
  // Keep the draft as entered for shareable search links; the query parser
  // trims user-facing contains terms before building the search predicate.
  const initialQuery = urlSearchParams.get("q") ?? ""
  const initialQueryIsValid = !initialQuery || isGlobalAssetSearchQueryValid(initialQuery)
  const initialQueryDiagnosticCode = initialQuery && !initialQueryIsValid
    ? getQueryDiagnosticCode(initialQuery)
    : null
  const initialSearchParams: SearchParams | undefined = initialQuery && initialQueryIsValid
    ? { q: initialQuery, assetType: "website", pageSize: GLOBAL_ASSET_SEARCH_DEFAULT_PAGE_SIZE }
    : undefined
  const [searchState, setSearchState] = React.useState<SearchState>(() => (
    initialSearchParams ? "searching" : "initial"
  ))
  const [query, setQueryState] = React.useState(initialQuery)
  const [assetType, setAssetType] = React.useState<AssetType>("website")
  const [submittedQuery, setSubmittedQuery] = React.useState(initialSearchParams?.q ?? "")
  const [pageSize, setPageSize] = React.useState(GLOBAL_ASSET_SEARCH_DEFAULT_PAGE_SIZE)
  const [pageTokens, setPageTokens] = React.useState<Array<string | undefined>>([undefined])
  const [pageIndex, setPageIndex] = React.useState(0)
  const [recentSearches, setRecentSearches] = React.useState<string[]>([])
  const [queryDiagnosticCode, setQueryDiagnosticCode] = React.useState<GlobalAssetSearchDiagnosticCode | null>(initialQueryDiagnosticCode)
  const lastUrlQueryRef = React.useRef(initialQuery)

  const setQuery = React.useCallback((nextQuery: React.SetStateAction<string>) => {
    setQueryState(nextQuery)
  }, [])

  const queryError = queryDiagnosticCode ? getDiagnosticText(t, queryDiagnosticCode) : null

  React.useEffect(() => {
    setRecentSearches(getRecentSearches())
  }, [])

  const beginSearch = React.useCallback((nextQuery: string, nextAssetType: AssetType, nextPageSize = pageSize) => {
    try {
      parseGlobalAssetSearchQuery(nextQuery)
    } catch (error) {
      setQueryDiagnosticCode(getGlobalAssetSearchDiagnosticCode(error))
      return false
    }

    setQueryState(nextQuery)
    setAssetType(nextAssetType)
    setSubmittedQuery(nextQuery)
    setPageSize(nextPageSize)
    setPageTokens([undefined])
    setPageIndex(0)
    setQueryDiagnosticCode(null)
    setSearchState("searching")
    saveRecentSearch(nextQuery)
    setRecentSearches(getRecentSearches())
    return true
  }, [pageSize])

  React.useEffect(() => {
    if (!queryDiagnosticCode) return
    const nextCode = getQueryDiagnosticCode(query)
    setQueryDiagnosticCode((current) => current === nextCode ? current : nextCode)
  }, [query, queryDiagnosticCode])

  React.useEffect(() => {
    const nextQuery = urlSearchParams.get("q") ?? ""
    if (nextQuery === lastUrlQueryRef.current) return
    lastUrlQueryRef.current = nextQuery
    setQueryState(nextQuery)

    if (!nextQuery) {
      setSubmittedQuery("")
      setPageTokens([undefined])
      setPageIndex(0)
      setSearchState("initial")
      setQueryDiagnosticCode(null)
      return
    }
    const nextCode = getQueryDiagnosticCode(nextQuery)
    if (nextCode) {
      setSubmittedQuery("")
      setPageTokens([undefined])
      setPageIndex(0)
      setSearchState("initial")
      setQueryDiagnosticCode(nextCode)
      return
    }
    beginSearch(nextQuery, "website")
  }, [beginSearch, urlSearchParams])

  const activeSearchParams = React.useMemo<SearchParams | undefined>(() => {
    if (!submittedQuery) return undefined
    const pageToken = pageTokens[pageIndex]
    return {
      q: submittedQuery,
      assetType,
      pageSize,
      ...(pageToken ? { pageToken } : {}),
    }
  }, [assetType, pageIndex, pageSize, pageTokens, submittedQuery])

  const { data, isLoading, error, isFetching, refetch } = useAssetSearch(activeSearchParams, {
    enabled: searchState === "searching" || searchState === "results",
  })

  const tPagination = useTranslations("common.pagination")
  // The capped summary is display-only: navigation stays token-driven, so the
  // summary never authorizes a page transition.
  const paginationSummary = React.useMemo(() => {
    if (data?.totalSize === undefined) return undefined
    return data.totalSizeCapped
      ? tPagination("totalCapped", { count: data.totalSize })
      : tPagination("total", { count: data.totalSize })
  }, [data?.totalSize, data?.totalSizeCapped, tPagination])

  React.useEffect(() => {
    if (searchState === "searching" && !isLoading && (data || error)) {
      setSearchState("results")
    }
  }, [data, error, isLoading, searchState])

  const handleSearch = React.useCallback(() => {
    beginSearch(query, assetType)
  }, [assetType, beginSearch, query])

  const handleQuickTagClick = React.useCallback((tagQuery: string) => {
    setQuery(tagQuery)
  }, [setQuery])

  const handleRecentSearchClick = React.useCallback((recentQuery: string) => {
    setQuery(recentQuery)
  }, [setQuery])

  const handleRemoveRecentSearch = React.useCallback((event: React.MouseEvent, searchQuery: string) => {
    event.stopPropagation()
    removeRecentSearch(searchQuery)
    setRecentSearches(getRecentSearches())
  }, [])

  const handleAssetTypeChange = React.useCallback((value: AssetType) => {
    if (value === assetType) return
    if (submittedQuery) {
      beginSearch(submittedQuery, value)
      return
    }
    setAssetType(value)
    setPageTokens([undefined])
    setPageIndex(0)
  }, [assetType, beginSearch, submittedQuery])

  const handlePageSizeChange = React.useCallback((nextPageSize: number) => {
    if (nextPageSize === pageSize) return
    if (submittedQuery) {
      beginSearch(submittedQuery, assetType, nextPageSize)
      return
    }
    setPageSize(nextPageSize)
    setPageTokens([undefined])
    setPageIndex(0)
  }, [assetType, beginSearch, pageSize, submittedQuery])

  const handlePreviousPage = React.useCallback(() => {
    if (pageIndex === 0) return
    setPageIndex((current) => current - 1)
    setSearchState("searching")
  }, [pageIndex])

  const handleFirstPage = React.useCallback(() => {
    if (pageIndex === 0) return
    setPageTokens([undefined])
    setPageIndex(0)
    setSearchState("searching")
  }, [pageIndex])

  const handleNextPage = React.useCallback(() => {
    const nextPageToken = data?.nextPageToken
    if (!nextPageToken) return
    setPageTokens((current) => [...current.slice(0, pageIndex + 1), nextPageToken])
    setPageIndex((current) => current + 1)
    setSearchState("searching")
  }, [data?.nextPageToken, pageIndex])

  return {
    t,
    searchState,
    query,
    assetType,
    pageSize,
    recentSearches,
    queryError,
    queryDiagnosticCode,
    data,
    isLoading,
    isFetching,
    error,
    refetch,
    canFirstPage: pageIndex > 0,
    canPreviousPage: pageIndex > 0,
    canNextPage: Boolean(data?.nextPageToken),
    paginationSummary,
    setQuery,
    handleSearch,
    handleQuickTagClick,
    handleRecentSearchClick,
    handleRemoveRecentSearch,
    handleAssetTypeChange,
    handlePageSizeChange,
    handlePreviousPage,
    handleFirstPage,
    handleNextPage,
  }
}

export type SearchPageState = ReturnType<typeof useSearchPageState>
