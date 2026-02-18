import { useQuery } from "@tanstack/react-query"
import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import type { PaginatedResponse } from "@/types/api-response.types"

export type FingerprintListParams = { page?: number; pageSize?: number; filter?: string }
export type QueryOptions = { enabled?: boolean }
export type FingerprintRecord = { id: number; createdAt: string }

export type FingerprintCreatePayload<T extends FingerprintRecord> = Omit<T, "id" | "createdAt">
export type FingerprintUpdatePayload<T extends FingerprintRecord> = {
  id: number
  data: Partial<T>
}

export type FingerprintKeyGroup = {
  all: () => readonly unknown[]
  list: (params: FingerprintListParams) => readonly unknown[]
  detail: (id: number) => readonly unknown[]
}

export type FingerprintCrudService<
  T extends FingerprintRecord,
  TImport = unknown,
  TBulk = unknown
> = {
  list: (params: FingerprintListParams) => Promise<PaginatedResponse<T>>
  detail: (id: number) => Promise<T>
  create: (data: FingerprintCreatePayload<T>) => Promise<T>
  update: (id: number, data: Partial<T>) => Promise<T>
  remove: (id: number) => Promise<void>
  importFromFile: (file: File) => Promise<TImport>
  bulkDelete: (ids: number[]) => Promise<TBulk>
  deleteAll: () => Promise<TBulk>
}

export type FingerprintHooksConfig<
  T extends FingerprintRecord,
  TImport = unknown,
  TBulk = unknown
> = {
  keys: FingerprintKeyGroup
  service: FingerprintCrudService<T, TImport, TBulk>
  statsKey: readonly unknown[]
}

export function createFingerprintHooks<
  T extends FingerprintRecord,
  TImport = unknown,
  TBulk = unknown
>(config: FingerprintHooksConfig<T, TImport, TBulk>) {
  const useList = (params: FingerprintListParams = {}, options?: QueryOptions) =>
    useQuery({
      queryKey: config.keys.list(params),
      queryFn: () => config.service.list(params),
      ...options,
    })

  const useDetail = (id: number, options?: QueryOptions) =>
    useQuery({
      queryKey: config.keys.detail(id),
      queryFn: () => config.service.detail(id),
      enabled: id > 0 && options?.enabled !== false,
    })

  const useCreate = () =>
    useResourceMutation({
      mutationFn: (data: FingerprintCreatePayload<T>) => config.service.create(data),
      invalidate: [
        { queryKey: config.keys.all() },
        { queryKey: config.statsKey },
      ],
      skipDefaultErrorHandler: true,
    })

  const useUpdate = () =>
    useResourceMutation({
      mutationFn: ({ id, data }: FingerprintUpdatePayload<T>) =>
        config.service.update(id, data),
      invalidate: [
        { queryKey: config.keys.all() },
        ({ variables }) => ({ queryKey: config.keys.detail(variables.id) }),
      ],
      skipDefaultErrorHandler: true,
    })

  const useDelete = () =>
    useResourceMutation({
      mutationFn: (id: number) => config.service.remove(id),
      invalidate: [
        { queryKey: config.keys.all() },
        { queryKey: config.statsKey },
      ],
      skipDefaultErrorHandler: true,
    })

  const useImport = () =>
    useResourceMutation({
      mutationFn: (file: File) => config.service.importFromFile(file),
      invalidate: [
        { queryKey: config.keys.all() },
        { queryKey: config.statsKey },
      ],
      skipDefaultErrorHandler: true,
    })

  const useBulkDelete = () =>
    useResourceMutation({
      mutationFn: (ids: number[]) => config.service.bulkDelete(ids),
      invalidate: [
        { queryKey: config.keys.all() },
        { queryKey: config.statsKey },
      ],
      skipDefaultErrorHandler: true,
    })

  const useDeleteAll = () =>
    useResourceMutation<TBulk, void>({
      mutationFn: () => config.service.deleteAll(),
      invalidate: [
        { queryKey: config.keys.all() },
        { queryKey: config.statsKey },
      ],
      skipDefaultErrorHandler: true,
    })

  return {
    useList,
    useDetail,
    useCreate,
    useUpdate,
    useDelete,
    useImport,
    useBulkDelete,
    useDeleteAll,
  }
}
