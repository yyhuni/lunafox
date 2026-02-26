"use client"

import React, { useState, useMemo, useCallback } from "react"
import { useTranslations } from "next-intl"
import { useVerticalResize } from "@/hooks/use-vertical-resize"
import { useVulnerabilitiesDetailViewState } from "./vulnerabilities-detail-view-state"
import { VulnerabilitiesVerticalHeader } from "./vulnerabilities-vertical-header"
import { VulnerabilitiesVerticalTable } from "./vulnerabilities-vertical-table"
import { VerticalResizeHandle } from "./vulnerabilities-vertical-resize-handle"
import { VulnerabilityVerticalDetail } from "./vulnerability-vertical-detail"
import { DataTableSkeleton } from "@/components/ui/data-table-skeleton"
import { Info } from "@/components/icons"
import {
  VulnerabilitiesDetailViewDialogs,
} from "./vulnerabilities-detail-view-sections"

import type { Vulnerability } from "@/types/vulnerability.types"

interface VulnerabilitiesVerticalViewProps {
  scanId?: number
  targetId?: number
}

export function VulnerabilitiesVerticalView({
  scanId,
  targetId,
}: VulnerabilitiesVerticalViewProps) {
  const state = useVulnerabilitiesDetailViewState({ scanId, targetId })
  const tVuln = useTranslations("vulnerabilities")

  // Vertical resize
  const { height, startResizing, containerRef, isResizing } = useVerticalResize(45)

  // Active vulnerability for bottom detail panel (separate from dialog selection)
  const [activeVulnerability, setActiveVulnerability] = useState<Vulnerability | null>(null)

  // Auto-select first vulnerability when data loads
  const displayedVulnerability = useMemo(() => {
    if (activeVulnerability) {
      // Ensure the active vulnerability still exists in data
      const exists = state.vulnerabilities.find(v => v.id === activeVulnerability.id)
      if (exists) return exists
    }
    return state.vulnerabilities[0] ?? null
  }, [activeVulnerability, state.vulnerabilities])

  const handleSelectVulnerability = useCallback((vulnerability: Vulnerability) => {
    setActiveVulnerability(vulnerability)
  }, [])

  const handleClearSelection = useCallback(() => {
    state.setSelectedVulnerabilities([])
  }, [state])

  // Loading state
  if ((state.isLoading || state.isQueryLoading) && !state.activeQuery.data) {
    return <DataTableSkeleton toolbarButtonCount={2} rows={6} columns={6} />
  }

  return (
    <>
      <div ref={containerRef} className="flex flex-col h-full min-h-0 bg-background border rounded-lg overflow-hidden select-none">
        {/* Top: List View */}
        <div
          className="flex flex-col border-b overflow-hidden relative min-h-[200px]"
          style={{ height: `${height}%` }}
        >
          <VulnerabilitiesVerticalHeader
            reviewFilter={state.reviewFilter}
            onReviewFilterChange={state.handleReviewFilterChange}
            severityFilter={state.severityFilter}
            onSeverityFilterChange={state.handleSeverityFilterChange}
            filterQuery={state.filterQuery}
            onFilterChange={state.handleFilterChange}
            totalCount={state.pendingCount + state.reviewedCount}
            pendingCount={state.pendingCount}
            reviewedCount={state.reviewedCount}
            selectedCount={state.selectedVulnerabilities.length}
            onBulkMarkAsReviewed={state.handleBulkMarkAsReviewed}
            onBulkMarkAsPending={state.handleBulkMarkAsPending}
            onClearSelection={handleClearSelection}
            pagination={state.pagination}
            paginationInfo={state.paginationInfo}
            onPaginationChange={state.handlePaginationChange}
          />

          <VulnerabilitiesVerticalTable
            items={state.vulnerabilities}
            selectedId={displayedVulnerability?.id ?? null}
            selectedRows={state.selectedVulnerabilities}
            onSelect={handleSelectVulnerability}
            onSelectionChange={state.setSelectedVulnerabilities}
          />

          <VerticalResizeHandle isResizing={isResizing} onMouseDown={startResizing} />
        </div>

        {/* Bottom: Detail View */}
        <div className="flex-1 flex flex-col min-h-0 bg-background relative overflow-hidden">
          {displayedVulnerability ? (
            <VulnerabilityVerticalDetail
              vulnerability={displayedVulnerability}
              className="h-full"
            />
          ) : (
            <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
              <Info className="h-10 w-10 mb-4 opacity-30" />
              <p className="text-sm">{tVuln("emptyDetail")}</p>
            </div>
          )}
        </div>
      </div>

      <VulnerabilitiesDetailViewDialogs state={state} />
    </>
  )
}
