"use client"

import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { useTranslations } from "next-intl"
import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { useFingerprintTableActions } from "@/components/fingerprints/fingerprint-table-actions"
import type { FilterField } from "@/components/common/smart-filter-input"
import type { WappalyzerFingerprint } from "@/types/fingerprint.types"
import type { PaginationInfo } from "@/types/common.types"

const WAPPALYZER_FILTER_EXAMPLES = [
  'name="WordPress"',
  'name=="React"',
  'website="wordpress.org"',
  'cpe="wordpress"',
]

interface WappalyzerFingerprintDataTableProps {
  data: WappalyzerFingerprint[]
  columns: ColumnDef<WappalyzerFingerprint>[]
  onSelectionChange?: (selectedRows: WappalyzerFingerprint[]) => void
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

export function WappalyzerFingerprintDataTable({
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
}: WappalyzerFingerprintDataTableProps) {
  const t = useTranslations("tools.fingerprints")

  // Wappalyzer filter field configuration (using translations)
  const wappalyzerFilterFields: FilterField[] = React.useMemo(() => [
    { key: "name", label: "Name", description: t("filter.wappalyzer.name") },
    { key: "website", label: "Website", description: t("filter.wappalyzer.website") },
    { key: "cpe", label: "CPE", description: t("filter.wappalyzer.cpe") },
  ], [t])

  const handleSmartSearch = (rawQuery: string) => {
    if (onFilterChange) {
      onFilterChange(rawQuery)
    }
  }

  const { handleSelectionChange, toolbarRight, dialogs } = useFingerprintTableActions<WappalyzerFingerprint>({
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
          filterFields: wappalyzerFilterFields,
          filterExamples: WAPPALYZER_FILTER_EXAMPLES,
          emptyMessage: "No results",
          toolbarRight,
        }}
      />
      {dialogs}
    </>
  )
}
