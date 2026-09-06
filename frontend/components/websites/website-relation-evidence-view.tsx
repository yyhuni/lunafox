"use client"

import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react"
import Link from "next/link"
import { useSearchParams } from "next/navigation"
import { useTranslations } from "next-intl"

import { ChevronRight, ExternalLink, IconFileExport, X, semanticIcons } from "@/components/icons"
import { buildExportOptions, SelectedRowActionBar } from "@/components/shared/data-table"
import { DataTableFacetPanel, type DataTableFacetPanelFacet } from "@/components/shared/data-table/faceted-filter"
import { ToolbarActionMenu } from "@/components/shared/data-table/menu-owners"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { CompactPaginationSkeleton } from "@/components/shared/loading/compact-pagination-skeleton"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context"
import { ResponseEvidencePanel, ResponseEvidencePanelLoadingState } from "@/components/shared/response-evidence"
import { SearchToolbarSkeleton } from "@/components/shared/loading/search-toolbar-skeleton"
import { SharedCompactPagination } from "@/components/shared/data-table/pagination"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import { useSimpleSearchState } from "@/components/shared/data-table/use-simple-search"
import { HttpStatusBadge } from "@/components/shared/status/http-status-badge"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { DropdownMenuItem } from "@/components/ui/dropdown-menu"
import { Skeleton } from "@/components/ui/skeleton"
import { useScreenshotImageUrlResolver } from "@/hooks/use-screenshots"
import { textRole } from "@/lib/typography"
import { normalizeError } from "@/lib/errors/normalize-error"
import { cn } from "@/lib/utils"
import { WebSitesManagementOverlays } from "./websites-view-sections"
import { useWebSitesViewState } from "./websites-view-state"

import type { WebSite, WebsiteFilterOptionField } from "@/types/website.types"

const WebsiteIcon = semanticIcons.concept.website
const ScreenshotIcon = semanticIcons.concept.screenshot
const FingerprintIcon = semanticIcons.concept.fingerprint
const AddIcon = semanticIcons.action.add
const DeleteIcon = semanticIcons.action.delete
const WEBSITE_EVIDENCE_RESTORE_QUERY = "restoreWebsiteState"
const WEBSITE_EVIDENCE_STATE_STORAGE_PREFIX = "lunafox:website-evidence-state:"
const WEBSITE_FILTER_FIELDS: WebsiteFilterOptionField[] = ["statusCode", "tech", "webserver", "contentType", "vhost"]
const TECHNOLOGY_TAG_GAP_PX = 6
// Three allowed technology rows plus identity metadata are 3px taller than the
// screenshot baseline. Reserve that shared upper bound in both handoff states.
const RELATION_EVIDENCE_IDENTITY_MIN_HEIGHT_PX = 211
const relationEvidenceIdentityMinHeightStyle = { minHeight: RELATION_EVIDENCE_IDENTITY_MIN_HEIGHT_PX }

export type RelationEvidenceFilters = Record<WebsiteFilterOptionField, string[]>
// This is a local presentation model. The Website resource remains the only
// transport model for the evidence workspace.
export type WebsiteEvidence = Pick<
  WebSite,
  "id" | "url" | "host" | "title" | "webserver" | "contentType" | "statusCode" | "contentLength" | "tech" | "vhost" | "responseBody" | "createdAt" | "screenshot"
> & {
  responseHeaders: string
}

export type RelationScreenshotPreview = {
  imageUrl: string
  statusCode: number | null
  url: string
}

type WebsiteEvidenceRestoreState = {
  query: ReturnType<typeof useWebSitesViewState>["query"]
  pageTokens: ReturnType<typeof useWebSitesViewState>["pageTokens"]
}

function websiteEvidenceStateStorageKey(targetId: number) {
  return `${WEBSITE_EVIDENCE_STATE_STORAGE_PREFIX}${targetId}`
}

