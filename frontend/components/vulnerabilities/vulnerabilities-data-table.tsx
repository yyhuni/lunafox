"use client"

import * as React from "react"
import type { ColumnDef, SortingState } from "@tanstack/react-table"
import { useLocale, useTranslations } from "next-intl"
import { CheckCircle2, Circle, semanticIcons } from "@/components/icons"
import { BusinessListDataTable } from "@/components/shared/data-table/business-list-data-table"
import { buildExportOptions } from "@/components/shared/data-table/data-table-export-helpers"
import {
  DataTableFacetPanel,
  type DataTableFacetPanelFacet,
  type DataTableFacetedFilterOption,
} from "@/components/shared/data-table/faceted-filter"
import {
  SelectedRowActionBar,
  type SelectedRowActionBarAction,
} from "@/components/shared/data-table/selected-row-action-bar"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import { useSimpleSearchState } from "@/components/shared/data-table/use-simple-search"
import { TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX } from "@/components/ui/table"
import { Tabs, TabsCountBadge, TabsList, TabsTrigger } from "@/components/ui/tabs"
import type { Vulnerability, VulnerabilitySeverity, VulnerabilitySeverityCounts } from "@/types/vulnerability.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  DataTableLoadingSlots,
  DataTableSortingMode,
} from "@/types/data-table.types"

// Review filter type
export type ReviewFilter = "all" | "pending" | "reviewed"

export type SeverityFilter = VulnerabilitySeverity[]
export type VulnerabilityTextFilter = string[]

const DeleteIcon = semanticIcons.action.delete


interface VulnerabilitiesDataTableProps {
  data: Vulnerability[]
  columns: ColumnDef<Vulnerability>[]
  filterValue?: string
  onFilterChange?: (value: string) => void
  pagination?: { pageIndex: number; pageSize: number }
  setPagination?: React.Dispatch<React.SetStateAction<{ pageIndex: number; pageSize: number }>>
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  sortingMode?: DataTableSortingMode
  sorting?: SortingState
  onSortingChange?: (sorting: SortingState) => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: Vulnerability[]) => void
  onExportAll?: () => void
  onExportSelected?: () => void
  hideToolbar?: boolean
  // Review status props
  reviewFilter?: ReviewFilter
  onReviewFilterChange?: (filter: ReviewFilter) => void
  totalCount?: number
  pendingCount?: number
  reviewedCount?: number
  selectedRows?: Vulnerability[]
  onBulkMarkAsReviewed?: () => void
  onBulkMarkAsPending?: () => void
  onRowClick?: (vulnerability: Vulnerability) => void
  // New: severity filter
  severityFilter?: SeverityFilter
  onSeverityFilterChange?: (filter: SeverityFilter) => void
  severityCounts?: VulnerabilitySeverityCounts
  sourceFilter?: VulnerabilityTextFilter
  onSourceFilterChange?: (filter: VulnerabilityTextFilter) => void
  sourceOptions?: Array<DataTableFacetedFilterOption<string>>
  vulnTypeFilter?: VulnerabilityTextFilter
  onVulnTypeFilterChange?: (filter: VulnerabilityTextFilter) => void
  vulnTypeOptions?: Array<DataTableFacetedFilterOption<string>>
  loading?: boolean
  loadingRowCount?: number
  stableSurfaceRowCount?: number
  loadingSlots?: DataTableLoadingSlots
}

function mergeSelectedFilterOptions(
  options: Array<DataTableFacetedFilterOption<string>>,
  selected: string[]
) {
  const optionByValue = new Map(options.map((option) => [option.value, option]))
  for (const value of selected) {
    const trimmed = value.trim()
    if (trimmed && !optionByValue.has(trimmed)) {
      optionByValue.set(trimmed, { value: trimmed, label: trimmed })
    }
  }
  return Array.from(optionByValue.values()).sort((left, right) =>
    left.label.localeCompare(right.label, undefined, { numeric: true })
  )
}

