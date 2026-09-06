import type { Target, TargetsResponse } from "@/types/target.types"

export type UseTargetsParams = {
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
}

export type UseTargetsOptions = { enabled?: boolean }

export type TargetSelectResponse = TargetsResponse & {
  targets: Target[]
  count: number
}

export type ResolvedTargetsQuery = {
  pageSize: number
  pageToken?: string
  filter?: string
  orderBy?: string
  enabled: boolean
}

export const resolveTargetsQueryInput = (
  params: UseTargetsParams = {},
  options: UseTargetsOptions = {}
): ResolvedTargetsQuery => {
  return {
    pageSize: params.pageSize ?? 10,
    pageToken: params.pageToken,
    filter: params.filter,
    orderBy: params.orderBy,
    enabled: options.enabled !== false,
  }
}

export const selectTargetsResponse = (
  response: TargetsResponse,
  params: { pageSize: number }
): TargetSelectResponse => {
  const { pageSize } = params
  const total = response.totalSize ?? response.total ?? 0

  return {
    ...response,
    targets: response.results,
    count: total,
    total,
    totalSize: total,
    page: response.page ?? 1,
    pageSize: response.pageSize ?? pageSize,
    totalPages: response.totalPages ?? Math.max(1, Math.ceil(total / pageSize)),
  }
}