function restoreWebsiteEvidenceState(targetId: number): WebsiteEvidenceRestoreState | undefined {
  if (typeof window === "undefined") return undefined

  try {
    const stored = window.sessionStorage.getItem(websiteEvidenceStateStorageKey(targetId))
    if (!stored) return undefined
    const parsed = JSON.parse(stored) as WebsiteEvidenceRestoreState
    if (!parsed.query || !parsed.pageTokens || typeof parsed.pageTokens !== "object") return undefined
    return parsed
  } catch {
    return undefined
  }
}

export function filterRelationEvidence(
  items: WebsiteEvidence[],
  search: string,
  filters: RelationEvidenceFilters
) {
  const normalizedSearch = search.trim().toLocaleLowerCase()

  return items.filter((item) => {
    const matchesSearch = !normalizedSearch || [item.url, item.host, item.title]
      .some((value) => value.toLocaleLowerCase().includes(normalizedSearch))
    if (!matchesSearch) return false

    if (filters.statusCode.length > 0 && !filters.statusCode.includes(String(item.statusCode))) return false
    if (filters.tech.length > 0 && !item.tech.some((technology) => filters.tech.includes(technology))) return false
    if (filters.webserver.length > 0 && !filters.webserver.includes(item.webserver)) return false
    if (filters.contentType.length > 0 && !filters.contentType.includes(item.contentType)) return false
    if (filters.vhost.length > 0 && !filters.vhost.includes(String(item.vhost))) return false

    return true
  })
}

export function toWebsiteEvidence(website: WebSite): WebsiteEvidence {
  return {
    id: website.id,
    url: website.url,
    host: website.host,
    title: website.title,
    webserver: website.webserver,
    contentType: website.contentType,
    statusCode: website.statusCode,
    contentLength: website.contentLength,
    tech: website.tech,
    vhost: website.vhost,
    responseHeaders: website.responseHeaders ?? "",
    responseBody: website.responseBody,
    createdAt: website.createdAt,
    screenshot: website.screenshot,
  }
}

