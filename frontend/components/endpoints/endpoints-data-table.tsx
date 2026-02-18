"use client"

import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { useTranslations } from "next-intl"
import { SmartFilterDataTable } from "@/components/ui/data-table/smart-filter-data-table"
import { buildDownloadOptions } from "@/components/ui/data-table/data-table-helpers"
import type { FilterField } from "@/components/common/smart-filter-input"
import type { PaginationState } from "@/types/data-table.types"
import type { PaginationInfo } from "@/types/common.types"
import { buildPaginationInfo } from "@/hooks/_shared/pagination"

// Endpoint page filter field configuration
const ENDPOINT_FILTER_FIELDS: FilterField[] = [
  { key: "url", label: "URL", description: "Endpoint URL" },
  { key: "host", label: "Host", description: "Hostname" },
  { key: "title", label: "Title", description: "Page title" },
  { key: "status", label: "Status", description: "HTTP status code" },
  { key: "tech", label: "Tech", description: "Technologies" },
  { key: "responseHeaders", label: "Headers", description: "Response headers" },
]

// Endpoint page filter examples
const ENDPOINT_FILTER_EXAMPLES = [
  'url="/api/" && status="200"',
  'host="api.example.com" || host="admin.example.com"',
  'title="Dashboard" && status!="404"',
  'tech="php" || tech="wordpress"',
]

interface EndpointsDataTableProps<TData extends { id: number | string }, TValue> {
  columns: ColumnDef<TData, TValue>[]
  data: TData[]
  // Smart filter
  filterValue?: string
  onFilterChange?: (value: string) => void
  isSearching?: boolean
  onAddNew?: () => void
  addButtonText?: string
  onSelectionChange?: (selectedRows: TData[]) => void
  onBulkDelete?: () => void
  pagination?: { pageIndex: number; pageSize: number }
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  totalCount?: number
  totalPages?: number
  onDownloadAll?: () => void
  onDownloadSelected?: () => void
  onBulkAdd?: () => void
}

export function EndpointsDataTable<TData extends { id: number | string }, TValue>({
  columns,
  data,
  filterValue,
  onFilterChange,
  isSearching = false,
  onAddNew,
  addButtonText = "Add",
  onSelectionChange,
  onBulkDelete,
  pagination: externalPagination,
  onPaginationChange,
  totalCount,
  totalPages,
  onDownloadAll,
  onDownloadSelected,
  onBulkAdd,
}: EndpointsDataTableProps<TData, TValue>) {
  const t = useTranslations("common.status")
  const tActions = useTranslations("common.actions")
  const tDownload = useTranslations("common.download")
  
  const [internalPagination, setInternalPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const pagination = externalPagination || internalPagination

  // Handle pagination change
  const handlePaginationChange = (newPagination: PaginationState) => {
    if (onPaginationChange) {
      onPaginationChange(newPagination)
    } else {
      setInternalPagination(newPagination)
    }
  }

  // Build paginationInfo
  const paginationInfo: PaginationInfo | undefined =
    externalPagination && totalCount !== undefined
      ? buildPaginationInfo({
        total: totalCount ?? 0,
        page: pagination.pageIndex + 1,
        pageSize: pagination.pageSize,
        totalPages,
        minTotalPages: 1,
      })
      : undefined

  const downloadOptions = buildDownloadOptions(tDownload, {
    onDownloadAll,
    onDownloadSelected,
  })

  return (
    <SmartFilterDataTable
      data={data}
      columns={columns as ColumnDef<TData>[]}
      getRowId={(row) => String(row.id)}
      filterFields={ENDPOINT_FILTER_FIELDS}
      filterExamples={ENDPOINT_FILTER_EXAMPLES}
      filterValue={filterValue}
      onFilterChange={onFilterChange}
      isSearching={isSearching}
      pagination={pagination}
      setPagination={onPaginationChange ? undefined : setInternalPagination}
      paginationInfo={paginationInfo}
      onPaginationChange={handlePaginationChange}
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
