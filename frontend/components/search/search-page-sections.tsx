"use client"

import dynamic from "next/dynamic"
import { AlertCircle, Download, History, Search, X } from "@/components/icons"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

import { QUICK_SEARCH_TAGS, type SearchPageState } from "./search-page-state"

const SearchResultCard = dynamic(
  () => import("./search-result-card").then((mod) => mod.SearchResultCard),
  {
    ssr: false,
    loading: () => <Skeleton className="h-32 w-full" />,
  }
)

const SearchResultsTable = dynamic(
  () => import("./search-results-table").then((mod) => mod.SearchResultsTable),
  {
    ssr: false,
    loading: () => <Skeleton className="h-64 w-full" />,
  }
)

const VulnerabilityDetailDialog = dynamic(
  () => import("@/components/vulnerabilities/vulnerability-detail-dialog").then((mod) => mod.VulnerabilityDetailDialog),
  { ssr: false }
)

const SmartFilterInput = dynamic(
  () => import("@/components/common/smart-filter-input").then((mod) => mod.SmartFilterInput),
  {
    ssr: false,
    loading: () => <Skeleton className="h-10 w-full" />,
  }
)

const SearchPagination = dynamic(
  () => import("./search-pagination").then((mod) => mod.SearchPagination),
  { ssr: false }
)

function SearchAssetTypeSelector({ state }: { state: SearchPageState }) {
  return (
    <Select value={state.assetType} onValueChange={state.handleAssetTypeChange}>
      <SelectTrigger size="sm" className="w-[100px]">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="website">{state.t("assetTypes.website")}</SelectItem>
        <SelectItem value="endpoint">{state.t("assetTypes.endpoint")}</SelectItem>
      </SelectContent>
    </Select>
  )
}

