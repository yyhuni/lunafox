/**
 * Distributed management hooks
 */

import { keepPreviousData, useQuery, type UseQueryResult } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { agentService } from '@/services/agent.service'
import { getErrorCode, getErrorResponseData } from '@/lib/response-parser'
import { agentName } from '@/lib/resource-name'
import type {
  AgentFilterOptionField,
  AgentListQueryParams,
  RegistrationTokenResponse,
  UpdateAgentConfigRequest,
} from '@/types/agent.types'

type RegistrationTokenIdentity = Pick<RegistrationTokenResponse, 'resourceName'>
type RetainedQueryOptions = {
  enabled?: boolean
  refetchInterval?: number | false
}

const agentKeyBase = createResourceKeys("agentNodes", {
  list: (params: AgentListQueryParams) => params,
  detail: (resourceName: string) => resourceName,
})

export const agentKeys = {
  ...agentKeyBase,
  scanPicker: (normalizedSearch: string) =>
    [...agentKeyBase.lists(), "scanPicker", normalizedSearch] as const,
  filterOptions: (field: AgentFilterOptionField) => [...agentKeyBase.all, "filterOptions", field] as const,
  clusterSummary: () => [...agentKeyBase.all, "clusterSummary"] as const,
  locationMap: () => [...agentKeyBase.all, "locationMap"] as const,
  registrationToken: (identity: RegistrationTokenIdentity) =>
    [...agentKeyBase.all, "registrationToken", identity.resourceName] as const,
}

function withRetainedQueryState<TData, TError>(query: UseQueryResult<TData, TError>) {
  const hasSuccessfulData = query.data !== undefined && query.dataUpdatedAt > 0
  return {
    ...query,
    isInitialError: query.isError && !hasSuccessfulData,
    isRefetchStale: query.isError && hasSuccessfulData,
    lastSuccessfulAt: hasSuccessfulData ? query.dataUpdatedAt : null,
  }
}

export function useAgents(params: AgentListQueryParams = {}, options?: { enabled?: boolean }) {
  const resolved = {
    pageSize: params.pageSize ?? 10,
    pageToken: params.pageToken,
    filter: params.filter,
    orderBy: params.orderBy,
  }
  return useQuery({
    queryKey: agentKeys.list(resolved),
    queryFn: () => agentService.getAgents(resolved),
    enabled: options?.enabled ?? true,
    refetchInterval: options?.enabled === false ? false : 15000,
    placeholderData: keepPreviousData,
  })
}

export function useAgentFilterOptions(field: AgentFilterOptionField) {
  return useQuery({
    queryKey: agentKeys.filterOptions(field),
    queryFn: () => agentService.getAgentFilterOptions(field),
    placeholderData: keepPreviousData,
  })
}

export function useAgent(id: number) {
  const resourceName = agentName(id)
  return useQuery({
    queryKey: agentKeys.detail(resourceName),
    queryFn: ({ signal }) => agentService.getAgent(resourceName, signal),
    enabled: id > 0,
  })
}

export function useSelectedAgentDetail(resourceName: string | null | undefined, options?: { enabled?: boolean }) {
  const selectedResourceName = resourceName ?? ''
  const query = useQuery({
    queryKey: agentKeys.detail(selectedResourceName),
    queryFn: ({ signal }) => agentService.getAgent(selectedResourceName, signal),
    enabled: Boolean(selectedResourceName) && (options?.enabled ?? true),
  })
  return withRetainedQueryState(query)
}

export function useAgentClusterSummary(options?: RetainedQueryOptions) {
  const query = useQuery({
    queryKey: agentKeys.clusterSummary(),
    queryFn: ({ signal }) => agentService.getAgentClusterSummary(signal),
    enabled: options?.enabled ?? true,
    refetchInterval: options?.refetchInterval ?? false,
  })
  return withRetainedQueryState(query)
}

export function useAgentLocationMap(options?: RetainedQueryOptions) {
  const query = useQuery({
    queryKey: agentKeys.locationMap(),
    queryFn: ({ signal }) => agentService.getAgentLocationMap(signal),
    enabled: options?.enabled ?? true,
    refetchInterval: options?.refetchInterval ?? false,
  })
  return withRetainedQueryState(query)
}

export function useRegistrationToken(
  identity: RegistrationTokenIdentity | null | undefined,
  options?: RetainedQueryOptions,
) {
  const safeIdentity = identity ?? { resourceName: '' }
  const query = useQuery({
    queryKey: agentKeys.registrationToken(safeIdentity),
    queryFn: ({ signal }) => agentService.getRegistrationToken(safeIdentity.resourceName, signal),
    enabled: Boolean(identity?.resourceName) && (options?.enabled ?? true),
    refetchInterval: options?.refetchInterval ?? false,
    retry: false,
  })
  return withRetainedQueryState(query)
}

export function useCreateRegistrationToken() {
  return useResourceMutation<Awaited<ReturnType<typeof agentService.createRegistrationToken>>, void>({
    mutationFn: () => agentService.createRegistrationToken(),
    loadingToast: {
      key: 'common.status.creating',
      params: {},
      id: 'agent-token',
    },
    onSuccess: ({ toast }) => {
      toast.success('toast.agent.token.success', {}, 'agent-token')
    },
    onError: ({ error, toast }) => {
      toast.errorFromCode(
        getErrorCode(getErrorResponseData(error)),
        'toast.agent.token.error',
        'agent-token'
      )
    },
  })
}

export function useUpdateAgentConfig() {
  return useResourceMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateAgentConfigRequest }) =>
      agentService.updateDistributedConfig(id, data),
    loadingToast: {
      key: 'common.status.updating',
      params: {},
      id: ({ id }) => `update-agent-config-${id}`,
    },
    invalidate: [
      { queryKey: agentKeys.lists() },
      ({ variables }) => ({ queryKey: agentKeys.detail(agentName(variables.id)) }),
      { queryKey: agentKeys.clusterSummary() },
    ],
    onSuccess: ({ toast }) => {
      toast.success('toast.agent.config.success')
    },
    errorFallbackKey: 'toast.agent.config.error',
  })
}


export function useDeleteAgent() {
  return useResourceMutation({
    mutationFn: (id: number) => agentService.deleteAgent(id),
    loadingToast: {
      key: 'common.status.deleting',
      params: {},
      id: (id) => `delete-agent-node-${id}`,
    },
    invalidate: [
      { queryKey: agentKeys.lists(), refetchType: 'active' },
      { queryKey: agentKeys.clusterSummary(), refetchType: 'active' },
      { queryKey: agentKeys.locationMap(), refetchType: 'active' },
    ],
    onSuccess: ({ toast }) => {
      toast.success('toast.agent.delete.success')
    },
    errorFallbackKey: 'toast.agent.delete.error',
  })
}