export function VulnerabilitiesDataTable({
  data = [],
  columns,
  filterValue,
  onFilterChange,
  pagination,
  setPagination,
  cursorPaginationSummary,
  paginationNavigation,
  onPaginationChange,
  sortingMode = "none",
  sorting,
  onSortingChange,
  onBulkDelete,
  onSelectionChange,
  onExportAll,
  onExportSelected,
  hideToolbar = false,
  reviewFilter = "all",
  onReviewFilterChange,
  totalCount,
  pendingCount = 0,
  reviewedCount = 0,
  selectedRows = [],
  onBulkMarkAsReviewed,
  onBulkMarkAsPending,
  onRowClick,
  severityFilter = [],
  onSeverityFilterChange,
  severityCounts,
  sourceFilter = [],
  onSourceFilterChange,
  sourceOptions = [],
  vulnTypeFilter = [],
  onVulnTypeFilterChange,
  vulnTypeOptions = [],
  loading = false,
  loadingRowCount,
  stableSurfaceRowCount,
  loadingSlots,
}: VulnerabilitiesDataTableProps) {
  const t = useTranslations("common.status")
  const tExport = useTranslations("common.export")
  const tActions = useTranslations("common.actions")
  const tDataTable = useTranslations("dataTable")
  const tVuln = useTranslations("vulnerabilities")
  const tColumns = useTranslations("columns")
  const tSeverity = useTranslations("severity")
  const locale = useLocale()
  const {
    value: localSearchValue,
    handleSearchInputChange,
    commitNow: commitSearch,
  } = useSimpleSearchState({ searchValue: filterValue, onSearch: onFilterChange })

  const exportOptions = buildExportOptions(tExport, {
    onExportAll,
    onExportSelected,
  })

  const severityOptions: Array<DataTableFacetedFilterOption<VulnerabilitySeverity>> = [
    { value: "critical", label: tSeverity("critical"), count: severityCounts?.critical ?? 0 },
    { value: "high", label: tSeverity("high"), count: severityCounts?.high ?? 0 },
    { value: "medium", label: tSeverity("medium"), count: severityCounts?.medium ?? 0 },
    { value: "low", label: tSeverity("low"), count: severityCounts?.low ?? 0 },
    { value: "info", label: tSeverity("info"), count: severityCounts?.info ?? 0 },
  ]
  const mergedSourceOptions = React.useMemo(
    () => mergeSelectedFilterOptions(sourceOptions, sourceFilter),
    [sourceFilter, sourceOptions]
  )
  const mergedVulnTypeOptions = React.useMemo(
    () => mergeSelectedFilterOptions(vulnTypeOptions, vulnTypeFilter),
    [vulnTypeFilter, vulnTypeOptions]
  )
  const vulnerabilityFacetPanelItems: DataTableFacetPanelFacet[] = []
  if (onSeverityFilterChange) {
    vulnerabilityFacetPanelItems.push({
      id: "severity",
      label: tVuln("severityLabel"),
      values: severityFilter,
      onValuesChange: (values) => onSeverityFilterChange(values as SeverityFilter),
      options: severityOptions,
      emptyLabel: tVuln("reviewStatus.all"),
      clearLabel: tDataTable("clearFilter"),
    })
  }
  if (onSourceFilterChange) {
    vulnerabilityFacetPanelItems.push({
      id: "source",
      label: tColumns("vulnerability.source"),
      values: sourceFilter,
      onValuesChange: onSourceFilterChange,
      options: mergedSourceOptions,
      emptyLabel: tVuln("reviewStatus.all"),
      clearLabel: tDataTable("clearFilter"),
    })
  }
  if (onVulnTypeFilterChange) {
    vulnerabilityFacetPanelItems.push({
      id: "vulnType",
      label: tColumns("vulnerability.vulnType"),
      values: vulnTypeFilter,
      onValuesChange: onVulnTypeFilterChange,
      options: mergedVulnTypeOptions,
      emptyLabel: tVuln("reviewStatus.all"),
      clearLabel: tDataTable("clearFilter"),
    })
  }
  const activeFacetFilterCount = severityFilter.length + sourceFilter.length + vulnTypeFilter.length

  // Left toolbar content - smart filter + faceted filters
  const leftToolbarContent = (
    <div className="flex w-full flex-wrap items-center gap-2 sm:flex-1">
      <SimpleSearchToolbar
        placeholder={tActions("searchURL")}
        value={localSearchValue}
        onChange={handleSearchInputChange}
        onSubmit={commitSearch}
        className="w-full sm:w-auto"
        toolbarDensity="compact"
      />
      {vulnerabilityFacetPanelItems.length > 0 ? (
        <DataTableFacetPanel
          title={tDataTable("filter")}
          facets={vulnerabilityFacetPanelItems}
          activeCount={activeFacetFilterCount}
        />
      ) : null}
    </div>
  )

  // Floating action bar for bulk operations
  const selectedCountContent = locale.startsWith("zh") ? (
    <>
      <span>已选</span>
      <span className="px-1 text-highlight">{selectedRows.length}</span>
      <span>项</span>
    </>
  ) : (
    <>
      <span className="text-highlight">{selectedRows.length}</span>
      <span> </span>
      <span>selected</span>
    </>
  )

  const hasSelectedRowActions = onBulkMarkAsReviewed || onBulkMarkAsPending || onBulkDelete
  const selectedRowActions: SelectedRowActionBarAction[] = [
    ...(onBulkMarkAsReviewed
      ? [{
          key: "mark-reviewed",
          label: tVuln("markAsReviewed"),
          icon: CheckCircle2,
          tone: "success" as const,
          group: "status",
          onClick: onBulkMarkAsReviewed,
        }]
      : []),
    ...(onBulkMarkAsPending
      ? [{
          key: "mark-pending",
          label: tVuln("markAsPending"),
          icon: Circle,
          tone: "muted" as const,
          group: "status",
          onClick: onBulkMarkAsPending,
        }]
      : []),
    ...(onBulkDelete
      ? [{
          key: "delete",
          label: tActions("delete"),
          icon: DeleteIcon,
          tone: "destructive" as const,
          group: "danger",
          onClick: onBulkDelete,
        }]
      : []),
  ]

  return (
    <>
      <div className="space-y-4">
        {onReviewFilterChange && (
          <VulnerabilityReviewTabs
            value={reviewFilter}
            onValueChange={onReviewFilterChange}
            totalCount={totalCount ?? cursorPaginationSummary?.total ?? data.length}
            pendingCount={pendingCount}
            reviewedCount={reviewedCount}
          />
        )}

        <BusinessListDataTable
          data={data}
          columns={columns}
          getRowId={(row) => String(row.id)}
          state={{
            pagination,
            setPagination,
            cursorPaginationSummary,
            paginationNavigation,
            onPaginationChange,
            sortingMode,
            sorting,
            onSortingChange,
            onSelectionChange,
            selectedRows,
          }}
          behavior={{
            expandColumnIds: ["vulnType", "url"],
            onRowClick: onRowClick
              ? (row) => {
                  onRowClick(row as Vulnerability)
                }
              : undefined,
            getRowActionLabel: (row) => {
              const vulnerability = row as Vulnerability
              return tVuln("openDetailFor", { name: vulnerability.vulnType })
            },
          }}
          actions={{
            showAddButton: false,
            exportOptions: exportOptions.length > 0 ? exportOptions : undefined,
          }}
          ui={{
            toolbarLeft: leftToolbarContent,
            hideToolbar,
            emptyMessage: t("noData"),
            loading,
            loadingPresentation: loading ? "initial" : undefined,
            initialLoadingToolbarFilterCount: vulnerabilityFacetPanelItems.length > 0 ? 1 : 0,
            loadingRowCount,
            loadingRowHeightEstimate: TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX,
            stableSurfaceRowCount,
            loadingSlots,
          }}
        />
      </div>
      {hasSelectedRowActions && (
        <SelectedRowActionBar
          selectedCount={selectedRows.length}
          ariaLabel={tVuln("selected", { count: selectedRows.length })}
          countLabel={selectedCountContent}
          actions={selectedRowActions}
          onClearSelection={onSelectionChange ? () => onSelectionChange([]) : undefined}
          clearSelectionLabel={tActions("deselectAll")}
        />
      )}
    </>
  )
}

function VulnerabilityReviewTabs({
  value,
  onValueChange,
  totalCount,
  pendingCount,
  reviewedCount,
}: {
  value: ReviewFilter
  onValueChange: (filter: ReviewFilter) => void
  totalCount?: number
  pendingCount: number
  reviewedCount: number
}) {
  const tVuln = useTranslations("vulnerabilities")

  return (
    <Tabs value={value} onValueChange={(nextValue) => onValueChange(nextValue as ReviewFilter)}>
      <TabsList variant="content" size="sm">
        <TabsTrigger variant="content" value="all" size="sm">
          {tVuln("reviewStatus.allVulnerabilities")}
          <TabsCountBadge>{totalCount ?? 0}</TabsCountBadge>
        </TabsTrigger>
        <TabsTrigger variant="content" value="pending" size="sm">
          {tVuln("reviewStatus.pending")}
          <TabsCountBadge>{pendingCount}</TabsCountBadge>
        </TabsTrigger>
        <TabsTrigger variant="content" value="reviewed" size="sm">
          {tVuln("reviewStatus.reviewed")}
          <TabsCountBadge>{reviewedCount}</TabsCountBadge>
        </TabsTrigger>
      </TabsList>
    </Tabs>
  )
}