export function TargetWebsiteEvidenceView({ targetId }: { targetId: number }) {
  const searchParams = useSearchParams()
  const restoresWebsiteState = searchParams.get(WEBSITE_EVIDENCE_RESTORE_QUERY) === "1"
  const [restoredState] = useState(() => restoresWebsiteState ? restoreWebsiteEvidenceState(targetId) : undefined)
  const state = useWebSitesViewState({ targetId, initialTargetState: restoredState })
  const [screenshotPreview, setScreenshotPreview] = useState<RelationScreenshotPreview | null>(null)
  const tActions = useTranslations("common.actions")
  const tExport = useTranslations("common.export")
  const tPagination = useTranslations("common.pagination")
  const tDataTable = useTranslations("dataTable")
  const tColumns = useTranslations("columns")
  const tStatus = useTranslations("common.status")
  const tRelation = useTranslations("pages.targetDetail.relations")

  useEffect(() => {
    if (typeof window === "undefined") return
    window.sessionStorage.setItem(websiteEvidenceStateStorageKey(targetId), JSON.stringify({ query: state.query, pageTokens: state.pageTokens }))
  }, [state.pageTokens, state.query, targetId])

  const {
    value: searchValue,
    handleSearchInputChange,
    commitNow: commitSearch,
  } = useSimpleSearchState({
    searchValue: state.filterQuery,
    onSearch: state.commitFilterSearch,
  })
  const isInitialLoading = state.isLoading && !state.data
  const detailShellReady = useDetailShellReadySignal(!isInitialLoading)
  const rows = useMemo(
    () => state.websites.map(toWebsiteEvidence),
    [state.websites]
  )
  const selectedIds = useMemo(() => new Set(state.selectedWebSites.map((website) => website.id)), [state.selectedWebSites])
  const activeFilters: RelationEvidenceFilters = {
    statusCode: state.statusCodeFilter,
    tech: state.techFilter,
    webserver: state.webserverFilter,
    contentType: state.contentTypeFilter,
    vhost: state.vhostFilter,
  }
  const filterOptions = {
    statusCode: state.statusCodeOptions,
    tech: state.techOptions,
    webserver: state.webserverOptions,
    contentType: state.contentTypeOptions,
    vhost: state.vhostOptions,
  }
  const filterChangeHandlers: Record<WebsiteFilterOptionField, (values: string[]) => void> = {
    statusCode: state.handleStatusCodeFilterChange,
    tech: state.handleTechFilterChange,
    webserver: state.handleWebserverFilterChange,
    contentType: state.handleContentTypeFilterChange,
    vhost: state.handleVhostFilterChange,
  }
  const activeFilterCount = WEBSITE_FILTER_FIELDS.reduce((count, field) => count + activeFilters[field].length, 0)
  const facetLabels: Record<WebsiteFilterOptionField, string> = {
    statusCode: tColumns("website.statusCode"),
    tech: tColumns("endpoint.technologies"),
    webserver: tColumns("endpoint.webServer"),
    contentType: tColumns("endpoint.contentType"),
    vhost: tColumns("endpoint.vhost"),
  }
  const facetItems: DataTableFacetPanelFacet[] = WEBSITE_FILTER_FIELDS.map((field) => ({
    id: field,
    label: facetLabels[field],
    values: activeFilters[field],
    onValuesChange: filterChangeHandlers[field],
    options: filterOptions[field],
    emptyLabel: tStatus("noData"),
    clearLabel: tDataTable("clearFilter"),
  }))
  const exportOptions = buildExportOptions(tExport, {
    onExportAll: state.handleExportAll,
    onExportSelected: state.handleExportSelected,
  })
  const cursorPaginationSummary = state.cursorPaginationSummary
  const selectedCount = state.selectedWebSites.length
  const allCurrentRowsSelected = state.websites.length > 0 && state.websites.every((website) => selectedIds.has(website.id))

  const handleRowSelectionChange = (website: WebSite, checked: boolean) => {
    state.handleSelectionChange(checked
      ? [...state.selectedWebSites.filter((item) => item.id !== website.id), website]
      : state.selectedWebSites.filter((item) => item.id !== website.id))
  }

  const handleCurrentPageSelectionChange = (checked: boolean) => {
    const currentIds = new Set(state.websites.map((website) => website.id))
    const retained = state.selectedWebSites.filter((website) => !currentIds.has(website.id))
    state.handleSelectionChange(checked ? [...retained, ...state.websites] : retained)
  }

  if (detailShellReady?.deferInitialSkeleton && isInitialLoading) return null
  if (state.error) return <AppErrorState error={normalizeError(state.error)} onRetry={() => state.refetch()} />

  const toolbar = <RelationEvidenceToolbar
    searchValue={searchValue}
    onSearchChange={handleSearchInputChange}
    onSearchSubmit={commitSearch}
    isSearching={state.isSearching}
    filterCount={activeFilterCount}
    facets={facetItems}
    searchPlaceholder={tActions("searchURL")}
    filterLabel={tDataTable("filter")}
    exportOptions={exportOptions}
    selectedCount={selectedCount}
    onBulkAdd={() => state.setBulkAddDialogOpen(true)}
  />
  const paginationFooter = <SharedCompactPagination
    mode="cursor"
    pageSize={state.pagination.pageSize}
    canFirstPage={state.paginationNavigation.canFirstPage}
    canPreviousPage={state.paginationNavigation.canPreviousPage}
    canNextPage={state.paginationNavigation.canNextPage}
    onFirstPage={() => state.handlePaginationChange({ pageIndex: 0, pageSize: state.pagination.pageSize })}
    onPreviousPage={() => state.handlePaginationChange({ pageIndex: state.pagination.pageIndex - 1, pageSize: state.pagination.pageSize })}
    onNextPage={() => state.handlePaginationChange({ pageIndex: state.pagination.pageIndex + 1, pageSize: state.pagination.pageSize })}
    summary={`${tDataTable("selected", { count: selectedCount })} / ${tPagination("total", { count: cursorPaginationSummary.total })}`}
    onPageSizeChange={(pageSize) => state.handlePaginationChange({ pageIndex: 0, pageSize })}
  />
  const hasAppliedControls = Boolean(state.filterQuery) || activeFilterCount > 0
  const resolvedContent = rows.length === 0 ? (
    <>
      <div className="space-y-4" data-loading-slot="website-relation-evidence-list">
        {toolbar}
        <div className="radius-surface flex min-h-64 flex-col items-center justify-center gap-2 border border-dashed border-border bg-card px-6 text-center">
          <WebsiteIcon className="size-6 text-muted-foreground" aria-hidden="true" />
          <p className={textRole.bodyStrong}>{hasAppliedControls ? tRelation("noResults") : tRelation("emptyTitle")}</p>
          <p className={textRole.bodySubtle}>{hasAppliedControls ? tRelation("noResultsHint") : tRelation("emptyDescription")}</p>
        </div>
        <div data-slot="data-table-pagination-surface">{paginationFooter}</div>
      </div>
      <WebSitesManagementOverlays state={state} />
    </>
  ) : (
    <>
      <RelationEvidenceListFrame
        toolbar={toolbar}
        selectionControl={<Checkbox checked={allCurrentRowsSelected} onCheckedChange={(checked) => handleCurrentPageSelectionChange(checked === true)} aria-label={allCurrentRowsSelected ? tActions("deselectAll") : tActions("selectAll")} />}
        footer={paginationFooter}
      >
        {state.websites.map((website) => {
          const item = toWebsiteEvidence(website)
          return <WebsiteRelationEvidenceRow key={website.resourceName ?? website.id} targetId={targetId} item={item} website={website} selected={selectedIds.has(website.id)} onSelectedChange={(checked) => handleRowSelectionChange(website, checked)} onPreviewScreenshot={setScreenshotPreview} />
        })}
      </RelationEvidenceListFrame>
      <RelationScreenshotPreviewDialog preview={screenshotPreview} onOpenChange={(open) => { if (!open) setScreenshotPreview(null) }} />
      <SelectedRowActionBar
        selectedCount={selectedCount}
        ariaLabel={tDataTable("selected", { count: selectedCount })}
        countLabel={tDataTable("selected", { count: selectedCount })}
        actions={[{
          key: "delete",
          label: tActions("delete"),
          icon: DeleteIcon,
          tone: "destructive",
          group: "danger",
          onClick: () => state.setDeleteDialogOpen(true),
        }]}
        onClearSelection={() => state.handleSelectionChange([])}
        clearSelectionLabel={tDataTable("deselectAll")}
      />
      <WebSitesManagementOverlays state={state} />
    </>
  )

  return (
    <ContentHandoff
      owner="target-website-evidence-view-content"
      isLoading={isInitialLoading}
      skeleton={<WebsiteRelationEvidenceLoadingState />}
    >
      <div className="min-w-0" data-loading-slot="website-relation-evidence-surface">
        {resolvedContent}
      </div>
    </ContentHandoff>
  )
}

