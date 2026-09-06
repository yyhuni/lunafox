"use client"

import * as React from "react"
import { useSearchParams } from "next/navigation"
import { useTranslations } from "next-intl"
import { readJsonStorage, writeJsonStorage } from "@/lib/browser-storage"
import {
  GLOBAL_ASSET_SEARCH_DEFAULT_PAGE_SIZE,
  isGlobalAssetSearchQueryValid,
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

export function useSearchPageState() {
  const t = useTranslations("search")
  const urlSearchParams = useSearchParams()
  // A URL in q is an exact observed-value lookup. Do not trim it while moving
  // through client route state, otherwise a valid stored payload cannot round
  // trip from a shareable search link.
  const initialQuery = urlSearchParams.get("q") ?? ""
  const initialQueryIsValid = !initialQuery || isGlobalAssetSearchQueryValid(initialQuery)
  const initialSearchParams: SearchParams | undefined = initialQuery && initialQueryIsValid
    ? { q: initialQuery, assetType: "website", pageSize: GLOBAL_ASSET_SEARCH_DEFAULT_PAGE_SIZE }
    : undefined
  const [searchState, setSearchState] = React.useState<SearchState>(() => (
    initialSearchParams ? "searching" : "initial"
  ))
  const [query, setQuery] = React.useState(initialQuery)
  const [assetType, setAssetType] = React.useState<AssetType>("website")
  const [submittedQuery, setSubmittedQuery] = React.useState(initialSearchParams?.q ?? "")
  const [pageSize, setPageSize] = React.useState(GLOBAL_ASSET_SEARCH_DEFAULT_PAGE_SIZE)
  const [pageTokens, setPageTokens] = React.useState<Array<string | undefined>>([undefined])
  const [pageIndex, setPageIndex] = React.useState(0)
  const [recentSearches, setRecentSearches] = React.useState<string[]>([])
  const [queryError, setQueryError] = React.useState<string | null>(() => (
    initialQuery && !initialQueryIsValid ? t("invalidQuery") : null
  ))
  const lastUrlQueryRef = React.useRef(initialQuery)

  React.useEffect(() => {
    setRecentSearches(getRecentSearches())
  }, [])

  const beginSearch = React.useCallback((nextQuery: string, nextAssetType: AssetType, nextPageSize = pageSize) => {
    if (!isGlobalAssetSearchQueryValid(nextQuery)) {
      setQueryError(t("invalidQuery"))
      return false
    }

    setQuery(nextQuery)
    setAssetType(nextAssetType)
    setSubmittedQuery(nextQuery)
    setPageSize(nextPageSize)
    setPageTokens([undefined])
    setPageIndex(0)
    setQueryError(null)
    setSearchState("searching")
    saveRecentSearch(nextQuery)
    setRecentSearches(getRecentSearches())
    return true
  }, [pageSize, t])

  React.useEffect(() => {
    const nextQuery = urlSearchParams.get("q") ?? ""
    if (nextQuery === lastUrlQueryRef.current) return
    lastUrlQueryRef.current = nextQuery
    setQuery(nextQuery)

    if (!nextQuery) {
      setSubmittedQuery("")
      setPageTokens([undefined])
      setPageIndex(0)
      setSearchState("initial")
      setQueryError(null)
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
  }, [])

  const handleRecentSearchClick = React.useCallback((recentQuery: string) => {
    setQuery(recentQuery)
  }, [])

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
    data,
    isLoading,
    isFetching,
    error,
    refetch,
    canFirstPage: pageIndex > 0,
    canPreviousPage: pageIndex > 0,
    canNextPage: Boolean(data?.nextPageToken),
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
