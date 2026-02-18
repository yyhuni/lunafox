"use client"

import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { useTranslations } from "next-intl"
import { SmartFilterDataTable } from "@/components/ui/data-table/smart-filter-data-table"
import { buildDownloadOptions } from "@/components/ui/data-table/data-table-helpers"
import { PREDEFINED_FIELDS, type FilterField } from "@/components/common/smart-filter-input"
import type { IPAddress } from "@/types/ip-address.types"
import type { PaginationInfo } from "@/types/common.types"

// IP address page filter field configuration
const IP_ADDRESS_FILTER_FIELDS: FilterField[] = [
  PREDEFINED_FIELDS.ip,
  PREDEFINED_FIELDS.port,
  PREDEFINED_FIELDS.host,
]

// IP address page filter examples
const IP_ADDRESS_FILTER_EXAMPLES = [
  'ip="192.168.1." && port="80"',
  'port="443" || port="8443"',
  'host="api.example.com" && port!="22"',
]

interface IPAddressesDataTableProps {
  data: IPAddress[]
  columns: ColumnDef<IPAddress>[]
  filterValue?: string
  onFilterChange?: (value: string) => void
  pagination?: { pageIndex: number; pageSize: number }
  setPagination?: React.Dispatch<React.SetStateAction<{ pageIndex: number; pageSize: number }>>
  paginationInfo?: PaginationInfo
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: IPAddress[]) => void
  onDownloadAll?: () => void
  onDownloadSelected?: () => void
}

export function IPAddressesDataTable({
  data = [],
  columns,
  filterValue = "",
  onFilterChange,
  pagination,
  setPagination,
  paginationInfo,
  onPaginationChange,
  onBulkDelete,
  onSelectionChange,
  onDownloadAll,
  onDownloadSelected,
}: IPAddressesDataTableProps) {
  const t = useTranslations("common.status")
  const tDownload = useTranslations("common.download")
  const tActions = useTranslations("common.actions")

  // Download options
  const downloadOptions = buildDownloadOptions(tDownload, {
    onDownloadAll,
    onDownloadSelected,
  })

  return (
    <SmartFilterDataTable
      data={data}
      columns={columns}
      getRowId={(row) => row.ip}
      filterFields={IP_ADDRESS_FILTER_FIELDS}
      filterExamples={IP_ADDRESS_FILTER_EXAMPLES}
      filterValue={filterValue}
      onFilterChange={onFilterChange}
      pagination={pagination}
      setPagination={setPagination}
      paginationInfo={paginationInfo}
      onPaginationChange={onPaginationChange}
      onSelectionChange={onSelectionChange}
      onBulkDelete={onBulkDelete}
      bulkDeleteLabel={tActions("delete")}
      showAddButton={false}
      downloadOptions={downloadOptions}
      emptyMessage={t("noData")}
    />
  )
}