export function RelationEvidenceListFrame({
  toolbar,
  footer,
  selectionControl,
  children,
}: {
  toolbar: ReactNode
  footer: ReactNode
  selectionControl?: ReactNode
  children: ReactNode
}) {
  const t = useTranslations("pages.targetDetail.relations")

  return (
    <div className="space-y-4" data-loading-slot="website-relation-evidence-list">
      {toolbar}
      <div className="radius-surface overflow-hidden border border-border bg-card">
        <div className="hidden h-10 min-w-0 grid-cols-12 gap-4 border-b border-border bg-secondary px-4 xl:grid">
          <div className="col-span-4 flex items-center gap-3">{selectionControl}<span className={textRole.metadataLabel}>{t("columns.website")}</span></div>
          <div className="col-span-5 flex items-center"><span className={textRole.metadataLabel}>{t("columns.response")}</span></div>
          <div className="col-span-3 flex items-center"><span className={textRole.metadataLabel}>{t("columns.screenshot")}</span></div>
        </div>
        <div className="divide-y divide-border">{children}</div>
      </div>
      <div data-slot="data-table-pagination-surface">{footer}</div>
    </div>
  )
}

export function RelationEvidenceToolbar({
  searchValue,
  onSearchChange,
  onSearchSubmit,
  filterCount,
  facets,
  searchPlaceholder,
  filterLabel,
  isSearching = false,
  exportOptions = [],
  selectedCount = 0,
  onBulkAdd,
  showFacets = true,
  inputWidthMode = "fixed",
}: {
  searchValue: string
  onSearchChange: (value: string) => void
  onSearchSubmit: () => void
  filterCount: number
  facets: DataTableFacetPanelFacet[]
  searchPlaceholder: string
  filterLabel: string
  isSearching?: boolean
  exportOptions?: ReturnType<typeof buildExportOptions>
  selectedCount?: number
  onBulkAdd?: () => void
  showFacets?: boolean
  inputWidthMode?: "fixed" | "fill"
}) {
  const tActions = useTranslations("common.actions")

  return (
    <div className="flex w-full min-w-0 flex-wrap items-center justify-between gap-2">
      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
        <SimpleSearchToolbar value={searchValue} onChange={onSearchChange} onSubmit={onSearchSubmit} loading={isSearching} placeholder={searchPlaceholder} inputWidthMode={inputWidthMode} toolbarDensity="compact" />
        {showFacets ? <DataTableFacetPanel title={filterLabel} activeCount={filterCount} facets={facets} /> : null}
      </div>
      <div className="flex items-center gap-2">
        {exportOptions.length > 0 ? (
          <ToolbarActionMenu label={tActions("export")} icon={<IconFileExport aria-hidden="true" className="size-4" />} buttonSize="sm">
            {exportOptions.map((option) => {
              const disabled = typeof option.disabled === "function" ? option.disabled(selectedCount) : option.disabled
              return <DropdownMenuItem key={option.key} onClick={option.onClick} disabled={disabled}>{option.label}</DropdownMenuItem>
            })}
          </ToolbarActionMenu>
        ) : null}
        {onBulkAdd ? (
          <Button type="button" variant="outline" size="sm" onClick={onBulkAdd}>
            <AddIcon aria-hidden="true" />
            {tActions("add")}
          </Button>
        ) : null}
      </div>
    </div>
  )
}

