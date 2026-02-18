import { useQuery } from "@tanstack/react-query"
import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { getAssetDeletedCount } from "@/hooks/_shared/asset-mutation-helpers"
import { CommandService } from "@/services/command.service"
import {
  getMockCommandById,
  getMockCommandDeleteCount,
  getMockCommands,
} from "@/mock/data/commands"
import type {
  GetCommandsRequest,
  CreateCommandRequest,
  UpdateCommandRequest,
  GetCommandsResponse,
  Command,
} from "@/types/command.types"


// Query Keys
export const commandKeys = createResourceKeys("commands", {
  list: (params: GetCommandsRequest = {}) => params,
  detail: (id: number) => id,
})

/**
 * Get command list (using mock data)
 */
export function useCommands(params: GetCommandsRequest = {}) {
  return useQuery({
    queryKey: commandKeys.list(params),
    queryFn: async (): Promise<GetCommandsResponse> => {
      // Simulate network delay
      await new Promise(resolve => setTimeout(resolve, 500))

      return getMockCommands(params)
    },
  })
}

/**
 * Get single command (using mock data)
 */
export function useCommand(id: number) {
  return useQuery({
    queryKey: commandKeys.detail(id),
    queryFn: async (): Promise<Command | undefined> => {
      // Simulate network delay
      await new Promise(resolve => setTimeout(resolve, 300))
      return getMockCommandById(id)
    },
    enabled: !!id,
  })
}

/**
 * Create command
 */
export function useCreateCommand() {
  return useResourceMutation({
    mutationFn: (data: CreateCommandRequest) => CommandService.createCommand(data),
    invalidate: [{ queryKey: commandKeys.all }],
    onSuccess: ({ toast }) => {
      toast.success('toast.command.create.success')
    },
    errorFallbackKey: 'toast.command.create.error',
  })
}

/**
 * Update command
 */
export function useUpdateCommand() {
  return useResourceMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateCommandRequest }) =>
      CommandService.updateCommand(id, data),
    invalidate: [
      { queryKey: commandKeys.all },
      { queryKey: commandKeys.details() },
    ],
    onSuccess: ({ toast }) => {
      toast.success('toast.command.update.success')
    },
    errorFallbackKey: 'toast.command.update.error',
  })
}

/**
 * Delete command
 */
export function useDeleteCommand() {
  return useResourceMutation({
    mutationFn: (id: number) => CommandService.deleteCommand(id),
    invalidate: [{ queryKey: commandKeys.all }],
    onSuccess: ({ toast }) => {
      toast.success('toast.command.delete.success')
    },
    errorFallbackKey: 'toast.command.delete.error',
  })
}

/**
 * Batch delete commands (using mock data)
 */
export function useBatchDeleteCommands() {
  return useResourceMutation({
    mutationFn: async (ids: number[]) => {
      // Simulate network delay
      await new Promise(resolve => setTimeout(resolve, 800))

      // Filter out deleted commands from mock data
      const deletedCount = getMockCommandDeleteCount(ids)

      // Simulate deletion (won't actually delete mock data)
      return {
        deletedCount: deletedCount
      }
    },
    invalidate: [{ queryKey: commandKeys.all }],
    onSuccess: ({ data: response, toast }) => {
      toast.success('toast.command.delete.bulkSuccess', {
        count: getAssetDeletedCount(response),
      })
    },
    errorFallbackKey: 'toast.command.delete.error',
  })
}
