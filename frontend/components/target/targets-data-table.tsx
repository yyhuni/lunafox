"use client"

import { Filter } from "@/components/icons"
import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { SimpleSearchToolbar } from "@/components/ui/data-table/simple-search-toolbar"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { ColumnDef } from "@tanstack/react-table"
import type { Target } from "@/types/target.types"
import { useTargetTargetsDataTableState } from "./targets-data-table-state"

interface TargetsDataTableProps {
  data: Target[]
  columns: ColumnDef<Target>[]
  onAddNew?: () => void
  onAddHover?: () => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: Target[]) => void
  searchPlaceholder?: string
  searchValue?: string
  onSearch?: (value: string) => void
  isSearching?: boolean
  addButtonText?: string
  // Pagination related props
  pagination?: { pageIndex: number, pageSize: number }
  onPaginationChange?: (pagination: { pageIndex: number, pageSize: number }) => void
  totalCount?: number
  manualPagination?: boolean
  // Type filter
  typeFilter?: string
  onTypeFilterChange?: (value: string) => void
  // Styling
  className?: string
  tableClassName?: string
  hideToolbar?: boolean
  hidePagination?: boolean
}

/**
 * Targets data table component (target version)
 * Uses UnifiedDataTable unified component
 */
export function TargetsDataTable({
  data = [],
  columns,
  onAddNew,
  onAddHover,
  onBulkDelete,
  onSelectionChange,
  searchPlaceholder,
  searchValue,
  onSearch,
  isSearching = false,
  addButtonText,
  pagination: externalPagination,
  onPaginationChange,
  totalCount,
  manualPagination = false,
  typeFilter,
  onTypeFilterChange,
  className,
  tableClassName,
  hideToolbar = false,
  hidePagination = false,
}: TargetsDataTableProps) {
  const state = useTargetTargetsDataTableState({
    searchValue,
    onSearch,
    externalPagination,
    onPaginationChange,
    manualPagination,
    totalCount,
  })

  return (
    <UnifiedDataTable
      data={data}
      columns={columns}
      getRowId={(row) => String(row.id)}
      state={{
        pagination: state.pagination,
        setPagination: onPaginationChange ? undefined : state.setInternalPagination,
        paginationInfo: state.paginationInfo,
        onPaginationChange: state.handlePaginationChange,
        onSelectionChange,
      }}
      actions={{
        onBulkDelete,
        bulkDeleteLabel: state.tActions("delete"),
        onAddNew,
        onAddHover,
        addButtonLabel: addButtonText || state.tTarget("addTarget"),
        showAddButton: !!onAddNew,
      }}
      ui={{
        emptyMessage: state.t("noData"),
        toolbarLeft: (
          <SimpleSearchToolbar
            value={state.localSearchValue}
            onChange={state.setLocalSearchValue}
            onSubmit={state.handleSearchSubmit}
            loading={isSearching}
            placeholder={searchPlaceholder || state.tTarget("title")}
            after={onTypeFilterChange ? (
              <Select
                value={typeFilter || "all"}
                onValueChange={(value) => onTypeFilterChange(value === "all" ? "" : value)}
              >
                <SelectTrigger size="sm" className="w-auto">
                  <Filter className="h-4 w-4" />
                  <SelectValue placeholder={state.tActions("filter")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{state.tActions("all")}</SelectItem>
                  <SelectItem value="domain">{state.tTarget("types.domain")}</SelectItem>
                  <SelectItem value="ip">{state.tTarget("types.ip")}</SelectItem>
                  <SelectItem value="cidr">{state.tTarget("types.cidr")}</SelectItem>
                </SelectContent>
              </Select>
            ) : null}
          />
        ),
        className,
        tableClassName,
        hideToolbar,
        hidePagination,
      }}
    />
  )
}
