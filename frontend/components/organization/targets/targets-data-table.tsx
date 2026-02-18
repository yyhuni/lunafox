"use client"

import type { Dispatch, SetStateAction } from "react"
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
import type { PaginationInfo } from "@/types/common.types"
import { useOrganizationTargetsDataTableState } from "./targets-data-table-state"

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
  pagination?: { pageIndex: number; pageSize: number }
  setPagination?: Dispatch<SetStateAction<{ pageIndex: number; pageSize: number }>>
  paginationInfo?: PaginationInfo
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  typeFilter?: string
  onTypeFilterChange?: (value: string) => void
}

/**
 * Target data table component (organization version)
 * Unify components using UnifiedDataTable
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
  setPagination: setExternalPagination,
  paginationInfo,
  onPaginationChange,
  typeFilter,
  onTypeFilterChange,
}: TargetsDataTableProps) {
  const state = useOrganizationTargetsDataTableState({ searchValue, onSearch })

  return (
    <UnifiedDataTable
      data={data}
      columns={columns}
      getRowId={(row) => String(row.id)}
      state={{
        pagination: externalPagination,
        setPagination: setExternalPagination,
        paginationInfo,
        onPaginationChange,
        onSelectionChange,
      }}
      actions={{
        showBulkDelete: !!onBulkDelete,
        onBulkDelete,
        bulkDeleteLabel: state.tTooltips("unlinkTarget"),
        showAddButton: !!onAddNew,
        onAddNew,
        onAddHover,
        addButtonLabel: addButtonText || state.tTarget("addTarget"),
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
                  <SelectValue placeholder={state.tCommon("actions.filter")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{state.tCommon("actions.all")}</SelectItem>
                  <SelectItem value="domain">{state.tTarget("types.domain")}</SelectItem>
                  <SelectItem value="ip">{state.tTarget("types.ip")}</SelectItem>
                  <SelectItem value="cidr">{state.tTarget("types.cidr")}</SelectItem>
                </SelectContent>
              </Select>
            ) : null}
          />
        ),
      }}
    />
  )
}