function RelationEvidenceToolbarLoadingState() {
  return (
    <SearchToolbarSkeleton
      className="sm:w-full"
      after={(
        <>
          <ActionSkeleton size="sm" widthClassName="w-20" />
          <span className="ml-auto flex items-center gap-2">
            <ActionSkeleton size="sm" widthClassName="w-20" />
            <ActionSkeleton size="sm" widthClassName="w-20" />
          </span>
        </>
      )}
      toolbarDensity="compact"
    />
  )
}

export function WebsiteRelationEvidenceRow({
  targetId,
  item,
  website,
  selected,
  onSelectedChange,
  onPreviewScreenshot,
}: {
  targetId?: number
  item: WebsiteEvidence
  website: WebSite
  selected: boolean
  onSelectedChange: (checked: boolean) => void
  onPreviewScreenshot: (preview: RelationScreenshotPreview) => void
}) {
  const detailHref = `/targets/${targetId}/websites/${item.id}/?returnTo=website-list`
  const targetDetailHref = targetId === undefined ? undefined : detailHref

  return (
    <article className="grid min-w-0 items-stretch gap-4 p-4 xl:grid-cols-12">
      <div className="h-full min-w-0 xl:col-span-4" style={relationEvidenceIdentityMinHeightStyle}>
        <WebsiteIdentity item={item} website={website} detailHref={targetDetailHref} selected={selected} onSelectedChange={onSelectedChange} />
      </div>
      <div className="h-full min-w-0 xl:col-span-5"><WebsiteResponseEvidence item={item} /></div>
      <div className="h-full min-w-0 xl:col-span-3"><WebsiteScreenshot item={item} onPreview={onPreviewScreenshot} /></div>
    </article>
  )
}

