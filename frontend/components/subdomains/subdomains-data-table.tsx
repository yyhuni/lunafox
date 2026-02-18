"use client"

import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { useTranslations } from "next-intl"
import { SmartFilterDataTable } from "@/components/ui/data-table/smart-filter-data-table"
import { buildDownloadOptions } from "@/components/ui/data-table/data-table-helpers"
import type { FilterField } from "@/components/common/smart-filter-input"
import type { Subdomain } from "@/types/subdomain.types"
import type { PaginationInfo } from "@/types/common.types"

// Subdomain page filter field configuration
const SUBDOMAIN_FILTER_FIELDS: FilterField[] = [
  { key: "name", label: "Name", description: "Subdomain name" },
]

// Subdomain page filter examples
const SUBDOMAIN_FILTER_EXAMPLES = [
  'name="api.example.com"',
  'name=".test.com"',
]

// Component props type definition
interface SubdomainsDataTableProps {
  data: Subdomain[]
  columns: ColumnDef<Subdomain>[]
  onAddNew?: () => void
  onBulkAdd?: () => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: Subdomain[]) => void
  // Smart filter
  filterValue?: string
  onFilterChange?: (value: string) => void
  isSearching?: boolean
  addButtonText?: string
  // Download callback functions
  onDownloadAll?: () => void
  onDownloadInteresting?: () => void
  onDownloadImportant?: () => void
  onDownloadSelected?: () => void
  // Server-side pagination support
  pagination?: { pageIndex: number; pageSize: number }
  setPagination?: React.Dispatch<React.SetStateAction<{ pageIndex: number; pageSize: number }>>
  paginationInfo?: PaginationInfo
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
}

/**
 * Subdomain data table component
 * Uses UnifiedDataTable unified component
 */
export function SubdomainsDataTable({
  data = [],
  columns,
  onAddNew,
  onBulkAdd,
  onBulkDelete,
  onSelectionChange,
  filterValue,
  onFilterChange,
  isSearching = false,
  addButtonText = "Add",
  onDownloadAll,
  onDownloadInteresting,
  onDownloadImportant,
  onDownloadSelected,
  pagination: externalPagination,
  setPagination: setExternalPagination,
  paginationInfo,
  onPaginationChange,
}: SubdomainsDataTableProps) {
  const t = useTranslations("common.status")
  const tActions = useTranslations("common.actions")
  const tDownload = useTranslations("common.download")

  // Download options
  const downloadOptions = buildDownloadOptions(tDownload, {
    onDownloadAll,
    onDownloadSelected,
    onDownloadImportant,
    onDownloadInteresting,
  })

  return (
    <SmartFilterDataTable
      data={data}
      columns={columns}
      getRowId={(row) => String(row.id)}
      filterFields={SUBDOMAIN_FILTER_FIELDS}
      filterExamples={SUBDOMAIN_FILTER_EXAMPLES}
      filterValue={filterValue}
      onFilterChange={onFilterChange}
      isSearching={isSearching}
      pagination={externalPagination}
      setPagination={setExternalPagination}
      paginationInfo={paginationInfo}
      onPaginationChange={onPaginationChange}
      onSelectionChange={onSelectionChange}
      onBulkDelete={onBulkDelete}
      bulkDeleteLabel={tActions("delete")}
      onAddNew={onAddNew}
      addButtonLabel={addButtonText}
      onBulkAdd={onBulkAdd}
      bulkAddLabel={tActions("add")}
      downloadOptions={downloadOptions}
      emptyMessage={t("noData")}
    />
  )
}
