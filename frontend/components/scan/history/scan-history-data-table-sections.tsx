"use client"

import { DataTableFacetedFilter, DataTableFacetedFilterGroup } from "@/components/shared/data-table/faceted-filter"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"

import type { ScanStatus } from "@/types/scan.types"
import type { ScanHistoryDataTableState } from "./scan-history-data-table-state"
import type { ScanStatusFilter } from "./scan-history-list-view-state"

interface ScanHistoryToolbarProps {
  state: ScanHistoryDataTableState
  loading: boolean
  placeholder?: string
  status: ScanStatusFilter
  onStatusChange?: (status: ScanStatusFilter) => void
}

export function ScanHistoryToolbar({
  state,
  loading,
  placeholder,
  status,
  onStatusChange,
}: ScanHistoryToolbarProps) {
  return (
    <SimpleSearchToolbar
      value={state.localSearchValue}
      onChange={state.handleSearchInputChange}
      onSubmit={state.commitSearch}
      loading={loading}
      placeholder={placeholder || state.tScan("searchPlaceholder")}
      after={onStatusChange ? (
        <DataTableFacetedFilterGroup hasSelectedValues={status.length > 0} onReset={() => onStatusChange([])}>
          <DataTableFacetedFilter<ScanStatus>
            title={state.tScan("statusLabel")}
            values={status}
            onValuesChange={onStatusChange}
            options={state.statusOptions}
            emptyLabel={state.tScan("allStatus")}
            clearLabel={state.tDataTable("clearFilter")}
          />
        </DataTableFacetedFilterGroup>
      ) : null}
    />
  )
}
