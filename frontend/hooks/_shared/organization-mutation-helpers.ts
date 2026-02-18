import type { QueryClient, QueryKey } from "@tanstack/react-query"
import type { Organization } from "@/types/organization.types"

export type OrganizationListCache = { organizations?: Organization[] } & Record<string, unknown>
export type OrganizationQuerySnapshot = Array<[QueryKey, OrganizationListCache | undefined]>

export const ORGANIZATION_QUERY_KEY = ["organizations"] as const
export const TARGET_QUERY_KEY = ["targets"] as const

export const getOrganizationDeleteToastId = (id: number): string => `delete-${id}`
export const ORGANIZATION_BATCH_DELETE_TOAST_ID = "batch-delete"

export const applyOrganizationOptimisticDelete = (
  queryClient: QueryClient,
  deletedIds: number[]
): OrganizationQuerySnapshot => {
  const previousData = queryClient.getQueriesData<OrganizationListCache>({
    queryKey: ORGANIZATION_QUERY_KEY,
  })

  queryClient.setQueriesData(
    { queryKey: ORGANIZATION_QUERY_KEY },
    (old: OrganizationListCache | undefined) => {
      if (!old?.organizations) return old
      return {
        ...old,
        organizations: old.organizations.filter(
          (org) => !deletedIds.includes(org.id)
        ),
      }
    }
  )

  return previousData
}

export const rollbackOrganizationQueries = (
  queryClient: QueryClient,
  previousData: OrganizationQuerySnapshot | undefined
): void => {
  if (!previousData) return
  previousData.forEach(([queryKey, data]) => {
    queryClient.setQueryData(queryKey, data)
  })
}

export const invalidateOrganizationTargets = async (
  queryClient: QueryClient
): Promise<void> => {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ORGANIZATION_QUERY_KEY }),
    queryClient.invalidateQueries({ queryKey: TARGET_QUERY_KEY }),
  ])
}
