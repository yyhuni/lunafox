import React from "react"
import { useSearchParams } from "next/navigation"
import { useTranslations } from "next-intl"
import { toast } from "sonner"

import { useAssetSearch } from "@/hooks/use-search"
import { buildPaginationInfo } from "@/hooks/_shared/pagination"
import { VulnerabilityService } from "@/services/vulnerability.service"
import { SearchService } from "@/services/search.service"

import type { FilterField } from "@/components/common/smart-filter-input"
import type {
  AssetType,
  SearchParams,
  SearchState,
  Vulnerability as SearchVuln,
} from "@/types/search.types"
import type { Vulnerability } from "@/types/vulnerability.types"

const WEBSITE_SEARCH_EXAMPLES = [
  'host="api"',
  'title="Dashboard"',
  'tech="nginx"',
  'status=="200"',
  'host="api" && status=="200"',
  'tech="vue" || tech="react"',
  'host="admin" && tech="php" && status=="200"',
  'status!="404"',
]

const ENDPOINT_SEARCH_EXAMPLES = [
  'host="api"',
  'url="/api/v1"',
  'title="Dashboard"',
  'tech="nginx"',
  'status=="200"',
  'host="api" && status=="200"',
  'url="/admin" && status=="200"',
  'tech="vue" || tech="react"',
]

export const QUICK_SEARCH_TAGS = [
  { label: 'status=="200"', query: 'status=="200"' },
  { label: 'tech="nginx"', query: 'tech="nginx"' },
  { label: 'tech="php"', query: 'tech="php"' },
  { label: 'tech="vue"', query: 'tech="vue"' },
  { label: 'tech="react"', query: 'tech="react"' },
  { label: 'status=="403"', query: 'status=="403"' },
]

const RECENT_SEARCHES_KEY = "star_patrol_recent_searches"
const MAX_RECENT_SEARCHES = 5

function getRecentSearches(): string[] {
  if (typeof window === "undefined") return []
  try {
    const saved = localStorage.getItem(RECENT_SEARCHES_KEY)
    return saved ? JSON.parse(saved) : []
  } catch {
    return []
  }
}

function saveRecentSearch(query: string) {
  if (typeof window === "undefined" || !query.trim()) return
  try {
    const searches = getRecentSearches().filter((search) => search !== query)
    searches.unshift(query)
    localStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(searches.slice(0, MAX_RECENT_SEARCHES)))
  } catch {
    // ignore
  }
}

function removeRecentSearch(query: string) {
  if (typeof window === "undefined") return
  try {
    const searches = getRecentSearches().filter((search) => search !== query)
    localStorage.setItem(RECENT_SEARCHES_KEY, JSON.stringify(searches))
  } catch {
    // ignore
  }
}

