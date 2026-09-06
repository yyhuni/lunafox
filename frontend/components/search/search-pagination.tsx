"use client"

import * as React from "react"
import { SharedCompactPagination } from "@/components/shared/data-table/pagination"

interface SearchPaginationProps {
  pageSize: number
  canFirstPage: boolean
  canPreviousPage: boolean
  canNextPage: boolean
  onPreviousPage: () => void
  onNextPage: () => void
  onFirstPage: () => void
  onPageSizeChange: (pageSize: number) => void
  pageSizeOptions?: number[]
  className?: React.ComponentProps<"div">["className"]
}

const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

export function SearchPagination({
  pageSize,
  canFirstPage,
  canPreviousPage,
  canNextPage,
  onPreviousPage,
  onNextPage,
  onFirstPage,
  onPageSizeChange,
  pageSizeOptions = DEFAULT_PAGE_SIZE_OPTIONS,
  className,
}: SearchPaginationProps) {
  return (
    <SharedCompactPagination
      mode="cursor"
      pageSize={pageSize}
      canFirstPage={canFirstPage}
      canPreviousPage={canPreviousPage}
      canNextPage={canNextPage}
      onFirstPage={onFirstPage}
      onPreviousPage={onPreviousPage}
      onNextPage={onNextPage}
      onPageSizeChange={onPageSizeChange}
      pageSizeOptions={pageSizeOptions}
      className={className}
    />
  )
}
