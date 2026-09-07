"use client"

import React, { useCallback, useState } from "react"

import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import {
  deferredInteractionUnmountDelayMs,
  useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount"
import { useVulnerabilitiesDetailViewState } from "./vulnerabilities-detail-view-state"
import { VulnerabilitiesDataTable } from "./vulnerabilities-data-table"
import { VulnerabilityStatCards, VulnerabilityStatCardsLoadingState } from "./vulnerability-stat-cards"
import { VulnerabilityDetailDrawer } from "./vulnerability-detail-drawer"
import {
  VulnerabilitiesDetailViewDialogs,
} from "./vulnerabilities-detail-view-sections"

import type { Vulnerability } from "@/types/vulnerability.types"

interface VulnerabilitiesVerticalViewProps {
  scanId?: number
  targetId?: number
}

const VULNERABILITIES_LOADING_SLOTS = {
  toolbar: "vulnerabilities-table-toolbar",
  body: "vulnerabilities-rows",
  pagination: "vulnerabilities-pagination",
}

export function VulnerabilitiesVerticalView({
  scanId,
  targetId,
}: VulnerabilitiesVerticalViewProps) {
  const [activeVulnerability, setActiveVulnerability] = useState<Vulnerability | null>(null)

  const handleSelectVulnerability = useCallback((vulnerability: Vulnerability) => {
    setActiveVulnerability(vulnerability)
  }, [])

  const state = useVulnerabilitiesDetailViewState({ scanId, targetId })

  const handleDetailOpenChange = useCallback((open: boolean) => {
    if (!open) {
      setActiveVulnerability(null)
    }
  }, [])
  const shouldMountDetailDrawer = useDeferredInteractionMount(Boolean(activeVulnerability), {
    unmountDelayMs: deferredInteractionUnmountDelayMs,
  })
  const isInitialLoading = (state.isLoading || state.isQueryLoading) && !state.activeQuery.data
  const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)

  const loadingState = (
    <div className="flex min-h-0 flex-1 flex-col gap-4 md:gap-6">
      <div {...getLoadingStructureSlotAttributes("vulnerabilities-severity-summary")}>
        <VulnerabilityStatCardsLoadingState />
      </div>
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
        onSelectionChange={state.setSelectedVulnerabilities}
        hideToolbar={false}
        reviewFilter={state.reviewFilter}
        onReviewFilterChange={state.onReviewFilterChange}
        totalCount={state.totalCount}
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
        onBulkMarkAsReviewed={state.handleBulkMarkAsReviewed}
        onBulkMarkAsPending={state.handleBulkMarkAsPending}
        onBulkDelete={state.handleOpenBulkDeleteDialog}
        onRowClick={handleSelectVulnerability}
        loading
        loadingRowCount={loadingRowCount}
        stableSurfaceRowCount={loadingRowCount}
        loadingSlots={VULNERABILITIES_LOADING_SLOTS}
      />
    </div>
  )

  return (
    <ContentHandoff
      owner="vulnerabilities-vertical-view-content"
      layer="workspace"
      isLoading={isInitialLoading}
      skeleton={loadingState}
      className="flex min-h-0 flex-1 flex-col"
      skeletonClassName="flex min-h-0 flex-1 flex-col"
      contentClassName="flex min-h-0 flex-1 flex-col"
    >
      <div className="flex min-h-0 flex-1 flex-col gap-4 md:gap-6">
        <div {...getLoadingStructureSlotAttributes("vulnerabilities-severity-summary")}>
          <VulnerabilityStatCards
            counts={state.severityCounts}
            pendingCount={state.pendingCount}
            reviewedCount={state.reviewedCount}
          />
        </div>

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
          onSelectionChange={state.setSelectedVulnerabilities}
          hideToolbar={false}
          reviewFilter={state.reviewFilter}
          onReviewFilterChange={state.onReviewFilterChange}
          totalCount={state.totalCount}
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
          onBulkMarkAsReviewed={state.handleBulkMarkAsReviewed}
          onBulkMarkAsPending={state.handleBulkMarkAsPending}
          onBulkDelete={state.handleOpenBulkDeleteDialog}
          onRowClick={handleSelectVulnerability}
          stableSurfaceRowCount={loadingRowCount}
          loadingSlots={VULNERABILITIES_LOADING_SLOTS}
        />

        {shouldMountDetailDrawer ? (
          <VulnerabilityDetailDrawer
            vulnerability={activeVulnerability}
            open={Boolean(activeVulnerability)}
            onOpenChange={handleDetailOpenChange}
          />
        ) : null}

        <VulnerabilitiesDetailViewDialogs state={state} />
      </div>
    </ContentHandoff>
  )
}
