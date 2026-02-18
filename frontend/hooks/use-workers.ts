/**
 * Worker node management Hooks
 */

import { useQuery } from '@tanstack/react-query'
import {
  useResourceMutation,
  type UseResourceMutationOptions,
} from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { workerService } from '@/services/worker.service'
import type { CreateWorkerRequest, UpdateWorkerRequest } from '@/types/worker.types'

// Query Keys
export const workerKeys = createResourceKeys("workers", {
  list: (page: number, pageSize: number) => ({ page, pageSize }),
  detail: (id: number) => id,
})

type WorkerMutationConfig<TData, TVariables> = {
  mutationFn: UseResourceMutationOptions<TData, TVariables>['mutationFn']
  invalidate?: UseResourceMutationOptions<TData, TVariables>['invalidate']
  successKey: string
  errorFallbackKey: string
}

function useWorkerMutation<TData, TVariables>({
  mutationFn,
  invalidate,
  successKey,
  errorFallbackKey,
}: WorkerMutationConfig<TData, TVariables>) {
  return useResourceMutation<TData, TVariables>({
    mutationFn,
    invalidate,
    onSuccess: ({ toast }) => {
      toast.success(successKey)
    },
    errorFallbackKey,
  })
}

/**
 * Get Worker list
 */
export function useWorkers(page = 1, pageSize = 10) {
  return useQuery({
    queryKey: workerKeys.list(page, pageSize),
    queryFn: () => workerService.getWorkers(page, pageSize),
  })
}

/**
 * Get individual Worker details
 */
export function useWorker(id: number) {
  return useQuery({
    queryKey: workerKeys.detail(id),
    queryFn: () => workerService.getWorker(id),
    enabled: id > 0,
  })
}

/**
 * Create Worker
 */
export function useCreateWorker() {
  return useWorkerMutation({
    mutationFn: (data: CreateWorkerRequest) => workerService.createWorker(data),
    invalidate: [{ queryKey: workerKeys.lists() }],
    successKey: 'toast.worker.create.success',
    errorFallbackKey: 'toast.worker.create.error',
  })
}

/**
 * Update Worker
 */
export function useUpdateWorker() {
  return useWorkerMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateWorkerRequest }) =>
      workerService.updateWorker(id, data),
    invalidate: [
      { queryKey: workerKeys.lists() },
      ({ variables }) => ({ queryKey: workerKeys.detail(variables.id) }),
    ],
    successKey: 'toast.worker.update.success',
    errorFallbackKey: 'toast.worker.update.error',
  })
}

/**
 * Delete Worker
 */
export function useDeleteWorker() {
  return useWorkerMutation({
    mutationFn: (id: number) => workerService.deleteWorker(id),
    invalidate: [
      {
        queryKey: workerKeys.lists(),
        refetchType: 'active',
      },
    ],
    successKey: 'toast.worker.delete.success',
    errorFallbackKey: 'toast.worker.delete.error',
  })
}

/**
 * Deploy Worker
 */
export function useDeployWorker() {
  return useWorkerMutation({
    mutationFn: (id: number) => workerService.deployWorker(id),
    invalidate: [
      ({ variables }) => ({ queryKey: workerKeys.detail(variables) }),
      { queryKey: workerKeys.lists() },
    ],
    successKey: 'toast.worker.deploy.success',
    errorFallbackKey: 'toast.worker.deploy.error',
  })
}

/**
 * Restart Worker
 */
export function useRestartWorker() {
  return useWorkerMutation({
    mutationFn: (id: number) => workerService.restartWorker(id),
    invalidate: [
      ({ variables }) => ({ queryKey: workerKeys.detail(variables) }),
      { queryKey: workerKeys.lists() },
    ],
    successKey: 'toast.worker.restart.success',
    errorFallbackKey: 'toast.worker.restart.error',
  })
}

/**
 * Stop Worker
 */
export function useStopWorker() {
  return useWorkerMutation({
    mutationFn: (id: number) => workerService.stopWorker(id),
    invalidate: [
      ({ variables }) => ({ queryKey: workerKeys.detail(variables) }),
      { queryKey: workerKeys.lists() },
    ],
    successKey: 'toast.worker.stop.success',
    errorFallbackKey: 'toast.worker.stop.error',
  })
}
