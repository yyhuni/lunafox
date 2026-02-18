"use client"

import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { useFingerprintTableActions } from "@/components/fingerprints/fingerprint-table-actions"
import type { FilterField } from "@/components/common/smart-filter-input"
import type { FingerPrintHubFingerprint } from "@/types/fingerprint.types"
import type { PaginationInfo } from "@/types/common.types"
import { useTranslations } from "next-intl"

const FINGERPRINTHUB_FILTER_EXAMPLES = [
  'name="Apache"',
  'severity="high"',
  'author="pdteam"',
  'fpId="apache-detect"',
]

interface FingerPrintHubFingerprintDataTableProps {
  data: FingerPrintHubFingerprint[]
  columns: ColumnDef<FingerPrintHubFingerprint>[]
  onSelectionChange?: (selectedRows: FingerPrintHubFingerprint[]) => void
  filterValue?: string
  onFilterChange?: (value: string) => void
  isSearching?: boolean
  onAddSingle?: () => void
  onAddImport?: () => void
  onExport?: () => void
  onBulkDelete?: () => void
  onDeleteAll?: () => void
  totalCount?: number
  pagination?: { pageIndex: number; pageSize: number }
  paginationInfo?: PaginationInfo
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
}

export function FingerPrintHubFingerprintDataTable({
  data = [],
  columns,
  onSelectionChange,
  filterValue,
  onFilterChange,
  isSearching = false,
  onAddSingle,
  onAddImport,
  onExport,
  onBulkDelete,
  onDeleteAll,
  totalCount = 0,
  pagination: externalPagination,
  paginationInfo,
  onPaginationChange,
}: FingerPrintHubFingerprintDataTableProps) {
  const t = useTranslations("tools.fingerprints")

  // FingerPrintHub filter field configuration
  const filterFields: FilterField[] = React.useMemo(() => [
    { key: "fpId", label: "ID", description: t("filter.fingerprinthub.fpId") },
    { key: "name", label: "Name", description: t("filter.fingerprinthub.name") },
    { key: "author", label: "Author", description: t("filter.fingerprinthub.author") },
    { key: "severity", label: "Severity", description: t("filter.fingerprinthub.severity") },
  ], [t])

  const handleSmartSearch = (rawQuery: string) => {
    if (onFilterChange) {
      onFilterChange(rawQuery)
    }
  }

  const { handleSelectionChange, toolbarRight, dialogs } = useFingerprintTableActions<FingerPrintHubFingerprint>({
    onSelectionChange,
    onAddSingle,
    onAddImport,
    onExport,
    onBulkDelete,
    onDeleteAll,
    totalCount,
  })

  return (
    <>
      <UnifiedDataTable
        data={data}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination: externalPagination,
          paginationInfo,
          onPaginationChange,
          searchValue: filterValue,
          isSearching,
          onSelectionChange: handleSelectionChange,
        }}
        behavior={{
          searchMode: "smart",
          onSearch: handleSmartSearch,
        }}
        actions={{
          showBulkDelete: false,
          showAddButton: false,
        }}
        ui={{
          filterFields,
          filterExamples: FINGERPRINTHUB_FILTER_EXAMPLES,
          emptyMessage: "No results",
          toolbarRight,
        }}
      />
      {dialogs}
    </>
  )
}
