"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { AlertDialog, AlertDialogClose, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { WordlistCatalogCard, WordlistCatalogCardLoadingState } from "@/components/tools/wordlist-catalog-card"
import { WordlistDetailDrawer } from "@/components/tools/wordlist-detail-drawer"
import { WordlistUploadDialog } from "@/components/tools/wordlist-upload-dialog"
import {
  WORDLISTS_CATALOG_CONTROLS_CLASS,
  WORDLISTS_CATALOG_FOOTER_CLASS,
  WORDLISTS_CATALOG_GRID_CLASS,
  WORDLISTS_WORKSPACE_HANDOFF_CLASS,
  WORDLISTS_WORKSPACE_SURFACE_CLASS,
} from "@/components/tools/wordlists-page-layout"
import { DataTableFacetedFilter, DataTableFacetedFilterGroup, type DataTableFacetedFilterOption } from "@/components/shared/data-table/faceted-filter"
import {
  applyBusinessListControlChange,
  compileBusinessListFilter,
  compileBusinessListOrderBy,
  createBusinessListQuery,
  getCurrentCursorNextPageToken,
  getCursorPaginationNavigation,
  getCursorPageTransition,
  setBusinessListPage,
  type BusinessListFilterCompilerConfig,
  type BusinessListSortableFieldConfig,
  type BusinessListQuery,
  type BusinessListSorting,
} from "@/components/shared/data-table/business-list-query"
import { SharedCompactPagination } from "@/components/shared/data-table/pagination"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { CompactPaginationSkeleton } from "@/components/shared/loading/compact-pagination-skeleton"
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { SearchToolbarSkeleton } from "@/components/shared/loading/search-toolbar-skeleton"
import { SearchInput } from "@/components/shared/search-input"
import { useDeleteWordlist, useWordlistTags, useWordlists } from "@/hooks/use-wordlists"
import { textRole } from "@/lib/typography"
import type { Wordlist } from "@/types/wordlist.types"
import { useLocale, useTranslations } from "next-intl"

const EMPTY_WORDLISTS: Wordlist[] = []
const WORDLIST_FILTER_FIELDS: BusinessListFilterCompilerConfig = {
  search: { field: "fileName", operator: "=" },
  facets: { tags: { field: "tags", operator: "==" } },
}
const WORDLIST_SORTABLE_FIELDS: Record<string, BusinessListSortableFieldConfig> = {
  fileName: { orderBy: "fileName", firstDirection: "asc" },
  lineCount: { orderBy: "lineCount", firstDirection: "desc" },
  fileSize: { orderBy: "fileSize", firstDirection: "desc" },
  updatedAt: { orderBy: "updatedAt", firstDirection: "desc" },
}
const WORDLIST_DEFAULT_SORTING: BusinessListSorting = { field: "updatedAt", direction: "desc" }
const WORDLIST_SEARCH_DEBOUNCE_MS = 300
export const MAX_ONLINE_EDIT_BYTES = 5 * 1024 * 1024

export function canEditContent(wordlist: Wordlist) {
  return wordlist.fileSize !== undefined && wordlist.fileSize <= MAX_ONLINE_EDIT_BYTES
}

function WordlistsPageLoadingState() {
  return (
    <div data-slot="wordlists-page-loading-state" className={WORDLISTS_WORKSPACE_SURFACE_CLASS}>
      <div {...getLoadingStructureSlotAttributes("wordlists-controls")} className={WORDLISTS_CATALOG_CONTROLS_CLASS}>
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
          <div className="min-w-48 flex-1 sm:w-72 sm:flex-none lg:w-80">
            <SearchToolbarSkeleton className="w-full sm:w-full" groupClassName="w-full" placeholderWidthClassName="w-28" />
          </div>
          <ActionSkeleton size="sm" widthClassName="w-20" />
        </div>
        <ActionSkeleton size="sm" widthClassName="w-24" emphasis="primary" />
      </div>
      <div {...getLoadingStructureSlotAttributes("wordlists-list")} className="min-h-0 flex-1 overflow-y-auto">
        <div className={WORDLISTS_CATALOG_GRID_CLASS}>
          {Array.from({ length: 6 }, (_, index) => <WordlistCatalogCardLoadingState key={index} />)}
        </div>
      </div>
      <div className={WORDLISTS_CATALOG_FOOTER_CLASS}>
        <CompactPaginationSkeleton mode="cursor" showSummary buttonCount={3} summaryWidthClassName="w-20" rowsPerPageLabelWidthClassName="w-20" className="px-0" />
      </div>
    </div>
  )
}

