"use client"

import type { ColumnDef } from "@tanstack/react-table"

import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { SimpleSearchToolbar } from "@/components/ui/data-table/simple-search-toolbar"
import { useEngineDataTableState } from "./engine-data-table-state"

import type { ScanEngine } from "@/types/engine.types"

interface EngineDataTableProps {
  data: ScanEngine[]
  columns: ColumnDef<ScanEngine>[]
  onAddNew?: () => void
  searchPlaceholder?: string
  searchColumn?: string
  addButtonText?: string
}

/**
 * Scan engine data table component
 * Uses UnifiedDataTable unified component
 */
export function EngineDataTable({
  data = [],
  columns,
  onAddNew,
  searchPlaceholder,
  addButtonText,
}: EngineDataTableProps) {
  const state = useEngineDataTableState({ data })

  return (
    <UnifiedDataTable
      data={state.filteredData}
      columns={columns}
      getRowId={(row) => String(row.id)}
      behavior={{ enableRowSelection: false }}
      actions={{
        onAddNew,
        addButtonLabel: addButtonText || state.tEngine("createEngine"),
        showBulkDelete: false,
      }}
      ui={{
        emptyMessage: state.t("noData"),
        toolbarLeft: (
          <SimpleSearchToolbar
            value={state.searchValue}
            onChange={state.setSearchValue}
            placeholder={searchPlaceholder || state.tEngine("searchPlaceholder")}
            showButton={false}
          />
        ),
      }}
    />
  )
}
