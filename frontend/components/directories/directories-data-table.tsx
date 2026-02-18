"use client"

import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { useTranslations } from "next-intl"
import { SmartFilterDataTable } from "@/components/ui/data-table/smart-filter-data-table"
import { buildDownloadOptions } from "@/components/ui/data-table/data-table-helpers"
import type { FilterField } from "@/components/common/smart-filter-input"
import type { Directory } from "@/types/directory.types"
import type { PaginationInfo } from "@/types/common.types"

// Directory page filter field configuration
const DIRECTORY_FILTER_FIELDS: FilterField[] = [
  { key: "url", label: "URL", description: "Directory URL" },
  { key: "status", label: "Status", description: "HTTP status code" },
]

// Directory page filter examples
const DIRECTORY_FILTER_EXAMPLES = [
  'url="/admin" && status="200"',
  'url="/api/" || url="/config/"',
  'status="200" && url!="/index.html"',
]

interface DirectoriesDataTableProps {
  data: Directory[]
  columns: ColumnDef<Directory>[]
  // Smart filter
  filterValue?: string
  onFilterChange?: (value: string) => void
  isSearching?: boolean
  pagination?: { pageIndex: number; pageSize: number }
  setPagination?: React.Dispatch<React.SetStateAction<{ pageIndex: number; pageSize: number }>>
  paginationInfo?: PaginationInfo
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: Directory[]) => void
  // Download callback functions
  onDownloadAll?: () => void
  onDownloadSelected?: () => void
  onBulkAdd?: () => void
}

export function DirectoriesDataTable({
  data = [],
  columns,
  filterValue,
  onFilterChange,
  isSearching = false,
  pagination,
  setPagination,
  paginationInfo,
  onPaginationChange,
  onBulkDelete,
  onSelectionChange,
  onDownloadAll,
  onDownloadSelected,
  onBulkAdd,
}: DirectoriesDataTableProps) {
  const t = useTranslations("common.status")
  const tActions = useTranslations("common.actions")
  const tDownload = useTranslations("common.download")
  const downloadOptions = buildDownloadOptions(tDownload, {
    onDownloadAll,
    onDownloadSelected,
  })

  return (
    <SmartFilterDataTable
      data={data}
      columns={columns}
      getRowId={(row) => String(row.id)}
      filterFields={DIRECTORY_FILTER_FIELDS}
      filterExamples={DIRECTORY_FILTER_EXAMPLES}
      filterValue={filterValue}
      onFilterChange={onFilterChange}
      isSearching={isSearching}
      pagination={pagination}
      setPagination={setPagination}
      paginationInfo={paginationInfo}
      onPaginationChange={onPaginationChange}
      onSelectionChange={onSelectionChange}
      onBulkDelete={onBulkDelete}
      bulkDeleteLabel={tActions("delete")}
      onBulkAdd={onBulkAdd}
      bulkAddLabel={tActions("add")}
      showAddButton={false}
      downloadOptions={downloadOptions}
      emptyMessage={t("noData")}
    />
  )
}
