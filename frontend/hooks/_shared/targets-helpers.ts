import type { Target, TargetsResponse } from "@/types/target.types"

export type UseTargetsParams = {
  page?: number
  pageSize?: number
  organizationId?: number
  filter?: string
}

export type UseTargetsOptions = { enabled?: boolean }

export type TargetSelectResponse = TargetsResponse & {
  targets: Target[]
  count: number
}

export type ResolvedTargetsQuery = {
  page: number
  pageSize: number
  organizationId?: number
  filter?: string
  type?: string
  enabled: boolean
}

export const resolveTargetsQueryInput = (
  pageOrParams: number | UseTargetsParams = 1,
  pageSizeOrOptions: number | UseTargetsOptions = 10,
  type?: string,
  filter?: string
): ResolvedTargetsQuery => {
  const isObjectParams = typeof pageOrParams === "object"
  const objectParams = isObjectParams ? pageOrParams : undefined

  const page = objectParams?.page ?? (pageOrParams as number)
  const pageSize = isObjectParams
    ? (objectParams?.pageSize ?? 10)
    : (typeof pageSizeOrOptions === "number" ? pageSizeOrOptions : 10)
  const organizationId = objectParams?.organizationId
  const resolvedFilter = objectParams?.filter ?? filter
  const resolvedType = isObjectParams ? undefined : type
  const enabled = isObjectParams && typeof pageSizeOrOptions === "object"
    ? pageSizeOrOptions.enabled !== false
    : true

  return {
    page,
    pageSize,
    organizationId,
    filter: resolvedFilter,
    type: resolvedType,
    enabled,
  }
}

export const selectTargetsResponse = (
  response: TargetsResponse,
  params: { page: number; pageSize: number; organizationId?: number }
): TargetSelectResponse => {
  const { organizationId, page, pageSize } = params

  if (organizationId) {
    const filteredResults = response.results.filter((target) =>
      target.organizations?.some((org) => org.id === organizationId)
    )
    return {
      ...response,
      results: filteredResults,
      total: filteredResults.length,
      count: filteredResults.length,
      targets: filteredResults,
      page,
      pageSize,
      totalPages: Math.ceil(filteredResults.length / pageSize),
    }
  }

  return {
    ...response,
    targets: response.results,
    count: response.total,
    total: response.total,
    page: response.page,
    pageSize: response.pageSize,
    totalPages: response.totalPages,
  }
}
