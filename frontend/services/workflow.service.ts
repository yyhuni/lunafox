import apiClient from '@/lib/api-client'
import type { ScanWorkflow, WorkflowProfile } from '@/types/workflow.types'
import {
  USE_MOCK,
  mockDelay,
  getMockWorkflows,
  getMockWorkflowProfiles,
  getMockWorkflowProfileById,
} from '@/mock'

/**
 * Workflow catalog API service (read-only)
 */

export async function getWorkflowProfiles(): Promise<WorkflowProfile[]> {
  if (USE_MOCK) {
    await mockDelay()
    return getMockWorkflowProfiles()
  }
  const response = await apiClient.get('/workflows/profiles/')
  return response.data
}

export async function getWorkflowProfile(id: string): Promise<WorkflowProfile> {
  if (USE_MOCK) {
    await mockDelay()
    const profile = getMockWorkflowProfileById(id)
    if (!profile) throw new Error('Workflow profile not found')
    return profile
  }
  const response = await apiClient.get(`/workflows/profiles/${id}/`)
  return response.data
}

export async function getWorkflows(): Promise<ScanWorkflow[]> {
  if (USE_MOCK) {
    await mockDelay()
    return normalizeWorkflows(getMockWorkflows())
  }
  const response = await apiClient.get('/workflows/')
  return normalizeWorkflows(response.data.results || response.data)
}

type WorkflowPayload = Partial<ScanWorkflow> & {
  name?: string
}

function normalizeWorkflows(payload: WorkflowPayload[]): ScanWorkflow[] {
  return (payload || [])
    .filter((item): item is WorkflowPayload & { name: string } => typeof item?.name === 'string' && item.name.trim().length > 0)
    .map((item, index) => ({
      id: typeof item.id === 'number' ? item.id : index + 1,
      name: item.name.trim(),
      title: item.title,
      description: item.description,
      version: item.version,
      configuration: item.configuration,
      isValid: item.isValid,
      createdAt: item.createdAt,
      updatedAt: item.updatedAt,
    }))
}
