"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { McpKeyService } from "@/services/mcp-key.service"
import type { McpKeyGeneration, McpKeyStatus } from "@/types/mcp-key.types"

export const mcpKeyKeys = {
  status: ["mcp-key", "status"] as const,
}

function toStatus(generation: McpKeyGeneration): McpKeyStatus {
  return {
    configured: generation.configured,
    createdAt: generation.createdAt,
    updatedAt: generation.updatedAt,
  }
}

export function useMcpKeyStatus(enabled: boolean) {
  return useQuery({
    queryKey: mcpKeyKeys.status,
    queryFn: () => McpKeyService.getStatus(),
    enabled,
  })
}

export function useGenerateMcpKey(onGenerated: (generation: McpKeyGeneration) => void) {
  const queryClient = useQueryClient()

  return useMutation<McpKeyStatus, unknown, void>({
    mutationFn: async () => {
      const generation = await McpKeyService.generate()
      onGenerated(generation)
      return toStatus(generation)
    },
    onSuccess: (status) => {
      queryClient.setQueryData(mcpKeyKeys.status, status)
    },
    retry: false,
  })
}
