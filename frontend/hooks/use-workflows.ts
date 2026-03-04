import { useQuery } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import {
  getPresetWorkflows,
  getPresetWorkflow,
  getWorkflows,
  getWorkflow,
  createWorkflow,
  updateWorkflow,
  deleteWorkflow,
} from '@/services/workflow.service'

/**
 * Get preset workflow list (system-defined, read-only)
 */
export function usePresetWorkflows() {
  return useQuery({
    queryKey: ['preset-workflows'],
    queryFn: getPresetWorkflows,
  })
}

/**
 * Get preset workflow by ID
 */
export function usePresetWorkflow(id: string) {
  return useQuery({
    queryKey: ['preset-workflows', id],
    queryFn: () => getPresetWorkflow(id),
    enabled: !!id,
  })
}

/**
 * Get user workflow template list (stored in database, editable)
 */
export function useWorkflows() {
  return useQuery({
    queryKey: ['workflows'],
    queryFn: getWorkflows,
  })
}

/**
 * Get workflow template details
 */
export function useWorkflow(id: number) {
  return useQuery({
    queryKey: ['workflows', id],
    queryFn: () => getWorkflow(id),
    enabled: !!id,
  })
}

/**
 * Create workflow template
 */
export function useCreateWorkflow() {
  return useResourceMutation({
    mutationFn: createWorkflow,
    invalidate: [{ queryKey: ['workflows'] }],
    onSuccess: ({ toast }) => {
      toast.success('toast.workflow.create.success')
    },
    errorFallbackKey: 'toast.workflow.create.error',
  })
}

/**
 * Update workflow template
 */
export function useUpdateWorkflow() {
  return useResourceMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof updateWorkflow>[1] }) =>
      updateWorkflow(id, data),
    invalidate: [
      { queryKey: ['workflows'] },
      ({ variables }) => ({ queryKey: ['workflows', variables.id] }),
    ],
    onSuccess: ({ toast }) => {
      toast.success('toast.workflow.update.success')
    },
    errorFallbackKey: 'toast.workflow.update.error',
  })
}

/**
 * Delete workflow template
 */
export function useDeleteWorkflow() {
  return useResourceMutation({
    mutationFn: deleteWorkflow,
    invalidate: [{ queryKey: ['workflows'] }],
    onSuccess: ({ toast }) => {
      toast.success('toast.workflow.delete.success')
    },
    errorFallbackKey: 'toast.workflow.delete.error',
  })
}
