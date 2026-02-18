"use client"

import * as React from "react"
import type { ColumnDef } from "@tanstack/react-table"
import { useTranslations } from "next-intl"
import { CheckCircle, Circle, X, Filter } from "@/components/icons"
import { UnifiedDataTable } from "@/components/ui/data-table/unified-data-table"
import { buildDownloadOptions } from "@/components/ui/data-table/data-table-helpers"
import { SmartFilterInput, PREDEFINED_FIELDS, type FilterField, type ParsedFilter } from "@/components/common/smart-filter-input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { Vulnerability, VulnerabilitySeverity } from "@/types/vulnerability.types"
import type { PaginationInfo } from "@/types/common.types"

// Review filter type
export type ReviewFilter = "all" | "pending" | "reviewed"

// Severity filter type
export type SeverityFilter = VulnerabilitySeverity | "all"

// Vulnerability page filter fields
const VULNERABILITY_FILTER_FIELDS: FilterField[] = [
  { key: "type", label: "Type", description: "Vulnerability type" },
  PREDEFINED_FIELDS.severity,
  { key: "source", label: "Source", description: "Scanner source" },
  PREDEFINED_FIELDS.url,
]

// Vulnerability page examples
const VULNERABILITY_FILTER_EXAMPLES = [
  'type="xss" || type="sqli"',
  'severity="critical" || severity="high"',
  'source="nuclei" && severity="high"',
  'type="xss" && url="/api/"',
]


interface VulnerabilitiesDataTableProps {
  data: Vulnerability[]
  columns: ColumnDef<Vulnerability>[]
  filterValue?: string
  onFilterChange?: (value: string) => void
  pagination?: { pageIndex: number; pageSize: number }
  setPagination?: React.Dispatch<React.SetStateAction<{ pageIndex: number; pageSize: number }>>
  paginationInfo?: PaginationInfo
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: Vulnerability[]) => void
  onDownloadAll?: () => void
  onDownloadSelected?: () => void
  hideToolbar?: boolean
  // Review status props
  reviewFilter?: ReviewFilter
  onReviewFilterChange?: (filter: ReviewFilter) => void
  pendingCount?: number
  reviewedCount?: number
  selectedRows?: Vulnerability[]
  onBulkMarkAsReviewed?: () => void
  onBulkMarkAsPending?: () => void
  // New: severity filter
  severityFilter?: SeverityFilter
  onSeverityFilterChange?: (filter: SeverityFilter) => void
  // New: source filter
  sourceFilter?: string
  onSourceFilterChange?: (source: string) => void
  availableSources?: string[]
}