export function SearchPageContent({ state }: { state: SearchPageState }) {
  return (
    <div className="flex-1 w-full flex flex-col">
      {state.searchState === "initial" ? (
        <div className="flex-1 flex flex-col items-center justify-center px-4 relative overflow-hidden animate-in fade-in slide-in-from-bottom-4 duration-300">
          <div className="absolute inset-0 -z-10 overflow-hidden pointer-events-none">
            <div className="absolute left-1/2 top-1/4 -translate-x-1/2 h-[400px] w-[600px] rounded-full bg-primary/5 blur-3xl" />
            <div className="absolute right-1/4 top-1/2 h-[200px] w-[300px] rounded-full bg-primary/3 blur-2xl" />
          </div>

          <div className="flex flex-col items-center gap-6 w-full max-w-3xl -mt-16">
            <div className="flex flex-col items-center gap-2">
              <div className="flex items-center justify-center w-16 h-16 rounded-2xl bg-primary/10 mb-2">
                <Search className="h-8 w-8 text-primary" />
              </div>
              <h1 className="text-3xl font-semibold text-foreground">{state.t("title")}</h1>
              <p className="text-sm text-muted-foreground">{state.t("hint")}</p>
            </div>

            <div className="flex items-center gap-3 w-full">
              <SearchAssetTypeSelector state={state} />
              <SmartFilterInput
                fields={state.searchFilterFields}
                examples={state.searchExamples}
                placeholder='host="api" && tech="nginx" && status=="200"'
                value={state.query}
                onSearch={state.handleSearch}
                className="flex-1"
              />
            </div>

            <div className="flex flex-wrap justify-center gap-2">
              {QUICK_SEARCH_TAGS.map((tag) => (
                <Badge
                  asChild
                  key={tag.query}
                  variant="outline"
                  className="hover:bg-accent transition-colors px-3 py-1"
                >
                  <button
                    type="button"
                    onClick={() => state.handleQuickTagClick(tag.query)}
                  >
                    {tag.label}
                  </button>
                </Badge>
              ))}
            </div>

            {state.recentSearches.length > 0 ? (
              <div className="w-full max-w-xl mt-2 animate-in fade-in duration-300 delay-300">
                <div className="flex items-center gap-2 text-xs text-muted-foreground mb-2">
                  <History className="h-3.5 w-3.5" />
                  <span>{state.t("recentSearches")}</span>
                </div>
                <div className="flex flex-wrap gap-2">
                  {state.recentSearches.map((search) => (
                    <Badge
                      key={search}
                      variant="secondary"
                      className={cn(
                        "hover:bg-secondary/80 transition-colors",
                        "pl-3 pr-1.5 py-1 gap-1 group"
                      )}
                    >
                      <button
                        type="button"
                        onClick={() => state.handleRecentSearchClick(search)}
                        className="font-mono text-xs truncate max-w-[200px] text-left"
                      >
                        {search}
                      </button>
                      <button
                        type="button"
                        onClick={(e) => state.handleRemoveRecentSearch(e, search)}
                        className="ml-1 p-0.5 rounded hover:bg-muted-foreground/20 opacity-0 group-hover:opacity-100 transition-opacity"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  ))}
                </div>
              </div>
            ) : null}
          </div>
        </div>
      ) : null}

      {state.searchState === "searching" && state.isLoading ? (
        <div className="h-full flex flex-col items-center justify-center animate-in fade-in duration-200">
          <div className="flex flex-col items-center gap-4">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
            <span className="text-muted-foreground">{state.t("searching")}</span>
          </div>
        </div>
      ) : null}

      {(state.searchState === "results" || (state.searchState === "searching" && !state.isLoading)) ? (
        <div className="h-full flex flex-col animate-in fade-in duration-300">
          <div className="sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-b px-4 py-3 animate-in fade-in slide-in-from-top-2 duration-300 delay-100">
            <div className="flex items-center gap-3">
              <SearchAssetTypeSelector state={state} />
              <SmartFilterInput
                fields={state.searchFilterFields}
                examples={state.searchExamples}
                placeholder='host="api" && tech="nginx" && status=="200"'
                value={state.query}
                onSearch={state.handleSearch}
                className="flex-1"
              />
              <span className="text-sm text-muted-foreground whitespace-nowrap">
                {state.isFetching
                  ? state.t("loading")
                  : state.t("resultsCount", { count: state.data?.total ?? 0 })}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={state.handleExportCSV}
                disabled={!state.data?.results || state.data.results.length === 0 || state.isExporting}
              >
                <Download className="h-4 w-4 mr-1.5" />
                {state.isExporting ? state.t("exporting") : state.t("export")}
              </Button>
            </div>
          </div>

          {state.error ? (
            <div className="p-4 w-full">
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{state.t("error")}</AlertDescription>
              </Alert>
            </div>
          ) : null}

          {!state.error && state.data?.results.length === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center p-4">
              <div className="text-center">
                <Search className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                <h3 className="text-lg font-medium mb-2">{state.t("noResults")}</h3>
                <p className="text-sm text-muted-foreground">{state.t("noResultsHint")}</p>
              </div>
            </div>
          ) : null}

          {!state.error && state.data && state.data.results.length > 0 ? (
            <>
              <div className="flex-1 overflow-auto p-4">
                {state.assetType === "website" ? (
                  <div className="space-y-4 max-w-4xl mx-auto">
                    {state.data.results.map((result) => (
                      <SearchResultCard
                        key={result.id}
                        result={result}
                        onViewVulnerability={state.handleViewVulnerability}
                      />
                    ))}
                  </div>
                ) : (
                  <SearchResultsTable
                    results={state.data.results}
                    assetType={state.assetType}
                    onViewVulnerability={state.handleViewVulnerability}
                  />
                )}
              </div>

              <div className="border-t px-4 py-3">
                <SearchPagination
                  page={state.page}
                  pageSize={state.pageSize}
                  total={state.paginationInfo.total}
                  totalPages={state.paginationInfo.totalPages}
                  onPageChange={state.handlePageChange}
                  onPageSizeChange={state.handlePageSizeChange}
                />
              </div>
            </>
          ) : null}
        </div>
      ) : null}

      {state.vulnDialogOpen ? (
        <VulnerabilityDetailDialog
          vulnerability={state.selectedVuln}
          open={state.vulnDialogOpen}
          onOpenChange={state.setVulnDialogOpen}
        />
      ) : null}
    </div>
  )
}
