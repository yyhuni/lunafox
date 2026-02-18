"use client"

import { useQuery } from "@tanstack/react-query"
import {
  useResourceMutation,
  type UseResourceMutationOptions,
} from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { ToolService } from "@/services/tool.service"
import type { GetToolsParams, CreateToolRequest, UpdateToolRequest } from "@/types/tool.types"

// Query Keys
export const toolKeys = createResourceKeys("tools", {
  list: (params: GetToolsParams) => params,
})

type ToolMutationConfig<TData, TVariables> = {
  mutationFn: UseResourceMutationOptions<TData, TVariables>['mutationFn']
  loadingToast: UseResourceMutationOptions<TData, TVariables>['loadingToast']
  successKey: string
  errorFallbackKey: string
}

function useToolMutation<TData, TVariables>({
  mutationFn,
  loadingToast,
  successKey,
  errorFallbackKey,
}: ToolMutationConfig<TData, TVariables>) {
  return useResourceMutation<TData, TVariables>({
    mutationFn,
    loadingToast,
    invalidate: [{ queryKey: toolKeys.all, refetchType: 'active' }],
    onSuccess: ({ toast }) => {
      toast.success(successKey)
    },
    errorFallbackKey,
  })
}

// Get a list of tools
export function useTools(params: GetToolsParams = {}) {
  return useQuery({
    queryKey: toolKeys.list(params),
    queryFn: () => ToolService.getTools(params),
    select: (response) => {
      // REST-style response: return data directly.
      return {
        tools: response.tools || [],
        pagination: {
          total: response.total || 0,
          page: response.page || 1,
          pageSize: response.pageSize || 10,
          totalPages: response.totalPages || 0,
        }
      }
    },
  })
}

// Create tools
export function useCreateTool() {
  return useToolMutation({
    mutationFn: (data: CreateToolRequest) => ToolService.createTool(data),
    loadingToast: {
      key: 'common.status.creating',
      params: {},
      id: 'create-tool',
    },
    successKey: 'toast.tool.create.success',
    errorFallbackKey: 'toast.tool.create.error',
  })
}

// Update tool
export function useUpdateTool() {
  return useToolMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateToolRequest }) => 
      ToolService.updateTool(id, data),
    loadingToast: {
      key: 'common.status.updating',
      params: {},
      id: 'update-tool',
    },
    successKey: 'toast.tool.update.success',
    errorFallbackKey: 'toast.tool.update.error',
  })
}

// removal tool
export function useDeleteTool() {
  return useToolMutation({
    mutationFn: (id: number) => ToolService.deleteTool(id),
    loadingToast: {
      key: 'common.status.deleting',
      params: {},
      id: 'delete-tool',
    },
    successKey: 'toast.tool.delete.success',
    errorFallbackKey: 'toast.tool.delete.error',
  })
}
