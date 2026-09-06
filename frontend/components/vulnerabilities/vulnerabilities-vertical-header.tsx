"use client"

import React from "react"
import { useTranslations } from "next-intl"
import { CheckCircle2, Circle, X } from "@/components/icons"
import {
  DataTableFacetPanel,
  type DataTableFacetPanelFacet,
  type DataTableFacetedFilterOption,
} from "@/components/shared/data-table/faceted-filter"
import { SharedCompactPagination } from "@/components/shared/data-table/pagination"
import { SearchInput } from "@/components/shared/search-input"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsCountBadge, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { getStatusToneInteractiveOutlineClass, getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
} from "@/types/data-table.types"
import type { VulnerabilitySeverity, VulnerabilitySeverityCounts } from "@/types/vulnerability.types"
import type { ReviewFilter, SeverityFilter } from "./vulnerabilities-data-table"

interface VulnerabilitiesVerticalHeaderProps {
  // Filter state
  reviewFilter: ReviewFilter
  onReviewFilterChange: (filter: ReviewFilter) => void
  severityFilter: SeverityFilter
  onSeverityFilterChange: (filter: SeverityFilter) => void
  filterQuery: string
  onFilterChange: (value: string) => void
  // Counts
  totalCount: number
  pendingCount: number
  reviewedCount: number
  severityCounts?: VulnerabilitySeverityCounts
  // Selection & bulk actions
  selectedCount: number
  onBulkMarkAsReviewed: () => void
  onBulkMarkAsPending: () => void
  onClearSelection: () => void
  // Pagination
  pagination: { pageIndex: number; pageSize: number }
  cursorPaginationSummary: CursorPaginationSummary
  paginationNavigation: CursorPaginationNavigation
  onPaginationChange: (pagination: { pageIndex: number; pageSize: number }) => void
}

