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
import { cn } from "@/lib/utils"
import type { PaginationInfo } from "@/types/data-table.types"

interface DataTablePaginationProps<TData> {
  table: Table<TData>
  paginationInfo?: PaginationInfo
  pageSizeOptions?: number[]
  className?: string
}

const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 50, 100, 200, 500, 1000]

/**
 * Unified pagination component
 * 
 * Updates pagination state through table.setPageIndex/setPageSize,
 * handled uniformly by useReactTable's onPaginationChange for state synchronization.
 */
export function DataTablePagination<TData>({
  table,
  paginationInfo,
  pageSizeOptions = DEFAULT_PAGE_SIZE_OPTIONS,
  className,
}: DataTablePaginationProps<TData>) {
  const t = useTranslations('common.pagination')
  const tDataTable = useTranslations('dataTable')
  const { pageIndex, pageSize } = table.getState().pagination
  
  // Server-side pagination mode
  const isServerSide = !!paginationInfo
  
  // Calculate total count and total pages
  const total = paginationInfo?.total ?? table.getFilteredRowModel().rows.length
  const totalPages = paginationInfo?.totalPages ?? table.getPageCount()
  const maxPageIndex = Math.max(0, totalPages - 1)
  const displayTotalPages = Math.max(1, totalPages)
  const selectedCount = table.getFilteredSelectedRowModel().rows.length

  // Use useCallback to wrap handler functions to avoid unnecessary re-renders
  const handlePageSizeChange = React.useCallback((value: string) => {
    const newPageSize = Number(value)
    table.setPageSize(newPageSize)
  }, [table])

  const handleFirstPage = React.useCallback(() => {
    table.setPageIndex(0)
  }, [table])

  const handlePreviousPage = React.useCallback(() => {
    if (isServerSide) {
      table.setPageIndex(Math.max(0, pageIndex - 1))
    } else {
      table.previousPage()
    }
  }, [table, isServerSide, pageIndex])

  const handleNextPage = React.useCallback(() => {
    if (isServerSide) {
      table.setPageIndex(Math.min(maxPageIndex, pageIndex + 1))
    } else {
      table.nextPage()
    }
  }, [table, isServerSide, pageIndex, maxPageIndex])

  const handleLastPage = React.useCallback(() => {
    table.setPageIndex(maxPageIndex)
  }, [table, maxPageIndex])

  // For server-side pagination use our calculated values, for client-side pagination use table methods
  const canPreviousPage = isServerSide ? pageIndex > 0 : table.getCanPreviousPage()
  const canNextPage = isServerSide ? pageIndex < maxPageIndex : table.getCanNextPage()

  return (
    <div className={cn("flex items-center justify-between px-2", className)}>
      {/* Selected rows info */}
      <div className="flex-1 text-sm text-muted-foreground">
        {tDataTable('selected', { count: selectedCount })} / {t('total', { count: total })}
      </div>

      {/* Pagination controls */}
      <div className="flex items-center space-x-6 lg:space-x-8">
        {/* Rows per page selection */}
        <div className="flex items-center space-x-2">
          <Label htmlFor="rows-per-page" className="text-sm font-medium">
            {t('rowsPerPage')}
          </Label>
          <Select
            value={`${pageSize}`}
            onValueChange={handlePageSizeChange}
          >
            <SelectTrigger className="h-8 w-[90px]" id="rows-per-page">
              <SelectValue placeholder={pageSize} />
            </SelectTrigger>
            <SelectContent side="top">
              {pageSizeOptions.map((size) => (
                <SelectItem key={size} value={`${size}`}>
                  {size}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Page info */}
        <div className="flex items-center justify-center text-sm font-medium whitespace-nowrap">
          {t('page', { current: pageIndex + 1, total: displayTotalPages })}
        </div>

        {/* Pagination buttons */}
        <div className="flex items-center space-x-2">
          <Button
            variant="outline"
            className="hidden h-8 w-8 p-0 lg:flex"
            onClick={handleFirstPage}
            disabled={!canPreviousPage}
          >
            <span className="sr-only">{t('first')}</span>
            <IconChevronsLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            className="h-8 w-8 p-0"
            onClick={handlePreviousPage}
            disabled={!canPreviousPage}
          >
            <span className="sr-only">{t('previous')}</span>
            <IconChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            className="h-8 w-8 p-0"
            onClick={handleNextPage}
            disabled={!canNextPage}
          >
            <span className="sr-only">{t('next')}</span>
            <IconChevronRight className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            className="hidden h-8 w-8 p-0 lg:flex"
            onClick={handleLastPage}
            disabled={!canNextPage}
          >
            <span className="sr-only">{t('last')}</span>
            <IconChevronsRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}
