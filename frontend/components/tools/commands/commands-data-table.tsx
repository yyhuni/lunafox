"use client"

import type { ColumnDef } from "@tanstack/react-table"

import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { SimpleSearchToolbar } from "@/components/ui/data-table/simple-search-toolbar"
import { useCommandsDataTableState } from "./commands-data-table-state"

interface CommandsDataTableProps<TData extends { id: number; displayName?: string }> {
  columns: ColumnDef<TData, unknown>[]
  data: TData[]
  onBulkDelete?: (selectedIds: number[]) => void
  onAdd?: () => void
}

export function CommandsDataTable<TData extends { id: number; displayName?: string }>({
  columns,
  data,
  onBulkDelete,
  onAdd,
}: CommandsDataTableProps<TData>) {
  const state = useCommandsDataTableState({ data, onBulkDelete })

  return (
    <UnifiedDataTable
      data={state.filteredData}
      columns={columns}
      getRowId={(row) => String(row.id)}
      state={{
        onSelectionChange: state.setSelectedRows,
      }}
      actions={{
        onBulkDelete: onBulkDelete ? state.handleBulkDelete : undefined,
        onAddNew: onAdd,
        addButtonLabel: state.tCommon("actions.add"),
        bulkDeleteLabel: state.tCommon("actions.delete"),
      }}
      ui={{
        emptyMessage: state.tCommon("status.noData"),
        toolbarLeft: (
          <SimpleSearchToolbar
            value={state.searchValue}
            onChange={state.setSearchValue}
            placeholder={state.t("searchPlaceholder")}
            showButton={false}
          />
        ),
      }}
    />
  )
}