export function WebsiteScreenshot({ item, onPreview }: { item: WebsiteEvidence; onPreview: (preview: RelationScreenshotPreview) => void }) {
  const t = useTranslations("pages.targetDetail.relations")
  const resolveScreenshotImageUrl = useScreenshotImageUrlResolver()

  if (!item.screenshot) {
    return (
      <div className="radius-surface flex h-full min-h-52 min-w-0 flex-col items-center justify-center gap-2 border border-dashed border-border bg-muted/20 px-4 text-center">
        <ScreenshotIcon aria-hidden="true" className="size-5 text-muted-foreground" />
        <p className={textRole.bodyStrong}>{t("noScreenshot")}</p>
        <p className={textRole.caption}>{t("noScreenshotHint")}</p>
      </div>
    )
  }
  const screenshot = item.screenshot
  const imageUrl = resolveScreenshotImageUrl(screenshot.id)
  return (
    <figure className="radius-surface group flex h-full min-w-0 flex-col overflow-hidden border border-border bg-muted/20">
      <Button type="button" variant="ghost" size="content" onClick={() => onPreview({ imageUrl, statusCode: screenshot.statusCode, url: item.url })} className="radius-none min-h-44 flex-1 overflow-hidden bg-muted p-0 hover:bg-muted" aria-label={t("viewScreenshot", { url: item.url })}>
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img className="group-hover:scale-105 h-full w-full object-cover object-top transition-transform" src={imageUrl} alt={t("screenshotAlt", { url: item.url })} />
      </Button>
      <figcaption className={cn("flex items-center gap-1.5 border-t border-border px-2.5 py-2", textRole.caption)}>
        <ScreenshotIcon aria-hidden="true" className="size-3.5 shrink-0 text-muted-foreground" />
        <span className="truncate">{t("latestScreenshot")}</span>
      </figcaption>
    </figure>
  )
}

function WebsiteIdentity({
  item,
  website,
  detailHref,
  selected,
  onSelectedChange,
}: {
  item: WebsiteEvidence
  website: WebSite
  detailHref?: string
  selected: boolean
  onSelectedChange: (checked: boolean) => void
}) {
  const t = useTranslations("pages.targetDetail.relations")
  const tActions = useTranslations("common.actions")

  return (
    <section className="flex h-full min-w-0 flex-col">
      <div className="space-y-3">
        <div className="flex min-w-0 items-center gap-2">
          <Checkbox checked={selected} onCheckedChange={(checked) => onSelectedChange(checked === true)} aria-label={`${tActions("selectRow")}: ${website.url}`} />
          <div className="flex min-w-0 items-center gap-2">
            <HttpStatusBadge statusCode={item.statusCode ?? undefined} />
            <a href={item.url} target="_blank" rel="noreferrer" className={cn("flex min-w-0 items-center gap-1.5 text-primary underline-offset-2 hover:underline", textRole.metadataValueStrong)}>
              <WebsiteIcon aria-hidden="true" className="size-3.5 shrink-0" />
              <span className="truncate">{item.url}</span>
              <ExternalLink aria-hidden="true" className="size-3 shrink-0 opacity-70" />
            </a>
          </div>
          {detailHref ? (
            <Button variant="ghost" size="sm" render={<Link href={detailHref} />} className="ml-auto shrink-0">
              {t("viewDetails")}
              <ChevronRight aria-hidden="true" data-icon="inline-end" />
            </Button>
          ) : null}
        </div>
        <div className="min-w-0">
          <p className={cn("truncate", textRole.bodyStrong)} title={item.title}>{item.title || t("unknownTitle")}</p>
          <p className={cn("mt-1 truncate", textRole.caption)}>{item.host} · {item.webserver || t("unknownServer")}</p>
        </div>
        <div className="space-y-2">
          <div className="flex items-center gap-1.5"><FingerprintIcon aria-hidden="true" className="size-3.5 text-muted-foreground" /><span className={textRole.monoLabel}>{t("technology")}</span></div>
          <WebsiteTechnologySummary technologies={item.tech} maxRows={3} />
        </div>
      </div>
    </section>
  )
}

