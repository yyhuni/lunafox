import { useQuery, keepPreviousData, type UseQueryResult } from "@tanstack/react-query"
import {
  resolveTargetsQueryInput,
  selectTargetsResponse,
  type TargetSelectResponse,
  type UseTargetsOptions,
  type UseTargetsParams,
} from "@/hooks/_shared/targets-helpers"
import {
  getTargets,
  getTargetById,
  getTargetOrganizations,
  getTargetEndpoints,
} from "@/services/target.service"
import type { TargetsResponse } from "@/types/target.types"
import type { EndpointListQueryParams, GetEndpointsResponse } from "@/types/endpoint.types"
import type { WebsiteAssetScope } from "@/types/website.types"
import { targetKeys } from "./keys"

export type UseTargetsResult = UseQueryResult<TargetSelectResponse, Error>

/**
 * Get a list of all targets
 * Canonical target collection controls are pageSize/pageToken/filter/orderBy.
 */
export function useTargets(
  params?: UseTargetsParams,
  options?: UseTargetsOptions
): UseTargetsResult

export function useTargets(
  params: UseTargetsParams = {},
  options: UseTargetsOptions = {}
): UseTargetsResult {
  const resolved = resolveTargetsQueryInput(params, options)

  return useQuery<TargetsResponse, Error, TargetSelectResponse>({
    queryKey: targetKeys.list({
      pageSize: resolved.pageSize,
      pageToken: resolved.pageToken,
      filter: resolved.filter,
      orderBy: resolved.orderBy,
    }),
    queryFn: () => getTargets({
      pageSize: resolved.pageSize,
      pageToken: resolved.pageToken,
      filter: resolved.filter,
      orderBy: resolved.orderBy,
    }),
    enabled: resolved.enabled,
    select: (response) => selectTargetsResponse(response, {
      pageSize: resolved.pageSize,
    }),
    placeholderData: keepPreviousData,
  })
}

/**
 * Get individual target details
 */
export function useTarget(id: number, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: targetKeys.detail(id),
    queryFn: () => getTargetById(id),
    enabled: options?.enabled !== undefined ? options.enabled : !!id,
  })
}

/**
 * Get the target's list of organizations
 */
export function useTargetOrganizations(targetId: number, page = 1, pageSize = 10) {
  return useQuery({
    queryKey: targetKeys.organizations(targetId, page, pageSize),
    queryFn: () => getTargetOrganizations(targetId, page, pageSize),
    enabled: !!targetId,
  })
}

/**
 * Get a list of endpoints for a target
 */
export function useTargetEndpoints(
  targetId: number,
  params?: EndpointListQueryParams,
  options?: {
    enabled?: boolean
    websiteScope?: WebsiteAssetScope
  }
)
{
  const resolved = {
    pageSize: params?.pageSize ?? 10,
    pageToken: params?.pageToken,
    filter: params?.filter,
    orderBy: params?.orderBy,
  }

  return useQuery({
    queryKey: targetKeys.endpoints(targetId, resolved, options?.websiteScope),
    queryFn: () => getTargetEndpoints(targetId, resolved),
    enabled: options?.enabled !== undefined ? options.enabled : !!targetId,
    select: (response: GetEndpointsResponse) => response,
    placeholderData: keepPreviousData,
  })
}
