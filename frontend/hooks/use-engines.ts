import { useQuery } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import {
  getPresetEngines,
  getPresetEngine,
  getEngines,
  getEngine,
  createEngine,
  updateEngine,
  deleteEngine,
} from '@/services/engine.service'

/**
 * Get preset engine list (system-defined, read-only)
 */
export function usePresetEngines() {
  return useQuery({
    queryKey: ['preset-engines'],
    queryFn: getPresetEngines,
  })
}

/**
 * Get preset engine by ID
 */
export function usePresetEngine(id: string) {
  return useQuery({
    queryKey: ['preset-engines', id],
    queryFn: () => getPresetEngine(id),
    enabled: !!id,
  })
}

/**
 * Get user engine list (stored in database, editable)
 */
export function useEngines() {
  return useQuery({
    queryKey: ['engines'],
    queryFn: getEngines,
  })
}

/**
 * Get engine details
 */
export function useEngine(id: number) {
  return useQuery({
    queryKey: ['engines', id],
    queryFn: () => getEngine(id),
    enabled: !!id,
  })
}

/**
 * Create engine
 */
export function useCreateEngine() {
  return useResourceMutation({
    mutationFn: createEngine,
    invalidate: [{ queryKey: ['engines'] }],
    onSuccess: ({ toast }) => {
      toast.success('toast.engine.create.success')
    },
    errorFallbackKey: 'toast.engine.create.error',
  })
}

/**
 * Update engine
 */
export function useUpdateEngine() {
  return useResourceMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof updateEngine>[1] }) =>
      updateEngine(id, data),
    invalidate: [
      { queryKey: ['engines'] },
      ({ variables }) => ({ queryKey: ['engines', variables.id] }),
    ],
    onSuccess: ({ toast }) => {
      toast.success('toast.engine.update.success')
    },
    errorFallbackKey: 'toast.engine.update.error',
  })
}

/**
 * Delete engine
 */
export function useDeleteEngine() {
  return useResourceMutation({
    mutationFn: deleteEngine,
    invalidate: [{ queryKey: ['engines'] }],
    onSuccess: ({ toast }) => {
      toast.success('toast.engine.delete.success')
    },
    errorFallbackKey: 'toast.engine.delete.error',
  })
}
