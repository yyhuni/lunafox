import { useQuery } from "@tanstack/react-query"
import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { getAssetDeletedCount } from "@/hooks/_shared/asset-mutation-helpers"
import { CommandService } from "@/services/command.service"
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
 * Get command list
 */
export function useCommands(params: GetCommandsRequest = {}) {
  return useQuery({
    queryKey: commandKeys.list(params),
    queryFn: (): Promise<GetCommandsResponse> =>
      CommandService.getCommands(params),
  })
}

/**
 * Get single command
 */
export function useCommand(id: number) {
  return useQuery({
    queryKey: commandKeys.detail(id),
    queryFn: async (): Promise<Command | undefined> =>
      (await CommandService.getCommandById(id)).command,
    enabled: !!id,
  })
}

/**
 * Create command
 */
export function useCreateCommand() {
  return useResourceMutation({
    mutationFn: (data: CreateCommandRequest) => CommandService.createCommand(data),
    loadingToast: {
      key: 'common.status.creating',
      params: {},
      id: 'create-command',
    },
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
    loadingToast: {
      key: 'common.status.updating',
      params: {},
      id: ({ id }) => `update-command-${id}`,
    },
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
    loadingToast: {
      key: 'common.status.deleting',
      params: {},
      id: (id) => `delete-command-${id}`,
    },
    invalidate: [{ queryKey: commandKeys.all }],
    onSuccess: ({ toast }) => {
      toast.success('toast.command.delete.success')
    },
    errorFallbackKey: 'toast.command.delete.error',
  })
}

/**
 * Batch delete commands
 */
export function useBatchDeleteCommands() {
  return useResourceMutation({
    mutationFn: (ids: number[]) => CommandService.batchDeleteCommands(ids),
    loadingToast: {
      key: 'common.status.batchDeleting',
      params: {},
      id: 'batch-delete-commands',
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
