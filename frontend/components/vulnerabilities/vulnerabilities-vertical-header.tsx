"use client"

import React from "react"
import { useTranslations } from "next-intl"
import { IconSearch, IconChevronDown, CheckCircle2, Circle, X, Filter } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { PaginationInfo } from "@/types/common.types"
import type { ReviewFilter, SeverityFilter } from "./vulnerabilities-data-table"

interface VulnerabilitiesVerticalHeaderProps {
  // Filter state
  reviewFilter: ReviewFilter
  onReviewFilterChange: (filter: ReviewFilter) => void
  severityFilter: SeverityFilter
  onSeverityFilterChange: (filter: SeverityFilter) => void
  filterQuery: string
  onFilterChange: (value: string) => void
  // Counts
  totalCount: number
  pendingCount: number
  reviewedCount: number
  // Selection & bulk actions
  selectedCount: number
  onBulkMarkAsReviewed: () => void
  onBulkMarkAsPending: () => void
  onClearSelection: () => void
  // Pagination
  pagination: { pageIndex: number; pageSize: number }
  paginationInfo?: PaginationInfo
  onPaginationChange: (pagination: { pageIndex: number; pageSize: number }) => void
}

export function VulnerabilitiesVerticalHeader({
  reviewFilter,
  onReviewFilterChange,
  severityFilter,
  onSeverityFilterChange,
  filterQuery,
  onFilterChange,
  totalCount,
  pendingCount,
  reviewedCount,
  selectedCount,
  onBulkMarkAsReviewed,
  onBulkMarkAsPending,
  onClearSelection,
  pagination,
  paginationInfo,
  onPaginationChange,
}: VulnerabilitiesVerticalHeaderProps) {
  const tVuln = useTranslations("vulnerabilities")
  const tSeverity = useTranslations("severity")
  const tActions = useTranslations("common.actions")
  const tPagination = useTranslations("common.pagination")

  const currentPage = pagination.pageIndex + 1
  const totalPages = paginationInfo?.totalPages ?? 1

  const severityOptions: { value: SeverityFilter; label: string }[] = [
    { value: "all", label: tVuln("reviewStatus.all") },
    { value: "critical", label: tSeverity("critical") },
    { value: "high", label: tSeverity("high") },
    { value: "medium", label: tSeverity("medium") },
    { value: "low", label: tSeverity("low") },
    { value: "info", label: tSeverity("info") },
  ]

  return (
    <div className="border-b bg-card px-3 py-2 md:h-[var(--vuln-toolbar-h)] md:px-4 md:py-0 shrink-0 flex flex-col md:flex-row md:items-center md:justify-between gap-2 md:gap-4">
      {/* Left: Filters or Bulk Actions */}
      {selectedCount > 0 ? (
        <div className="flex flex-wrap items-center gap-2 md:gap-3 animate-in fade-in duration-200 min-w-0">
          <Badge variant="secondary" className="h-7 px-2 font-mono text-xs">
            {tVuln("selected", { count: selectedCount })}
          </Badge>
          <div className="h-4 w-px bg-border/60 mx-1" />
          <Button
            size="sm"
            variant="outline"
            className="h-11 md:h-8 gap-2 text-green-600 hover:text-green-700 hover:bg-green-50 border-green-200/50"
            onClick={onBulkMarkAsReviewed}
          >
            <CheckCircle2 className="h-3.5 w-3.5" />
            {tVuln("markAsReviewed")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            className="h-11 md:h-8 gap-2 text-blue-600 hover:text-blue-700 hover:bg-blue-50 border-blue-200/50"
            onClick={onBulkMarkAsPending}
          >
            <Circle className="h-3.5 w-3.5" />
            {tVuln("markAsPending")}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={onClearSelection}
            className="h-11 w-11 md:h-7 md:w-7 text-muted-foreground hover:text-foreground"
            aria-label={tActions("deselectAll")}
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>
      ) : (
        <Tabs value={reviewFilter} onValueChange={(v) => onReviewFilterChange(v as ReviewFilter)} className="w-full md:w-auto overflow-x-auto">
          <TabsList className="h-12 md:h-9 p-0 bg-muted/50 border w-max min-w-full md:min-w-0 md:w-auto">
            <TabsTrigger value="all" className="text-xs px-3 !h-11 md:!h-7 whitespace-nowrap data-[state=active]:shadow-sm">
              <span className="inline-flex items-center leading-none">{tVuln("reviewStatus.all")}</span>
              <Badge variant="secondary" className="bg-background/50 text-[10px] h-4 min-w-[1.25rem] font-mono tabular-nums leading-none !py-0 px-1 rounded-sm border-0">{totalCount}</Badge>
            </TabsTrigger>
            <TabsTrigger value="pending" className="text-xs px-3 !h-11 md:!h-7 whitespace-nowrap data-[state=active]:shadow-sm">
              <span className="inline-flex items-center leading-none">{tVuln("reviewStatus.pending")}</span>
              <Badge variant="secondary" className="bg-background/50 text-[10px] h-4 min-w-[1.25rem] font-mono tabular-nums leading-none !py-0 px-1 rounded-sm border-0">{pendingCount}</Badge>
            </TabsTrigger>
            <TabsTrigger value="reviewed" className="text-xs px-3 !h-11 md:!h-7 whitespace-nowrap data-[state=active]:shadow-sm">
              <span className="inline-flex items-center leading-none">{tVuln("reviewStatus.reviewed")}</span>
              <Badge variant="secondary" className="bg-background/50 text-[10px] h-4 min-w-[1.25rem] font-mono tabular-nums leading-none !py-0 px-1 rounded-sm border-0">{reviewedCount}</Badge>
            </TabsTrigger>
          </TabsList>
        </Tabs>
      )}

      {/* Right: Search + Severity Filter + Pagination */}
      <div className="flex w-full md:flex-1 items-center gap-2 md:gap-3 min-w-0 md:justify-end">
        <div className="relative flex-1 w-full md:max-w-sm md:ml-auto">
          <IconSearch className="absolute left-2.5 top-3 md:top-2.5 size-4 text-muted-foreground" />
          <Input
            type="search"
            value={filterQuery}
            onChange={(e) => onFilterChange(e.target.value)}
            placeholder={tActions("search")}
            className="h-11 md:h-9 pl-9 bg-background/50"
          />
        </div>

        <Select
          value={severityFilter}
          onValueChange={(value) => onSeverityFilterChange(value as SeverityFilter)}
        >
          <SelectTrigger size="sm" className="shrink-0 data-[size=sm]:h-11 md:data-[size=sm]:h-9 w-[92px] md:w-auto">
            <Filter className="h-3.5 w-3.5" />
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

        <div className="flex items-center gap-1.5 md:gap-2 shrink-0 border-l pl-2 md:pl-3">
          <span className="hidden sm:inline text-xs text-muted-foreground tabular-nums">
            {currentPage} / {totalPages}
          </span>
          <div className="flex gap-1 sm:ml-1">
            <Button
              variant="outline"
              size="icon"
              className="h-11 w-11 md:h-7 md:w-7"
              disabled={currentPage <= 1}
              onClick={() => onPaginationChange({ ...pagination, pageIndex: pagination.pageIndex - 1 })}
              aria-label={tPagination("previous")}
            >
              <IconChevronDown className="h-3.5 w-3.5 rotate-90" />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="h-11 w-11 md:h-7 md:w-7"
              disabled={currentPage >= totalPages}
              onClick={() => onPaginationChange({ ...pagination, pageIndex: pagination.pageIndex + 1 })}
              aria-label={tPagination("next")}
            >
              <IconChevronDown className="h-3.5 w-3.5 -rotate-90" />
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
