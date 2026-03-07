import { useMemo } from "react"
import {
  DEFAULT_PAGE,
  DEFAULT_PAGE_SIZE,
  normalizePagination,
  type PaginationResponse,
} from "@/hooks/_shared/pagination"
import type { PaginationInfo } from "@/types/common.types"

type StablePaginationOptions = {
  fallbackPage?: number
  fallbackPageSize?: number
  fallbackTotalPages?: number
}

export const useStablePaginationInfo = (
  response?: PaginationResponse | null,
  {
    fallbackPage = DEFAULT_PAGE,
    fallbackPageSize = DEFAULT_PAGE_SIZE,
    fallbackTotalPages,
  }: StablePaginationOptions = {}
): PaginationInfo => {
  const normalized = normalizePagination(response, fallbackPage, fallbackPageSize)
  const resolvedTotalPages =
    response?.totalPages ??
    fallbackTotalPages ??
    normalized.totalPages

  return useMemo(
    () => ({
      total: normalized.total,
      page: normalized.page,
      pageSize: normalized.pageSize,
      totalPages: resolvedTotalPages,
    }),
    [normalized.total, normalized.page, normalized.pageSize, resolvedTotalPages]
  )
}
