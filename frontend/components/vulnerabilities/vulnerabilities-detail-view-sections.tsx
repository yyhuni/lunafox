"use client"

import { useTranslations } from "next-intl"
import {
  AlertDialog,
  AlertDialogClose, AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsCountBadge, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { VulnerabilitiesDataTable } from "./vulnerabilities-data-table"
import { createVulnerabilityColumns, type VulnerabilityTranslations } from "./vulnerabilities-columns"

import type { VulnerabilitiesDetailViewState } from "./vulnerabilities-detail-view-state"
import type { Vulnerability } from "@/types/vulnerability.types"

const vulnerabilityFallbackTranslations: VulnerabilityTranslations = {
  columns: {
    status: "",
    severity: "",
    source: "",
    vulnType: "",
    url: "",
    createdAt: "",
  },
  actions: {
    selectAll: "",
    selectRow: "",
  },
  tooltips: {
    vulnDetails: "",
    reviewed: "",
    pending: "",
  },
  severity: {
    critical: "",
    high: "",
    medium: "",
    low: "",
    info: "",
  },
}

const VULNERABILITIES_ROUTE_FALLBACK_PAGE_SIZE = 10

function VulnerabilitiesRouteFallbackReviewTabs() {
  const tVulnerabilities = useTranslations("vulnerabilities")

  return (
    <div data-slot="vulnerabilities-review-tabs-loading-state">
      <Tabs value="all">
        <TabsList variant="content" size="sm" aria-hidden="true">
          {[
            {
              value: "all",
              label: tVulnerabilities("reviewStatus.allVulnerabilities"),
              countWidth: "w-7",
            },
            {
              value: "pending",
              label: tVulnerabilities("reviewStatus.pending"),
              countWidth: "w-6",
            },
            {
              value: "reviewed",
              label: tVulnerabilities("reviewStatus.reviewed"),
              countWidth: "w-6",
            },
          ].map((item) => (
            <TabsTrigger key={item.value} variant="content" value={item.value} size="sm" disabled>
              {item.label}
              <TabsCountBadge>
                <Skeleton className={`${item.countWidth} h-3 rounded-full`} />
              </TabsCountBadge>
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
    </div>
  )
}

function VulnerabilitiesRouteFallbackTable({ rows }: { rows: number }) {
  const columns = createVulnerabilityColumns({
    formatDate: (value) => value,
    t: vulnerabilityFallbackTranslations,
  })

  return (
    <VulnerabilitiesDataTable
      data={[]}
      columns={columns}
      filterValue=""
      onFilterChange={() => {}}
      pagination={{ pageIndex: 0, pageSize: VULNERABILITIES_ROUTE_FALLBACK_PAGE_SIZE }}
      paginationNavigation={{ mode: "cursor", canFirstPage: false, canPreviousPage: false, canNextPage: false }}
      cursorPaginationSummary={{ total: 0 }}
      onPaginationChange={() => {}}
      onSelectionChange={() => {}}
      severityFilter={[]}
      onSeverityFilterChange={() => {}}
      selectedRows={[]}
      loading
      loadingRowCount={rows}
    />
  )
}

export function VulnerabilitiesDetailViewLoadingState({
  state,
  rowCount,
  hideToolbar,
  onRowClick,
}: {
  state: VulnerabilitiesDetailViewState
  rowCount: number
  hideToolbar: boolean
  onRowClick?: (vulnerability: Vulnerability) => void
}) {
  return (
    <VulnerabilitiesDataTable
      data={[]}
      columns={state.vulnerabilityColumns}
      filterValue={state.filterQuery}
      onFilterChange={state.commitFilterSearch}
      pagination={state.pagination}
      cursorPaginationSummary={state.cursorPaginationSummary}
      paginationNavigation={state.paginationNavigation}
      onPaginationChange={state.handlePaginationChange}
      sortingMode="server"
      sorting={state.sorting}
      onSortingChange={state.handleSortingChange}
      onSelectionChange={state.isReadOnly ? undefined : state.setSelectedVulnerabilities}
      hideToolbar={hideToolbar}
      reviewFilter={state.reviewFilter}
      onReviewFilterChange={state.isReadOnly ? undefined : state.onReviewFilterChange}
      pendingCount={state.pendingCount}
      reviewedCount={state.reviewedCount}
      severityFilter={state.severityFilter}
      onSeverityFilterChange={state.handleSeverityFilterChange}
      severityCounts={state.severityCounts}
      sourceFilter={state.sourceFilter}
      onSourceFilterChange={state.handleSourceFilterChange}
      sourceOptions={state.sourceOptions}
      vulnTypeFilter={state.vulnTypeFilter}
      onVulnTypeFilterChange={state.handleVulnTypeFilterChange}
      vulnTypeOptions={state.vulnTypeOptions}
      selectedRows={[]}
      onBulkMarkAsReviewed={state.isReadOnly ? undefined : state.handleBulkMarkAsReviewed}
      onBulkMarkAsPending={state.isReadOnly ? undefined : state.handleBulkMarkAsPending}
      onBulkDelete={state.isReadOnly ? undefined : state.handleOpenBulkDeleteDialog}
      onRowClick={onRowClick}
      loading
      loadingRowCount={rowCount}
    />
  )
}

export function VulnerabilitiesDetailViewRouteFallback({ rowCount }: { rowCount: number }) {
  return (
    <div className="space-y-4">
      <VulnerabilitiesRouteFallbackReviewTabs />
      <VulnerabilitiesRouteFallbackTable rows={rowCount} />
    </div>
  )
}

export function VulnerabilitiesDetailViewContent({
  state,
  hideToolbar,
  onRowClick,
}: {
  state: VulnerabilitiesDetailViewState
  hideToolbar: boolean
  onRowClick?: (vulnerability: Vulnerability) => void
}) {
  return (
    <>
      <VulnerabilitiesDataTable
        data={state.vulnerabilities}
        columns={state.vulnerabilityColumns}
        filterValue={state.filterQuery}
        onFilterChange={state.commitFilterSearch}
        pagination={state.pagination}
        cursorPaginationSummary={state.cursorPaginationSummary}
        paginationNavigation={state.paginationNavigation}
        onPaginationChange={state.handlePaginationChange}
        sortingMode="server"
        sorting={state.sorting}
        onSortingChange={state.handleSortingChange}
        onSelectionChange={state.isReadOnly ? undefined : state.setSelectedVulnerabilities}
        hideToolbar={hideToolbar}
        reviewFilter={state.reviewFilter}
        onReviewFilterChange={state.isReadOnly ? undefined : state.onReviewFilterChange}
        pendingCount={state.pendingCount}
        reviewedCount={state.reviewedCount}
        severityFilter={state.severityFilter}
        onSeverityFilterChange={state.handleSeverityFilterChange}
        severityCounts={state.severityCounts}
        sourceFilter={state.sourceFilter}
        onSourceFilterChange={state.handleSourceFilterChange}
        sourceOptions={state.sourceOptions}
        vulnTypeFilter={state.vulnTypeFilter}
        onVulnTypeFilterChange={state.handleVulnTypeFilterChange}
        vulnTypeOptions={state.vulnTypeOptions}
        selectedRows={state.selectedVulnerabilities}
        onBulkMarkAsReviewed={state.isReadOnly ? undefined : state.handleBulkMarkAsReviewed}
        onBulkMarkAsPending={state.isReadOnly ? undefined : state.handleBulkMarkAsPending}
        onBulkDelete={state.isReadOnly ? undefined : state.handleOpenBulkDeleteDialog}
        onRowClick={onRowClick}
      />
    </>
  )
}

export function VulnerabilitiesDetailViewDialogs({ state }: { state: VulnerabilitiesDetailViewState }) {
  if (state.isReadOnly) return null

  return (
    <>
      <AlertDialog open={state.deleteDialogOpen} onOpenChange={state.setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("deleteVulnMessage", { name: state.vulnerabilityToDelete?.vulnType ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{state.tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={state.confirmDelete}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
            >
              {state.tCommon("actions.delete")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={state.bulkDeleteDialogOpen} onOpenChange={state.setBulkDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state.tConfirm("bulkDeleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {state.tConfirm("bulkDeleteVulnMessage", { count: state.selectedVulnerabilities.length })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose variant="outline">{state.tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose
              onClick={state.confirmBulkDelete}
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
            >
              {state.tConfirm("confirmDelete")}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