export function VulnerabilitiesVerticalHeader({
  reviewFilter,
  onReviewFilterChange,
  severityFilter,
  onSeverityFilterChange,
  filterQuery,
  onFilterChange,
  totalCount,
  pendingCount,
  reviewedCount,
  severityCounts,
  selectedCount,
  onBulkMarkAsReviewed,
  onBulkMarkAsPending,
  onClearSelection,
  pagination,
  cursorPaginationSummary,
  paginationNavigation,
  onPaginationChange,
}: VulnerabilitiesVerticalHeaderProps) {
  const tVuln = useTranslations("vulnerabilities")
  const tSeverity = useTranslations("severity")
  const tActions = useTranslations("common.actions")
  const tDataTable = useTranslations("dataTable")
  const severityOptions: Array<DataTableFacetedFilterOption<VulnerabilitySeverity>> = [
    { value: "critical", label: tSeverity("critical"), count: severityCounts?.critical ?? 0 },
    { value: "high", label: tSeverity("high"), count: severityCounts?.high ?? 0 },
    { value: "medium", label: tSeverity("medium"), count: severityCounts?.medium ?? 0 },
    { value: "low", label: tSeverity("low"), count: severityCounts?.low ?? 0 },
    { value: "info", label: tSeverity("info"), count: severityCounts?.info ?? 0 },
  ]
  const severityFacetPanelItems: DataTableFacetPanelFacet[] = [{
    id: "severity",
    label: tVuln("severityLabel"),
    values: severityFilter,
    onValuesChange: (values) => onSeverityFilterChange(values as SeverityFilter),
    options: severityOptions,
    emptyLabel: tVuln("reviewStatus.all"),
    clearLabel: tDataTable("clearFilter"),
  }]

  return (
    <div className="flex flex-col gap-3 px-1 py-1 shrink-0 md:flex-row md:items-start md:justify-between md:gap-4">
      {/* Left: Filters or Bulk Actions */}
      {selectedCount > 0 ? (
        <div className="animate-in duration-200 fade-in flex flex-wrap gap-2 items-center md:gap-3 min-w-0">
          <Badge variant="count" className="h-7 px-2 text-xs">
            {tVuln("selected", { count: selectedCount })}
          </Badge>
          <div className="bg-border/60 h-4 mx-1 w-px" />
          <Button
            variant="outline"
            size="sm"
            className={cn(
              "gap-2",
              getStatusToneInteractiveOutlineClass("success")
            )}
            onClick={onBulkMarkAsReviewed}
          >
            <CheckCircle2 className={cn("h-3.5 w-3.5", getStatusToneTextClass("success"))} />
            {tVuln("markAsReviewed")}
          </Button>
          <Button
            variant="outline"
            size="sm"
            className={cn(
              "gap-2",
              getStatusToneInteractiveOutlineClass("muted")
            )}
            onClick={onBulkMarkAsPending}
          >
            <Circle className={cn("h-3.5 w-3.5", getStatusToneTextClass("muted"))} />
            {tVuln("markAsPending")}
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={onClearSelection}
            className="hover:text-foreground text-muted-foreground"
            aria-label={tActions("deselectAll")}
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>
      ) : (
        <Tabs value={reviewFilter} onValueChange={(v) => onReviewFilterChange(v as ReviewFilter)} className="md:w-auto overflow-x-auto w-full">
          <TabsList variant="filter" size="sm" className="md:min-w-0 md:w-auto min-w-full w-max">
            <TabsTrigger variant="filter" size="sm" value="all" className="px-3 whitespace-nowrap">
              <span className={cn("inline-flex items-center leading-none", textRole.tab)}>{tVuln("reviewStatus.all")}</span>
              <TabsCountBadge>{totalCount}</TabsCountBadge>
            </TabsTrigger>
            <TabsTrigger variant="filter" size="sm" value="pending" className="px-3 whitespace-nowrap">
              <span className={cn("inline-flex items-center leading-none", textRole.tab)}>{tVuln("reviewStatus.pending")}</span>
              <TabsCountBadge>{pendingCount}</TabsCountBadge>
            </TabsTrigger>
            <TabsTrigger variant="filter" size="sm" value="reviewed" className="px-3 whitespace-nowrap">
              <span className={cn("inline-flex items-center leading-none", textRole.tab)}>{tVuln("reviewStatus.reviewed")}</span>
              <TabsCountBadge>{reviewedCount}</TabsCountBadge>
            </TabsTrigger>
          </TabsList>
        </Tabs>
      )}

      {/* Right: Search + Severity Filter + Pagination */}
      <div className="flex gap-2 items-center md:flex-1 md:gap-2 md:justify-end min-w-0 w-full">
        <div className="flex-1 md:max-w-sm md:ml-auto w-full">
          <SearchInput
            value={filterQuery}
            onChange={(e) => onFilterChange(e.target.value)}
            placeholder={tActions("search")}
            toolbarDensity="compact"
          />
        </div>

        <DataTableFacetPanel
          title={tDataTable("filter")}
          facets={severityFacetPanelItems}
          activeCount={severityFilter.length}
        />

        <div className="shrink-0">
          <SharedCompactPagination
            mode="cursor"
            pageSize={pagination.pageSize}
            canFirstPage={paginationNavigation.canFirstPage}
            canPreviousPage={paginationNavigation.canPreviousPage}
            canNextPage={paginationNavigation.canNextPage}
            onFirstPage={() => onPaginationChange({ ...pagination, pageIndex: 0 })}
            onPreviousPage={() => onPaginationChange({ ...pagination, pageIndex: pagination.pageIndex - 1 })}
            onNextPage={() => onPaginationChange({ ...pagination, pageIndex: pagination.pageIndex + 1 })}
            onPageSizeChange={(pageSize) => onPaginationChange({ pageIndex: 0, pageSize })}
            summary={tDataTable("total", { count: cursorPaginationSummary.total })}
            className="px-0"
          />
        </div>
      </div>
    </div>
  )
}
