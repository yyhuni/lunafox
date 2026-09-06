"use client"

import React from "react"
import dynamic from "next/dynamic"
import { useTranslations } from "next-intl"
import { semanticIcons } from "@/components/icons"

import {
  DataTableSkeleton,
  getDataTableSkeletonRowCount,
} from "@/components/shared/loading/data-table-skeleton"
import {
  getLoadingOwnerAttributes,
  getLoadingStructureSlotAttributes,
} from "@/components/shared/loading/loading-owner"
import { ScheduledScanDataTable } from "@/components/scan/scheduled/scheduled-scan-data-table"
import { TargetsDataTable } from "./targets/targets-data-table"
import {
  DataTableFacetPanel,
  type DataTableFacetPanelFacet,
} from "@/components/shared/data-table/faceted-filter"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { InteractionLoadingDialog } from "@/components/shared/loading/interaction-loading-dialog"
import {
  AlertDialog,
  AlertDialogClose, AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Tabs, TabsContent, TabsCountBadge, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import {
  deferredInteractionUnmountDelayMs,
  useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount"

import {
  type OrganizationDetailTab,
  type OrganizationDetailViewState,
  type OrganizationScheduledStatusFilter,
} from "./organization-detail-view-state"

const EditOrganizationFormPanel = dynamic(
  () =>
    import("./edit-organization-dialog").then((mod) => ({
      default: mod.EditOrganizationFormPanel,
    })),
  { ssr: false, loading: () => null }
)

const LinkTargetFormPanel = dynamic(
  () =>
    import("./targets/link-target-dialog").then((mod) => ({
      default: mod.LinkTargetFormPanel,
    })),
  { ssr: false, loading: () => null }
)

const CreateScheduledScanDialog = dynamic(
  () =>
    import("@/components/scan/scheduled/create-scheduled-scan-dialog").then(
      (mod) => mod.CreateScheduledScanDialog
    ),
  { ssr: false, loading: () => null }
)

const EditScheduledScanDialog = dynamic(
  () =>
    import("@/components/scan/scheduled/edit-scheduled-scan-dialog").then(
      (mod) => mod.EditScheduledScanDialog
    ),
  { ssr: false, loading: () => null }
)

const EditIcon = semanticIcons.action.edit
const UpdatedAtIcon = semanticIcons.action.refresh
const CreatedAtIcon = semanticIcons.metric.latestScan
const TotalTargetsIcon = semanticIcons.concept.target
const DomainIcon = semanticIcons.concept.domain
const IpIcon = semanticIcons.concept.ip
const CidrIcon = semanticIcons.concept.cidr
const LatestScanIcon = semanticIcons.metric.latestScan
const ScheduledScanIcon = semanticIcons.concept.scheduledScan
const ORGANIZATION_DETAIL_FALLBACK_TARGETS_TABLE_COLUMN_COUNT = 4
const ORGANIZATION_DETAIL_FALLBACK_TARGETS_TABLE_TOOLBAR_BUTTON_COUNT = 1

function OrganizationDetailTargetsTableLoadingState({
  state,
}: {
  state?: OrganizationDetailViewState
}) {
  if (!state) {
    return (
      <DataTableSkeleton
        owner="organization-detail-overview"
        layer="section"
        rows={1}
        columns={ORGANIZATION_DETAIL_FALLBACK_TARGETS_TABLE_COLUMN_COUNT}
        toolbarButtonCount={ORGANIZATION_DETAIL_FALLBACK_TARGETS_TABLE_TOOLBAR_BUTTON_COUNT}
        withPadding={false}
      />
    )
  }

  const targetStableSurfaceRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)

  return (
    <div
      {...getLoadingOwnerAttributes({ owner: "organization-detail-overview", layer: "section", intent: "data" })}
      data-slot="organization-detail-targets-table-loading-state"
      className="w-full"
    >
      <TargetsDataTable
        data={[]}
        columns={state.targetColumns}
        onAddNew={state.handleAddTarget}
        onBulkDelete={state.handleBulkDelete}
        onSelectionChange={state.setSelectedTargets}
        selectedRows={state.selectedTargets}
        searchPlaceholder={state.tColumns("target.target")}
        searchValue={state.searchQuery}
        onSearch={state.commitSearch}
        isSearching={state.isSearching}
        addButtonText={state.tTarget("addTarget")}
        pagination={state.pagination}
        cursorPaginationSummary={state.cursorPaginationSummary}
        paginationNavigation={state.paginationNavigation}
        onPaginationChange={state.handlePaginationChange}
        typeFilter={state.typeFilter}
        onTypeFilterChange={state.handleTypeFilterChange}
        loading
        loadingRowCount={targetStableSurfaceRowCount}
        stableSurfaceRowCount={targetStableSurfaceRowCount}
      />
    </div>
  )
}

type OrganizationDetailSummarySurface = "page" | "drawer"
type OrganizationTranslation = (key: string) => string

function OrganizationDetailSummaryRegion({ children }: { children: React.ReactNode }) {
  return (
    <div
      {...getLoadingStructureSlotAttributes("organization-detail-summary")}
      className="space-y-5"
    >
      {children}
    </div>
  )
}

function OrganizationDetailPrimaryTableRegion({ children }: { children: React.ReactNode }) {
  return (
    <div {...getLoadingStructureSlotAttributes("organization-detail-primary-table")}>
      {children}
    </div>
  )
}

function InlineSlotLoadingState({
  text,
  className,
  skeletonClassName,
}: {
  text: string
  className?: string
  skeletonClassName?: string
}) {
  return (
    <span
      aria-hidden="true"
      className={cn("relative inline-block max-w-full select-none text-transparent", className)}
    >
      {text}
      <span
        data-slot="skeleton"
        className={cn(
          "loading-skeleton !absolute block rounded-md",
          skeletonClassName ?? "inset-y-0 left-0 w-full"
        )}
      />
    </span>
  )
}

export function OrganizationDetailViewLoadingState({
  state,
  previewName,
  previewDescription,
  summarySurface = "page",
}: {
  state?: OrganizationDetailViewState
  previewName?: string
  previewDescription?: string
  summarySurface?: OrganizationDetailSummarySurface
}) {
  const tOrg = useTranslations("organization") as OrganizationTranslation

  // The page workbench owns this inset; the drawer already supplies it outside ContentHandoff.
  return (
    <div
      data-slot="organization-detail-view-loading-state"
      className={cn("space-y-5", summarySurface === "page" && "py-5")}
    >
      <OrganizationDetailSummaryRegion>
        <OrganizationDetailHeader
          state={state}
          tOrg={tOrg}
          previewName={previewName}
          previewDescription={previewDescription}
          loading
        />
        <OrganizationSummaryStrip state={state} tOrg={tOrg} surface={summarySurface} loading />
      </OrganizationDetailSummaryRegion>

      <OrganizationDetailPrimaryTableRegion>
        <Tabs value="targets" className="gap-4 px-4 lg:px-6">
          <TabsList variant="content" size="md">
            <TabsTrigger value="targets" variant="content" size="md" disabled>
              <Skeleton className="h-4 w-16 rounded-full" />
              <Skeleton className="h-5 w-5 rounded-full" />
            </TabsTrigger>
            <TabsTrigger value="scheduled" variant="content" size="md" disabled>
              <Skeleton className="h-4 w-20 rounded-full" />
              <Skeleton className="h-5 w-5 rounded-full" />
            </TabsTrigger>
          </TabsList>

          <OrganizationDetailTargetsTableLoadingState state={state} />
        </Tabs>
      </OrganizationDetailPrimaryTableRegion>
    </div>
  )
}

function OrganizationInitial({ name, loading = false }: { name: string; loading?: boolean }) {
  const initial = name.trim().charAt(0).toUpperCase() || "O"

  return (
    <div className="flex size-14 shrink-0 items-center justify-center rounded-md bg-muted">
      <span className={textRole.pageTitle}>
        {loading ? <InlineSlotLoadingState text={initial} className="w-4" /> : initial}
      </span>
    </div>
  )
}

function OrganizationMetaItem({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode
  label: string
  value: React.ReactNode
}) {
  const valueTitle = typeof value === "string" || typeof value === "number" ? String(value) : undefined

  return (
    <div className="flex min-w-0 items-center gap-2">
      <span className="shrink-0 text-muted-foreground">{icon}</span>
      <span className={textRole.metadataLabel}>{label}</span>
      <span className={cn("min-w-0 truncate", textRole.metadataValue)} title={valueTitle}>
        {value}
      </span>
    </div>
  )
}

function OrganizationDetailHeader({
  state,
  tOrg,
  previewName,
  previewDescription,
  loading = false,
}: {
  state?: OrganizationDetailViewState
  tOrg?: OrganizationTranslation
  previewName?: string
  previewDescription?: string
  loading?: boolean
}) {
  const organization = state?.organization
  if (!loading && !organization) return null

  const translateOrg = (state?.tOrg ?? tOrg) as OrganizationTranslation | undefined
  if (!translateOrg) return null

  const organizationName = organization?.name ?? previewName ?? "Organization"
  const organizationDescription = organization?.description || previewDescription || translateOrg("noDescription")
  // The invisible loading text must use the resolved date string when it is
  // available. A fixed-width date shell can cross the mobile wrap threshold
  // under a different font fallback and change the summary height at handoff.
  const createdAtText = organization
    ? state?.formatDate(organization.createdAt) ?? "2024/09/18 21:35:00"
    : "2024/09/18 21:35:00"
  const updatedAtText = organization
    ? state?.formatDate(organization.updatedAt) ?? "2024/12/19 20:15:00"
    : "2024/12/19 20:15:00"
  const createdAtValue = loading
    ? <InlineSlotLoadingState text={createdAtText} />
    : createdAtText
  const updatedAtValue = loading
    ? <InlineSlotLoadingState text={updatedAtText} />
    : updatedAtText

  return (
    <div className="space-y-5 px-4 lg:px-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div className="min-w-0 space-y-4">
          <div className="flex min-w-0 items-center gap-4">
            <OrganizationInitial name={organizationName} loading={loading} />
            <div className="min-w-0">
              <h1 className={cn("truncate", textRole.pageTitle)}>
                {loading ? (
                  <InlineSlotLoadingState text={organizationName} />
                ) : (
                  organizationName
                )}
              </h1>
              {/* The clamp must own this text directly; an inline-block wrapper can add a third CJK line. */}
              <p
                aria-hidden={loading || undefined}
                className={cn(
                  "relative mt-1 line-clamp-2 min-h-12 md:min-h-0",
                  textRole.bodySubtle,
                  loading && "select-none text-transparent"
                )}
              >
                {organizationDescription}
                {loading ? (
                  <span
                    data-slot="skeleton"
                    className="loading-skeleton !absolute inset-0 block rounded-md"
                  />
                ) : null}
              </p>
            </div>
          </div>

          <div className="flex flex-wrap gap-x-8 gap-y-2">
            <OrganizationMetaItem
              icon={<CreatedAtIcon className="h-4 w-4" />}
              label={translateOrg("detail.createdAt")}
              value={createdAtValue}
            />
            <OrganizationMetaItem
              icon={<UpdatedAtIcon className="h-4 w-4" />}
              label={translateOrg("detail.updatedAt")}
              value={updatedAtValue}
            />
          </div>
        </div>

        <div className="flex shrink-0 flex-wrap items-center gap-2 md:justify-end">
          {loading ? (
            <ActionSkeleton widthClassName="w-28" className="shrink-0" />
          ) : state ? (
            <Button type="button" variant="outline" onClick={state.handleEditOrganization}>
              <EditIcon className="h-4 w-4" />
              {state.tOrg("editOrganization")}
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  )
}

function OrganizationMetric({
  icon,
  label,
  value,
  valueClassName,
  className,
  loading = false,
  loadingText = "0",
  loadingClassName,
  loadingTextUsesContentWidth = false,
}: {
  icon: React.ReactNode
  label: string
  value?: React.ReactNode
  valueClassName?: string
  className?: string
  loading?: boolean
  loadingText?: string
  loadingClassName?: string
  loadingTextUsesContentWidth?: boolean
}) {
  const valueTitle = !loading && (typeof value === "string" || typeof value === "number")
    ? String(value)
    : undefined

  return (
    <div className={cn("min-w-0 px-4 py-3", className)}>
      <div className="flex min-w-0 items-center gap-2">
        <span className="shrink-0 text-muted-foreground">{icon}</span>
        <span
          className={cn("min-w-0 max-w-full truncate", textRole.metadataLabel)}
          title={label}
        >
          {label}
        </span>
      </div>
      <div
        className={cn("mt-2 min-w-0 truncate", valueClassName ?? textRole.pageTitle)}
        title={valueTitle}
      >
        {loading ? (
          <InlineSlotLoadingState
            text={loadingText}
            className={loadingTextUsesContentWidth ? undefined : loadingClassName ?? "w-10"}
          />
        ) : (
          value
        )}
      </div>
    </div>
  )
}

function OrganizationSummaryStrip({
  state,
  tOrg,
  surface = "page",
  loading = false,
}: {
  state?: OrganizationDetailViewState
  tOrg?: OrganizationTranslation
  surface?: OrganizationDetailSummarySurface
  loading?: boolean
}) {
  const translateOrg = (state?.tOrg ?? tOrg) as OrganizationTranslation | undefined
  if (!translateOrg) return null

  const summary = state?.organizationSummary
  const latestScanValue = summary?.latestScanAt
    ? state?.formatDate(summary.latestScanAt) ?? translateOrg("detail.noScan")
    : translateOrg("detail.noScan")
  const summaryGridClassName =
    surface === "drawer"
      ? "grid min-w-0 overflow-hidden rounded-md border bg-card sm:grid-cols-2 lg:grid-cols-4"
      : "grid min-w-0 overflow-hidden rounded-md border bg-card sm:grid-cols-2 xl:grid-cols-8"
  const latestScanClassName =
    surface === "drawer"
      ? "sm:col-span-2 lg:col-span-2"
      : "sm:col-span-2 xl:col-span-2"

  return (
    <div className="px-4 lg:px-6">
      <div className={summaryGridClassName}>
        <OrganizationMetric
          icon={<TotalTargetsIcon className="h-4 w-4" />}
          label={translateOrg("detail.metrics.totalTargets")}
          value={summary?.totalTargets}
          loading={loading}
        />
        <OrganizationMetric
          icon={<DomainIcon className="h-4 w-4" />}
          label={translateOrg("detail.metrics.domains")}
          value={summary?.domains}
          loading={loading}
        />
        <OrganizationMetric
          icon={<IpIcon className="h-4 w-4" />}
          label={translateOrg("detail.metrics.ips")}
          value={summary?.ips}
          loading={loading}
        />
        <OrganizationMetric
          icon={<CidrIcon className="h-4 w-4" />}
          label={translateOrg("detail.metrics.cidrs")}
          value={summary?.cidrs}
          loading={loading}
        />
        <OrganizationMetric
          icon={<LatestScanIcon className="h-4 w-4" />}
          label={translateOrg("detail.metrics.latestScan")}
          value={latestScanValue}
          valueClassName={textRole.metadataValueStrong}
          className={latestScanClassName}
          loading={loading}
          loadingText={latestScanValue}
          loadingTextUsesContentWidth
        />
        <OrganizationMetric
          icon={<ScheduledScanIcon className="h-4 w-4" />}
          label={translateOrg("detail.metrics.scheduledTotal")}
          value={summary?.scheduledTotal}
          loading={loading}
        />
      </div>
    </div>
  )
}

function OrganizationScheduledScanFilters({ state }: { state: OrganizationDetailViewState }) {
  const tDataTable = useTranslations("dataTable")
  const statusOptions = [
    { value: "enabled", label: state.tScan("scheduled.workbench.filters.enabled") },
    { value: "paused", label: state.tScan("scheduled.workbench.filters.paused") },
  ]
  const facets: DataTableFacetPanelFacet[] = [
    {
      id: "status",
      label: state.tOrg("detail.filters.status"),
      values: state.scheduledStatusFilter,
      onValuesChange: (values) =>
        state.handleScheduledStatusFilterChange(values as OrganizationScheduledStatusFilter),
      options: statusOptions,
      emptyLabel: tDataTable("noResults"),
      clearLabel: tDataTable("clearFilter"),
    },
    {
      id: "workflow",
      label: state.tOrg("detail.filters.workflow"),
      values: state.scheduledWorkflowFilter,
      onValuesChange: state.handleScheduledWorkflowFilterChange,
      options: state.scheduledWorkflowOptions.map((workflowName) => ({
        value: workflowName,
        label: workflowName,
      })),
      emptyLabel: tDataTable("noResults"),
      clearLabel: tDataTable("clearFilter"),
    },
  ]

  return (
    <DataTableFacetPanel
      title={tDataTable("filter")}
      facets={facets}
      activeCount={state.scheduledStatusFilter.length + state.scheduledWorkflowFilter.length}
    />
  )
}

function OrganizationDetailScheduledScansTableLoadingState({
  state,
}: {
  state: OrganizationDetailViewState
}) {
  return (
    <div
      {...getLoadingOwnerAttributes({ owner: "organization-detail-scheduled-scans", layer: "section", intent: "data" })}
      data-slot="organization-detail-scheduled-scans-table-loading-state"
      className="w-full"
    >
      <ScheduledScanDataTable
        data={[]}
        columns={state.scheduledColumns}
        onAddNew={state.handleAddScheduledScan}
        searchAfter={<OrganizationScheduledScanFilters state={state} />}
        searchPlaceholder={state.tScan("scheduled.searchPlaceholder")}
        searchValue={state.scheduledSearchQuery}
        onSearch={state.commitScheduledSearch}
        isSearching={state.isSearchingScheduledScans}
        addButtonText={state.tScan("scheduled.createTitle")}
        page={state.scheduledPagination.pageIndex + 1}
        pageSize={state.scheduledPagination.pageSize}
        total={state.scheduledPaginationInfo.total}
        totalPages={state.scheduledPaginationInfo.totalPages}
        onPageChange={(page) =>
          state.handleScheduledPaginationChange({
            ...state.scheduledPagination,
            pageIndex: page - 1,
          })
        }
        onPageSizeChange={(pageSize) =>
          state.handleScheduledPaginationChange({
            pageIndex: 0,
            pageSize,
          })
        }
        showQuickFilters={false}
        enableRowSelection={false}
        loading
        loadingRowCount={getDataTableSkeletonRowCount(state.scheduledPagination.pageSize)}
      />
    </div>
  )
}

export function OrganizationDetailViewTable({
  state,
  summarySurface = "page",
}: {
  state: OrganizationDetailViewState
  summarySurface?: OrganizationDetailSummarySurface
}) {
  const targetStableSurfaceRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)

  return (
    <div className="space-y-5">
      <OrganizationDetailSummaryRegion>
        <OrganizationDetailHeader state={state} />
        <OrganizationSummaryStrip state={state} surface={summarySurface} />
      </OrganizationDetailSummaryRegion>

      <OrganizationDetailPrimaryTableRegion>
        <Tabs
          value={state.activeTab}
          onValueChange={(value) => state.setActiveTab(value as OrganizationDetailTab)}
          className="gap-4 px-4 lg:px-6"
        >
          <TabsList variant="content" size="md">
            <TabsTrigger value="targets" variant="content" size="md">
              {state.tOrg("detail.tabs.targets")}
              <TabsCountBadge>{state.organizationSummary.totalTargets}</TabsCountBadge>
            </TabsTrigger>
            <TabsTrigger value="scheduled" variant="content" size="md">
              {state.tOrg("detail.tabs.scheduled")}
              <TabsCountBadge>{state.organizationSummary.scheduledTotal}</TabsCountBadge>
            </TabsTrigger>
          </TabsList>

          <TabsContent value="targets" className="mt-0">
            <TargetsDataTable
              data={state.targetRows}
              columns={state.targetColumns}
              onAddNew={state.handleAddTarget}
              onBulkDelete={state.handleBulkDelete}
              onSelectionChange={state.setSelectedTargets}
              selectedRows={state.selectedTargets}
              searchPlaceholder={state.tColumns("target.target")}
              searchValue={state.searchQuery}
              onSearch={state.commitSearch}
              isSearching={state.isSearching}
              addButtonText={state.tTarget("addTarget")}
              pagination={state.pagination}
              cursorPaginationSummary={state.cursorPaginationSummary}
              paginationNavigation={state.paginationNavigation}
              onPaginationChange={state.handlePaginationChange}
              typeFilter={state.typeFilter}
              onTypeFilterChange={state.handleTypeFilterChange}
              stableSurfaceRowCount={targetStableSurfaceRowCount}
            />
          </TabsContent>

          <TabsContent value="scheduled" className="mt-0">
            {state.isLoadingScheduledScans ? (
              <OrganizationDetailScheduledScansTableLoadingState state={state} />
            ) : (
              <ScheduledScanDataTable
                data={state.scheduledRows}
                columns={state.scheduledColumns}
                onAddNew={state.handleAddScheduledScan}
                searchAfter={<OrganizationScheduledScanFilters state={state} />}
                searchPlaceholder={state.tScan("scheduled.searchPlaceholder")}
                searchValue={state.scheduledSearchQuery}
                onSearch={state.commitScheduledSearch}
                isSearching={state.isSearchingScheduledScans}
                addButtonText={state.tScan("scheduled.createTitle")}
                page={state.scheduledPagination.pageIndex + 1}
                pageSize={state.scheduledPagination.pageSize}
                total={state.scheduledPaginationInfo.total}
                totalPages={state.scheduledPaginationInfo.totalPages}
                onPageChange={(page) =>
                  state.handleScheduledPaginationChange({
                    ...state.scheduledPagination,
                    pageIndex: page - 1,
                  })
                }
                onPageSizeChange={(pageSize) =>
                  state.handleScheduledPaginationChange({
                    pageIndex: 0,
                    pageSize,
                  })
                }
                showQuickFilters={false}
                enableRowSelection={false}
              />
            )}
          </TabsContent>
        </Tabs>
      </OrganizationDetailPrimaryTableRegion>
    </div>
  )
}

export function OrganizationDetailActionPanel({
  state,
}: {
  state: OrganizationDetailViewState
}) {
  const activeActionPanel =
    state.isEditDialogOpen ? "edit" :
    state.isAddDialogOpen ? "add-target" :
    null

  if (!activeActionPanel) return null

  if (activeActionPanel === "edit" && state.organization) {
    return (
      <EditOrganizationFormPanel
        organization={state.organization}
        onOpenChange={state.setIsEditDialogOpen}
        onEdit={state.handleEditOrganizationSuccess}
      />
    )
  }

  if (activeActionPanel === "add-target") {
    return (
      <LinkTargetFormPanel
        organizationId={state.organization?.id ?? 0}
        organizationName={state.organization?.name ?? ""}
        onAdd={state.handleAddSuccess}
        onOpenChange={state.setIsAddDialogOpen}
      />
    )
  }

  return null
}

export function OrganizationDetailViewWorkbench({
  state,
}: {
  state: OrganizationDetailViewState
}) {
  const activeActionPanel =
    state.isEditDialogOpen || state.isAddDialogOpen
  const shouldMountActionPanel = useDeferredInteractionMount(activeActionPanel, {
    unmountDelayMs: deferredInteractionUnmountDelayMs,
  })

  return (
    <div className="flex min-h-0 flex-1 flex-col lg:flex-row">
      <div className={cn("min-h-0 flex-1 overflow-y-auto py-5", activeActionPanel && "hidden lg:block")}>
        <OrganizationDetailViewTable state={state} />
      </div>

      {shouldMountActionPanel && activeActionPanel ? (
        <aside className="flex min-h-0 flex-1 overflow-hidden bg-card lg:max-w-lg lg:border-l">
          <OrganizationDetailActionPanel state={state} />
        </aside>
      ) : null}
    </div>
  )
}

export function OrganizationDetailViewDialogs({
  state,
}: {
  state: OrganizationDetailViewState
}) {
  return (
    <>
      <InteractionLoadingDialog
        open={state.pendingInteraction === "edit-dialog"}
        onOpenChange={(open) => {
          if (!open) {
            state.cancelPendingInteraction("edit-dialog")
          }
        }}
        owner="organization-detail-edit-dialog-loading"
        title={state.tScan("editTask")}
      />

      {state.isCreateScheduledScanDialogOpen ? (
        <CreateScheduledScanDialog
          open={state.isCreateScheduledScanDialogOpen}
          onOpenChange={state.setIsCreateScheduledScanDialogOpen}
          presetOrganizationId={state.organizationId}
          presetOrganizationName={state.organization?.name}
          onSuccess={state.handleScheduledScanMutationSuccess}
        />
      ) : null}

      {state.isEditScheduledScanDialogOpen ? (
        <EditScheduledScanDialog
          open={state.isEditScheduledScanDialogOpen}
          onOpenChange={state.setIsEditScheduledScanDialogOpen}
          scheduledScan={state.scheduledScanToEdit}
          onSuccess={state.handleScheduledScanMutationSuccess}
        />
      ) : null}

      <AlertDialog open={state.deleteDialogOpen} onOpenChange={state.setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("unlinkTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("unlinkTargetMessage", { name: state.targetToDelete?.name ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{state.tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={state.confirmDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {state.tConfirm("confirmUnlink")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={state.bulkDeleteDialogOpen} onOpenChange={state.setBulkDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("bulkUnlinkTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("bulkUnlinkTargetMessage", { count: state.selectedTargets.length })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{state.tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={state.confirmBulkDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {state.tConfirm("confirmUnlink")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={state.scheduledScanDeleteDialogOpen}
        onOpenChange={state.setScheduledScanDeleteDialogOpen}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("deleteScheduledScanMessage", {
                name: state.scheduledScanToDelete?.name ?? "",
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{state.tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={state.confirmDeleteScheduledScan}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {state.tCommon("actions.delete")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