export function WebsiteResponseEvidence({ item }: { item: WebsiteEvidence }) {
  const t = useTranslations("pages.targetDetail.relations")
  return (
    <ResponseEvidencePanel
      responseBody={item.responseBody}
      responseHeaders={item.responseHeaders}
      labels={{
        body: t("responseBody"),
        headers: t("responseHeaders"),
        location: t("location"),
        emptyLabel: t("noResponseContent"),
      }}
      showMetadata
      metadata={{
        contentType: item.contentType,
        contentLength: item.contentLength,
        labels: {
          contentType: t("contentType"),
          responseSize: t("responseSize"),
        },
      }}
    />
  )
}

export function WebsiteTechnologySummary({
  technologies,
  maxRows = 1,
}: {
  technologies: string[]
  maxRows?: number
}) {
  const t = useTranslations("pages.targetDetail.relations")
  const [dialogOpen, setDialogOpen] = useState(false)
  const [visibleCount, setVisibleCount] = useState(technologies.length)
  const summaryRef = useRef<HTMLDivElement>(null)
  const measurementRef = useRef<HTMLDivElement>(null)

  const recalculateVisibleCount = useCallback(() => {
    const summary = summaryRef.current
    const measurements = measurementRef.current
    if (!summary || !measurements) return

    const tagWidths = Array.from(measurements.querySelectorAll<HTMLElement>("[data-technology-measurement=tag]"))
      .map((element) => element.getBoundingClientRect().width)
    const remainingWidths = Array.from(measurements.querySelectorAll<HTMLElement>("[data-technology-measurement=remaining]"))
      .map((element) => element.getBoundingClientRect().width)

    setVisibleCount(resolveVisibleTechnologyCount(summary.clientWidth, tagWidths, remainingWidths, maxRows))
  }, [maxRows])

  useEffect(() => {
    recalculateVisibleCount()
    if (!summaryRef.current || typeof ResizeObserver === "undefined") return

    const observer = new ResizeObserver(recalculateVisibleCount)
    observer.observe(summaryRef.current)
    return () => observer.disconnect()
  }, [maxRows, recalculateVisibleCount, technologies])

  if (technologies.length === 0) return <p className={textRole.caption}>{t("noTechnology")}</p>

  const visibleTechnologies = technologies.slice(0, visibleCount)
  const remainingCount = technologies.length - visibleTechnologies.length
  const remainingLabel = t("viewRemainingTechnologies", { count: remainingCount })

  return (
    <>
      <div ref={summaryRef} className="flex min-w-0 flex-wrap gap-1.5">
        {visibleTechnologies.map((technology, index) => <Badge key={`${technology}-${index}`} variant="outline">{technology}</Badge>)}
        {remainingCount > 0 ? (
          <Badge
            variant="outline"
            className="cursor-pointer hover:bg-accent"
            title={remainingLabel}
            render={<button type="button" onClick={() => setDialogOpen(true)} aria-label={remainingLabel} />}
          >
            +{remainingCount}
          </Badge>
        ) : null}
      </div>
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("technology")}</DialogTitle>
          </DialogHeader>
          <div className="flex flex-wrap gap-2">
            {technologies.map((technology, index) => <Badge key={`${technology}-${index}`} variant="outline">{technology}</Badge>)}
          </div>
        </DialogContent>
      </Dialog>
      <div ref={measurementRef} className="pointer-events-none absolute -z-10 flex gap-1.5 opacity-0" aria-hidden="true">
        {technologies.map((technology, index) => <Badge key={`${technology}-${index}`} data-technology-measurement="tag" variant="outline">{technology}</Badge>)}
        {technologies.map((_, index) => {
          const count = index + 1
          return <Badge key={count} data-technology-measurement="remaining" variant="outline">+{count}</Badge>
        })}
      </div>
    </>
  )
}

export function resolveVisibleTechnologyCount(
  containerWidth: number,
  tagWidths: number[],
  remainingWidths: number[],
  maxRows = 1
) {
  for (let count = tagWidths.length; count >= 0; count -= 1) {
    const remainingWidth = count === tagWidths.length ? undefined : remainingWidths[tagWidths.length - count - 1]
    if (fitsTechnologyRows(containerWidth, tagWidths.slice(0, count), remainingWidth, maxRows)) return count
  }

  return 0
}

