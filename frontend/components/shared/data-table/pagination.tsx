"use client"

import * as React from "react"
import type { Table } from "@tanstack/react-table"
import {
  IconChevronLeft,
  IconChevronRight,
  IconChevronsLeft,
  IconChevronsRight,
} from "@/components/icons"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  PaginationInfo,
} from "@/types/data-table.types"

interface SharedCompactPaginationBaseProps {
  pageSize: number
  onPageSizeChange: (pageSize: number) => void
  pageSizeOptions?: number[]
  summary?: React.ReactNode
  className?: string
}

interface NumberedSharedCompactPaginationProps extends SharedCompactPaginationBaseProps {
  mode?: "numbered"
  total: number
  page: number
  totalPages: number
  onPageChange: (page: number) => void
}

interface CursorSharedCompactPaginationProps extends SharedCompactPaginationBaseProps {
  mode: "cursor"
  canFirstPage?: boolean
  canPreviousPage: boolean
  canNextPage: boolean
  onFirstPage: () => void
  onPreviousPage: () => void
  onNextPage: () => void
}

type SharedCompactPaginationProps = NumberedSharedCompactPaginationProps | CursorSharedCompactPaginationProps

interface DataTablePaginationProps<TData> {
  table: Table<TData>
  paginationInfo?: PaginationInfo
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  pageSizeOptions?: number[]
  className?: string
}

const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 50, 100, 200, 500, 1000]
const MOBILE_STACK_LAYOUT_CLASSNAME = "flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"

