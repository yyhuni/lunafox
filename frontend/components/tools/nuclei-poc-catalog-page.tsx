"use client"

import * as React from "react"
import { useLocale, useTranslations } from "next-intl"
import type { ColumnDef } from "@tanstack/react-table"

import { AlertTriangle, Ban, CheckCircle2, ChevronDown, Circle, Layers, RefreshCw, semanticIcons } from "@/components/icons"
import { BusinessListDataTable } from "@/components/shared/data-table/business-list-data-table"
import {
  getCurrentCursorNextPageToken,
  getCursorPageTransition,
  getCursorPaginationNavigation,
  type BusinessListQuery,
  createBusinessListQuery,
  applyBusinessListControlChange,
  setBusinessListPage,
} from "@/components/shared/data-table/business-list-query"
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import {
  DataTableFacetPanel,
  type DataTableFacetedFilterOption,
  type DataTableFacetPanelFacet,
} from "@/components/shared/data-table/faceted-filter"
import { DenseRowActionMenu } from "@/components/shared/data-table/menu-owners"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell"
import type { SelectedRowActionBarAction } from "@/components/shared/data-table/selected-row-action-bar"
import { CopyButton } from "@/components/shared/feedback/copy-button"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { DetailDrawer, DetailDrawerTabs, DetailDrawerTabsContent, DetailDrawerTabsList, DetailDrawerTabsTrigger } from "@/components/shared/detail-drawer"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { RefreshSpinner } from "@/components/shared/loading/spinner"
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { AlertDialog, AlertDialogClose, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { FieldError } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { getNucleiPocQueryActiveSyncTaskName, getNucleiPocQueryErrorReason, getNucleiPocQueryErrorStatus, isNucleiPocSyncTaskTerminal, useNucleiPocDetail, useNucleiPocFilterOptions, useNucleiPocs, useNucleiPocSource, useNucleiPocSyncTask, useSetNucleiPocActivation, useSyncNucleiPocSource, useUpdateNucleiPocEnabled } from "@/hooks/use-nuclei-pocs"
import type { NucleiPocListItem, NucleiPocSourceType, NucleiPocSyncState, NucleiPocSyncTask } from "@/types/nuclei-poc.types"
import { formatDateByLocale } from "@/lib/date-utils"
import { getSeverityVariant, SEVERITY_LEVELS } from "@/lib/severity-config"
import { normalizeError } from "@/lib/errors/normalize-error"
import { getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

type NucleiPocSyncSourceKind = NucleiPocSourceType

export const DEFAULT_NUCLEI_TEMPLATES_GIT_URL = "https://github.com/projectdiscovery/nuclei-templates.git"

const DETAIL_ACTION_ICON = semanticIcons.action.view
const REFERENCE_ICON = semanticIcons.concept.endpoint
const SYNC_ACTION_ICON = semanticIcons.action.refresh
const SYNC_SOURCE_ICON = semanticIcons.concept.nucleiRepository
const PAGE_SIZE = 10
const NUCLEI_POC_CATALOG_LOADING_ROW_COUNT = getDataTableSkeletonRowCount(PAGE_SIZE)
const EMPTY_NUCLEI_POC_ITEMS: NucleiPocListItem[] = []
const SEARCH_DEBOUNCE_MS = 300

const SYNC_PHASES: NucleiPocSyncState[] = [
  "VALIDATING_SOURCE",
  "CLONING",
  "SCANNING_FILES",
  "VALIDATING_TEMPLATES",
  "COMMITTING",
  "CLEANING",
]

export function isNucleiPocGitRepositoryUrl(value: string): boolean {
  return isPublicHttpsGitUrl(value)
}

export function isNucleiPocGiteeSyncSourceUrl(value: string): boolean {
  return isPublicHttpsGitUrl(value, "gitee.com")
}

function isPublicHttpsGitUrl(value: string, requiredHost?: string) {
  try {
    const url = new URL(value.trim())
    if (url.protocol !== "https:" || url.username || url.password || !url.hostname) return false
    if ((url.port && url.port !== "443") || url.search || url.hash || url.pathname.includes("\\")) return false
    const hostname = url.hostname.toLowerCase().replace(/\.$/, "")
    if (requiredHost && hostname !== requiredHost) return false
    if (isIpLiteral(hostname) || hostname === "localhost" || hostname.endsWith(".localhost") || hostname.endsWith(".local")) return false
    if (!/^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$/.test(hostname)) return false
    let decodedPath = url.pathname
    try {
      decodedPath = decodeURIComponent(decodedPath)
    } catch {
      return false
    }
    if (!decodedPath || decodedPath === "/" || /[\u0000-\u001f\u007f]/.test(decodedPath)) return false
    return !decodedPath.split("/").some((segment) => segment === "." || segment === "..")
  } catch {
    return false
  }
}

function isIpLiteral(hostname: string) {
  const value = hostname.replace(/^\[|\]$/g, "")
  return /^\d{1,3}(?:\.\d{1,3}){3}$/.test(value) || value.includes(":")
}

export default function NucleiPocCatalogPage() {
  return <NucleiPocCatalogWorkspace />
}

function NucleiPocCatalogWorkspace() {
  const t = useTranslations("pages.nucleiCatalog")
  const tCommon = useTranslations("common.actions")
  const tSeverity = useTranslations("severity")
  const locale = useLocale()
  const [query, setQuery] = React.useState<BusinessListQuery>(() => createBusinessListQuery({ pageSize: PAGE_SIZE }))
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const [searchInput, setSearchInput] = React.useState("")
  const [activeName, setActiveName] = React.useState<string | null>(null)
  const [syncDialogOpen, setSyncDialogOpen] = React.useState(false)
  const [syncSourceKind, setSyncSourceKind] = React.useState<NucleiPocSyncSourceKind>("git")
  const [gitSourceUrl, setGitSourceUrl] = React.useState(DEFAULT_NUCLEI_TEMPLATES_GIT_URL)
  const [giteeSourceUrl, setGiteeSourceUrl] = React.useState("")
  const [customSourceUrl, setCustomSourceUrl] = React.useState("")
  const [syncSubmitted, setSyncSubmitted] = React.useState(false)
  const [activeTaskName, setActiveTaskName] = React.useState<string | null>(null)
  const [selectedRows, setSelectedRows] = React.useState<NucleiPocListItem[]>([])
  const [selectedActivation, setSelectedActivation] = React.useState<{ enabled: boolean; names: `nucleiPocs/${string}`[] } | null>(null)

  const selectedSeverities = React.useMemo(
    () => (query.filters?.severities ?? []) as string[],
    [query.filters],
  )
  const selectedTags = React.useMemo(
    () => (query.filters?.tags ?? []) as string[],
    [query.filters],
  )
  const searchQuery = query.search ?? ""
  const page = query.pageIndex ?? 1
  const pageSize = query.pageSize ?? PAGE_SIZE
  const filter = buildNucleiPocFilter(searchQuery, selectedSeverities, selectedTags)
  const listQuery = useNucleiPocs({ pageSize, pageToken: query.pageToken, filter, orderBy: "templateId asc" })
  const tagOptionsQuery = useNucleiPocFilterOptions("tags")
  const sourceQuery = useNucleiPocSource()
  const syncMutation = useSyncNucleiPocSource()
  const enabledMutation = useUpdateNucleiPocEnabled()
	const activationMutation = useSetNucleiPocActivation()
	const updateEnabled = enabledMutation.mutate
	const isUpdatingEnabled = enabledMutation.isPending
	const enabledVariables = enabledMutation.variables
  const taskQuery = useNucleiPocSyncTask(activeTaskName)
  const activeTask = taskQuery.data ?? syncMutation.data ?? null
  const syncIsRunning = Boolean(activeTask && !isNucleiPocSyncTaskTerminal(activeTask) && !taskQuery.isExpired)
	const batchActivationDisabled = syncIsRunning || syncMutation.isPending || activationMutation.isPending
	const [activationTarget, setActivationTarget] = React.useState<boolean | null>(null)

  React.useEffect(() => {
    if (activeTask?.state === "SUCCEEDED") setSelectedRows([])
  }, [activeTask?.name, activeTask?.state])

  const items = listQuery.data?.results ?? EMPTY_NUCLEI_POC_ITEMS
  const nextPageToken = getCurrentCursorNextPageToken(listQuery.data?.nextPageToken, listQuery.isPlaceholderData)
  const paginationNavigation = getCursorPaginationNavigation({ currentPage: page, pageTokens, nextPageToken })
  React.useEffect(() => {
    if (nextPageToken) setPageTokens((tokens) => ({ ...tokens, [page + 1]: nextPageToken }))
  }, [nextPageToken, page])

  React.useEffect(() => {
    const timer = window.setTimeout(() => {
      const normalized = searchInput.trim()
      if ((query.search ?? "") === normalized) return
      setSelectedRows([])
      setPageTokens({ 1: undefined })
      setQuery((current) => applyBusinessListControlChange(current, { search: normalized || undefined }))
    }, SEARCH_DEBOUNCE_MS)
    return () => window.clearTimeout(timer)
  }, [query.search, searchInput])

  const handlePageChange = (nextPage: number) => {
    const transition = getCursorPageTransition({ currentPage: page, pageTokens, nextPageToken, requestedPage: nextPage })
    if (!transition.reachable) return
    setSelectedRows([])
    setQuery((current) => setBusinessListPage(current, { pageIndex: nextPage, pageToken: transition.pageToken }))
  }

  const handleFirstPage = () => {
    if (page === 1) return
    setSelectedRows([])
    setPageTokens({ 1: undefined })
    setQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
  }

  const updateFilters = (filters: Record<string, string[]>) => {
    setSelectedRows([])
    setPageTokens({ 1: undefined })
    setQuery((current) => applyBusinessListControlChange(current, { filters: { ...current.filters, ...filters } }))
  }

  const activeSourceUrl = syncSourceKind === "git" ? gitSourceUrl : syncSourceKind === "gitee" ? giteeSourceUrl : customSourceUrl
  const activeSourceValid = syncSourceKind === "gitee"
    ? isNucleiPocGiteeSyncSourceUrl(activeSourceUrl)
    : isNucleiPocGitRepositoryUrl(activeSourceUrl)

  const submitSync = () => {
    setSyncSubmitted(true)
    if (!activeSourceValid || syncMutation.isPending) return
    syncMutation.mutate({
      requestId: createNucleiPocRequestId(),
      sourceType: syncSourceKind,
      repoUrl: activeSourceUrl.trim(),
    }, {
      onSuccess: (task) => {
        setSelectedRows([])
        setActiveTaskName(task.name)
        setSyncSubmitted(false)
      },
      onError: (error) => {
        const existingTaskName = getNucleiPocQueryActiveSyncTaskName(error)
        if (!existingTaskName) return
        syncMutation.reset()
        setActiveTaskName(existingTaskName)
        setSyncDialogOpen(true)
        setSyncSubmitted(false)
      },
    })
  }

  const handleSyncDialogOpenChange = (open: boolean) => {
    setSyncDialogOpen(open)
    if (open && taskQuery.isExpired) setActiveTaskName(null)
    if (!open) {
      setSyncSubmitted(false)
      if (!activeTaskName) syncMutation.reset()
    }
  }

  const startNewSync = () => {
    setSelectedRows([])
    setActiveTaskName(null)
    setSyncSubmitted(false)
    syncMutation.reset()
  }

  const confirmActivation = () => {
    if (activationTarget === null || activationMutation.isPending) return
    activationMutation.mutate({ enabled: activationTarget }, {
      onSuccess: () => setSelectedRows([]),
      onSettled: () => setActivationTarget(null),
    })
  }

  const openSelectedActivation = React.useCallback((enabled: boolean) => {
    if (batchActivationDisabled || selectedRows.length === 0) return
    setSelectedActivation({
      enabled,
      names: selectedRows.map((row) => row.name),
    })
  }, [batchActivationDisabled, selectedRows])

  const confirmSelectedActivation = React.useCallback(() => {
    if (!selectedActivation || activationMutation.isPending) return
    activationMutation.mutate(selectedActivation, {
      onSuccess: () => setSelectedRows([]),
      onSettled: () => setSelectedActivation(null),
    })
  }, [activationMutation, selectedActivation])

  const selectedRowActions = React.useMemo<SelectedRowActionBarAction[]>(() => [
    {
      key: "enable-selected-pocs",
      label: t("actions.enableSelected"),
      icon: CheckCircle2,
      tone: "success",
      group: "activation",
      disabled: batchActivationDisabled,
      onClick: () => openSelectedActivation(true),
    },
    {
      key: "disable-selected-pocs",
      label: t("actions.disableSelected"),
      icon: Ban,
      tone: "muted",
      group: "activation",
      disabled: batchActivationDisabled,
      onClick: () => openSelectedActivation(false),
    },
  ], [batchActivationDisabled, openSelectedActivation, t])

  const columns = React.useMemo<ColumnDef<NucleiPocListItem>[]>(() => [
    {
      id: "select",
      size: 44,
      minSize: 44,
      maxSize: 44,
      enableResizing: false,
      enableSorting: false,
      enableHiding: false,
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          indeterminate={!table.getIsAllPageRowsSelected() && table.getIsSomePageRowsSelected()}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label={t("actions.selectAll")}
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          onClick={(event) => event.stopPropagation()}
          aria-label={t("actions.selectRow", { name: row.original.displayName })}
        />
      ),
    },
    {
      accessorKey: "displayName",
      size: 300,
      minSize: 230,
      meta: { title: t("columns.name") },
      header: ({ column }) => <DataTableColumnHeader column={column} title={t("columns.name")} />,
      cell: ({ row }) => (
        <div className="flex min-w-0 flex-col gap-1">
          <span className={cn("break-words", textRole.tableCellPrimary)}>{row.original.displayName}</span>
          <span className={cn("truncate font-mono", textRole.tableCellSecondary)} title={row.original.templateId}>{row.original.templateId}</span>
        </div>
      ),
    },
    {
      accessorKey: "severity",
      size: 100,
      minSize: 92,
      maxSize: 120,
      meta: {
        title: t("columns.severity"),
        singleBadge: true,
        singleBadgeValue: (row) => tSeverity(row.severity),
        singleBadgeValues: SEVERITY_LEVELS.map((severity) => tSeverity(severity)),
      },
      header: ({ column }) => <DataTableColumnHeader column={column} title={t("columns.severity")} />,
      cell: ({ row }) => <Badge variant={getSeverityVariant(row.original.severity)}>{tSeverity(row.original.severity)}</Badge>,
    },
    {
      accessorKey: "tags",
      size: 240,
      minSize: 160,
      meta: { title: t("columns.tags") },
      header: ({ column }) => <DataTableColumnHeader column={column} title={t("columns.tags")} />,
      cell: ({ row }) => <PocTagCell tags={row.original.tags} />,
    },
    {
      accessorKey: "isEnabled",
      size: 138,
      minSize: 128,
      maxSize: 154,
      enableSorting: false,
      meta: { title: t("columns.status") },
      header: ({ column }) => <DataTableColumnHeader column={column} title={t("columns.status")} />,
      cell: ({ row }) => {
        const item = row.original
        const pending = isUpdatingEnabled && enabledVariables?.name === item.name
        return (
          <div className="flex items-center gap-2">
            <Switch
              checked={item.isEnabled}
              disabled={pending}
              onClick={(event) => event.stopPropagation()}
              onCheckedChange={(isEnabled) => {
                updateEnabled({ name: item.name, isEnabled, updateMask: ["isEnabled"] })
              }}
              aria-label={t("actions.setEnabled", { name: item.displayName, state: t(item.isEnabled ? "status.enabled" : "status.disabled") })}
            />
            <PocEnabledStatus enabled={item.isEnabled} />
          </div>
        )
      },
    },
    {
      accessorKey: "updatedAt",
      size: 172,
      minSize: 172,
      maxSize: 188,
      enableResizing: false,
      meta: { title: t("columns.updated") },
      header: ({ column }) => <DataTableColumnHeader column={column} title={t("columns.updated")} />,
      cell: ({ row }) => <TimestampCell value={formatDateByLocale(row.original.updatedAt, locale)} />,
    },
    {
      id: "actions",
      size: 52,
      minSize: 52,
      maxSize: 52,
      enableResizing: false,
      enableSorting: false,
      enableHiding: false,
      cell: ({ row }) => (
        <DenseRowActionMenu ariaLabel={t("actions.open")}>
          <DropdownMenuItem onClick={() => setActiveName(row.original.name)}>
            <DETAIL_ACTION_ICON />
            {t("actions.open")}
          </DropdownMenuItem>
        </DenseRowActionMenu>
      ),
    },
  ], [enabledVariables?.name, isUpdatingEnabled, locale, t, tSeverity, updateEnabled])

  const tagOptions = React.useMemo(() => {
    const optionByValue = new Map<string, DataTableFacetedFilterOption<string>>(
      (tagOptionsQuery.data?.results ?? []).map((option) => [option.value, option]),
    )
    for (const tag of selectedTags) {
      if (!optionByValue.has(tag)) optionByValue.set(tag, { value: tag, label: tag })
    }
    return [...optionByValue.values()]
  }, [selectedTags, tagOptionsQuery.data?.results])

  const facets = React.useMemo<DataTableFacetPanelFacet[]>(() => [
    {
      id: "severity",
      label: t("severity"),
      values: selectedSeverities,
      onValuesChange: (values) => updateFilters({ severities: values }),
      options: SEVERITY_LEVELS.map((severity) => ({ value: severity, label: tSeverity(severity) })),
      emptyLabel: t("all"),
      clearLabel: tCommon("reset"),
    },
    {
      id: "tags",
      label: t("tag"),
      values: selectedTags,
      onValuesChange: (values) => updateFilters({ tags: values }),
      options: tagOptions,
      emptyLabel: t("all"),
      clearLabel: tCommon("reset"),
    },
  ], [selectedSeverities, selectedTags, t, tCommon, tSeverity, tagOptions])

  const emptyMessage = sourceQuery.isError
    ? t("sourceError")
	  : items.length === 0
	      ? (searchQuery || selectedSeverities.length || selectedTags.length ? t("empty") : t("noCommittedSource"))
      : t("empty")

  return (
    <ContentHandoff
      owner="nuclei-poc-catalog-content"
      layer="workspace"
      isLoading={listQuery.isPending || sourceQuery.isPending}
      skeleton={<NucleiPocCatalogLoadingState columns={columns} />}
    >
      <div className="flex flex-col px-4 lg:px-6">
        <div {...getLoadingStructureSlotAttributes("nuclei-poc-catalog-source")}>
          <NucleiPocSyncSourceStatus source={sourceQuery.data} loading={sourceQuery.isPending} error={sourceQuery.isError} locale={locale} />
        </div>
        <div {...getLoadingStructureSlotAttributes("nuclei-poc-catalog-table")} className="min-w-0">
          <BusinessListDataTable
        data={items}
        columns={columns}
        getRowId={(item) => item.name}
        paginationNavigation={{
          ...paginationNavigation,
          onFirstPage: handleFirstPage,
          canPreviousPage: paginationNavigation.canPreviousPage,
          canNextPage: paginationNavigation.canNextPage,
        }}
        cursorPaginationSummary={{ total: listQuery.data?.totalSize ?? 0 }}
        behavior={{
          onRowClick: (item) => setActiveName((item as NucleiPocListItem).name),
          getRowActionLabel: () => t("actions.open"),
          expandColumnIds: ["displayName"],
          enableRowSelection: true,
        }}
        state={{
          sortingMode: "none",
          pagination: { pageIndex: page - 1, pageSize },
          onSelectionChange: setSelectedRows,
          selectedRows,
          onPaginationChange: (next) => {
            const current = { pageIndex: page - 1, pageSize }
            if (next.pageIndex !== current.pageIndex) handlePageChange(next.pageIndex + 1)
          },
        }}
        actions={{ selectedRowActions }}
        ui={{
          emptyMessage,
          emptyComponent: listQuery.isError ? <CatalogError error={listQuery.error} onRetry={() => void listQuery.refetch()} /> : undefined,
          loading: listQuery.isPending,
          loadingPresentation: "rows",
          loadingRowCount: NUCLEI_POC_CATALOG_LOADING_ROW_COUNT,
          stableSurfaceRowCount: NUCLEI_POC_CATALOG_LOADING_ROW_COUNT,
          showColumnVisibility: true,
          toolbarRight: <Button type="button" size="sm" variant="outline" onClick={() => setSyncDialogOpen(true)}>{syncIsRunning ? <RefreshSpinner className="size-4" aria-hidden="true" /> : <SYNC_ACTION_ICON className="size-4" aria-hidden="true" />}{syncIsRunning ? t("sync.viewProgress") : t("sync.action")}</Button>,
           toolbarLeft: <div className="flex w-full flex-wrap items-center gap-2 sm:flex-1"><SimpleSearchToolbar value={searchInput} onChange={setSearchInput} placeholder={t("search")} toolbarDensity="compact" /><DataTableFacetPanel title={t("filter")} facets={facets} activeCount={selectedSeverities.length + selectedTags.length} /><NucleiPocBatchActivationControls activationTarget={activationTarget} onActivationTargetChange={setActivationTarget} disabled={batchActivationDisabled} isPending={activationMutation.isPending} onConfirm={confirmActivation} /></div>,
        }}
          />
        </div>
        <NucleiPocDetailDrawer name={activeName} open={Boolean(activeName)} onOpenChange={(open) => !open && setActiveName(null)} />
        <NucleiPocSelectedActivationDialog
          activation={selectedActivation}
          isPending={activationMutation.isPending}
          onCancel={() => setSelectedActivation(null)}
          onConfirm={confirmSelectedActivation}
        />
        <NucleiPocSyncDialog
        open={syncDialogOpen}
        onOpenChange={handleSyncDialogOpenChange}
        sourceKind={syncSourceKind}
        sourceUrl={activeSourceUrl}
        onSourceKindChange={(value) => {
          if (value === "git" || value === "gitee" || value === "custom") {
            setSyncSourceKind(value)
            setSyncSubmitted(false)
          }
        }}
        onSourceUrlChange={(value) => {
          if (syncSourceKind === "git") setGitSourceUrl(value)
          else if (syncSourceKind === "gitee") setGiteeSourceUrl(value)
          else setCustomSourceUrl(value)
          setSyncSubmitted(false)
        }}
        submitted={syncSubmitted}
        isSubmitting={syncMutation.isPending}
        task={activeTask}
        taskPending={Boolean(activeTaskName) && taskQuery.isPending}
        taskError={taskQuery.error ?? syncMutation.error ?? null}
        errorKind={taskQuery.error ? "task" : syncMutation.error ? "create" : null}
        taskExpired={taskQuery.isExpired}
        onSubmit={submitSync}
        onStartNew={startNewSync}
        />
      </div>
    </ContentHandoff>
  )
}

function NucleiPocCatalogLoadingState({
  columns,
}: {
  columns: ColumnDef<NucleiPocListItem, unknown>[]
}) {
  const locale = useLocale()
  return (
    <div className="flex flex-col px-4 lg:px-6">
      <div {...getLoadingStructureSlotAttributes("nuclei-poc-catalog-source")}>
        <NucleiPocSyncSourceStatus source={undefined} loading error={false} locale={locale} />
      </div>
      <div {...getLoadingStructureSlotAttributes("nuclei-poc-catalog-table")} className="min-w-0">
        <BusinessListDataTable
          data={[]}
          columns={columns}
          paginationNavigation={{
            mode: "cursor",
            canFirstPage: false,
            canPreviousPage: false,
            canNextPage: false,
          }}
          cursorPaginationSummary={{ total: 0 }}
          state={{
            sortingMode: "none",
            pagination: { pageIndex: 0, pageSize: PAGE_SIZE },
          }}
          behavior={{
            enableRowSelection: true,
            expandColumnIds: ["displayName"],
          }}
          ui={{
            loading: true,
            loadingPresentation: "initial",
            initialLoadingToolbarFilterCount: 1,
            loadingRowCount: NUCLEI_POC_CATALOG_LOADING_ROW_COUNT,
            stableSurfaceRowCount: NUCLEI_POC_CATALOG_LOADING_ROW_COUNT,
            showColumnVisibility: true,
            toolbarRight: <span aria-hidden="true" />,
          }}
        />
      </div>
    </div>
  )
}

export function NucleiPocBatchActivationControls({
  activationTarget,
  onActivationTargetChange,
  disabled,
  isPending,
  onConfirm,
}: {
  activationTarget: boolean | null
  onActivationTargetChange: (target: boolean | null) => void
  disabled: boolean
  isPending: boolean
  onConfirm: () => void
}) {
  const t = useTranslations("pages.nucleiCatalog")
  const tCommon = useTranslations("common.actions")
  const controlsDisabled = disabled || isPending

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={controlsDisabled}
              aria-label={t("actions.batch")}
            />
          }
        >
          <Layers className="size-4" aria-hidden="true" />
          {t("actions.batch")}
          <ChevronDown className="size-4" aria-hidden="true" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuItem
            disabled={controlsDisabled}
            onClick={() => onActivationTargetChange(true)}
          >
            <CheckCircle2 className="size-4" aria-hidden="true" />
            {t("actions.enableAll")}
          </DropdownMenuItem>
          <DropdownMenuItem
            disabled={controlsDisabled}
            onClick={() => onActivationTargetChange(false)}
          >
            <Ban className="size-4" aria-hidden="true" />
            {t("actions.disableAll")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <AlertDialog
        open={activationTarget !== null}
        onOpenChange={(open) => {
          if (!open && !isPending) onActivationTargetChange(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {activationTarget === true ? t("batch.enableTitle") : t("batch.disableTitle")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {activationTarget === true ? t("batch.enableDescription") : t("batch.disableDescription")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline" disabled={isPending}>
              {tCommon("cancel")}
            </AlertDialogClose>
            <Button
              type="button"
              onClick={onConfirm}
              loading={isPending}
              loadingLabel={t("batch.processing")}
            >
              {t("batch.confirm")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

function NucleiPocSelectedActivationDialog({
  activation,
  isPending,
  onCancel,
  onConfirm,
}: {
  activation: { enabled: boolean; names: `nucleiPocs/${string}`[] } | null
  isPending: boolean
  onCancel: () => void
  onConfirm: () => void
}) {
  const t = useTranslations("pages.nucleiCatalog")
  const tCommon = useTranslations("common.actions")
  const open = activation !== null
  return (
    <AlertDialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen && !isPending) onCancel() }}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {activation?.enabled ? t("selected.enableTitle") : t("selected.disableTitle")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {activation ? t("selected.description", { count: activation.names.length }) : null}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogClose variant="outline" disabled={isPending} onClick={onCancel}>
            {tCommon("cancel")}
          </AlertDialogClose>
          <Button type="button" onClick={onConfirm} loading={isPending} loadingLabel={t("selected.processing")}>
            {t("selected.confirm")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

function NucleiPocSourceStatusLoadingText({
  children,
  className,
}: React.PropsWithChildren<{ className: string }>) {
  return (
    <div
      aria-hidden="true"
      data-slot="nuclei-poc-source-status-loading-text"
      className={cn("relative min-w-0 max-w-full text-transparent", className)}
    >
      <span className="invisible">{children}</span>
      <Skeleton className="absolute inset-0" />
    </div>
  )
}

export function NucleiPocSyncSourceStatus({ source, loading, error, locale }: {
  source: { sourceType: NucleiPocSourceType; repoUrl: string; commitSha: string; syncedAt: string | null } | null | undefined
  loading: boolean
  error: boolean
  locale: string
}) {
  const t = useTranslations("pages.nucleiCatalog")
  const isInitialLoading = loading && !source
  const displayedSource = source ?? (isInitialLoading ? {
    sourceType: "git" as const,
    repoUrl: DEFAULT_NUCLEI_TEMPLATES_GIT_URL,
    commitSha: "000000000000",
    syncedAt: "2026-08-18T08:00:00.000Z",
  } : null)
  const sourceLabel = displayedSource
    ? t(`sync.source${displayedSource.sourceType[0].toUpperCase()}${displayedSource.sourceType.slice(1)}` as "sync.sourceGit")
    : t("sync.noSource")
  const commitInfo = displayedSource
    ? t("sync.commitInfo", {
      commit: displayedSource.commitSha.slice(0, 12),
      time: displayedSource.syncedAt ? formatDateByLocale(displayedSource.syncedAt, locale) : t("detail.noData"),
    })
    : null
  return (
    <div
      className="mb-3 flex min-w-0 flex-col gap-2 border-y border-border/70 bg-muted/20 px-3 py-3 sm:flex-row sm:items-center sm:justify-between"
      role="status"
      aria-label={t("sync.currentSource")}
    >
      <div className="flex min-w-0 items-start gap-3">
        <SYNC_SOURCE_ICON className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className={textRole.sectionTitle}>{t("sync.currentSource")}</span>
            <Badge variant={displayedSource ? "outline" : "secondary"} className={isInitialLoading ? "relative text-transparent" : undefined}>
              {isInitialLoading ? t("sync.committedLabel") : displayedSource ? t("sync.committedLabel") : t("sync.noSourceLabel")}
              {isInitialLoading ? <Skeleton className="absolute inset-0 radius-badge" /> : null}
            </Badge>
          </div>
          <div className="mt-1 flex min-w-0 flex-wrap items-baseline gap-x-2 gap-y-1">
            {displayedSource ? (
              isInitialLoading ? (
                <NucleiPocSourceStatusLoadingText className={textRole.bodyStrong}>
                  {sourceLabel}
                </NucleiPocSourceStatusLoadingText>
              ) : (
                <span className={textRole.bodyStrong}>{sourceLabel}</span>
              )
            ) : (
              <span className={textRole.bodyStrong}>{sourceLabel}</span>
            )}
            {displayedSource ? (
              isInitialLoading ? (
                <NucleiPocSourceStatusLoadingText
                  className={cn("break-all font-mono", textRole.caption)}
                >
                  {displayedSource.repoUrl}
                </NucleiPocSourceStatusLoadingText>
              ) : (
                <span className={cn("min-w-0 max-w-full break-all font-mono", textRole.caption)} title={displayedSource.repoUrl}>
                  {displayedSource.repoUrl}
                </span>
              )
            ) : (
              <span className={textRole.helperText}>{error ? t("sync.sourceLoadError") : t("sync.noSourceDescription")}</span>
            )}
          </div>
        </div>
      </div>
      <div className={cn("shrink-0 sm:text-right", textRole.caption)}>
        {displayedSource ? (
          isInitialLoading ? (
            <NucleiPocSourceStatusLoadingText className={textRole.caption}>
              {commitInfo}
            </NucleiPocSourceStatusLoadingText>
          ) : (
            <span>{commitInfo}</span>
          )
        ) : (
          <span>{t("sync.notSynced")}</span>
        )}
      </div>
    </div>
  )
}

export function NucleiPocSyncDialog({
  open,
  onOpenChange,
  sourceKind,
  sourceUrl,
  onSourceKindChange,
  onSourceUrlChange,
  submitted,
  isSubmitting,
  task,
  taskPending = false,
  taskError,
  errorKind,
  taskExpired,
  onSubmit,
  onStartNew,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  sourceKind: NucleiPocSyncSourceKind
  sourceUrl: string
  onSourceKindChange: (value: string) => void
  onSourceUrlChange: (value: string) => void
  submitted: boolean
  isSubmitting: boolean
  task: NucleiPocSyncTask | null
  taskPending?: boolean
  taskError: unknown
  errorKind?: "create" | "task" | null
  taskExpired: boolean
  onSubmit: () => void
  onStartNew: () => void
}) {
  const t = useTranslations("pages.nucleiCatalog")
  const tCommon = useTranslations("common.actions")
  const lastObservedPhase = React.useRef<{ taskName: string; phase: NucleiPocSyncState } | null>(null)
  React.useEffect(() => {
    if (task && !isNucleiPocSyncTaskTerminal(task) && SYNC_PHASES.includes(task.phase)) {
      lastObservedPhase.current = { taskName: task.name, phase: task.phase }
    }
  }, [task])
  const failedPhase = task?.state === "FAILED" && lastObservedPhase.current?.taskName === task.name
    ? lastObservedPhase.current.phase
    : null
  const showProgress = taskPending || Boolean(task) || Boolean(taskError)
  const isTerminal = task?.state === "SUCCEEDED" || task?.state === "FAILED"
  const hasRecoverableError = Boolean(taskError) && !task
  const isGiteeSource = sourceKind === "gitee"
  const sourceValid = isGiteeSource ? isNucleiPocGiteeSyncSourceUrl(sourceUrl) : isNucleiPocGitRepositoryUrl(sourceUrl)
  const hasSourceError = submitted && !sourceValid
  const sourceUrlLabel = sourceKind === "git" ? t("sync.gitUrl") : isGiteeSource ? t("sync.giteeUrl") : t("sync.customUrl")
  const sourceUrlPlaceholder = sourceKind === "git" ? DEFAULT_NUCLEI_TEMPLATES_GIT_URL : isGiteeSource ? t("sync.giteeUrlPlaceholder") : t("sync.customUrlPlaceholder")
  const sourceOptions = [
    { value: "git", label: t("sync.sourceGit"), description: t("sync.sourceGitDescription") },
    { value: "gitee", label: t("sync.sourceGitee"), description: t("sync.sourceGiteeDescription") },
    { value: "custom", label: t("sync.sourceCustom"), description: t("sync.sourceCustomDescription") },
  ] as const
  return <Dialog open={open} onOpenChange={onOpenChange}><DialogContent className="max-h-[min(90vh,720px)] overflow-y-auto sm:max-w-lg"><DialogHeader><DialogTitle>{showProgress ? t("sync.progressTitle") : t("sync.title")}</DialogTitle><DialogDescription>{showProgress ? t("sync.progressDescription") : t("sync.sourceDescription")}</DialogDescription></DialogHeader>{showProgress ? <SyncTaskProgress task={task} failedPhase={failedPhase} error={taskError} errorKind={errorKind} expired={taskExpired} /> : <form noValidate className="grid gap-5" onSubmit={(event) => { event.preventDefault(); onSubmit() }}><div className="grid gap-2"><Label id="nuclei-poc-sync-source-kind-label">{t("sync.sourceLabel")}</Label><RadioGroup value={sourceKind} onValueChange={onSourceKindChange} aria-labelledby="nuclei-poc-sync-source-kind-label" disabled={isSubmitting} className="gap-2">{sourceOptions.map((option) => { const optionId = `nuclei-poc-sync-source-${option.value}`; const selected = sourceKind === option.value; return <label key={option.value} htmlFor={optionId} className={cn("radius-control flex min-w-0 cursor-pointer items-start gap-3 border px-3 py-2.5 transition-colors", selected ? "border-primary/60 bg-primary/10" : "border-border", isSubmitting && "cursor-not-allowed opacity-60")}><RadioGroupItem id={optionId} value={option.value} disabled={isSubmitting} className="mt-0.5" /><span className="min-w-0"><span className={textRole.bodyStrong}>{option.label}</span><span className={cn("mt-0.5 block", textRole.helperText)}>{option.description}</span></span></label> })}</RadioGroup></div><div className="grid gap-2"><Label htmlFor="nuclei-poc-sync-source-url">{sourceUrlLabel}</Label><Input id="nuclei-poc-sync-source-url" type="url" value={sourceUrl} onChange={(event) => onSourceUrlChange(event.target.value)} placeholder={sourceUrlPlaceholder} autoComplete="url" inputMode="url" maxLength={512} disabled={isSubmitting} aria-invalid={hasSourceError} aria-describedby={hasSourceError ? "nuclei-poc-sync-source-url-error" : undefined} required />{hasSourceError ? <FieldError id="nuclei-poc-sync-source-url-error">{isGiteeSource ? t("sync.invalidGiteeUrl") : t("sync.invalidUrl")}</FieldError> : null}</div><DialogFooter><Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isSubmitting}>{tCommon("cancel")}</Button><Button type="submit" loading={isSubmitting} loadingLabel={t("sync.submitting")}>{t("sync.submit")}</Button></DialogFooter></form>}{showProgress ? <DialogFooter>{isTerminal || taskExpired || hasRecoverableError ? <Button type="button" variant="outline" onClick={onStartNew}>{t("sync.newSync")}</Button> : null}<Button type="button" onClick={() => onOpenChange(false)}>{tCommon("close")}</Button></DialogFooter> : null}</DialogContent></Dialog>
}

const SYNC_FAILURE_CODES = new Set([
  "UNSAFE_SOURCE", "GIT_CLONE_FAILED", "FILE_QUOTA_EXCEEDED", "YAML_FILE_QUOTA_EXCEEDED",
  "YAML_QUOTA_EXCEEDED", "YAML_SIZE_EXCEEDED", "WORKSPACE_QUOTA_EXCEEDED", "TEMPLATE_INVALID",
  "DUPLICATE_TEMPLATE_ID", "EMPTY_CANDIDATE", "DEADLINE_EXCEEDED", "PROCESS_INTERRUPTED",
  "SYNC_FAILED", "CANDIDATE_STAGE_FAILED", "PATH_ESCAPE", "SYMLINK_REJECTED", "SUBMODULE_REJECTED",
  "GIT_QUOTA_EXCEEDED", "GIT_COMMIT_LOOKUP_FAILED", "WORKSPACE_UNAVAILABLE",
])

function SyncTaskProgress({ task, failedPhase, error, errorKind, expired }: { task: NucleiPocSyncTask | null; failedPhase: NucleiPocSyncState | null; error: unknown; errorKind?: "create" | "task" | null; expired: boolean }) {
  const t = useTranslations("pages.nucleiCatalog")
  const taskStatus = task?.state === "SUCCEEDED" ? "success" : task?.state === "FAILED" || expired || error ? "error" : "running"
  const statusLabel = task?.state === "SUCCEEDED" ? t("sync.completed") : task?.state === "FAILED" ? t("sync.failed") : expired ? t("sync.expired") : error ? t("sync.error") : t("sync.running")
  const phaseLabel = taskStatus === "running" && task
    ? (SYNC_PHASES.includes(task.phase) ? t(`sync.phases.${task.phase}`) : t("sync.waiting"))
    : statusLabel
  const statusMessage = expired || getNucleiPocQueryErrorStatus(error) === 404 || getNucleiPocQueryErrorStatus(error) === 410
    ? t("sync.expiredSummary")
    : error
      ? getSyncErrorMessage(error, errorKind, t)
      : task?.state === "SUCCEEDED"
        ? t("sync.successSummary", { count: task.committedPocCount ?? 0, skipped: task.diagnostics.total, commit: task.commitSha?.slice(0, 12) ?? "-" })
        : task?.state === "FAILED"
          ? getSyncFailureMessage(task, t)
          : task
            ? t("sync.runningSummary", { phase: phaseLabel })
            : t("sync.waiting")
  return <div className="grid gap-4" role="status" aria-live="polite"><div className="flex items-start justify-between gap-3 border-b border-border pb-3"><div className="flex min-w-0 items-center gap-2"><SyncStatusIcon status={taskStatus} /><span className={textRole.bodyStrong}>{statusLabel}</span></div><Badge variant={taskStatus === "error" ? "destructive" : taskStatus === "success" ? "secondary" : "outline"}>{phaseLabel}</Badge></div><p className={cn("break-words", textRole.body, taskStatus === "error" && "text-error")}>{statusMessage}</p>{task?.state === "FAILED" ? <p className={textRole.bodySubtle}>{t("sync.failedSummary")}</p> : null}{task && (task.state !== "FAILED" || failedPhase) ? <SyncPhaseTimeline task={task} failedPhase={failedPhase} t={t} /> : null}{task ? <div className="grid grid-cols-2 gap-3 border-y border-border/70 py-3 sm:grid-cols-4">{(["filesSeen", "yamlFilesSeen", "templatesValidated", "bytesRead"] as const).map((key) => <SyncCounter key={key} label={t(`sync.counters.${key}`)} value={task.counters[key]} />)}</div> : null}{task && task.diagnostics.total > 0 ? <SyncDiagnostics diagnostics={task.diagnostics} t={t} /> : null}</div>
}

function SyncStatusIcon({ status }: { status: "running" | "success" | "error" }) {
  if (status === "success") return <CheckCircle2 aria-hidden="true" className={cn("size-5", getStatusToneTextClass("success"))} />
  if (status === "error") return <AlertTriangle aria-hidden="true" className={cn("size-5", getStatusToneTextClass("error"))} />
  return <RefreshSpinner aria-hidden="true" className={cn("size-5", getStatusToneTextClass("info"))} />
}

function SyncPhaseTimeline({ task, failedPhase, t }: { task: NucleiPocSyncTask; failedPhase: NucleiPocSyncState | null; t: ReturnType<typeof useTranslations> }) {
  const currentIndex = SYNC_PHASES.indexOf(task.state === "FAILED" && failedPhase ? failedPhase : task.phase)
  return <ol className="grid gap-2" aria-label={t("sync.phaseTimeline")}>{SYNC_PHASES.map((phase, index) => { const state = getSyncPhaseState(task, index, currentIndex); const Icon = state === "complete" ? CheckCircle2 : state === "failed" ? AlertTriangle : state === "active" ? RefreshCw : Circle; return <li key={phase} className="relative flex min-w-0 items-center gap-2" aria-current={state === "active" || state === "failed" ? "step" : undefined}><span className="flex size-5 shrink-0 items-center justify-center">{state === "active" ? <RefreshSpinner aria-hidden="true" className={cn("size-4", getStatusToneTextClass("info"))} /> : <Icon aria-hidden="true" className={cn("size-4", state === "complete" && getStatusToneTextClass("success"), state === "failed" && getStatusToneTextClass("error"), state === "pending" && getStatusToneTextClass("muted"))} />}</span><span className={cn(textRole.helperText, state === "active" && textRole.bodyStrong, state === "failed" && "text-error")}>{t(`sync.phases.${phase}`)}</span></li> })}</ol>
}

function getSyncPhaseState(task: NucleiPocSyncTask, index: number, currentIndex: number): "complete" | "active" | "failed" | "pending" {
  if (task.state === "SUCCEEDED") return "complete"
  if (task.state === "FAILED") return index < currentIndex ? "complete" : index === currentIndex ? "failed" : "pending"
  if (index < currentIndex) return "complete"
  return index === currentIndex ? "active" : "pending"
}

function getSyncFailureMessage(task: NucleiPocSyncTask, t: ReturnType<typeof useTranslations>) {
  const code = task.failureCode && SYNC_FAILURE_CODES.has(task.failureCode) ? task.failureCode : "SYNC_FAILED"
  return t(`sync.failureCodes.${code}`)
}

function getSyncErrorMessage(error: unknown, errorKind: "create" | "task" | null | undefined, t: ReturnType<typeof useTranslations>) {
  if (errorKind === "task") return t("sync.taskReadError")
  const reason = getNucleiPocQueryErrorReason(error)
  if (reason === "SYNC_ALREADY_RUNNING") return t("sync.activeTaskUnavailable")
  if (reason === "SYNC_REQUEST_CONFLICT") return t("sync.requestConflict")
  if (reason === "SYNC_REQUEST_EXPIRED") return t("sync.expiredSummary")
  return t("sync.createError")
}

function SyncDiagnostics({ diagnostics, t }: { diagnostics: NucleiPocSyncTask["diagnostics"]; t: ReturnType<typeof useTranslations> }) {
  return <Collapsible className="grid gap-2 border-t border-border/70 pt-3"><div className="flex items-center justify-between gap-3"><span className={textRole.metadataLabel}>{t("sync.skippedSummary", { count: diagnostics.total })}</span>{diagnostics.samples.length ? <CollapsibleTrigger render={<Button type="button" size="sm" variant="ghost" className="shrink-0 gap-1" />}><span>{t("sync.viewSamples")}</span><ChevronDown aria-hidden="true" className="size-4" /></CollapsibleTrigger> : null}</div><CollapsibleContent className="grid gap-1"><ul className="grid max-h-40 gap-1 overflow-y-auto">{diagnostics.samples.map((sample, index) => <li key={`${sample.reasonCode}-${index}`} className={cn("break-all", textRole.caption)}>{sample.relativePath ? `${sample.relativePath}: ` : ""}{sample.reasonCode}</li>)}</ul>{diagnostics.truncated ? <p className={textRole.helperText}>{t("sync.diagnosticsTruncated")}</p> : null}</CollapsibleContent></Collapsible>
}

function SyncCounter({ label, value }: { label: string; value: number | null }) {
  return <div className="min-w-0 space-y-1"><span className={textRole.metadataLabel}>{label}</span><span className={cn("block font-mono", textRole.bodyStrong)}>{value === null ? "-" : value.toLocaleString()}</span></div>
}

function NucleiPocDetailDrawer({ name, open, onOpenChange }: { name: string | null; open: boolean; onOpenChange: (open: boolean) => void }) {
  const t = useTranslations("pages.nucleiCatalog")
  const tSeverity = useTranslations("severity")
  const locale = useLocale()
  const query = useNucleiPocDetail(name, open)
  const item = query.data
  return <DetailDrawer open={open} onOpenChange={onOpenChange} title={item?.displayName ?? name?.replace(/^nucleiPocs\//, "") ?? ""} description={item?.templateId ?? name ?? undefined} titleMeta={item ? <Badge variant={getSeverityVariant(item.severity)}>{tSeverity(item.severity)}</Badge> : null} headerMeta={item ? <span className={cn("min-w-0 truncate font-mono", textRole.caption)} title={item.name}>{item.name}</span> : null}>{query.isPending ? <DetailLoading /> : query.isError ? <CatalogError error={query.error} onRetry={() => void query.refetch()} /> : item ? <DetailDrawerTabs defaultValue="overview"><DetailDrawerTabsList className="px-6"><DetailDrawerTabsTrigger value="overview">{t("detail.overview")}</DetailDrawerTabsTrigger><DetailDrawerTabsTrigger value="yaml">{t("detail.yaml")}</DetailDrawerTabsTrigger></DetailDrawerTabsList><div className="flex min-h-0 flex-1 flex-col overflow-y-auto px-6 py-5"><DetailDrawerTabsContent value="overview" className="space-y-7"><section className="space-y-3"><h3 className={textRole.sectionTitle}>{t("detail.metadata")}</h3><dl className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2"><PocDetailField label={t("detail.templateId")} value={item.templateId} mono /><PocDetailField label={t("detail.path")} value={item.relativePath} mono /><PocDetailField label={t("detail.author")} value={item.author || t("detail.noData")} /><PocDetailField label={t("columns.status")} value={<PocEnabledStatus enabled={item.isEnabled} />} /><PocDetailField label={t("columns.updated")} value={formatDateByLocale(item.updatedAt, locale)} /><PocDetailField label={t("columns.tags")} value={<PocTagCell tags={item.tags} showAll />} className="sm:col-span-2" /><PocDetailField label={t("detail.description")} value={item.description || t("detail.noData")} className="sm:col-span-2" /></dl></section><section className="space-y-3"><h3 className={textRole.sectionTitle}>{t("detail.classification")}</h3><dl className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2"><PocDetailField label={t("detail.cve")} value={item.cve.length ? item.cve.join(", ") : t("detail.noData")} mono /><PocDetailField label={t("detail.cwe")} value={item.cwe.length ? item.cwe.join(", ") : t("detail.noData")} mono /><PocDetailField label={t("detail.contentHash")} value={item.contentSha256} mono /></dl></section>{item.references.length ? <section className="space-y-3"><h3 className={textRole.sectionTitle}>{t("detail.references")}</h3><div className="space-y-2">{item.references.map((reference) => <a key={reference} href={reference} target="_blank" rel="noreferrer" className={cn("flex min-w-0 items-start gap-2 break-all text-primary hover:underline", textRole.body)}><REFERENCE_ICON className="mt-0.5 size-4 shrink-0" />{reference}</a>)}</div></section> : null}{item.remediation ? <section className="space-y-3"><h3 className={textRole.sectionTitle}>{t("detail.remediation")}</h3><p className={cn("whitespace-pre-wrap break-words", textRole.body)}>{item.remediation}</p></section> : null}</DetailDrawerTabsContent><DetailDrawerTabsContent value="yaml" className="flex min-h-0 flex-col"><CodeRegion value={item.content} copyLabel={t("detail.copy")} copiedLabel={t("detail.copied")} /></DetailDrawerTabsContent></div></DetailDrawerTabs> : null}</DetailDrawer>
}

function buildNucleiPocFilter(keyword: string, severities: string[], tags: string[]) {
  const groups: string[] = []
  const normalizedKeyword = keyword.trim().toLowerCase()
  if (normalizedKeyword) groups.push(["templateId", "name", "author", "description", "tags", "cve", "cwe"].map((field) => `${field}="${escapeFilterValue(normalizedKeyword)}"`).join(" OR "))
  if (severities.length) groups.push(severities.map((severity) => `severity=="${escapeFilterValue(severity)}"`).join(" OR "))
  if (tags.length) groups.push(tags.map((tag) => `tags=="${escapeFilterValue(tag)}"`).join(" OR "))
  return groups.join(" AND ")
}

function escapeFilterValue(value: string) {
  return value.replaceAll("\\", "\\\\").replaceAll('"', '\\"')
}

function createNucleiPocRequestId() {
  if (typeof globalThis.crypto?.randomUUID !== "function") {
    throw new Error("The browser must support crypto.randomUUID for Nuclei POC sync requests")
  }
  return globalThis.crypto.randomUUID()
}

function CatalogError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  return <AppErrorState error={normalizeError(error, { notFoundKind: "unexpected-error" })} resourceLabel="Nuclei POC" variant="section" onRetry={onRetry} />
}

function DetailLoading() {
  return <div className="grid gap-4 px-6 py-5"><Skeleton className="h-5 w-32" /><Skeleton className="h-24 w-full" /><Skeleton className="h-24 w-full" /></div>
}

function PocTagCell({ tags, showAll = false }: { tags: string[]; showAll?: boolean }) {
  const visibleTags = showAll ? tags : tags.slice(0, 2)
  return <div className="flex min-w-0 flex-wrap gap-1">{visibleTags.map((tag) => <Badge key={tag} variant="outline">{tag}</Badge>)}{tags.length > visibleTags.length ? <Badge variant="count">+{tags.length - visibleTags.length}</Badge> : null}</div>
}

function PocEnabledStatus({ enabled }: { enabled: boolean }) {
  const t = useTranslations("pages.nucleiCatalog")
  return <span className={textRole.tableCellSecondary}>{t(enabled ? "status.enabled" : "status.disabled")}</span>
}

function PocDetailField({ label, value, mono = false, className }: { label: string; value: React.ReactNode; mono?: boolean; className?: string }) {
  return <div className={cn("min-w-0 space-y-1", className)}><dt className={textRole.metadataLabel}>{label}</dt><dd className={cn("break-words", textRole.metadataValueStrong, mono && "font-mono")}>{value}</dd></div>
}

function CodeRegion({ value, copyLabel, copiedLabel }: { value: string; copyLabel: string; copiedLabel: string }) {
  return <div className="relative flex min-h-0 flex-1 flex-col"><div className="absolute top-2 right-2 z-10"><CopyButton value={value} copyLabel={copyLabel} copiedLabel={copiedLabel} toastId="nuclei-poc-yaml" /></div><pre className="min-h-0 flex-1 overflow-auto rounded-md border bg-muted/30 p-3 pr-12 font-mono text-xs leading-5 whitespace-pre-wrap break-words">{value}</pre></div>
}