function fitsTechnologyRows(
  containerWidth: number,
  tagWidths: number[],
  remainingWidth: number | undefined,
  maxRows: number
) {
  let rows = 1
  let rowWidth = 0

  for (const width of [...tagWidths, remainingWidth].filter((width): width is number => width !== undefined)) {
    if (rowWidth === 0) {
      rowWidth = width
      continue
    }

    if (rowWidth + TECHNOLOGY_TAG_GAP_PX + width <= containerWidth) {
      rowWidth += TECHNOLOGY_TAG_GAP_PX + width
      continue
    }

    rows += 1
    rowWidth = width
  }

  return rows <= maxRows
}

export function RelationScreenshotPreviewDialog({
  preview,
  onOpenChange,
}: {
  preview: RelationScreenshotPreview | null
  onOpenChange: (open: boolean) => void
}) {
  const t = useTranslations("pages.targetDetail.relations")

  return (
    <Dialog open={Boolean(preview)} onOpenChange={onOpenChange}>
      <DialogContent className="bg-[var(--media-overlay-background)] border-none max-h-[90vh] max-w-[90vw] p-0" showCloseButton={preview === null}>
        <DialogTitle className="sr-only">{t("screenshotPreviewTitle")}</DialogTitle>
        {preview ? (
          <div className="relative flex h-full w-full items-center justify-center">
            <Button
              type="button"
              variant="ghost"
              size="icon-lg"
              onClick={() => onOpenChange(false)}
              className="absolute right-4 top-4 z-50 rounded-full bg-background/10 hover:bg-background/20"
              aria-label={t("closeScreenshotPreview")}
            >
              <X className="size-6 text-background" />
            </Button>
            <div className="flex flex-col items-center gap-4 p-8">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={preview.imageUrl}
                alt={t("screenshotPreviewAlt", { url: preview.url })}
                className="max-h-[70vh] max-w-full object-contain"
                width={1920}
                height={1080}
              />
              <div className="flex items-center justify-center gap-2 text-[var(--media-overlay-foreground)]">
                {preview.statusCode ? <HttpStatusBadge statusCode={preview.statusCode} size="overlay" /> : null}
                <a href={preview.url} target="_blank" rel="noopener noreferrer" className="flex min-w-0 items-center gap-1 text-info hover:underline">
                  <span className="truncate">{preview.url}</span>
                  <ExternalLink aria-hidden="true" className="size-3 shrink-0" />
                </a>
              </div>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}

export function WebsiteRelationEvidenceLoadingState({
  toolbar = <RelationEvidenceToolbarLoadingState />,
  footer = <CompactPaginationSkeleton />,
  rowCount = 3,
}: {
  toolbar?: ReactNode
  footer?: ReactNode
  rowCount?: number
} = {}) {
  return (
    <div className="min-w-0" data-loading-slot="website-relation-evidence-surface">
      <RelationEvidenceListFrame
        toolbar={toolbar}
        selectionControl={<Skeleton className="size-4" />}
        footer={footer}
      >
        {Array.from({ length: rowCount }, (_, index) => <WebsiteRelationEvidenceLoadingRow key={index} />)}
      </RelationEvidenceListFrame>
    </div>
  )
}

function WebsiteRelationEvidenceLoadingRow() {
  return (
    <div className="grid min-w-0 items-stretch gap-4 p-4 xl:grid-cols-12" aria-hidden="true">
      <div className="flex h-52 flex-col space-y-3 xl:col-span-4" style={relationEvidenceIdentityMinHeightStyle}>
        <div className="flex items-center gap-3"><Skeleton className="size-4 shrink-0" /><Skeleton className="h-5 w-4/5" /></div>
        <Skeleton className="h-4 w-2/3" />
        <Skeleton className="h-12 w-full" />
      </div>
      <div className="h-full min-w-0 xl:col-span-5">
        <ResponseEvidencePanelLoadingState showMetadata />
      </div>
      <Skeleton className="h-52 min-w-0 xl:col-span-3" />
    </div>
  )
}