export function VulnerabilitiesDataTable({
  data = [],
  columns,
  filterValue,
  onFilterChange,
  pagination,
  setPagination,
  paginationInfo,
  onPaginationChange,
  onBulkDelete,
  onSelectionChange,
  onDownloadAll,
  onDownloadSelected,
  hideToolbar = false,
  reviewFilter = "all",
  onReviewFilterChange,
  pendingCount = 0,
  reviewedCount = 0,
  selectedRows = [],
  onBulkMarkAsReviewed,
  onBulkMarkAsPending,
  severityFilter = "all",
  onSeverityFilterChange,
}: VulnerabilitiesDataTableProps) {
  const t = useTranslations("common.status")
  const tDownload = useTranslations("common.download")
  const tActions = useTranslations("common.actions")
  const tVuln = useTranslations("vulnerabilities")
  const tSeverity = useTranslations("severity")
  
  // Handle smart filter search
  const handleSmartSearch = (_filters: ParsedFilter[], rawQuery: string) => {
    onFilterChange?.(rawQuery)
  }

  const downloadOptions = buildDownloadOptions(tDownload, {
    onDownloadAll,
    onDownloadSelected,
  })

  // Severity options for Select
  const severityOptions: { value: SeverityFilter; label: string }[] = [
    { value: "all", label: tVuln("reviewStatus.all") },
    { value: "critical", label: tSeverity("critical") },
    { value: "high", label: tSeverity("high") },
    { value: "medium", label: tSeverity("medium") },
    { value: "low", label: tSeverity("low") },
    { value: "info", label: tSeverity("info") },
  ]

  // Left toolbar content - smart filter + severity select
  const leftToolbarContent = (
    <div className="flex items-center gap-2 flex-1">
      <SmartFilterInput
        fields={VULNERABILITY_FILTER_FIELDS}
        examples={VULNERABILITY_FILTER_EXAMPLES}
        placeholder={tActions("search")}
        value={filterValue}
        onSearch={handleSmartSearch}
        className="flex-1 max-w-md"
      />
      {onSeverityFilterChange && (
        <Select
          value={severityFilter}
          onValueChange={(value) => onSeverityFilterChange(value as SeverityFilter)}
        >
          <SelectTrigger size="sm" className="w-auto">
            <Filter className="h-4 w-4" />
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {severityOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
    </div>
  )

  // Right toolbar content - review tabs
  const rightToolbarContent = (
    <>
      {/* Review filter tabs */}
      {onReviewFilterChange && (
        <Tabs value={reviewFilter} onValueChange={(v) => onReviewFilterChange(v as ReviewFilter)}>
          <TabsList>
            <TabsTrigger value="all">
              {tVuln("reviewStatus.all")}
            </TabsTrigger>
            <TabsTrigger value="pending">
              {tVuln("reviewStatus.pending")}
              {pendingCount > 0 && (
                <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                  {pendingCount}
                </Badge>
              )}
            </TabsTrigger>
            <TabsTrigger value="reviewed">
              {tVuln("reviewStatus.reviewed")}
              {reviewedCount > 0 && (
                <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                  {reviewedCount}
                </Badge>
              )}
            </TabsTrigger>
          </TabsList>
        </Tabs>
      )}
    </>
  )

  // Floating action bar for bulk operations
  const floatingActionBar = selectedRows.length > 0 && (onBulkMarkAsReviewed || onBulkMarkAsPending) && (
    <div className="fixed bottom-6 left-[calc(50vw+var(--sidebar-width,14rem)/2)] -translate-x-1/2 z-50 animate-in slide-in-from-bottom-4 fade-in duration-200">
      <div className="flex items-center gap-3 bg-background border rounded-lg shadow-lg px-4 py-2.5">
        <span className="text-sm text-muted-foreground">
          {tVuln("selected", { count: selectedRows.length })}
        </span>
        <div className="h-4 w-px bg-border" />
        {onBulkMarkAsReviewed && (
          <Button
            variant="outline"
            size="sm"
            onClick={onBulkMarkAsReviewed}
            className="h-8"
          >
            <CheckCircle className="h-4 w-4 mr-1.5" />
            {tVuln("markAsReviewed")}
          </Button>
        )}
        {onBulkMarkAsPending && (
          <Button
            variant="outline"
            size="sm"
            onClick={onBulkMarkAsPending}
            className="h-8"
          >
            <Circle className="h-4 w-4 mr-1.5" />
            {tVuln("markAsPending")}
          </Button>
        )}
        {onSelectionChange && (
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onSelectionChange([])}
            className="h-8 w-8 text-muted-foreground hover:text-foreground"
            aria-label={tActions("deselectAll")}
          >
            <X className="h-4 w-4" />
          </Button>
        )}
      </div>
    </div>
  )

  return (
    <>
      <UnifiedDataTable
        data={data}
        columns={columns}
        getRowId={(row) => String(row.id)}
        state={{
          pagination,
          setPagination,
          paginationInfo,
          onPaginationChange,
          onSelectionChange,
        }}
        actions={{
          onBulkDelete,
          bulkDeleteLabel: tActions("delete"),
          showAddButton: false,
          downloadOptions: downloadOptions.length > 0 ? downloadOptions : undefined,
        }}
        ui={{
          toolbarLeft: leftToolbarContent,
          hideToolbar,
          toolbarRight: rightToolbarContent,
          emptyMessage: t("noData"),
        }}
      />
      {floatingActionBar}
    </>
  )
}
