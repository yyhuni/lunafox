"use client"

import * as React from "react"
import { History, LayoutGrid, Search, semanticIcons, X } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/select"
import { Popover, PopoverAnchor, PopoverContent } from "@/components/ui/popover"
import { Skeleton } from "@/components/ui/skeleton"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { CompactPaginationSkeleton } from "@/components/shared/loading/compact-pagination-skeleton"
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { SelectShellSkeleton } from "@/components/shared/loading/select-shell-skeleton"
import { normalizeError } from "@/lib/errors/normalize-error"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import {
  appendGlobalAssetSearchCondition,
  getGlobalAssetSearchInlineCompletion,
  SearchSyntaxGuidance,
} from "./search-syntax-guidance"
import type { GlobalAssetSearchField } from "@/lib/global-asset-search-query"
import type { WebsiteSearchResult } from "@/types/search.types"
import { SearchPagination } from "./search-pagination"
import {
  SearchWebsiteExportMenu,
  SearchWebsitesDataTable,
  SearchWebsitesDataTableLoadingState,
  useSearchWebsitesResultModel,
} from "./search-websites-data-table"
import { SearchResultsTable } from "./search-results-table"
import {
  SearchAssetBarShell,
  SearchInitialHeading,
  SearchInitialPageShell,
  SearchQuickTags,
} from "./search-initial-page-shell"
import type { SearchPageState } from "./search-page-state"

const EMPTY_WEBSITE_RESULTS: WebsiteSearchResult[] = []

function SearchAssetTypeSelector({ state, integrated = false }: {
  state: SearchPageState
  integrated?: boolean
}) {
  const assetTypeLabel = state.assetType === "website" ? state.t("assetTypes.website") : state.t("assetTypes.endpoint")
  return (
    <Select value={state.assetType} onValueChange={state.handleAssetTypeChange}>
      <SelectTrigger
        size={integrated ? "default" : "sm"}
        className={cn(
          integrated
            ? "radius-none w-40 border-0 border-r bg-muted/30 px-4 text-sm shadow-none hover:bg-muted/50 focus-visible:z-10 focus-visible:ring-0"
            : "w-28"
        )}
      >
        {integrated ? <LayoutGrid className="h-4 w-4 text-muted-foreground" /> : null}
        <span className="truncate">{assetTypeLabel}</span>
      </SelectTrigger>
      <SelectContent width="content-fit">
        <SelectItem value="website">
          <span className="inline-flex min-w-0 items-center gap-1.5">
            <semanticIcons.concept.website className="h-4 w-4 shrink-0 text-muted-foreground" />
            <span className="truncate">{state.t("assetTypes.website")}</span>
          </span>
        </SelectItem>
        <SelectItem value="endpoint">
          <span className="inline-flex min-w-0 items-center gap-1.5">
            <semanticIcons.concept.endpoint className="h-4 w-4 shrink-0 text-muted-foreground" />
            <span className="truncate">{state.t("assetTypes.endpoint")}</span>
          </span>
        </SelectItem>
      </SelectContent>
    </Select>
  )
}

