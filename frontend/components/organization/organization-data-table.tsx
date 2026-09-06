"use client"

import { semanticIcons } from "@/components/icons"
import {
  BusinessListDataTable,
  SelectedRowActionBar,
  type SelectedRowActionBarAction,
} from "@/components/shared/data-table"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import type { OrganizationDataTableProps } from "@/types/organization.types"
import { useOrganizationDataTableState } from "./organization-data-table-state"

const RunIcon = semanticIcons.action.run
const DeleteIcon = semanticIcons.action.delete
const ORGANIZATION_LIST_LOADING_SLOTS = {
  toolbar: "organization-list-toolbar",
  body: "organization-list-body",
  pagination: "organization-list-pagination",
}
// Description overflow actions make resolved rows taller than the shared comfortable baseline.
const ORGANIZATION_LIST_LOADING_ROW_HEIGHT_PX = 79

export function OrganizationDataTable({
  data,
  columns,
  onAddNew,
  onBulkDelete,
  onBulkInitiateScan,
  onViewDetail,
  onDetailIntent,
  onAddIntent,
  onSelectionChange,
  selectedRows = [],
  searchPlaceholder,
  searchValue,
  onSearch,
  isSearching,
  pagination: externalPagination,
  paginationInfo,
  cursorPaginationSummary,
  onPaginationChange,
  paginationNavigation,
  sorting,
  onSortingChange,
  loading = false,
  loadingRowCount,
  stableSurfaceRowCount,
}: OrganizationDataTableProps) {
  const state = useOrganizationDataTableState({ searchValue, onSearch })
  const selectedRowActions: SelectedRowActionBarAction[] = [
    ...(onBulkInitiateScan
      ? [{
          key: "initiate-scan",
          label: state.tTooltips("initiateScan"),
          icon: RunIcon,
          group: "scan",
          onClick: onBulkInitiateScan,
        }]
      : []),
    ...(onBulkDelete
      ? [{
          key: "delete",
          label: state.tActions("delete"),
          icon: DeleteIcon,
          tone: "destructive" as const,
          group: "danger",
          onClick: onBulkDelete,
        }]
      : []),
  ]

  return (
    <>
      <BusinessListDataTable
        data={data}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: externalPagination,
          paginationInfo,
          cursorPaginationSummary,
          paginationNavigation,
          onPaginationChange,
          sortingMode: "server",
          sorting,
          onSortingChange,
          onSelectionChange,
          selectedRows,
        }}
        behavior={{
          expandColumnIds: ["name"],
          onRowClick: onViewDetail
            ? (row) => onViewDetail(row as OrganizationDataTableProps["data"][number])
            : undefined,
          onRowIntent: onDetailIntent,
        }}
        actions={{
          showBulkDelete: false,
          onAddNew,
          onAddHover: onAddIntent,
          addButtonLabel: state.t("addOrganization"),
        }}
        ui={{
          emptyMessage: state.t("noResults"),
          showColumnVisibility: false,
          rowDensity: "comfortable",
          loading,
          loadingPresentation: loading ? "initial" : undefined,
          loadingRowCount,
          loadingRowHeightEstimate: ORGANIZATION_LIST_LOADING_ROW_HEIGHT_PX,
          stableSurfaceRowCount,
          loadingSlots: ORGANIZATION_LIST_LOADING_SLOTS,
          toolbarLeft: (
            <SimpleSearchToolbar
              value={state.localSearchValue}
              onChange={state.handleSearchInputChange}
              onSubmit={state.commitSearch}
              loading={isSearching}
              placeholder={searchPlaceholder ?? state.t("searchPlaceholder")}
              toolbarDensity="compact"
            />
          ),
        }}
      />
      <SelectedRowActionBar
        selectedCount={selectedRows.length}
        ariaLabel={state.tDataTable("selected", { count: selectedRows.length })}
        countLabel={state.tDataTable("selected", { count: selectedRows.length })}
        actions={selectedRowActions}
        onClearSelection={onSelectionChange ? () => onSelectionChange([]) : undefined}
        clearSelectionLabel={state.tDataTable("deselectAll")}
      />
    </>
  )
}
