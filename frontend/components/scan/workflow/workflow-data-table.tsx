"use client"

import type { ColumnDef } from "@tanstack/react-table"

import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { SimpleSearchToolbar } from "@/components/ui/data-table/simple-search-toolbar"
import { useWorkflowDataTableState } from "./workflow-data-table-state"

import type { ScanWorkflow } from "@/types/workflow.types"

interface WorkflowDataTableProps {
  data: ScanWorkflow[]
  columns: ColumnDef<ScanWorkflow>[]
  onAddNew?: () => void
  searchPlaceholder?: string
  searchColumn?: string
  addButtonText?: string
}

/**
 * Scan workflow template data table component
 * Uses UnifiedDataTable unified component
 */
export function WorkflowDataTable({
  data = [],
  columns,
  onAddNew,
  searchPlaceholder,
  addButtonText,
}: WorkflowDataTableProps) {
  const state = useWorkflowDataTableState({ data })

  return (
    <UnifiedDataTable
      data={state.filteredData}
      columns={columns}
      getRowId={(row) => String(row.id)}
      behavior={{ enableRowSelection: false }}
      actions={{
        onAddNew,
        addButtonLabel: addButtonText || state.tWorkflow("createWorkflow"),
        showBulkDelete: false,
      }}
      ui={{
        emptyMessage: state.t("noData"),
        toolbarLeft: (
          <SimpleSearchToolbar
            value={state.searchValue}
            onChange={state.setSearchValue}
            placeholder={searchPlaceholder || state.tWorkflow("searchPlaceholder")}
            showButton={false}
          />
        ),
      }}
    />
  )
}
