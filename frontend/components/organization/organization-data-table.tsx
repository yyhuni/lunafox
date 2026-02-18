"use client"

import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { SimpleSearchToolbar } from "@/components/ui/data-table/simple-search-toolbar"
import type { OrganizationDataTableProps } from "@/types/organization.types"
import { useOrganizationDataTableState } from "./organization-data-table-state"

export function OrganizationDataTable({
  data,
  columns,
  onAddNew,
  onBulkDelete,
  onSelectionChange,
  searchPlaceholder,
  searchValue,
  onSearch,
  isSearching,
  pagination: externalPagination,
  paginationInfo,
  onPaginationChange,
}: OrganizationDataTableProps) {
  const state = useOrganizationDataTableState({ searchValue, onSearch })

  return (
    <UnifiedDataTable
      data={data}
      columns={columns}
      getRowId={(row) => String(row.id)}
      state={{
        pagination: externalPagination,
        paginationInfo,
        onPaginationChange,
        onSelectionChange,
        defaultSorting: state.defaultSorting,
      }}
      actions={{
        onBulkDelete,
        bulkDeleteLabel: state.tActions("delete"),
        onAddNew,
        addButtonLabel: state.t("addOrganization"),
      }}
      ui={{
        emptyMessage: state.t("noResults"),
        toolbarLeft: (
          <SimpleSearchToolbar
            value={state.localSearchValue}
            onChange={state.setLocalSearchValue}
            onSubmit={state.handleSearchSubmit}
            loading={isSearching}
            placeholder={searchPlaceholder ?? state.t("searchPlaceholder")}
          />
        ),
      }}
    />
  )
}