export function useSearchPageState() {
  const t = useTranslations("search")
  const urlSearchParams = useSearchParams()
  const [searchState, setSearchState] = React.useState<SearchState>("initial")
  const [query, setQuery] = React.useState("")
  const [assetType, setAssetType] = React.useState<AssetType>("website")
  const [searchParams, setSearchParams] = React.useState<SearchParams>({})
  const [page, setPage] = React.useState(1)
  const [pageSize, setPageSize] = React.useState(10)
  const [selectedVuln, setSelectedVuln] = React.useState<Vulnerability | null>(null)
  const [vulnDialogOpen, setVulnDialogOpen] = React.useState(false)
  const [, setLoadingVuln] = React.useState(false)
  const [recentSearches, setRecentSearches] = React.useState<string[]>([])
  const [isExporting, setIsExporting] = React.useState(false)
  const lastUrlSyncKeyRef = React.useRef<string | null>(null)

  React.useEffect(() => {
    setRecentSearches(getRecentSearches())
  }, [])

  React.useEffect(() => {
    const q = urlSearchParams.get("q")?.trim() ?? ""
    if (!q) {
      lastUrlSyncKeyRef.current = null
      return
    }

    const syncKey = `${assetType}:${q}`
    if (lastUrlSyncKeyRef.current === syncKey) {
      return
    }

    lastUrlSyncKeyRef.current = syncKey
    setQuery(q)
    setSearchParams({ q, asset_type: assetType })
    setPage(1)
    setSearchState("searching")
    saveRecentSearch(q)
    setRecentSearches(getRecentSearches())
  }, [assetType, urlSearchParams])

  const searchExamples = React.useMemo(() => {
    return assetType === "endpoint" ? ENDPOINT_SEARCH_EXAMPLES : WEBSITE_SEARCH_EXAMPLES
  }, [assetType])

  const searchFilterFields: FilterField[] = React.useMemo(
    () => [
      { key: "host", label: "Host", description: t("fields.host") },
      { key: "url", label: "URL", description: t("fields.url") },
      { key: "title", label: "Title", description: t("fields.title") },
      { key: "tech", label: "Tech", description: t("fields.tech") },
      { key: "status", label: "Status", description: t("fields.status") },
      { key: "body", label: "Body", description: t("fields.body") },
      { key: "header", label: "Header", description: t("fields.header") },
    ],
    [t]
  )

  const { data, isLoading, error, isFetching } = useAssetSearch(
    { ...searchParams, page, pageSize },
    { enabled: searchState === "results" || searchState === "searching" }
  )

  const paginationInfo = buildPaginationInfo({
    total: data?.total ?? 0,
    page,
    pageSize,
    totalPages: data?.totalPages || undefined,
    minTotalPages: data?.total ? 1 : 0,
  })

  const handleSearch = React.useCallback(
    (_filters: unknown, rawQuery: string) => {
      if (!rawQuery.trim()) return

      setQuery(rawQuery)
      setSearchParams({ q: rawQuery, asset_type: assetType })
      setPage(1)
      setSearchState("searching")

      saveRecentSearch(rawQuery)
      setRecentSearches(getRecentSearches())
    },
    [assetType]
  )

  const handleQuickTagClick = React.useCallback((tagQuery: string) => {
    setQuery(tagQuery)
  }, [])

  const handleRecentSearchClick = React.useCallback(
    (recentQuery: string) => {
      setQuery(recentQuery)
      setSearchParams({ q: recentQuery, asset_type: assetType })
      setPage(1)
      setSearchState("searching")
      saveRecentSearch(recentQuery)
      setRecentSearches(getRecentSearches())
    },
    [assetType]
  )

  const handleRemoveRecentSearch = React.useCallback(
    (e: React.MouseEvent, searchQuery: string) => {
      e.stopPropagation()
      removeRecentSearch(searchQuery)
      setRecentSearches(getRecentSearches())
    },
    []
  )

  const handleExportCSV = React.useCallback(async () => {
    if (!searchParams.q) return

    setIsExporting(true)
    try {
      await SearchService.exportCSV(searchParams.q, assetType)
      toast.success(t("exportSuccess"))
    } catch {
      toast.error(t("exportFailed"))
    } finally {
      setIsExporting(false)
    }
  }, [assetType, searchParams.q, t])

  React.useEffect(() => {
    if (searchState === "searching" && data && !isLoading) {
      setSearchState("results")
    }
  }, [data, isLoading, searchState])

  const handleAssetTypeChange = React.useCallback(
    (value: AssetType) => {
      setAssetType(value)
      if (searchState === "results") {
        setSearchState("initial")
        setSearchParams({})
        setQuery("")
      }
    },
    [searchState]
  )

  const handlePageChange = React.useCallback((newPage: number) => {
    setPage(newPage)
  }, [])

  const handlePageSizeChange = React.useCallback((newPageSize: number) => {
    setPageSize(newPageSize)
    setPage(1)
  }, [])

  const handleViewVulnerability = React.useCallback(
    async (vuln: SearchVuln) => {
      if (!vuln.id) return

      setLoadingVuln(true)
      try {
        const fullVuln = await VulnerabilityService.getVulnerabilityById(vuln.id)
        setSelectedVuln(fullVuln)
        setVulnDialogOpen(true)
      } catch {
        toast.error(t("vulnLoadError"))
      } finally {
        setLoadingVuln(false)
      }
    },
    [t]
  )

  return {
    t,
    searchState,
    query,
    assetType,
    searchFilterFields,
    searchExamples,
    recentSearches,
    data,
    isLoading,
    isFetching,
    error,
    paginationInfo,
    page,
    pageSize,
    isExporting,
    selectedVuln,
    vulnDialogOpen,
    setVulnDialogOpen,
    handleSearch,
    handleQuickTagClick,
    handleRecentSearchClick,
    handleRemoveRecentSearch,
    handleExportCSV,
    handleAssetTypeChange,
    handlePageChange,
    handlePageSizeChange,
    handleViewVulnerability,
  }
}

export type SearchPageState = ReturnType<typeof useSearchPageState>