function AssetSearchBar({ state, className }: { state: SearchPageState; className?: string }) {
  const inputRef = React.useRef<HTMLInputElement>(null)
  const closeTimerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null)
  const [syntaxOpen, setSyntaxOpen] = React.useState(false)
  const [caretIsAtDraftEnd, setCaretIsAtDraftEnd] = React.useState(true)
  const inlineCompletion = React.useMemo(
    () => caretIsAtDraftEnd ? getGlobalAssetSearchInlineCompletion(state.query) : "",
    [caretIsAtDraftEnd, state.query]
  )

  const clearCloseTimer = React.useCallback(() => {
    if (closeTimerRef.current) {
      clearTimeout(closeTimerRef.current)
      closeTimerRef.current = null
    }
  }, [])

  React.useEffect(() => clearCloseTimer, [clearCloseTimer])

  const openSyntaxGuidance = React.useCallback(() => {
    clearCloseTimer()
    setSyntaxOpen(true)
  }, [clearCloseTimer])

  const scheduleCloseSyntaxGuidance = React.useCallback((event: React.FocusEvent<HTMLInputElement>) => {
    const relatedTarget = event.relatedTarget as HTMLElement | null
    if (relatedTarget?.closest('[data-slot="popover-content"]')) return
    clearCloseTimer()
    // A syntax shortcut receives the input blur before its select handler can restore focus.
    closeTimerRef.current = setTimeout(() => {
      setSyntaxOpen(false)
      closeTimerRef.current = null
    }, 150)
  }, [clearCloseTimer])

  const focusInput = React.useCallback((cursorPosition?: number) => {
    window.requestAnimationFrame(() => {
      const input = inputRef.current
      if (!input) return
      input.focus()
      const position = cursorPosition ?? input.value.length
      input.setSelectionRange(position, position)
    })
  }, [])

  const syncCaretPosition = React.useCallback((input: HTMLInputElement) => {
    setCaretIsAtDraftEnd(
      input.selectionStart === input.value.length && input.selectionEnd === input.value.length
    )
  }, [])

  const handleInputChange = React.useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    state.setQuery(event.target.value)
    syncCaretPosition(event.currentTarget)
  }, [state, syncCaretPosition])

  const handleInputSelection = React.useCallback((event: React.SyntheticEvent<HTMLInputElement>) => {
    syncCaretPosition(event.currentTarget)
  }, [syncCaretPosition])

  const handleInputKeyDown = React.useCallback((event: React.KeyboardEvent<HTMLInputElement>) => {
    const input = event.currentTarget
    const caretIsAtEnd = input.selectionStart === input.value.length && input.selectionEnd === input.value.length
    const acceptsCompletion = event.key === "Tab" || event.key === "ArrowRight"
    if (!inlineCompletion || !acceptsCompletion || !caretIsAtEnd) return

    event.preventDefault()
    const nextQuery = `${state.query}${inlineCompletion}`
    state.setQuery(nextQuery)
    setCaretIsAtDraftEnd(true)
    focusInput(nextQuery.length)
  }, [focusInput, inlineCompletion, state])

  const handleSelectField = React.useCallback((field: GlobalAssetSearchField) => {
    const nextQuery = appendGlobalAssetSearchCondition(state.query, field)
    state.setQuery(nextQuery)
    openSyntaxGuidance()
    focusInput(nextQuery.length - 1)
  }, [focusInput, openSyntaxGuidance, state])

  const handleSelectExample = React.useCallback((example: string) => {
    state.setQuery(example)
    openSyntaxGuidance()
    focusInput()
  }, [focusInput, openSyntaxGuidance, state])

  return (
    <div className={cn("min-w-0", className)}>
      <Popover open={syntaxOpen} onOpenChange={setSyntaxOpen} modal={false}>
        <SearchAssetBarShell>
          <SearchAssetTypeSelector state={state} integrated />
          <form
            className="flex min-w-0 flex-1"
            onSubmit={(event) => {
              event.preventDefault()
              state.handleSearch()
            }}
          >
            <PopoverAnchor className="relative flex min-w-0 flex-1">
              <Input
                ref={inputRef}
                type="search"
                value={state.query}
                onChange={handleInputChange}
                onFocus={openSyntaxGuidance}
                onClick={openSyntaxGuidance}
                onBlur={scheduleCloseSyntaxGuidance}
                onSelect={handleInputSelection}
                onKeyDown={handleInputKeyDown}
                placeholder={state.t("searchPlaceholder")}
                aria-invalid={Boolean(state.queryError)}
                className="radius-none h-9 min-w-0 border-0 bg-transparent px-3 text-sm shadow-none focus-visible:border-transparent focus-visible:ring-0"
              />
              {inlineCompletion ? (
                <div aria-hidden="true" className="pointer-events-none absolute inset-0 flex items-center overflow-hidden px-3">
                  <span className="whitespace-pre text-sm">
                    <span className="invisible">{state.query}</span>
                    <span className="text-muted-foreground/40">{inlineCompletion}</span>
                  </span>
                </div>
              ) : null}
            </PopoverAnchor>
            <Button type="submit" size="default" className="radius-none shrink-0 border-y-0 border-r-0">
              <Search className="h-4 w-4" />
              <span className="hidden sm:inline">{state.t("searchAction")}</span>
            </Button>
          </form>
        </SearchAssetBarShell>
        <PopoverContent
          align="start"
          sideOffset={8}
          collisionPadding={16}
          initialFocus={false}
          finalFocus={false}
          className="w-[var(--anchor-width)] p-0"
        >
          <SearchSyntaxGuidance
            t={state.t}
            onSelectField={handleSelectField}
            onSelectExample={handleSelectExample}
          />
        </PopoverContent>
      </Popover>
      {state.queryError ? <p className={cn("mt-1", textRole.caption, "text-destructive")}>{state.queryError}</p> : null}
    </div>
  )
}