export function SharedCompactPagination({
  pageSize,
  onPageSizeChange,
  pageSizeOptions = DEFAULT_PAGE_SIZE_OPTIONS,
  summary,
  className,
  ...pagination
}: SharedCompactPaginationProps) {
  const t = useTranslations("common.pagination")
  const rowsPerPageId = React.useId()
  const isCursorPagination = pagination.mode === "cursor"
  const safeTotalPages = isCursorPagination ? 1 : Math.max(1, pagination.totalPages)
  const canFirstPage = isCursorPagination
    ? (pagination.canFirstPage ?? pagination.canPreviousPage)
    : pagination.page > 1
  const canPreviousPage = isCursorPagination ? pagination.canPreviousPage : pagination.page > 1
  const canNextPage = isCursorPagination ? pagination.canNextPage : pagination.page < safeTotalPages

  const handlePageSizeChange = React.useCallback((value: string) => {
    onPageSizeChange(Number(value))
  }, [onPageSizeChange])

  const handleFirstPage = React.useCallback(() => {
    if (isCursorPagination) {
      pagination.onFirstPage()
      return
    }
    pagination.onPageChange(1)
  }, [isCursorPagination, pagination])

  const handlePreviousPage = React.useCallback(() => {
    if (isCursorPagination) {
      pagination.onPreviousPage()
      return
    }
    pagination.onPageChange(Math.max(1, pagination.page - 1))
  }, [isCursorPagination, pagination])

  const handleNextPage = React.useCallback(() => {
    if (isCursorPagination) {
      pagination.onNextPage()
      return
    }
    pagination.onPageChange(Math.min(safeTotalPages, pagination.page + 1))
  }, [isCursorPagination, pagination, safeTotalPages])

  const handleLastPage = React.useCallback(() => {
    if (!isCursorPagination) pagination.onPageChange(safeTotalPages)
  }, [isCursorPagination, pagination, safeTotalPages])

  return (
    <div className={cn(MOBILE_STACK_LAYOUT_CLASSNAME, "px-2", className)}>
      {!isCursorPagination || summary ? (
        <div className={cn("w-full sm:flex-1", textRole.metadataLabel)}>
          {summary ?? (!isCursorPagination ? t("total", { count: pagination.total }) : null)}
        </div>
      ) : null}

      <div className="flex w-full flex-col gap-3 sm:w-auto sm:flex-row sm:items-center sm:gap-4">
        <div className="flex items-center gap-2">
          <Label htmlFor={rowsPerPageId} className={cn("whitespace-nowrap", textRole.metadataLabel)}>
            {t("rowsPerPage")}
          </Label>
          <Select value={`${pageSize}`} onValueChange={handlePageSizeChange}>
            <SelectTrigger
              size="sm"
              id={rowsPerPageId}
              className="w-24"
            >
              <SelectValue placeholder={pageSize} />
            </SelectTrigger>
            <SelectContent position="popper" width="content-fit">
              {pageSizeOptions.map((size) => (
                <SelectItem key={size} value={`${size}`}>
                  {size}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {!isCursorPagination ? (
          <div className={cn("flex items-center justify-center whitespace-nowrap", textRole.metadataValue)}>
            {t("page", { current: pagination.page, total: safeTotalPages })}
          </div>
        ) : null}

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon-sm"
            className={isCursorPagination ? undefined : "hidden lg:flex"}
            onClick={handleFirstPage}
            disabled={!canFirstPage}
          >
            <span className="sr-only">{t("first")}</span>
            <IconChevronsLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="icon-sm"
            onClick={handlePreviousPage}
            disabled={!canPreviousPage}
          >
            <span className="sr-only">{t("previous")}</span>
            <IconChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="icon-sm"
            onClick={handleNextPage}
            disabled={!canNextPage}
          >
            <span className="sr-only">{t("next")}</span>
            <IconChevronRight className="h-4 w-4" />
          </Button>
          {!isCursorPagination ? (
            <Button
              variant="outline"
              size="icon-sm"
              className="hidden lg:flex"
              onClick={handleLastPage}
              disabled={!canNextPage}
            >
              <span className="sr-only">{t("last")}</span>
              <IconChevronsRight className="h-4 w-4" />
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  )
}

/**
 * Unified pagination component
 * 
 * Updates pagination state through table.setPageIndex/setPageSize,
 * handled uniformly by useReactTable's onPaginationChange for state synchronization.
 */
export function DataTablePagination<TData>({
  table,
  paginationInfo,
  cursorPaginationSummary,
  paginationNavigation,
  pageSizeOptions = DEFAULT_PAGE_SIZE_OPTIONS,
  className,
}: DataTablePaginationProps<TData>) {
  const tPagination = useTranslations("common.pagination")
  const tDataTable = useTranslations('dataTable')
  const { pageIndex, pageSize } = table.getState().pagination

  // Result totals are summaries; cursor navigation uses only explicit token availability.
  const total = cursorPaginationSummary?.total
    ?? paginationInfo?.total
    ?? table.getFilteredRowModel().rows.length
  const selectedCount = table.getFilteredSelectedRowModel().rows.length
  const summary = table.options.enableRowSelection === false
    ? tPagination("total", { count: total })
    : `${tDataTable("selected", { count: selectedCount })} / ${tPagination("total", { count: total })}`

  // Use useCallback to wrap handler functions to avoid unnecessary re-renders
  const handlePageSizeChange = React.useCallback((newPageSize: number) => {
    table.setPageSize(newPageSize)
  }, [table])

  if (paginationNavigation?.mode === "cursor") {
    return (
      <SharedCompactPagination
        mode="cursor"
        pageSize={pageSize}
        canFirstPage={paginationNavigation.canFirstPage}
        canPreviousPage={paginationNavigation.canPreviousPage}
        canNextPage={paginationNavigation.canNextPage}
        onFirstPage={paginationNavigation.onFirstPage ?? (() => table.setPageIndex(0))}
        onPreviousPage={() => table.setPageIndex(Math.max(0, pageIndex - 1))}
        onNextPage={() => table.setPageIndex(pageIndex + 1)}
        onPageSizeChange={handlePageSizeChange}
        pageSizeOptions={pageSizeOptions}
        summary={summary}
        className={className}
      />
    )
  }

  const totalPages = paginationInfo?.totalPages ?? table.getPageCount()
  const maxPageIndex = Math.max(0, totalPages - 1)

  return (
    <SharedCompactPagination
      total={total}
      page={pageIndex + 1}
      pageSize={pageSize}
      totalPages={totalPages}
      onPageSizeChange={handlePageSizeChange}
      onPageChange={(nextPage) => {
        const nextPageIndex = Math.max(0, Math.min(maxPageIndex, nextPage - 1))

        table.setPageIndex(nextPageIndex)
      }}
      pageSizeOptions={pageSizeOptions}
      summary={summary}
      className={className}
    />
  )
}
