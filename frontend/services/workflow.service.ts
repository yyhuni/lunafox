import apiClient from '@/lib/api-client'
import { normalizeWorkflowConfiguration } from '@/lib/workflow-config'
import type { ScanWorkflow, WorkflowProfile } from '@/types/workflow.types'
import {
  USE_MOCK,
  mockDelay,
  getMockWorkflows,
  getMockWorkflowProfiles,
  getMockWorkflowProfileById,
} from '@/mock'

export async function getWorkflowProfiles(): Promise<WorkflowProfile[]> {
  if (USE_MOCK) {
    await mockDelay()
    return normalizeProfiles(getMockWorkflowProfiles())
  }
  const response = await apiClient.get('/workflows/profiles/')
  return normalizeProfiles(response.data)
}

export async function getWorkflowProfile(id: string): Promise<WorkflowProfile> {
  if (USE_MOCK) {
    await mockDelay()
    const profile = getMockWorkflowProfileById(id)
    if (!profile) throw new Error('Workflow profile not found')
    return normalizeProfile(profile)
  }
  const response = await apiClient.get(`/workflows/profiles/${id}/`)
  return normalizeProfile(response.data)
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
  workflowId?: string
  title?: string
  displayName?: string
  configuration?: unknown
}

type WorkflowProfilePayload = Partial<WorkflowProfile> & {
  workflowIds?: string[]
  workflowNames?: string[]
  configuration?: unknown
}

function normalizeWorkflows(payload: WorkflowPayload[]): ScanWorkflow[] {
  return (payload || []).reduce<ScanWorkflow[]>((result, item, index) => {
    const name = typeof item?.name === 'string' && item.name.trim().length > 0
      ? item.name.trim()
      : typeof item?.workflowId === 'string' && item.workflowId.trim().length > 0
        ? item.workflowId.trim()
        : ''
    if (!name) {
      return result
    }
    result.push({
      id: typeof item.id === 'number' ? item.id : index + 1,
      name,
      title: item.title ?? item.displayName,
      description: item.description,
      version: item.version,
      configuration: normalizeWorkflowConfiguration(item.configuration),
      isValid: item.isValid,
      createdAt: item.createdAt,
      updatedAt: item.updatedAt,
    })
    return result
  }, [])
}

function normalizeProfiles(payload: WorkflowProfilePayload[]): WorkflowProfile[] {
  return (payload || []).map(normalizeProfile)
}

function normalizeProfile(item: WorkflowProfilePayload): WorkflowProfile {
  return {
    id: item.id || '',
    name: item.name || '',
    description: item.description,
    workflowIds: (item.workflowIds || item.workflowNames || []).filter(Boolean),
    workflowNames: (item.workflowNames || item.workflowIds || []).filter(Boolean),
    configuration: normalizeWorkflowConfiguration(item.configuration),
  }
}