function SearchResultsToolbarRegion({ children }: { children: React.ReactNode }) {
  return (
    <div {...getLoadingStructureSlotAttributes("search-results-toolbar")} className="sticky top-0 z-10 border-b bg-background/95 px-4 py-3 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="flex flex-wrap items-center gap-3">{children}</div>
    </div>
  )
}

function SearchResultsBodyRegion({ children }: { children: React.ReactNode }) {
  return <div {...getLoadingStructureSlotAttributes("search-results-body")} className="flex min-h-0 flex-1 flex-col">{children}</div>
}

function SearchResultsPaginationRegion({ children }: { children: React.ReactNode }) {
  return <div {...getLoadingStructureSlotAttributes("search-results-pagination")} className="border-t px-4 py-3">{children}</div>
}

function SearchResultsLoadingState({
  showWebsiteExport,
  websiteLoadingRowCount,
}: {
  showWebsiteExport: boolean
  websiteLoadingRowCount: number
}) {
  const pagination = (
    <SearchResultsPaginationRegion>
      <CompactPaginationSkeleton mode="cursor" className="px-0" />
    </SearchResultsPaginationRegion>
  )

  return (
    <>
      <SearchResultsToolbarRegion>
        <SearchAssetBarShell className="w-full md:flex-1">
          <SelectShellSkeleton
            size="default"
            widthClassName="radius-none w-40 border-0 border-r bg-muted/30 px-4 text-sm shadow-none"
            valueWidthClassName="w-16"
          />
          <div className="relative flex min-w-0 flex-1 items-center">
            <Input
              type="search"
              disabled
              className="radius-none h-9 border-0 bg-transparent px-3 text-sm shadow-none disabled:cursor-default disabled:opacity-100 focus-visible:border-transparent focus-visible:ring-0"
            />
            <Skeleton className="pointer-events-none absolute left-3 h-4 w-32 rounded-full" />
          </div>
          <ActionSkeleton size="default" widthClassName="w-10" className="radius-none border-y-0 border-r-0" />
        </SearchAssetBarShell>
        {showWebsiteExport ? <ActionSkeleton size="default" widthClassName="w-20" className="shrink-0" /> : null}
      </SearchResultsToolbarRegion>

      <SearchResultsBodyRegion>
        <div className="flex-1 overflow-auto p-4">
          {showWebsiteExport ? <SearchWebsitesDataTableLoadingState pagination={pagination} rowCount={websiteLoadingRowCount} /> : (
            <div className="mx-auto max-w-4xl space-y-4">
              {Array.from({ length: 3 }).map((_, index) => <Skeleton key={index} className="h-56 w-full rounded-md" />)}
            </div>
          )}
        </div>
      </SearchResultsBodyRegion>

      {!showWebsiteExport ? pagination : null}
    </>
  )
}

