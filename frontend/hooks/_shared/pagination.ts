import type { PaginationInfo } from "@/types/common.types"

export type PaginationResponse = {
  total?: number
  page?: number
  pageSize?: number
  totalPages?: number
  page_size?: number
  total_pages?: number
}

export const DEFAULT_PAGE = 1
export const DEFAULT_PAGE_SIZE = 10

export const normalizePagination = (
  response?: PaginationResponse | null,
  fallbackPage: number = DEFAULT_PAGE,
  fallbackPageSize: number = DEFAULT_PAGE_SIZE
): PaginationInfo => {
  const total = response?.total ?? 0
  const page = response?.page ?? fallbackPage ?? DEFAULT_PAGE
  const pageSize =
    response?.pageSize ??
    response?.page_size ??
    fallbackPageSize ??
    DEFAULT_PAGE_SIZE
  const totalPages =
    response?.totalPages ??
    response?.total_pages ??
    0

  return {
    total,
    page,
    pageSize,
    totalPages,
  }
}

type PaginationInfoInput = {
  total: number
  page: number
  pageSize: number
  totalPages?: number
  minTotalPages?: number
}

export const buildPaginationInfo = ({
  total,
  page,
  pageSize,
  totalPages,
  minTotalPages = 0,
}: PaginationInfoInput): PaginationInfo => {
  const calculatedTotalPages = totalPages ?? Math.ceil(total / pageSize)
  return {
    total,
    page,
    pageSize,
    totalPages: Math.max(minTotalPages, calculatedTotalPages),
  }
}
