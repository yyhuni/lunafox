"use client"

import type { ColumnDef } from "@tanstack/react-table"

import { BusinessListDataTable } from "@/components/shared/data-table"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import { useWorkflowDataTableState } from "./workflow-data-table-state"

import type { ScanWorkflow } from "@/types/scan-workflow.types"

interface WorkflowDataTableProps {
  data: ScanWorkflow[]
  columns: ColumnDef<ScanWorkflow>[]
  onAddNew?: () => void
  searchPlaceholder?: string
  searchColumn?: string
  addButtonText?: string
}

export function WorkflowDataTable({
  data = [],
  columns,
  onAddNew,
  searchPlaceholder,
  addButtonText,
}: WorkflowDataTableProps) {
  const state = useWorkflowDataTableState({ data })

  return (
    <BusinessListDataTable
      data={state.filteredData}
      columns={columns}
      getRowId={(row) => row.name}
      behavior={{
        enableRowSelection: false,
        expandColumnIds: ["name"],
      }}
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
            toolbarDensity="compact"
          />
        ),
      }}
    />
  )
}