export function SearchPageContent({ state }: { state: SearchPageState }) {
  const websiteResults = state.assetType === "website" && state.data ? state.data.results : EMPTY_WEBSITE_RESULTS
  const websiteResultModel = useSearchWebsitesResultModel(websiteResults)
  const showWebsiteExport = !state.error && state.assetType === "website" && Boolean(state.data?.results.length)

  return (
    <div className="flex w-full flex-1 flex-col">
      {state.searchState === "initial" ? (
        <SearchInitialPageShell animated>
          <SearchInitialHeading title={state.t("title")} hint={state.t("hint")} />
          <div className="w-full"><AssetSearchBar state={state} /></div>
          <SearchQuickTags onTagClick={state.handleQuickTagClick} />

          {state.recentSearches.length > 0 ? (
            <div className="mt-2 w-full max-w-xl animate-in fade-in delay-300 duration-300">
              <div className={cn("mb-2 flex items-center gap-2", textRole.metadataLabel)}>
                <History className="h-3.5 w-3.5" />
                <span>{state.t("recentSearches")}</span>
              </div>
              <div className="flex flex-wrap gap-2">
                {state.recentSearches.map((search) => (
                  <Badge key={search} variant="secondary" className="group gap-1 py-1 pr-1.5 pl-3 hover:bg-secondary/80">
                    <button type="button" onClick={() => state.handleRecentSearchClick(search)} className="max-w-52 truncate text-left font-mono text-xs">
                      {search}
                    </button>
                    <button type="button" onClick={(event) => state.handleRemoveRecentSearch(event, search)} className="ml-1 p-0.5 opacity-0 transition-opacity hover:bg-muted-foreground/20 group-hover:opacity-100" aria-label={state.t("removeRecentSearch")}>
                      <X className="h-3 w-3" />
                    </button>
                  </Badge>
                ))}
              </div>
            </div>
          ) : null}
        </SearchInitialPageShell>
      ) : null}

      {state.searchState === "results" || state.searchState === "searching" ? (
        <ContentHandoff
          owner="search-results-content"
          isLoading={state.searchState === "searching" || state.isLoading}
          skeleton={(
            <SearchResultsLoadingState
              showWebsiteExport={state.assetType === "website"}
              websiteLoadingRowCount={state.pageSize}
            />
          )}
          className="flex h-full flex-col"
          skeletonClassName="flex h-full flex-col"
          contentClassName="flex h-full flex-col"
        >
          <SearchResultsToolbarRegion>
            <AssetSearchBar state={state} className="w-full md:flex-1" />
            {showWebsiteExport ? <SearchWebsiteExportMenu model={websiteResultModel} /> : null}
            {state.isFetching ? <span className={cn("whitespace-nowrap", textRole.bodySubtle)}>{state.t("loading")}</span> : null}
          </SearchResultsToolbarRegion>

          <SearchResultsBodyRegion>
            {state.error ? (
              <AppErrorState
                error={normalizeError(state.error, { notFoundKind: "unexpected-error" })}
                onRetry={state.refetch}
                variant="section"
                className="flex-1 px-4"
              />
            ) : null}

            {!state.error && state.data?.results.length === 0 ? (
              <div className="flex flex-1 flex-col items-center justify-center p-4">
                <div className="text-center">
                  <Search className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
                  <h3 className={cn("mb-2", textRole.panelTitle)}>{state.t("noResults")}</h3>
                  <p className={textRole.bodySubtle}>{state.t("noResultsHint")}</p>
                </div>
              </div>
            ) : null}

            {!state.error && state.data && state.data.results.length > 0 ? (
              <div className="flex-1 overflow-auto p-4">
                {state.assetType === "website" ? (
                  <SearchWebsitesDataTable
                    model={websiteResultModel}
                    pagination={(
                      <SearchResultsPaginationRegion>
                        <SearchPagination
                          pageSize={state.pageSize}
                          canFirstPage={state.canFirstPage}
                          canPreviousPage={state.canPreviousPage}
                          canNextPage={state.canNextPage}
                          onFirstPage={state.handleFirstPage}
                          onPreviousPage={state.handlePreviousPage}
                          onNextPage={state.handleNextPage}
                          onPageSizeChange={state.handlePageSizeChange}
                        />
                      </SearchResultsPaginationRegion>
                    )}
                  />
                ) : <SearchResultsTable results={state.data.results} />}
              </div>
            ) : null}
          </SearchResultsBodyRegion>

          {!state.error && state.data && state.data.results.length > 0 && state.assetType !== "website" ? (
            <SearchResultsPaginationRegion>
              <SearchPagination
                pageSize={state.pageSize}
                canFirstPage={state.canFirstPage}
                canPreviousPage={state.canPreviousPage}
                canNextPage={state.canNextPage}
                onFirstPage={state.handleFirstPage}
                onPreviousPage={state.handlePreviousPage}
                onNextPage={state.handleNextPage}
                onPageSizeChange={state.handlePageSizeChange}
              />
            </SearchResultsPaginationRegion>
          ) : null}
        </ContentHandoff>
      ) : null}
    </div>
  )
}
