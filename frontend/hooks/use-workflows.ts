import { useQuery } from '@tanstack/react-query'
import {
  getWorkflowProfiles,
  getWorkflowProfile,
  getWorkflows,
} from '@/services/workflow.service'

/**
 * Get workflow profile list (system-defined, read-only)
 */
export function useWorkflowProfiles() {
  return useQuery({
    queryKey: ['workflow-profiles'],
    queryFn: getWorkflowProfiles,
  })
}

/**
 * Get workflow profile by ID
 */
export function useWorkflowProfile(id: string) {
  return useQuery({
    queryKey: ['workflow-profiles', id],
    queryFn: () => getWorkflowProfile(id),
    enabled: !!id,
  })
}

/**
 * Get workflow catalog list (read-only)
 */
export function useWorkflows() {
  return useQuery({
    queryKey: ['workflows'],
    queryFn: getWorkflows,
  })
}
