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
    <div className="h-14 border-b bg-card flex items-center justify-between px-4 shrink-0 gap-4">
      {/* Left: Filters or Bulk Actions */}
      {selectedCount > 0 ? (
        <div className="flex items-center gap-3 animate-in fade-in duration-200">
          <Badge variant="secondary" className="h-7 px-2 font-mono text-xs">
            {tVuln("selected", { count: selectedCount })}
          </Badge>
          <div className="h-4 w-px bg-border/60 mx-1" />
          <Button
            size="sm"
            variant="outline"
            className="h-8 gap-2 text-green-600 hover:text-green-700 hover:bg-green-50 border-green-200/50"
            onClick={onBulkMarkAsReviewed}
          >
            <CheckCircle2 className="h-3.5 w-3.5" />
            {tVuln("markAsReviewed")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            className="h-8 gap-2 text-blue-600 hover:text-blue-700 hover:bg-blue-50 border-blue-200/50"
            onClick={onBulkMarkAsPending}
          >
            <Circle className="h-3.5 w-3.5" />
            {tVuln("markAsPending")}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={onClearSelection}
            className="h-7 w-7 text-muted-foreground hover:text-foreground"
            aria-label={tActions("deselectAll")}
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>
      ) : (
        <Tabs value={reviewFilter} onValueChange={(v) => onReviewFilterChange(v as ReviewFilter)} className="w-auto">
          <TabsList className="h-9 p-1 bg-muted/50 border">
            <TabsTrigger value="all" className="text-xs px-3 h-7 data-[state=active]:shadow-sm">
              {tVuln("reviewStatus.all")}
              <Badge variant="secondary" className="ml-1.5 bg-background/50 text-[10px] h-4 font-mono px-1 rounded-sm border-0">{totalCount}</Badge>
            </TabsTrigger>
            <TabsTrigger value="pending" className="text-xs px-3 h-7 data-[state=active]:shadow-sm">
              {tVuln("reviewStatus.pending")}
              <Badge variant="secondary" className="ml-1.5 bg-background/50 text-[10px] h-4 font-mono px-1 rounded-sm border-0">{pendingCount}</Badge>
            </TabsTrigger>
            <TabsTrigger value="reviewed" className="text-xs px-3 h-7 data-[state=active]:shadow-sm">
              {tVuln("reviewStatus.reviewed")}
              <Badge variant="secondary" className="ml-1.5 bg-background/50 text-[10px] h-4 font-mono px-1 rounded-sm border-0">{reviewedCount}</Badge>
            </TabsTrigger>
          </TabsList>
        </Tabs>
      )}

      {/* Right: Search + Severity Filter + Pagination */}
      <div className="flex flex-1 items-center justify-end gap-3 min-w-0">
        <div className="relative w-full max-w-sm ml-auto">
          <IconSearch className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
          <Input
            type="search"
            value={filterQuery}
            onChange={(e) => onFilterChange(e.target.value)}
            placeholder={tActions("search")}
            className="h-9 pl-9 bg-background/50"
          />
        </div>

        <Select
          value={severityFilter}
          onValueChange={(value) => onSeverityFilterChange(value as SeverityFilter)}
        >
          <SelectTrigger size="sm" className="w-auto shrink-0">
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

        <div className="flex items-center gap-2 shrink-0 border-l pl-3">
          <span className="text-xs text-muted-foreground tabular-nums">
            {currentPage} / {totalPages}
          </span>
          <div className="flex gap-1 ml-1">
            <Button
              variant="outline"
              size="icon"
              className="h-7 w-7"
              disabled={currentPage <= 1}
              onClick={() => onPaginationChange({ ...pagination, pageIndex: pagination.pageIndex - 1 })}
              aria-label="Previous page"
            >
              <IconChevronDown className="h-3.5 w-3.5 rotate-90" />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="h-7 w-7"
              disabled={currentPage >= totalPages}
              onClick={() => onPaginationChange({ ...pagination, pageIndex: pagination.pageIndex + 1 })}
              aria-label="Next page"
            >
              <IconChevronDown className="h-3.5 w-3.5 -rotate-90" />
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
