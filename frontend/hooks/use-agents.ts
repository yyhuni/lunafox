/**
 * Agent management hooks
 */

import { useQuery } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { agentService } from '@/services/agent.service'
import { getErrorCode, getErrorResponseData } from '@/lib/response-parser'
import type { UpdateAgentConfigRequest } from '@/types/agent.types'

export const agentKeys = createResourceKeys("agents", {
  list: (page: number, pageSize: number, status?: string) => ({ page, pageSize, status }),
  detail: (id: number) => id,
})

export function useAgents(page = 1, pageSize = 10, status?: string) {
  return useQuery({
    queryKey: agentKeys.list(page, pageSize, status),
    queryFn: () => agentService.getAgents(page, pageSize, status),
    refetchInterval: 15000,
  })
}

export function useAgent(id: number) {
  return useQuery({
    queryKey: agentKeys.detail(id),
    queryFn: () => agentService.getAgent(id),
    enabled: id > 0,
  })
}

export function useCreateRegistrationToken() {
  return useResourceMutation<Awaited<ReturnType<typeof agentService.createRegistrationToken>>, void>({
    mutationFn: () => agentService.createRegistrationToken(),
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
      agentService.updateAgentConfig(id, data),
    invalidate: [
      { queryKey: agentKeys.lists() },
      ({ variables }) => ({ queryKey: agentKeys.detail(variables.id) }),
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
    invalidate: [{ queryKey: agentKeys.lists(), refetchType: 'active' }],
    onSuccess: ({ toast }) => {
      toast.success('toast.agent.delete.success')
    },
    errorFallbackKey: 'toast.agent.delete.error',
  })
}
