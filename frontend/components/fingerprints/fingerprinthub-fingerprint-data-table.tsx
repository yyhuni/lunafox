"use client"

import * as React from "react"
import type { ColumnDef, SortingState } from "@tanstack/react-table"
import { useTranslations } from "next-intl"
import { useFingerprintTableActions } from "@/components/fingerprints/fingerprint-table-actions"
import { BusinessListDataTable } from "@/components/shared/data-table"
import type { BusinessListQuery } from "@/components/shared/data-table/business-list-query"
import {
  DataTableFacetedFilter,
  DataTableFacetedFilterGroup,
  type DataTableFacetedFilterOption,
} from "@/components/shared/data-table/faceted-filter"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import { useSimpleSearchState } from "@/components/shared/data-table/use-simple-search"
import type { FingerPrintHubFingerprint, FingerprintFilterOption } from "@/types/fingerprint.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  PaginationState,
} from "@/types/data-table.types"

const FINGERPRINTHUB_FINGERPRINT_LOADING_SLOTS = {
  toolbar: "fingerprinthub-fingerprint-toolbar",
  body: "fingerprinthub-fingerprint-rows",
  pagination: "fingerprinthub-fingerprint-pagination",
}

interface FingerPrintHubFingerprintDataTableProps {
  data: FingerPrintHubFingerprint[]
  columns: ColumnDef<FingerPrintHubFingerprint>[]
  query: BusinessListQuery
  filterOptions?: Partial<Record<"severity", FingerprintFilterOption[]>>
  onSearchChange?: (value: string) => void
  isSearching?: boolean
  onFacetChange?: (field: "severity", values: string[]) => void
  onSortingChange?: (sorting: SortingState) => void
  onSelectionChange?: (selectedRows: FingerPrintHubFingerprint[]) => void
  onRowClick?: (fingerprint: FingerPrintHubFingerprint) => void
  selectedRows?: FingerPrintHubFingerprint[]
  onImport?: () => void
  onExport?: () => void
  onBulkDelete?: () => void
  onDeleteAll?: () => void
  totalCount?: number
  pagination?: PaginationState
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  onPaginationChange?: (pagination: PaginationState) => void
  loading?: boolean
  initialLoading?: boolean
  loadingRowCount?: number
  stableSurfaceRowCount?: number
}

function mergeSelectedFacetOptions(
  options: DataTableFacetedFilterOption<string>[],
  selectedValues: string[],
  unsetLabel: string
) {
  const byValue = new Map(options.map((option) => [option.value, option]))
  for (const value of selectedValues) {
    if (!byValue.has(value)) {
      byValue.set(value, { value, label: value === "__unset__" ? unsetLabel : value })
    }
  }
  return Array.from(byValue.values())
}

export function FingerPrintHubFingerprintDataTable({
  data = [],
  columns,
  query,
  filterOptions = {},
  onSearchChange,
  isSearching = false,
  onFacetChange,
  onSortingChange,
  onSelectionChange,
  onRowClick,
  selectedRows,
  onImport,
  onExport,
  onBulkDelete,
  onDeleteAll,
  totalCount,
  pagination,
  cursorPaginationSummary,
  paginationNavigation,
  onPaginationChange,
  loading = false,
  initialLoading = false,
  loadingRowCount,
  stableSurfaceRowCount,
}: FingerPrintHubFingerprintDataTableProps) {
  const t = useTranslations("common.status")
  const tActions = useTranslations("common.actions")
  const tDataTable = useTranslations("dataTable")
  const tFingerprints = useTranslations("tools.fingerprints")
  const tForm = useTranslations("tools.fingerprints.form")
  const unsetLabel = tFingerprints("unset")
  const {
    value: searchValue,
    handleSearchInputChange,
    commitNow: commitSearch,
  } = useSimpleSearchState({
    searchValue: query.search,
    onSearch: onSearchChange,
  })
  const sorting = React.useMemo<SortingState>(
    () => query.sorting
      ? [{ id: query.sorting.field, desc: query.sorting.direction === "desc" }]
      : [],
    [query.sorting]
  )
  const severityValues = query.filters.severity ?? []
  const severityOptions = mergeSelectedFacetOptions(
    (filterOptions.severity ?? []).map((option) => ({
      ...option,
      label: option.value === "__unset__" ? unsetLabel : option.label,
    })),
    severityValues,
    unsetLabel
  )
  const { handleSelectionChange, selectedRowActions, toolbarRight, dialogs } = useFingerprintTableActions<FingerPrintHubFingerprint>({
    onSelectionChange,
    onImport,
    onExport,
    onBulkDelete,
    onDeleteAll,
    totalCount: totalCount ?? cursorPaginationSummary?.total ?? 0,
  })

  return (
    <>
      <BusinessListDataTable
        data={data}
        columns={columns}
        getRowId={(row) => row.name}
        state={{
          pagination,
          cursorPaginationSummary,
          paginationNavigation,
          onPaginationChange,
          sortingMode: "server",
          sorting,
          onSortingChange,
          onSelectionChange: handleSelectionChange,
          selectedRows,
        }}
        behavior={{
          onRowClick: onRowClick
            ? (row) => onRowClick(row as FingerPrintHubFingerprint)
            : undefined,
          getRowActionLabel: (row) => `${tActions("details")}: ${(row as FingerPrintHubFingerprint).displayName}`,
        }}
        actions={{
          selectedRowActions,
          showBulkDelete: false,
          showAddButton: false,
        }}
        ui={{
          toolbarLeft: (
            <div className="flex w-full min-w-0 flex-wrap items-center gap-2">
              <SimpleSearchToolbar
                value={searchValue}
                onChange={handleSearchInputChange}
                onSubmit={commitSearch}
                loading={isSearching}
                placeholder={tFingerprints("searchName")}
                toolbarDensity="compact"
              />
              <DataTableFacetedFilterGroup
                hasSelectedValues={severityValues.length > 0}
                onReset={() => onFacetChange?.("severity", [])}
              >
                <DataTableFacetedFilter
                  title={tForm("severity")}
                  values={severityValues}
                  onValuesChange={(values) => onFacetChange?.("severity", values)}
                  options={severityOptions}
                  emptyLabel={t("noData")}
                  clearLabel={tDataTable("clearFilter")}
                />
              </DataTableFacetedFilterGroup>
            </div>
          ),
          emptyMessage: t("noData"),
          toolbarRight,
          loading,
          loadingPresentation: initialLoading ? "initial" : "rows",
          initialLoadingToolbarFilterCount: 1,
          loadingRowCount,
          stableSurfaceRowCount,
          loadingSlots: FINGERPRINTHUB_FINGERPRINT_LOADING_SLOTS,
        }}
      />
      {dialogs}
    </>
  )
}