export default function WordlistsPage() {
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [query, setQuery] = useState<BusinessListQuery>(() => createBusinessListQuery({ pageSize: 20, sorting: WORDLIST_DEFAULT_SORTING }))
  const [pageTokens, setPageTokens] = useState<Record<number, string | undefined>>({ 1: undefined })
  const [searchInput, setSearchInput] = useState("")
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [wordlistToDelete, setWordlistToDelete] = useState<Wordlist | null>(null)
  const tCommon = useTranslations("common")
  const tConfirm = useTranslations("common.confirm")
  const t = useTranslations("pages.wordlists")
  const locale = useLocale()
  const wordlistSearchPlaceholder = locale.startsWith("zh")
    ? `${tCommon("actions.search")}${t("name")}...`
    : `${tCommon("actions.search")} ${t("name").toLowerCase()}...`
  const selectedTags = query.filters.tags ?? []
  const searchQuery = query.search ?? ""
  const page = query.pageIndex ?? 1
  const pageSize = query.pageSize
  const compiledFilter = compileBusinessListFilter({ search: query.search, filters: query.filters }, WORDLIST_FILTER_FIELDS)
  const compiledOrderBy = compileBusinessListOrderBy(query.sorting, WORDLIST_SORTABLE_FIELDS)
  const { data, isLoading, isPlaceholderData } = useWordlists({
    pageSize,
    pageToken: query.pageToken,
    filter: compiledFilter,
    orderBy: compiledOrderBy,
  })
  const nextPageToken = getCurrentCursorNextPageToken(data?.nextPageToken, isPlaceholderData)
  const { data: tagData } = useWordlistTags({ pageSize: 100 })
  const deleteMutation = useDeleteWordlist()
  const wordlists = data?.results ?? EMPTY_WORDLISTS
  const totalSize = data?.totalSize ?? 0
  const selectedWordlist = useMemo(() => wordlists.find((wordlist) => wordlist.id === selectedId) ?? null, [wordlists, selectedId])
  const paginationNavigation = getCursorPaginationNavigation({ currentPage: page, pageTokens, nextPageToken })
  const tagFilterOptions = useMemo<Array<DataTableFacetedFilterOption<string>>>(() => (tagData?.results ?? []).map((tag) => ({
    value: tag.displayName,
    label: tag.displayName,
    count: tag.wordlistCount,
  })), [tagData?.results])

  useEffect(() => {
    if (nextPageToken) setPageTokens((tokens) => ({ ...tokens, [page + 1]: nextPageToken }))
  }, [nextPageToken, page])

  useEffect(() => {
    if (selectedId && !selectedWordlist) setSelectedId(null)
  }, [selectedId, selectedWordlist])

  const commitSearch = useCallback((search: string) => {
    const normalizedSearch = search.trim()
    if ((query.search ?? "") === normalizedSearch) return
    setPageTokens({ 1: undefined })
    setQuery((current) => applyBusinessListControlChange(current, { search: normalizedSearch || undefined }))
  }, [query.search])

  useEffect(() => {
    const timeout = window.setTimeout(() => commitSearch(searchInput), WORDLIST_SEARCH_DEBOUNCE_MS)
    return () => window.clearTimeout(timeout)
  }, [commitSearch, searchInput])

  const handleTagsChange = (tags: string[]) => {
    setPageTokens({ 1: undefined })
    setQuery((current) => applyBusinessListControlChange(current, { filters: { ...current.filters, tags } }))
  }

  const handlePageChange = (nextPage: number) => {
    const transition = getCursorPageTransition({ currentPage: page, pageTokens, nextPageToken, requestedPage: nextPage })
    if (!transition.reachable) return
    setQuery((current) => setBusinessListPage(current, { pageIndex: nextPage, pageToken: transition.pageToken }))
  }

  const handleFirstPage = () => {
    if (page === 1) return
    setPageTokens({ 1: undefined })
    setQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
  }

  const handlePageSizeChange = (nextPageSize: number) => {
    setPageTokens({ 1: undefined })
    setQuery((current) => applyBusinessListControlChange(current, { pageSize: nextPageSize }))
  }

  const handleDelete = (wordlist: Wordlist) => {
    setSelectedId(null)
    setWordlistToDelete(wordlist)
    setDeleteDialogOpen(true)
  }

  const confirmDelete = () => {
    if (!wordlistToDelete) return
    deleteMutation.mutate(wordlistToDelete.id, {
      onSuccess: () => {
        setDeleteDialogOpen(false)
        setWordlistToDelete(null)
      },
    })
  }

  return (
    <ContentHandoff
      owner="wordlists-page-content"
      layer="workspace"
      isLoading={isLoading}
      skeleton={<WordlistsPageLoadingState />}
      className={WORDLISTS_WORKSPACE_HANDOFF_CLASS}
      skeletonClassName={WORDLISTS_WORKSPACE_HANDOFF_CLASS}
      contentClassName={WORDLISTS_WORKSPACE_HANDOFF_CLASS}
    >
      <div className={WORDLISTS_WORKSPACE_SURFACE_CLASS}>
        <div {...getLoadingStructureSlotAttributes("wordlists-controls")} className={WORDLISTS_CATALOG_CONTROLS_CLASS}>
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
            <div className="min-w-48 flex-1 sm:w-72 sm:flex-none lg:w-80">
              <SearchInput
                name="wordlistSearch"
                placeholder={wordlistSearchPlaceholder}
                value={searchInput}
                onChange={(event) => setSearchInput(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Enter") {
                    event.preventDefault()
                    commitSearch(searchInput)
                  }
                }}
                toolbarDensity="compact"
              />
            </div>
            <DataTableFacetedFilterGroup hasSelectedValues={selectedTags.length > 0} onReset={() => handleTagsChange([])}>
              <DataTableFacetedFilter title={t("tags")} values={selectedTags} onValuesChange={handleTagsChange} options={tagFilterOptions} emptyLabel={t("tagEmpty")} clearLabel={t("clearFilter")} contentSize="wide" />
            </DataTableFacetedFilterGroup>
          </div>
          <WordlistUploadDialog triggerSize="sm" />
        </div>

        <div {...getLoadingStructureSlotAttributes("wordlists-list")} className="min-h-0 flex-1 overflow-y-auto">
          {wordlists.length === 0 ? (
            <div className="flex min-h-full items-center justify-center px-4 py-8 text-center">
              <p className={textRole.bodySubtle}>{searchQuery || selectedTags.length ? t("noMatch") : t("noData")}</p>
            </div>
          ) : (
            <div className={WORDLISTS_CATALOG_GRID_CLASS}>
              {wordlists.map((wordlist) => <WordlistCatalogCard key={wordlist.id} wordlist={wordlist} locale={locale} onSelect={(selected) => setSelectedId(selected.id)} />)}
            </div>
          )}
        </div>

        <div className={WORDLISTS_CATALOG_FOOTER_CLASS}>
          <SharedCompactPagination
            mode="cursor"
            pageSize={pageSize}
            canFirstPage={paginationNavigation.canFirstPage}
            canPreviousPage={paginationNavigation.canPreviousPage}
            canNextPage={paginationNavigation.canNextPage}
            onFirstPage={handleFirstPage}
            onPreviousPage={() => handlePageChange(page - 1)}
            onNextPage={() => handlePageChange(page + 1)}
            onPageSizeChange={handlePageSizeChange}
            pageSizeOptions={[20, 50, 100]}
            summary={t("tableSummary", { count: totalSize })}
            className="w-full px-0"
          />
        </div>

        <WordlistDetailDrawer
          wordlist={selectedWordlist}
          locale={locale}
          open={selectedWordlist !== null}
          onOpenChange={(open) => !open && setSelectedId(null)}
          onDelete={handleDelete}
          canEditContent={canEditContent}
        />

        <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{tConfirm("deleteTitle")}</AlertDialogTitle>
              <AlertDialogDescription>{tConfirm("deleteWordlistMessage", { name: wordlistToDelete?.fileName ?? "" })}</AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogClose variant="outline">{tCommon("actions.cancel")}</AlertDialogClose>
              <AlertDialogClose onClick={confirmDelete} className="bg-destructive text-destructive-foreground hover:bg-destructive/90" disabled={deleteMutation.isPending}>
                {deleteMutation.isPending ? tConfirm("deleting") : tCommon("actions.delete")}
              </AlertDialogClose>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </ContentHandoff>
  )
}
