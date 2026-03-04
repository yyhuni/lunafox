import apiClient from '@/lib/api-client'
import type { ScanWorkflow, PresetWorkflow } from '@/types/workflow.types'
import { USE_MOCK, mockDelay, getMockWorkflows, getMockWorkflowById, getMockPresetWorkflows, getMockPresetWorkflowById } from '@/mock'

/**
 * Workflow catalog API service
 */

/**
 * Get preset workflow list (system-defined, read-only)
 */
export async function getPresetWorkflows(): Promise<PresetWorkflow[]> {
  if (USE_MOCK) {
    await mockDelay()
    return getMockPresetWorkflows()
  }
  const response = await apiClient.get('/workflows/presets/')
  return response.data
}

/**
 * Get preset workflow by ID
 */
export async function getPresetWorkflow(id: string): Promise<PresetWorkflow> {
	if (USE_MOCK) {
		await mockDelay()
		const preset = getMockPresetWorkflowById(id)
		if (!preset) throw new Error('Preset workflow not found')
		return preset
	}
  const response = await apiClient.get(`/workflows/presets/${id}/`)
  return response.data
}

/**
 * Get user workflow template list (stored in database, editable)
 */
export async function getWorkflows(): Promise<ScanWorkflow[]> {
  if (USE_MOCK) {
    await mockDelay()
    return getMockWorkflows()
  }
  // Workflow template entries are usually limited; fetch all in one page
  const response = await apiClient.get('/workflows/', {
    params: { pageSize: 1000 }
  })
  // Backend returns paginated data: { results: [...], total, page, pageSize, totalPages }
  return response.data.results || response.data
}

/**
 * Get user workflow template details
 */
export async function getWorkflow(id: number): Promise<ScanWorkflow> {
  if (USE_MOCK) {
    await mockDelay()
    const workflow = getMockWorkflowById(id)
    if (!workflow) throw new Error('Workflow not found')
    return workflow
  }
  const response = await apiClient.get(`/workflows/${id}/`)
  return response.data
}

/**
 * Create user workflow template
 */
export async function createWorkflow(data: {
  name: string
  configuration: string
}): Promise<ScanWorkflow> {
  if (USE_MOCK) {
    await mockDelay()
    // Mock create - return a new workflow template with generated ID
    const newWorkflow: ScanWorkflow = {
      id: Date.now(),
      name: data.name,
      configuration: data.configuration,
      isValid: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    }
    return newWorkflow
  }
  const response = await apiClient.post('/workflows/', data)
  return response.data
}

/**
 * Update user workflow template
 */
export async function updateWorkflow(
  id: number,
  data: Partial<{
    name: string
    configuration: string
  }>
): Promise<ScanWorkflow> {
  const response = await apiClient.patch(`/workflows/${id}/`, data)
  return response.data
}

/**
 * Delete user workflow template
 */
export async function deleteWorkflow(id: number): Promise<void> {
  await apiClient.delete(`/workflows/${id}/`)
}
